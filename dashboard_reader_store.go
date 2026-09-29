package main

// Durable, privacy-reduced export data for the Operator Workspace reader.
// Data-plane callbacks only enqueue compact events; disk I/O happens on the
// reader worker. A dirty-start marker turns an interrupted queue into an
// explicit coverage gap instead of silently claiming complete history.

import (
	"bufio"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	dashboardContract       = "operator-workspace.integration.v1"
	dashboardSchemaVersion  = "1.0"
	dashboardPrefix         = "/api/integrations/dashboard/v1"
	dashboardQueueSize      = 2048
	dashboardMaxJournalSize = 64 << 20
	dashboardMaxSnapshots   = 16
	dashboardSnapshotTTL    = 25 * time.Hour
	dashboardHistoryDays    = 30
)

type dashboardRef struct {
	SourceProduct  string `json:"source_product"`
	SourceInstance string `json:"source_instance_id"`
	ResourceKind   string `json:"resource_kind"`
	ExternalID     string `json:"external_id"`
}

type dashboardProvenance struct {
	Origin         dashboardRef   `json:"origin"`
	OriginRevision string         `json:"origin_revision"`
	OriginEventID  *string        `json:"origin_event_id"`
	ForwardedBy    []dashboardRef `json:"forwarded_by"` // empty; kept as a typed empty array
	DerivedFrom    []dashboardRef `json:"derived_from"`
	EvidenceRefs   []any          `json:"evidence_refs"`
	DetailPath     *string        `json:"detail_path"`
}

type dashboardRecord struct {
	ExternalID       string              `json:"external_id"`
	Revision         string              `json:"revision"`
	StreamSequence   string              `json:"stream_sequence"`
	Operation        string              `json:"operation"`
	SourceObservedAt string              `json:"source_observed_at"`
	SourceUpdatedAt  *string             `json:"source_updated_at"`
	Provenance       dashboardProvenance `json:"provenance"`
	Payload          map[string]any      `json:"payload"`
}

type dashboardJournalEntry struct {
	Kind   string          `json:"kind"`
	Record dashboardRecord `json:"record"`
}

type dashboardSnapshot struct {
	Kind      string            `json:"kind"`
	Principal string            `json:"principal"`
	CreatedAt time.Time         `json:"created_at"`
	Watermark uint64            `json:"watermark"`
	GapEpoch  uint64            `json:"gap_epoch"`
	Records   []dashboardRecord `json:"records"`
}

type dashboardStore struct {
	mu        sync.Mutex
	queueMu   sync.RWMutex
	dir       string
	instance  string
	journal   *os.File
	key       []byte
	state     map[string]map[string]dashboardRecord
	history   []dashboardRecord
	sequences map[string]uint64
	snapshots map[string]dashboardSnapshot
	gap       atomic.Bool
	gapEpoch  atomic.Uint64
	queue     chan dashboardDetection
	worker    sync.WaitGroup
	closed    atomic.Bool
}

type dashboardDetection struct {
	Site     string
	RuleID   int
	Severity string
	At       time.Time
}

func dashboardUTC(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func dashboardRandomID() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func dashboardStableID(prefix, native string) string {
	sum := sha256.Sum256([]byte(prefix + "\x00" + native))
	return prefix + "-" + fmt.Sprintf("%x", sum[:12])
}

func dashboardWriteAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".dashboard-reader-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func dashboardPrivatePath(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || directory && !info.IsDir() || !directory && !info.Mode().IsRegular() {
		return fmt.Errorf("dashboard reader path has unsafe type: %s", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("dashboard reader path is accessible outside its owner: %s", path)
	}
	return nil
}

