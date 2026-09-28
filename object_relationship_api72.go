package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	objectRelationshipStateVersion     = 1
	objectRelationshipDefaultTTL       = 30 * 24 * time.Hour
	objectRelationshipMaxRelations     = 8192
	objectRelationshipMaxPerIdentity   = 512
	objectRelationshipMaxPerObject     = 256
	objectRelationshipQueueCapacity    = 2048
	objectRelationshipMaxPathBytes     = 4096
	objectRelationshipMaxQueryBytes    = 8192
	objectRelationshipSourceAPI5       = "API5_VERIFIED_IDENTITY"
	objectRelationshipSourceAPI71      = "API71_KEYED_OBJECT"
	objectRelationshipSourceAPI8       = "API8_GRAPHQL_VARIABLE"
	objectRelationshipEvidenceObserved = "OBSERVED"
	objectRelationshipEvidenceRepeated = "REPEATED"
)

// IdentityObjectRelationship is API-7.2 visibility evidence only. It records
// that one cryptographically verified identity pseudonym was observed using one
// API-7.1 object selector/value fingerprint. It is not an ownership, tenant
// boundary, authorization, or BOLA verdict.
type IdentityObjectRelationship struct {
	ID                  string    `json:"id"`
	LocatorID           string    `json:"locator_id"`
	OperationID         string    `json:"operation_id"`
	Location            string    `json:"location"`
	Field               string    `json:"field"`
	IdentityKind        string    `json:"identity_kind"` // subject | client
	IdentityFingerprint string    `json:"identity_fingerprint"`
	TenantFingerprint   string    `json:"tenant_fingerprint,omitempty"`
	ClientFingerprint   string    `json:"client_fingerprint,omitempty"`
	ObjectFingerprint   string    `json:"object_fingerprint"`
	EvidenceLevel       string    `json:"evidence_level"` // OBSERVED | REPEATED
	ObservationCount    uint64    `json:"observation_count"`
	Sources             []string  `json:"sources"`
	FirstSeen           time.Time `json:"first_seen"`
	LastSeen            time.Time `json:"last_seen"`
	ExpiresAt           time.Time `json:"expires_at"`
}

type objectRelationshipStateFile struct {
	Version       int                          `json:"version"`
	Saved         time.Time                    `json:"saved"`
	Relationships []IdentityObjectRelationship `json:"relationships"`
}

type objectRelationshipObservation struct {
	OperationID             string
	Path                    string
	RawQuery                string
	ContentType             string
	BodyPrefix              []byte
	BodyTruncated           bool
	IdentityKind            string
	IdentityFingerprint     string
	TenantFingerprint       string
	ClientFingerprint       string
	DirectLocatorID         string
	DirectLocation          string
	DirectField             string
	DirectObjectFingerprint string
	DirectSource            string
	At                      time.Time
}

type ObjectRelationshipStatus struct {
	RelationshipCount int    `json:"relationship_count"`
	MaxRelationships  int    `json:"max_relationships"`
	MaxPerIdentity    int    `json:"max_per_identity"`
	MaxPerObject      int    `json:"max_per_object"`
	TTLSeconds        int64  `json:"ttl_seconds"`
	QueueDepth        int    `json:"queue_depth"`
	QueueCapacity     int    `json:"queue_capacity"`
	Dropped           uint64 `json:"dropped_observations"`
	Processed         uint64 `json:"processed_observations"`
	Accepting         bool   `json:"accepting"`
}

type objectRelationshipStore struct {
	mu            sync.RWMutex
	keyMu         sync.RWMutex
	key           []byte
	relationships map[string]*IdentityObjectRelationship
	locators      *objectLocatorStore
	detector      *bolaDetectionStore
	now           func() time.Time

	queue     chan objectRelationshipObservation
	stop      chan struct{}
	done      chan struct{}
	accepting atomic.Bool
	dropped   atomic.Uint64
	processed atomic.Uint64
	startOnce sync.Once
	stopOnce  sync.Once
}

func newObjectRelationshipStore() *objectRelationshipStore {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("object relationship key generation failed: " + err.Error())
	}
	return newObjectRelationshipStoreWithKey(key)
}

func newObjectRelationshipStoreWithKey(key []byte) *objectRelationshipStore {
	if len(key) != 32 {
		panic("object relationship key must be exactly 32 bytes")
	}
	return &objectRelationshipStore{
		key:           append([]byte(nil), key...),
		relationships: map[string]*IdentityObjectRelationship{},
		now:           func() time.Time { return time.Now().UTC() },
		queue:         make(chan objectRelationshipObservation, objectRelationshipQueueCapacity),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
}

func (s *objectRelationshipStore) start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		s.accepting.Store(true)
		go s.run()
	})
}

