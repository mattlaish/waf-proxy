package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const owiStateVersion = 1

type owiStoreLimits struct {
	SnapshotMaxCount   int
	SnapshotMaxRecords int
	SnapshotMaxBytes   int64
	DetectionMaxRows   int
	DetectionMaxBytes  int64
}

func defaultOWIStoreLimits() owiStoreLimits {
	return owiStoreLimits{256, 10000, 32 << 20, 200000, 128 << 20}
}

type owiStore struct {
	mu             sync.Mutex
	path           string
	sourceInstance string
	state          owiPersistedState
	limits         owiStoreLimits
	detectionBytes int64
	writeFailed    bool
}

func newOWIStore(path, sourceInstance string) (*owiStore, error) {
	return newOWIStoreWithLimits(path, sourceInstance, defaultOWIStoreLimits())
}

func newOWIStoreWithLimits(path, sourceInstance string, limits owiStoreLimits) (*owiStore, error) {
	s := &owiStore{path: path, sourceInstance: sourceInstance, limits: limits}
	s.state = owiPersistedState{
		Version:        owiStateVersion,
		SourceInstance: sourceInstance,
		NextSequence:   map[string]uint64{},
		Current:        map[string]map[string]owiRecord{},
		Journal:        map[string][]owiRecord{},
		Snapshots:      map[string]owiSnapshot{},
		CleanShutdown:  true,
	}
	var loaded owiPersistedState
	if err := readJSONIfExists(path, &loaded); err != nil {
		return nil, fmt.Errorf("read dashboard connector state: %w", err)
	}
	if loaded.Version != 0 {
		if loaded.Version != owiStateVersion {
			return nil, fmt.Errorf("dashboard connector state version %d unsupported", loaded.Version)
		}
		if loaded.SourceInstance != "" && loaded.SourceInstance != sourceInstance {
			return nil, fmt.Errorf("dashboard connector state belongs to source instance %q, configured %q", loaded.SourceInstance, sourceInstance)
		}
		s.state = loaded
		s.ensureMapsLocked()
		if !loaded.CleanShutdown {
			now := time.Now().UTC()
			s.state.DetectionGapAt = &now
		}
	}
	// Mark the process dirty before serving. A crash can lose an accepted but
	// not-yet-fsynced observation; the next process reports GAP rather than
	// silently asserting complete HISTORY coverage.
	s.state.Version = owiStateVersion
	s.state.SourceInstance = sourceInstance
	s.state.CleanShutdown = false
	for _, r := range s.state.Journal["DETECTION"] {
		s.detectionBytes += owiRecordEncodedBytes(r)
	}
	s.cleanupLocked(time.Now().UTC())
	if err := s.persistLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *owiStore) ensureMapsLocked() {
	if s.state.NextSequence == nil {
		s.state.NextSequence = map[string]uint64{}
	}
	if s.state.Current == nil {
		s.state.Current = map[string]map[string]owiRecord{}
	}
	if s.state.Journal == nil {
		s.state.Journal = map[string][]owiRecord{}
	}
	if s.state.Snapshots == nil {
		s.state.Snapshots = map[string]owiSnapshot{}
	}
}

func (s *owiStore) persistLocked() error {
	s.state.SavedAt = time.Now().UTC()
	err := atomicWriteJSON(s.path, s.state)
	s.writeFailed = err != nil
	return err
}

func (s *owiStore) closeClean() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.CleanShutdown = true
	return s.persistLocked()
}

func (s *owiStore) markDetectionDrop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.state.DetectionDrops++
	if s.state.DetectionGapAt == nil {
		s.state.DetectionGapAt = &now
	}
	_ = s.persistLocked()
}

func (s *owiStore) nextSeqLocked(kind string) uint64 {
	s.state.NextSequence[kind]++
	return s.state.NextSequence[kind]
}

func owiRecordRevision(r owiRecord) uint64 {
	n, _ := strconv.ParseUint(r.Revision, 10, 64)
	return n
}