func openDashboardStore(dir, instance string) (*dashboardStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := dashboardPrivatePath(dir, true); err != nil {
		return nil, err
	}
	s := &dashboardStore{
		dir: dir, instance: instance, state: make(map[string]map[string]dashboardRecord),
		sequences: make(map[string]uint64), snapshots: make(map[string]dashboardSnapshot),
		queue: make(chan dashboardDetection, dashboardQueueSize),
	}
	keyPath := filepath.Join(dir, "cursor.key")
	key, err := os.ReadFile(keyPath)
	if errors.Is(err, os.ErrNotExist) {
		if _, statErr := os.Stat(filepath.Join(dir, "journal.jsonl")); statErr == nil {
			return nil, errors.New("dashboard reader cursor key missing for existing journal")
		}
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		if err = dashboardWriteAtomic(keyPath, key); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if err := dashboardPrivatePath(keyPath, false); err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("invalid dashboard reader cursor key")
	}
	s.key = key
	journalPath := filepath.Join(dir, "journal.jsonl")
	if _, err := os.Lstat(journalPath); err == nil {
		if err := dashboardPrivatePath(journalPath, false); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f, err := os.OpenFile(journalPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	if err := dashboardPrivatePath(journalPath, false); err != nil {
		f.Close()
		return nil, err
	}
	s.journal = f
	if err := s.loadJournal(); err != nil {
		f.Close()
		return nil, err
	}
	if err := s.loadSnapshots(); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(dir, "gap")); err == nil {
		s.markGap()
	} else if !errors.Is(err, os.ErrNotExist) {
		f.Close()
		return nil, err
	}
	dirtyPath := filepath.Join(dir, "dirty")
	if _, err := os.Stat(dirtyPath); err == nil {
		s.markGap()
	} else if !errors.Is(err, os.ErrNotExist) {
		f.Close()
		return nil, err
	}
	if err := dashboardWriteAtomic(dirtyPath, []byte("reader active\n")); err != nil {
		f.Close()
		return nil, err
	}
	s.worker.Add(1)
	go s.run()
	return s, nil
}

func (s *dashboardStore) loadJournal() error {
	if _, err := s.journal.Seek(0, io.SeekStart); err != nil {
		return err
	}
	sc := bufio.NewScanner(s.journal)
	sc.Buffer(make([]byte, 64<<10), 512<<10)
	for sc.Scan() {
		var entry dashboardJournalEntry
		if err := json.Unmarshal(sc.Bytes(), &entry); err != nil {
			return fmt.Errorf("dashboard reader journal corrupt: %w", err)
		}
		if entry.Record.ExternalID == "" || entry.Record.StreamSequence == "" {
			return errors.New("dashboard reader journal contains invalid record")
		}
		seq, err := strconv.ParseUint(entry.Record.StreamSequence, 10, 64)
		if err != nil || seq <= s.sequences[entry.Kind] {
			return errors.New("dashboard reader journal sequence regression")
		}
		s.sequences[entry.Kind] = seq
		if entry.Kind == "DETECTION" {
			s.history = append(s.history, entry.Record)
		} else {
			if s.state[entry.Kind] == nil {
				s.state[entry.Kind] = make(map[string]dashboardRecord)
			}
			s.state[entry.Kind][entry.Record.ExternalID] = entry.Record
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	_, err := s.journal.Seek(0, io.SeekEnd)
	return err
}

func (s *dashboardStore) loadSnapshots() error {
	path := filepath.Join(s.dir, "snapshots.json")
	if _, err := os.Lstat(path); err == nil {
		if err := dashboardPrivatePath(path, false); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &s.snapshots); err != nil {
		return fmt.Errorf("dashboard reader snapshots corrupt: %w", err)
	}
	for id, snap := range s.snapshots {
		if time.Since(snap.CreatedAt) > dashboardSnapshotTTL {
			delete(s.snapshots, id)
		}
	}
	return nil
}

func (s *dashboardStore) markGap() {
	s.gap.Store(true)
	s.gapEpoch.Add(1)
}

func (s *dashboardStore) enqueueDetection(v dashboardDetection) {
	if s == nil {
		return
	}
	s.queueMu.RLock()
	defer s.queueMu.RUnlock()
	if s.closed.Load() {
		return
	}
	select {
	case s.queue <- v:
	default:
		s.markGap()
	}
}

func (s *dashboardStore) run() {
	defer s.worker.Done()
	for v := range s.queue {
		if err := s.appendDetection(v); err != nil {
			s.markGap()
		}
	}
}

func (s *dashboardStore) newRecord(kind, id string, payload map[string]any, at time.Time) dashboardRecord {
	return dashboardRecord{
		ExternalID: id, Operation: "UPSERT", SourceObservedAt: dashboardUTC(at), Payload: payload,
		Provenance: dashboardProvenance{
			Origin:      dashboardRef{"WAF", s.instance, kind, id},
			ForwardedBy: []dashboardRef{}, DerivedFrom: []dashboardRef{}, EvidenceRefs: []any{},
		},
	}
}

func (s *dashboardStore) appendDetection(v dashboardDetection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.RuleID <= 0 || v.Site == "" {
		return nil
	}
	id := fmt.Sprintf("det-%d", s.sequences["DETECTION"]+1)
	siteID := dashboardStableID("site", v.Site)
	payload := map[string]any{
		"entity_refs": []any{dashboardEntityRef(s.ref("ASSET", siteID), "SUBJECT", v.At)},
		"summary":     "Coraza rule matched on a protected site.",
		"attention":   nil, // a rule match alone does not prove a high-signal attack
		"detector_id": fmt.Sprintf("coraza-rule-%d", v.RuleID), "rule_version": nil,
		"category": "CORAZA_RULE_MATCH", "confidence": nil, "status": "OPEN",
		"lure_type": nil, "match_strength": nil,
	}
	rec := s.newRecord("DETECTION", id, payload, v.At)
	rec.Revision = "1"
	rec.Provenance.OriginRevision = "1"
	rec.Provenance.OriginEventID = &id
	return s.appendLocked("DETECTION", rec)
}

func dashboardSeverity(v string) string {
	switch strings.ToUpper(v) {
	case "EMERGENCY", "ALERT", "CRITICAL":
		return "CRITICAL"
	case "ERROR", "HIGH":
		return "HIGH"
	case "WARNING", "MEDIUM":
		return "MEDIUM"
	case "NOTICE", "LOW":
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

func (s *dashboardStore) ref(kind, id string) dashboardRef {
	return dashboardRef{"WAF", s.instance, kind, id}
}

func dashboardEntityRef(ref dashboardRef, role string, at time.Time) map[string]any {
	return map[string]any{
		"ref": ref, "role": role, "observed_at": dashboardUTC(at),
		"valid_until": nil, "network_namespace": nil, "confidence": 1.0,
	}
}

func (s *dashboardStore) appendLocked(kind string, rec dashboardRecord) error {
	info, err := s.journal.Stat()
	if err != nil {
		return err
	}
	if info.Size() >= dashboardMaxJournalSize || s.sequences[kind] >= 9999999999999999999 {
		s.markGap()
		return errors.New("dashboard reader export capacity reached")
	}
	rec.StreamSequence = strconv.FormatUint(s.sequences[kind]+1, 10)
	b, err := json.Marshal(dashboardJournalEntry{Kind: kind, Record: rec})
	if err != nil {
		return err
	}
	if len(b) > 256<<10 {
		s.markGap()
		return errors.New("dashboard reader record too large")
	}
	b = append(b, '\n')
	if _, err := s.journal.Write(b); err != nil {
		return err
	}
	if err := s.journal.Sync(); err != nil {
		return err
	}
	s.sequences[kind]++
	if kind == "DETECTION" {
		s.history = append(s.history, rec)
	} else {
		if s.state[kind] == nil {
			s.state[kind] = make(map[string]dashboardRecord)
		}
		s.state[kind][rec.ExternalID] = rec
	}
	return nil
}

func (s *dashboardStore) upsert(kind, id string, payload map[string]any, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.state[kind][id]; ok && current.Operation == "UPSERT" {
		oldPayload, _ := json.Marshal(current.Payload)
		newPayload, _ := json.Marshal(payload)
		if hmac.Equal(oldPayload, newPayload) {
			return nil
		}
	}
	rec := s.newRecord(kind, id, payload, at)
	if current, ok := s.state[kind][id]; ok {
		rev, _ := strconv.ParseUint(current.Revision, 10, 64)
		rec.Revision = strconv.FormatUint(rev+1, 10)
	} else {
		rec.Revision = "1"
	}
	rec.Provenance.OriginRevision = rec.Revision
	return s.appendLocked(kind, rec)
}

func (s *dashboardStore) deleteMissing(kind string, keep map[string]bool, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, current := range s.state[kind] {
		if keep[id] || current.Operation == "DELETE" {
			continue
		}
		rev, _ := strconv.ParseUint(current.Revision, 10, 64)
		rec := s.newRecord(kind, id, map[string]any{}, at)
		rec.Operation = "DELETE"
		rec.Revision = strconv.FormatUint(rev+1, 10)
		rec.Provenance.OriginRevision = rec.Revision
		if err := s.appendLocked(kind, rec); err != nil {
			return err
		}
	}
	return nil
}

func (s *dashboardStore) snapshot(kind, principal string) (string, dashboardSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, snap := range s.snapshots {
		if time.Since(snap.CreatedAt) > dashboardSnapshotTTL {
			delete(s.snapshots, id)
		}
	}
	if len(s.snapshots) >= dashboardMaxSnapshots {
		return "", dashboardSnapshot{}, errors.New("dashboard reader snapshot capacity reached")
	}
	id, err := dashboardRandomID()
	if err != nil {
		return "", dashboardSnapshot{}, err
	}
	snap := dashboardSnapshot{Kind: kind, Principal: principal, CreatedAt: time.Now().UTC(),
		Watermark: s.sequences[kind], GapEpoch: s.gapEpoch.Load(), Records: []dashboardRecord{}}
	if kind == "DETECTION" {
		cutoff := time.Now().Add(-dashboardHistoryDays * 24 * time.Hour)
		for _, rec := range s.history {
			at, err := time.Parse(time.RFC3339Nano, rec.SourceObservedAt)
			if err == nil && !at.Before(cutoff) {
				snap.Records = append(snap.Records, rec)
			}
		}
	} else {
		for _, rec := range s.state[kind] {
			snap.Records = append(snap.Records, rec)
		}
		sort.Slice(snap.Records, func(i, j int) bool {
			a, _ := strconv.ParseUint(snap.Records[i].StreamSequence, 10, 64)
			b, _ := strconv.ParseUint(snap.Records[j].StreamSequence, 10, 64)
			return a < b
		})
	}
	s.snapshots[id] = snap
	if err := s.saveSnapshotsLocked(); err != nil {
		delete(s.snapshots, id)
		return "", dashboardSnapshot{}, err
	}
	return id, snap, nil
}

func (s *dashboardStore) saveSnapshotsLocked() error {
	b, err := json.Marshal(s.snapshots)
	if err != nil {
		return err
	}
	return dashboardWriteAtomic(filepath.Join(s.dir, "snapshots.json"), b)
}

func (s *dashboardStore) getSnapshot(id string) (dashboardSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.snapshots[id]
	if !ok || time.Since(snap.CreatedAt) > dashboardSnapshotTTL {
		return dashboardSnapshot{}, false
	}
	return snap, true
}

// A one-page bootstrap has no outstanding snapshot cursor, so retaining its
// on-disk copy would exhaust the bounded snapshot pool during normal polling.
func (s *dashboardStore) dropSnapshot(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.snapshots, id)
	return s.saveSnapshotsLocked()
}

func (s *dashboardStore) delta(after uint64, limit int) ([]dashboardRecord, uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	watermark := s.sequences["DETECTION"]
	rows := make([]dashboardRecord, 0, limit)
	for _, rec := range s.history {
		seq, _ := strconv.ParseUint(rec.StreamSequence, 10, 64)
		if seq > after && seq <= watermark {
			rows = append(rows, rec)
			if len(rows) > limit {
				return rows[:limit], watermark, true
			}
		}
	}
	return rows, watermark, false
}

func (s *dashboardStore) close() error {
	s.queueMu.Lock()
	if s.closed.Swap(true) {
		s.queueMu.Unlock()
		return nil
	}
	close(s.queue)
	s.queueMu.Unlock()
	s.worker.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gap.Load() {
		if err := dashboardWriteAtomic(filepath.Join(s.dir, "gap"), []byte("coverage gap\n")); err != nil {
			return err
		}
	}
	if err := s.journal.Sync(); err != nil {
		return err
	}
	if err := s.journal.Close(); err != nil {
		return err
	}
	return os.Remove(filepath.Join(s.dir, "dirty"))
}