func (s *objectRelationshipStore) stopAndDrain() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.accepting.Store(false)
		close(s.stop)
	})
	<-s.done
}

func (s *objectRelationshipStore) run() {
	defer close(s.done)
	for {
		select {
		case ev := <-s.queue:
			s.process(ev)
		case <-s.stop:
			for {
				select {
				case ev := <-s.queue:
					s.process(ev)
				default:
					return
				}
			}
		}
	}
}

func (s *objectRelationshipStore) digestIdentity(kind string, values ...string) string {
	if s == nil {
		return ""
	}
	s.keyMu.RLock()
	key := append([]byte(nil), s.key...)
	s.keyMu.RUnlock()
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte("api72\x00" + kind + "\x00" + strings.Join(values, "\x00")))
	return hex.EncodeToString(h.Sum(nil)[:20])
}

func (s *objectRelationshipStore) identityObservation(r *http.Request) objectRelationshipObservation {
	if s == nil || r == nil {
		return objectRelationshipObservation{}
	}
	id, ok := verifiedAPIIdentity(r)
	if !ok {
		return objectRelationshipObservation{}
	}
	kind := ""
	identityFingerprint := ""
	if strings.TrimSpace(id.Subject) != "" {
		kind = "subject"
		identityFingerprint = s.digestIdentity("subject", id.Issuer, id.Subject)
	} else if strings.TrimSpace(id.ClientID) != "" {
		kind = "client"
		identityFingerprint = s.digestIdentity("client", id.Issuer, id.ClientID)
	}
	if identityFingerprint == "" {
		return objectRelationshipObservation{}
	}
	return objectRelationshipObservation{
		IdentityKind: kind, IdentityFingerprint: identityFingerprint,
		TenantFingerprint: s.digestOptionalIdentity("tenant", id.Issuer, id.TenantID),
		ClientFingerprint: s.digestOptionalIdentity("client", id.Issuer, id.ClientID),
		At:                s.now().UTC(),
	}
}

func (s *objectRelationshipStore) observation(r *http.Request, site string) objectRelationshipObservation {
	if s == nil || r == nil || r.URL == nil {
		return objectRelationshipObservation{}
	}
	base := s.identityObservation(r)
	if base.IdentityFingerprint == "" {
		return objectRelationshipObservation{}
	}
	path := r.URL.Path
	if len(path) > objectRelationshipMaxPathBytes || len(r.URL.RawQuery) > objectRelationshipMaxQueryBytes {
		return objectRelationshipObservation{}
	}
	prefix := requestBodyPrefixFromRequest(r)
	if len(prefix) > schemaObservationBodyMax {
		prefix = prefix[:schemaObservationBodyMax]
	}
	capture := requestCaptureFromRequest(r)
	base.OperationID = apiOperationID(site, r.Method, normalizeAPIOperationPath(path))
	base.Path = path
	base.RawQuery = r.URL.RawQuery
	base.ContentType = r.Header.Get("Content-Type")
	base.BodyPrefix = append([]byte(nil), prefix...)
	base.BodyTruncated = capture != nil && capture.bodyTruncated
	return base
}

func (s *objectRelationshipStore) digestOptionalIdentity(kind, issuer, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return s.digestIdentity(kind, issuer, value)
}