func owiSameRecordContent(a, b owiRecord) bool {
	// Stream sequence/revision and provenance.origin_revision are export
	// metadata. Source timestamps and the rest of provenance are source facts.
	a.Provenance.OriginRevision = ""
	b.Provenance.OriginRevision = ""
	aa := struct {
		Observed   string         `json:"observed"`
		Updated    *string        `json:"updated"`
		Provenance owiProvenance  `json:"provenance"`
		Payload    map[string]any `json:"payload"`
	}{a.SourceObservedAt, a.SourceUpdatedAt, a.Provenance, a.Payload}
	bb := struct {
		Observed   string         `json:"observed"`
		Updated    *string        `json:"updated"`
		Provenance owiProvenance  `json:"provenance"`
		Payload    map[string]any `json:"payload"`
	}{b.SourceObservedAt, b.SourceUpdatedAt, b.Provenance, b.Payload}
	ab, _ := json.Marshal(aa)
	bbuf, _ := json.Marshal(bb)
	return bytes.Equal(ab, bbuf)
}

func (s *owiStore) syncStateKindLocked(kind string, desired []owiRecord, now time.Time) bool {
	cur := s.state.Current[kind]
	if cur == nil {
		cur = map[string]owiRecord{}
		s.state.Current[kind] = cur
	}
	seen := make(map[string]struct{}, len(desired))
	changed := false
	for _, candidate := range desired {
		seen[candidate.ExternalID] = struct{}{}
		old, ok := cur[candidate.ExternalID]
		if ok && owiSameRecordContent(old, candidate) {
			continue
		}
		rev := uint64(1)
		if ok {
			rev = owiRecordRevision(old) + 1
		}
		seq := s.nextSeqLocked(kind)
		candidate.Revision = strconv.FormatUint(rev, 10)
		candidate.StreamSequence = strconv.FormatUint(seq, 10)
		candidate.Operation = "UPSERT"
		candidate.Provenance.OriginRevision = candidate.Revision
		cur[candidate.ExternalID] = candidate
		s.state.Journal[kind] = append(s.state.Journal[kind], candidate)
		changed = true
	}
	for id, old := range cur {
		if _, ok := seen[id]; ok {
			continue
		}
		rev := owiRecordRevision(old) + 1
		seq := s.nextSeqLocked(kind)
		observed := now.Format(time.RFC3339)
		tomb := owiRecord{
			ExternalID: id, Revision: strconv.FormatUint(rev, 10), StreamSequence: strconv.FormatUint(seq, 10), Operation: "DELETE",
			SourceObservedAt: observed, SourceUpdatedAt: nil,
			Provenance: old.Provenance, Payload: map[string]any{},
		}
		tomb.Provenance.OriginRevision = tomb.Revision
		tomb.Provenance.OriginEventID = nil
		s.state.Journal[kind] = append(s.state.Journal[kind], tomb)
		delete(cur, id)
		changed = true
	}
	return changed
}

func (s *owiStore) lockContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if s.mu.TryLock() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *owiStore) syncStateBatchContext(ctx context.Context, desired map[string][]owiRecord, now time.Time) error {
	if err := s.lockContext(ctx); err != nil {
		return err
	}
	defer s.mu.Unlock()
	changed := false
	for _, kind := range []string{"ASSET", "POLICY", "HEALTH"} {
		changed = s.syncStateKindLocked(kind, desired[kind], now) || changed
	}
	changed = s.cleanupLocked(now) || changed
	if changed || s.writeFailed {
		return s.persistLocked()
	}
	return nil
}

func (s *owiStore) syncStateBatch(desired map[string][]owiRecord, now time.Time) error {
	return s.syncStateBatchContext(context.Background(), desired, now)
}

func (s *owiStore) appendDetection(candidate owiRecord, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	rowBytes := owiRecordEncodedBytes(candidate)
	if len(s.state.Journal["DETECTION"]) >= s.limits.DetectionMaxRows || s.detectionBytes+rowBytes > s.limits.DetectionMaxBytes {
		return errors.New("detection export capacity exceeded")
	}
	seq := s.nextSeqLocked("DETECTION")
	candidate.Revision = "1"
	candidate.StreamSequence = strconv.FormatUint(seq, 10)
	candidate.Operation = "UPSERT"
	candidate.Provenance.OriginRevision = "1"
	s.state.Journal["DETECTION"] = append(s.state.Journal["DETECTION"], candidate)
	s.detectionBytes += owiRecordEncodedBytes(candidate)
	return s.persistLocked()
}

func (s *owiStore) cleanupLocked(now time.Time) bool {
	changed := false
	for id, snap := range s.state.Snapshots {
		if !snap.ExpiresAt.After(now) {
			delete(s.state.Snapshots, id)
			changed = true
		}
	}
	for kind, rows := range s.state.Journal {
		retention := owiStateLogRetain
		if kind == "DETECTION" {
			retention = owiHistoryRetain
		}
		cutoff := now.Add(-retention)
		idx := 0
		for idx < len(rows) {
			t, err := time.Parse(time.RFC3339, rows[idx].SourceObservedAt)
			if err != nil || !t.Before(cutoff) {
				break
			}
			idx++
		}
		if idx > 0 {
			if kind == "DETECTION" {
				for _, r := range rows[:idx] {
					s.detectionBytes -= owiRecordEncodedBytes(r)
				}
				if s.detectionBytes < 0 {
					s.detectionBytes = 0
				}
			}
			s.state.Journal[kind] = append([]owiRecord(nil), rows[idx:]...)
			changed = true
		}
	}
	if s.state.DetectionGapAt != nil && now.Sub(*s.state.DetectionGapAt) >= owiHistoryRetain {
		s.state.DetectionGapAt = nil
		s.state.DetectionDrops = 0
		changed = true
	}
	return changed
}

func owiRecordEncodedBytes(r owiRecord) int64 {
	b, _ := json.Marshal(r)
	return int64(len(b))
}

func cloneOWIRecords(in []owiRecord) []owiRecord {
	out := make([]owiRecord, len(in))
	for i := range in {
		b, _ := json.Marshal(in[i])
		_ = json.Unmarshal(b, &out[i])
	}
	return out
}

func (s *owiStore) newSnapshotContext(ctx context.Context, kind string, p owiPrincipal, scopeHash string, now time.Time) (owiSnapshot, error) {
	if err := s.lockContext(ctx); err != nil {
		return owiSnapshot{}, err
	}
	defer s.mu.Unlock()
	if s.writeFailed {
		return owiSnapshot{}, errors.New("dashboard connector durability is unavailable")
	}
	s.cleanupLocked(now)
	var rows []owiRecord
	if kind == "DETECTION" {
		rows = cloneOWIRecords(s.state.Journal[kind])
	} else {
		cur := s.state.Current[kind]
		rows = make([]owiRecord, 0, len(cur))
		for _, r := range cur {
			rows = append(rows, r)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		a, _ := strconv.ParseUint(rows[i].StreamSequence, 10, 64)
		b, _ := strconv.ParseUint(rows[j].StreamSequence, 10, 64)
		return a < b
	})
	id, err := randomOWIID("snap")
	if err != nil {
		return owiSnapshot{}, err
	}
	snap := owiSnapshot{
		ID: id, TenantID: p.TenantID, Principal: p.Principal, ScopeHash: scopeHash, Kind: kind,
		CreatedAt: now, ExpiresAt: now.Add(owiCursorTTL), Watermark: s.state.NextSequence[kind], Records: rows,
	}
	// Bound retained snapshot material. Capacity rejection never advances a checkpoint.
	if len(s.state.Snapshots) >= s.limits.SnapshotMaxCount {
		return owiSnapshot{}, errors.New("snapshot count capacity exceeded")
	}
	if len(rows) > s.limits.SnapshotMaxRecords {
		return owiSnapshot{}, errors.New("snapshot record capacity exceeded")
	}
	b, _ := json.Marshal(rows)
	if int64(len(b)) > s.limits.SnapshotMaxBytes {
		return owiSnapshot{}, errors.New("snapshot byte capacity exceeded")
	}
	s.state.Snapshots[id] = snap
	if err := s.persistLocked(); err != nil {
		return owiSnapshot{}, err
	}
	return snap, nil
}

func (s *owiStore) newSnapshot(kind string, p owiPrincipal, scopeHash string, now time.Time) (owiSnapshot, error) {
	return s.newSnapshotContext(context.Background(), kind, p, scopeHash, now)
}

func (s *owiStore) getSnapshotContext(ctx context.Context, id string) (owiSnapshot, bool, error) {
	if err := s.lockContext(ctx); err != nil {
		return owiSnapshot{}, false, err
	}
	defer s.mu.Unlock()
	if s.writeFailed {
		return owiSnapshot{}, false, errors.New("dashboard connector durability is unavailable")
	}
	snap, ok := s.state.Snapshots[id]
	if !ok || !snap.ExpiresAt.After(time.Now().UTC()) {
		if ok {
			delete(s.state.Snapshots, id)
			_ = s.persistLocked()
		}
		return owiSnapshot{}, false, nil
	}
	return snap, true, nil
}

func (s *owiStore) getSnapshot(id string) (owiSnapshot, bool) {
	snap, ok, _ := s.getSnapshotContext(context.Background(), id)
	return snap, ok
}