func (s *objectRelationshipStore) wrap(site string, next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.accepting.Load() {
			ev := s.observation(r, site)
			if ev.OperationID != "" && ev.IdentityFingerprint != "" {
				select {
				case s.queue <- ev:
				default:
					s.dropped.Add(1)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// enqueueKeyed accepts already-keyed API-8 GraphQL variable evidence. It is
// non-blocking and carries no raw object or identity value.
func (s *objectRelationshipStore) enqueueKeyed(ev objectRelationshipObservation, locator ObjectLocator, objectFingerprint, source string) {
	if s == nil || !s.accepting.Load() || ev.IdentityFingerprint == "" || objectFingerprint == "" || locator.ID == "" {
		return
	}
	ev.OperationID = locator.OperationID
	ev.DirectLocatorID = locator.ID
	ev.DirectLocation = locator.Location
	ev.DirectField = locator.Field
	ev.DirectObjectFingerprint = objectFingerprint
	ev.DirectSource = source
	select {
	case s.queue <- ev:
	default:
		s.dropped.Add(1)
	}
}

func (s *objectRelationshipStore) process(ev objectRelationshipObservation) {
	if s == nil || s.locators == nil || ev.OperationID == "" || ev.IdentityFingerprint == "" || ev.At.IsZero() {
		return
	}
	if ev.DirectLocatorID != "" && ev.DirectObjectFingerprint != "" {
		locator, ok := s.locators.activeLocator(ev.OperationID, ev.DirectLocation, ev.DirectField)
		if ok && locator.ID == ev.DirectLocatorID {
			if s.detector != nil {
				s.detector.observe(ev, locator, ev.DirectObjectFingerprint, s)
			}
			s.mergeWithSource(ev, locator, ev.DirectObjectFingerprint, ev.DirectSource)
		}
		s.processed.Add(1)
		return
	}
	meta := apiObservationMeta{BodyPrefix: ev.BodyPrefix, BodyTruncated: ev.BodyTruncated}
	samples := collectSchemaSamplesFromObservation(ev.Path, ev.RawQuery, ev.ContentType, meta)
	for _, sample := range samples {
		if sample.Location != "path" && sample.Location != "query" && sample.Location != "body" {
			continue
		}
		if sample.Location == "body" && strings.HasPrefix(strings.ToLower(sample.Path), "variables.") {
			continue
		}
		if sample.objectValue == "" {
			continue
		}
		locator, ok := s.locators.activeLocator(ev.OperationID, sample.Location, sample.Path)
		if !ok {
			continue
		}
		objectFingerprint := s.locators.digestValue(ev.OperationID, sample.Location, sample.Path, sample.objectValue)
		if objectFingerprint == "" {
			continue
		}
		if s.detector != nil {
			s.detector.observe(ev, locator, objectFingerprint, s)
		}
		s.mergeWithSource(ev, locator, objectFingerprint, objectRelationshipSourceAPI71)
	}
	s.processed.Add(1)
}

func objectRelationshipID(identityFingerprint, locatorID, objectFingerprint string) string {
	h := sha256.Sum256([]byte("api72-relationship\x00" + identityFingerprint + "\x00" + locatorID + "\x00" + objectFingerprint))
	return hex.EncodeToString(h[:20])
}

func (s *objectRelationshipStore) countIdentityLocked(identityFingerprint string) int {
	n := 0
	for _, v := range s.relationships {
		if v.IdentityFingerprint == identityFingerprint {
			n++
		}
	}
	return n
}

func (s *objectRelationshipStore) countObjectLocked(locatorID, objectFingerprint string) int {
	n := 0
	for _, v := range s.relationships {
		if v.LocatorID == locatorID && v.ObjectFingerprint == objectFingerprint {
			n++
		}
	}
	return n
}

func (s *objectRelationshipStore) merge(ev objectRelationshipObservation, locator ObjectLocator, objectFingerprint string) {
	s.mergeWithSource(ev, locator, objectFingerprint, objectRelationshipSourceAPI71)
}

func (s *objectRelationshipStore) mergeWithSource(ev objectRelationshipObservation, locator ObjectLocator, objectFingerprint, source string) {
	id := objectRelationshipID(ev.IdentityFingerprint, locator.ID, objectFingerprint)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(ev.At)
	v := s.relationships[id]
	if v == nil {
		if len(s.relationships) >= objectRelationshipMaxRelations ||
			s.countIdentityLocked(ev.IdentityFingerprint) >= objectRelationshipMaxPerIdentity ||
			s.countObjectLocked(locator.ID, objectFingerprint) >= objectRelationshipMaxPerObject {
			return
		}
		v = &IdentityObjectRelationship{
			ID: id, LocatorID: locator.ID, OperationID: locator.OperationID, Location: locator.Location, Field: locator.Field,
			IdentityKind: ev.IdentityKind, IdentityFingerprint: ev.IdentityFingerprint, TenantFingerprint: ev.TenantFingerprint,
			ClientFingerprint: ev.ClientFingerprint, ObjectFingerprint: objectFingerprint,
			Sources: []string{objectRelationshipSourceAPI5, source}, FirstSeen: ev.At,
		}
		s.relationships[id] = v
	}
	if v.ObservationCount < ^uint64(0) {
		v.ObservationCount++
	}
	v.EvidenceLevel = objectRelationshipEvidenceObserved
	if v.ObservationCount >= 2 {
		v.EvidenceLevel = objectRelationshipEvidenceRepeated
	}
	v.LastSeen = ev.At
	v.ExpiresAt = ev.At.Add(objectRelationshipDefaultTTL)
}

func validObjectRelationship(v IdentityObjectRelationship, now time.Time) bool {
	if len(v.ID) != 40 || len(v.LocatorID) != 32 || !validSequenceOperationID(v.OperationID) ||
		!validObjectLocatorLocation(v.Location) || !validObjectLocatorField(v.Field) ||
		(v.IdentityKind != "subject" && v.IdentityKind != "client") || len(v.IdentityFingerprint) != 40 || len(v.ObjectFingerprint) != 40 ||
		v.ObservationCount == 0 || v.FirstSeen.IsZero() || v.LastSeen.IsZero() || !v.ExpiresAt.After(now) ||
		(v.EvidenceLevel != objectRelationshipEvidenceObserved && v.EvidenceLevel != objectRelationshipEvidenceRepeated) {
		return false
	}
	for _, digest := range []string{v.ID, v.IdentityFingerprint, v.ObjectFingerprint} {
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	for _, digest := range []string{v.TenantFingerprint, v.ClientFingerprint} {
		if digest != "" {
			if len(digest) != 40 {
				return false
			}
			if _, err := hex.DecodeString(digest); err != nil {
				return false
			}
		}
	}
	return len(v.Sources) == 2 && containsString(v.Sources, objectRelationshipSourceAPI5) && (containsString(v.Sources, objectRelationshipSourceAPI71) || containsString(v.Sources, objectRelationshipSourceAPI8))
}

func (s *objectRelationshipStore) pruneLocked(now time.Time) {
	for id, v := range s.relationships {
		if v == nil || !v.ExpiresAt.After(now) {
			delete(s.relationships, id)
		}
	}
}

func cloneObjectRelationship(v IdentityObjectRelationship) IdentityObjectRelationship {
	v.Sources = append([]string(nil), v.Sources...)
	return v
}

func (s *objectRelationshipStore) snapshot() []IdentityObjectRelationship {
	if s == nil {
		return []IdentityObjectRelationship{}
	}
	now := s.now()
	s.mu.Lock()
	s.pruneLocked(now)
	out := make([]IdentityObjectRelationship, 0, len(s.relationships))
	for _, p := range s.relationships {
		out = append(out, cloneObjectRelationship(*p))
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastSeen.Equal(out[j].LastSeen) {
			return out[i].LastSeen.Before(out[j].LastSeen)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *objectRelationshipStore) status() ObjectRelationshipStatus {
	rows := s.snapshot()
	status := ObjectRelationshipStatus{
		RelationshipCount: len(rows), MaxRelationships: objectRelationshipMaxRelations, MaxPerIdentity: objectRelationshipMaxPerIdentity,
		MaxPerObject: objectRelationshipMaxPerObject, TTLSeconds: int64(objectRelationshipDefaultTTL / time.Second),
		QueueCapacity: objectRelationshipQueueCapacity, Dropped: s.dropped.Load(), Processed: s.processed.Load(), Accepting: s.accepting.Load(),
	}
	if s != nil {
		status.QueueDepth = len(s.queue)
	}
	return status
}

func (s *objectRelationshipStore) ensureKey(configPath string) error {
	path := apiSecurityStatePath(configPath, "api-object-relationship.key")
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("api-object-relationship.key must be a regular file")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(b) != 32 {
			return errors.New("api-object-relationship.key must contain exactly 32 bytes")
		}
		s.keyMu.Lock()
		s.key = append(s.key[:0], b...)
		s.keyMu.Unlock()
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	s.keyMu.RLock()
	key := append([]byte(nil), s.key...)
	s.keyMu.RUnlock()
	f, err := os.CreateTemp(filepath.Dir(path), ".api-object-relationship.key.tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	if _, err := f.Write(key); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *objectRelationshipStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	if err := s.ensureKey(configPath); err != nil {
		return err
	}
	state := objectRelationshipStateFile{Version: objectRelationshipStateVersion, Saved: time.Now().UTC(), Relationships: s.snapshot()}
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-object-relationships.json"), state)
}

func (s *objectRelationshipStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	if err := s.ensureKey(configPath); err != nil {
		return err
	}
	var state objectRelationshipStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-object-relationships.json"), &state); err != nil {
		return err
	}
	if state.Version != 0 && state.Version != objectRelationshipStateVersion {
		return fmt.Errorf("unsupported object relationship state version %d", state.Version)
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range state.Relationships {
		if len(s.relationships) >= objectRelationshipMaxRelations || !validObjectRelationship(v, now) ||
			s.countIdentityLocked(v.IdentityFingerprint) >= objectRelationshipMaxPerIdentity ||
			s.countObjectLocked(v.LocatorID, v.ObjectFingerprint) >= objectRelationshipMaxPerObject {
			continue
		}
		cp := cloneObjectRelationship(v)
		s.relationships[v.ID] = &cp
	}
	return nil
}

func (a *adminServer) handleObjectRelationships(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.objectRelationships.snapshot()
	a.audit.add(who(r).user, "api_object_relationship.view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}

func (a *adminServer) handleObjectRelationshipStatus(w http.ResponseWriter, r *http.Request) {
	status := a.srv.objectRelationships.status()
	a.audit.add(who(r).user, "api_object_relationship.status_view", fmt.Sprintf("count=%d", status.RelationshipCount))
	writeJSON(w, status)
}