func (s *owiStore) journalAfterContext(ctx context.Context, kind string, watermark uint64) (rows []owiRecord, current uint64, min uint64, coverage owiCoverage, err error) {
	if err = s.lockContext(ctx); err != nil {
		return nil, 0, 0, owiCoverage{}, err
	}
	defer s.mu.Unlock()
	if s.writeFailed {
		return nil, 0, 0, owiCoverage{}, errors.New("dashboard connector durability is unavailable")
	}
	now := time.Now().UTC()
	if s.cleanupLocked(now) {
		if err = s.persistLocked(); err != nil {
			return nil, 0, 0, owiCoverage{}, err
		}
	}
	journal := s.state.Journal[kind]
	current = s.state.NextSequence[kind]
	if len(journal) > 0 {
		min, _ = strconv.ParseUint(journal[0].StreamSequence, 10, 64)
	}
	for _, r := range journal {
		seq, _ := strconv.ParseUint(r.StreamSequence, 10, 64)
		if seq > watermark {
			rows = append(rows, r)
		}
	}
	coverage = s.coverageLocked(kind, now)
	return cloneOWIRecords(rows), current, min, coverage, nil
}

func (s *owiStore) journalAfter(kind string, watermark uint64) (rows []owiRecord, current uint64, min uint64, coverage owiCoverage) {
	rows, current, min, coverage, _ = s.journalAfterContext(context.Background(), kind, watermark)
	return
}

func (s *owiStore) coverageContext(ctx context.Context, kind string) (owiCoverage, error) {
	if err := s.lockContext(ctx); err != nil {
		return owiCoverage{}, err
	}
	defer s.mu.Unlock()
	if s.writeFailed {
		return owiCoverage{}, errors.New("dashboard connector durability is unavailable")
	}
	return s.coverageLocked(kind, time.Now().UTC()), nil
}

func (s *owiStore) coverage(kind string) owiCoverage {
	cov, _ := s.coverageContext(context.Background(), kind)
	return cov
}

func (s *owiStore) coverageLocked(kind string, now time.Time) owiCoverage {
	if kind == "DETECTION" && s.state.DetectionGapAt != nil && now.Sub(*s.state.DetectionGapAt) < owiHistoryRetain {
		return owiCoverage{State: "GAP", ReasonCodes: []string{"EXPORT_COVERAGE_GAP"}}
	}
	return owiCoverage{State: "COMPLETE", ReasonCodes: []string{}}
}

func (s *owiStore) stats() (snapshots int, journalRows int, detectionDrops uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rows := range s.state.Journal {
		journalRows += len(rows)
	}
	return len(s.state.Snapshots), journalRows, s.state.DetectionDrops
}

func randomOWIID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(b), nil
}

func owiScopeHash(scopes map[string]struct{}) string {
	keys := make([]string, 0, len(scopes))
	for k := range scopes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return hex.EncodeToString(sum[:])
}

func encodeOWICursor(secret []byte, c owiCursor) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	blob := append(append([]byte(nil), payload...), mac.Sum(nil)...)
	return base64.RawURLEncoding.EncodeToString(blob), nil
}

func decodeOWICursor(secret []byte, encoded string) (owiCursor, error) {
	if encoded == "" || len(encoded) > 2048 || strings.Contains(encoded, "=") {
		return owiCursor{}, errors.New("invalid cursor encoding")
	}
	blob, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(blob) <= sha256.Size {
		return owiCursor{}, errors.New("invalid cursor encoding")
	}
	payload := blob[:len(blob)-sha256.Size]
	got := blob[len(blob)-sha256.Size:]
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	want := mac.Sum(nil)
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return owiCursor{}, errors.New("invalid cursor signature")
	}
	var c owiCursor
	if err := json.Unmarshal(payload, &c); err != nil || c.Version != 1 {
		return owiCursor{}, errors.New("invalid cursor payload")
	}
	return c, nil
}

func loadOWISecretFile(path, label string) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", label, err)
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() || st.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%s must be a regular owner-only non-symlink file", label)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	b = bytes.TrimSpace(b)
	if strings.HasPrefix(string(b), "base64:") {
		b, err = base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(string(b), "base64:")))
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", label, err)
		}
	}
	if len(b) < 32 {
		return nil, fmt.Errorf("%s must contain at least 32 bytes of high-entropy secret", label)
	}
	return append([]byte(nil), b...), nil
}

func defaultOWIStatePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "dashboard-connector-state.json")
}
