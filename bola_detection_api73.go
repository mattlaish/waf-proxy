package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

const (
	bolaCandidateStateVersion       = 1
	bolaCandidateDefaultTTL         = 7 * 24 * time.Hour
	bolaCandidateMaxCandidates      = 4096
	bolaCandidateMaxPerIdentity     = 256
	bolaCandidateMaxPerObject       = 128
	bolaCandidateMinBaselineObs     = 2
	bolaCandidateEnumerationWindow  = 10 * time.Minute
	bolaCandidateEnumerationMinObjs = 20

	bolaCandidateIdentityDivergence = "IDENTITY_OBJECT_DIVERGENCE"
	bolaCandidateTenantDivergence   = "TENANT_OBJECT_DIVERGENCE"
	bolaCandidateEnumeration        = "OBJECT_ENUMERATION"

	bolaConfidenceMedium = "MEDIUM"
	bolaConfidenceHigh   = "HIGH"
)

// BOLACandidate is API-7.3 DETECT-only evidence. It is deliberately not an
// ownership, tenant-boundary, authorization, or enforcement verdict. Every
// identity/object value carried here is already pseudonymous/keyed evidence
// from API-7.2/API-7.1.
type BOLACandidate struct {
	ID                    string    `json:"id"`
	Type                  string    `json:"type"`
	OperationID           string    `json:"operation_id"`
	LocatorID             string    `json:"locator_id"`
	Location              string    `json:"location"`
	Field                 string    `json:"field"`
	IdentityFingerprint   string    `json:"identity_fingerprint"`
	TenantFingerprint     string    `json:"tenant_fingerprint,omitempty"`
	ObjectFingerprint     string    `json:"object_fingerprint"`
	Confidence            string    `json:"confidence"`
	BaselineIdentities    int       `json:"baseline_identities"`
	BaselineTenants       int       `json:"baseline_tenants"`
	BaselineObservations  uint64    `json:"baseline_observations"`
	RecentIdentityObjects int       `json:"recent_identity_objects"`
	EvidenceCount         uint64    `json:"evidence_count"`
	Sources               []string  `json:"sources"`
	FirstSeen             time.Time `json:"first_seen"`
	LastSeen              time.Time `json:"last_seen"`
	ExpiresAt             time.Time `json:"expires_at"`
}

type bolaCandidateStateFile struct {
	Version    int             `json:"version"`
	Saved      time.Time       `json:"saved"`
	Candidates []BOLACandidate `json:"candidates"`
}

type BOLADetectionStatus struct {
	CandidateCount       int    `json:"candidate_count"`
	MaxCandidates        int    `json:"max_candidates"`
	MaxPerIdentity       int    `json:"max_per_identity"`
	MaxPerObject         int    `json:"max_per_object"`
	TTLSeconds           int64  `json:"ttl_seconds"`
	MinBaselineObs       int    `json:"min_baseline_observations"`
	EnumerationWindowSec int64  `json:"enumeration_window_seconds"`
	EnumerationMinObjs   int    `json:"enumeration_min_objects"`
	DetectionOnly        bool   `json:"detection_only"`
	Generated            uint64 `json:"generated_candidates"`
}

type bolaRelationshipContext struct {
	CurrentRelationship   bool
	ForeignRepeated       bool
	ForeignTenantRepeated bool
	BaselineIdentities    int
	BaselineTenants       int
	BaselineObservations  uint64
	RecentIdentityObjects int
}

type bolaDetectionStore struct {
	mu         sync.RWMutex
	candidates map[string]*BOLACandidate
	now        func() time.Time
	generated  uint64
}

func newBOLADetectionStore() *bolaDetectionStore {
	return &bolaDetectionStore{
		candidates: map[string]*BOLACandidate{},
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func bolaCandidateID(kind, identityFingerprint, locatorID, objectFingerprint, tenantFingerprint string) string {
	sum := sha256.Sum256([]byte("api73-bola-candidate\x00" + kind + "\x00" + identityFingerprint + "\x00" + locatorID + "\x00" + objectFingerprint + "\x00" + tenantFingerprint))
	return hex.EncodeToString(sum[:20])
}

func (s *objectRelationshipStore) bolaContext(locatorID, objectFingerprint, identityFingerprint, tenantFingerprint string, now time.Time) bolaRelationshipContext {
	var ctx bolaRelationshipContext
	if s == nil {
		return ctx
	}
	identities := map[string]struct{}{}
	tenants := map[string]struct{}{}
	recentObjects := map[string]struct{}{}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rel := range s.relationships {
		if rel == nil || !rel.ExpiresAt.After(now) {
			continue
		}
		if rel.IdentityFingerprint == identityFingerprint && rel.LocatorID == locatorID && rel.LastSeen.After(now.Add(-bolaCandidateEnumerationWindow)) {
			recentObjects[rel.ObjectFingerprint] = struct{}{}
		}
		if rel.LocatorID != locatorID || rel.ObjectFingerprint != objectFingerprint {
			continue
		}
		identities[rel.IdentityFingerprint] = struct{}{}
		if rel.TenantFingerprint != "" {
			tenants[rel.TenantFingerprint] = struct{}{}
		}
		if rel.ObservationCount <= ^uint64(0)-ctx.BaselineObservations {
			ctx.BaselineObservations += rel.ObservationCount
		} else {
			ctx.BaselineObservations = ^uint64(0)
		}
		if rel.IdentityFingerprint == identityFingerprint {
			ctx.CurrentRelationship = true
		} else if rel.ObservationCount >= bolaCandidateMinBaselineObs {
			ctx.ForeignRepeated = true
		}
		if tenantFingerprint != "" && rel.TenantFingerprint != "" && rel.TenantFingerprint != tenantFingerprint && rel.ObservationCount >= bolaCandidateMinBaselineObs {
			ctx.ForeignTenantRepeated = true
		}
	}
	ctx.BaselineIdentities = len(identities)
	ctx.BaselineTenants = len(tenants)
	ctx.RecentIdentityObjects = len(recentObjects)
	return ctx
}

func (s *bolaDetectionStore) observe(ev objectRelationshipObservation, locator ObjectLocator, objectFingerprint string, relationships *objectRelationshipStore) {
	if s == nil || relationships == nil || ev.IdentityFingerprint == "" || objectFingerprint == "" || ev.At.IsZero() {
		return
	}
	now := ev.At.UTC()
	ctx := relationships.bolaContext(locator.ID, objectFingerprint, ev.IdentityFingerprint, ev.TenantFingerprint, now)

	// A new verified identity touching an object that already has repeated
	// evidence under another verified identity is a candidate, not an ownership
	// mismatch verdict. Shared-resource applications may legitimately trigger it.
	if !ctx.CurrentRelationship && ctx.ForeignRepeated && ctx.BaselineObservations >= bolaCandidateMinBaselineObs {
		confidence := bolaConfidenceMedium
		if ctx.BaselineObservations >= 5 && ctx.BaselineIdentities == 1 {
			confidence = bolaConfidenceHigh
		}
		s.record(bolaCandidateIdentityDivergence, confidence, ev, locator, objectFingerprint, ctx)
	}

	// A different cryptographically verified tenant context on an established
	// object is separate evidence. Caller-supplied tenant headers never reach
	// this path because API-7.2 only uses API-5 verified claims.
	if ev.TenantFingerprint != "" && !ctx.CurrentRelationship && ctx.ForeignTenantRepeated && ctx.BaselineObservations >= bolaCandidateMinBaselineObs {
		confidence := bolaConfidenceMedium
		if ctx.BaselineTenants == 1 && ctx.BaselineObservations >= 5 {
			confidence = bolaConfidenceHigh
		}
		s.record(bolaCandidateTenantDivergence, confidence, ev, locator, objectFingerprint, ctx)
	}

	// Enumeration is based on bounded recent relationship state under the same
	// locator. It never fires merely because one new object is observed.
	if !ctx.CurrentRelationship && ctx.RecentIdentityObjects >= bolaCandidateEnumerationMinObjs-1 {
		confidence := bolaConfidenceMedium
		if ctx.RecentIdentityObjects >= (bolaCandidateEnumerationMinObjs*2)-1 {
			confidence = bolaConfidenceHigh
		}
		s.record(bolaCandidateEnumeration, confidence, ev, locator, objectFingerprint, ctx)
	}
}

func (s *bolaDetectionStore) record(kind, confidence string, ev objectRelationshipObservation, locator ObjectLocator, objectFingerprint string, ctx bolaRelationshipContext) {
	id := bolaCandidateID(kind, ev.IdentityFingerprint, locator.ID, objectFingerprint, ev.TenantFingerprint)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(ev.At)
	row := s.candidates[id]
	if row == nil {
		if len(s.candidates) >= bolaCandidateMaxCandidates || s.countIdentityLocked(ev.IdentityFingerprint) >= bolaCandidateMaxPerIdentity || s.countObjectLocked(locator.ID, objectFingerprint) >= bolaCandidateMaxPerObject {
			return
		}
		row = &BOLACandidate{
			ID: id, Type: kind, OperationID: locator.OperationID, LocatorID: locator.ID, Location: locator.Location, Field: locator.Field,
			IdentityFingerprint: ev.IdentityFingerprint, TenantFingerprint: ev.TenantFingerprint, ObjectFingerprint: objectFingerprint,
			Sources: []string{objectRelationshipSourceAPI5, objectRelationshipSourceAPI71, "API72_RELATIONSHIP_BASELINE"}, FirstSeen: ev.At,
		}
		s.candidates[id] = row
	}
	row.Confidence = confidence
	row.BaselineIdentities = ctx.BaselineIdentities
	row.BaselineTenants = ctx.BaselineTenants
	row.BaselineObservations = ctx.BaselineObservations
	row.RecentIdentityObjects = ctx.RecentIdentityObjects + 1
	if row.EvidenceCount < ^uint64(0) {
		row.EvidenceCount++
	}
	row.LastSeen = ev.At
	row.ExpiresAt = ev.At.Add(bolaCandidateDefaultTTL)
	s.generated++
}

func (s *bolaDetectionStore) countIdentityLocked(identityFingerprint string) int {
	n := 0
	for _, row := range s.candidates {
		if row != nil && row.IdentityFingerprint == identityFingerprint {
			n++
		}
	}
	return n
}

func (s *bolaDetectionStore) countObjectLocked(locatorID, objectFingerprint string) int {
	n := 0
	for _, row := range s.candidates {
		if row != nil && row.LocatorID == locatorID && row.ObjectFingerprint == objectFingerprint {
			n++
		}
	}
	return n
}

func (s *bolaDetectionStore) pruneLocked(now time.Time) {
	for id, row := range s.candidates {
		if row == nil || !row.ExpiresAt.After(now) {
			delete(s.candidates, id)
		}
	}
}

func validBOLACandidate(row BOLACandidate, now time.Time) bool {
	if len(row.ID) != 40 || !validSequenceOperationID(row.OperationID) || len(row.LocatorID) != 32 || !validObjectLocatorLocation(row.Location) || !validObjectLocatorField(row.Field) ||
		len(row.IdentityFingerprint) != 40 || len(row.ObjectFingerprint) != 40 || row.EvidenceCount == 0 || row.FirstSeen.IsZero() || row.LastSeen.IsZero() || !row.ExpiresAt.After(now) {
		return false
	}
	if row.Type != bolaCandidateIdentityDivergence && row.Type != bolaCandidateTenantDivergence && row.Type != bolaCandidateEnumeration {
		return false
	}
	if row.Confidence != bolaConfidenceMedium && row.Confidence != bolaConfidenceHigh {
		return false
	}
	if row.TenantFingerprint != "" && len(row.TenantFingerprint) != 40 {
		return false
	}
	for _, digest := range []string{row.ID, row.IdentityFingerprint, row.ObjectFingerprint, row.TenantFingerprint} {
		if digest == "" {
			continue
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	return len(row.Sources) == 3 && containsString(row.Sources, objectRelationshipSourceAPI5) && containsString(row.Sources, objectRelationshipSourceAPI71) && containsString(row.Sources, "API72_RELATIONSHIP_BASELINE")
}

func cloneBOLACandidate(row BOLACandidate) BOLACandidate {
	row.Sources = append([]string(nil), row.Sources...)
	return row
}

func (s *bolaDetectionStore) snapshot() []BOLACandidate {
	if s == nil {
		return []BOLACandidate{}
	}
	now := s.now()
	s.mu.Lock()
	s.pruneLocked(now)
	out := make([]BOLACandidate, 0, len(s.candidates))
	for _, row := range s.candidates {
		out = append(out, cloneBOLACandidate(*row))
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastSeen.Equal(out[j].LastSeen) {
			return out[i].LastSeen.After(out[j].LastSeen)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *bolaDetectionStore) status() BOLADetectionStatus {
	rows := s.snapshot()
	s.mu.RLock()
	generated := s.generated
	s.mu.RUnlock()
	return BOLADetectionStatus{
		CandidateCount: len(rows), MaxCandidates: bolaCandidateMaxCandidates, MaxPerIdentity: bolaCandidateMaxPerIdentity,
		MaxPerObject: bolaCandidateMaxPerObject, TTLSeconds: int64(bolaCandidateDefaultTTL / time.Second), MinBaselineObs: bolaCandidateMinBaselineObs,
		EnumerationWindowSec: int64(bolaCandidateEnumerationWindow / time.Second), EnumerationMinObjs: bolaCandidateEnumerationMinObjs,
		DetectionOnly: true, Generated: generated,
	}
}

func (s *bolaDetectionStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	state := bolaCandidateStateFile{Version: bolaCandidateStateVersion, Saved: time.Now().UTC(), Candidates: s.snapshot()}
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-bola-candidates.json"), state)
}

func (s *bolaDetectionStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state bolaCandidateStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-bola-candidates.json"), &state); err != nil {
		return err
	}
	if state.Version != 0 && state.Version != bolaCandidateStateVersion {
		return fmt.Errorf("unsupported BOLA candidate state version %d", state.Version)
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range state.Candidates {
		if len(s.candidates) >= bolaCandidateMaxCandidates || !validBOLACandidate(row, now) || s.countIdentityLocked(row.IdentityFingerprint) >= bolaCandidateMaxPerIdentity || s.countObjectLocked(row.LocatorID, row.ObjectFingerprint) >= bolaCandidateMaxPerObject {
			continue
		}
		cp := cloneBOLACandidate(row)
		s.candidates[row.ID] = &cp
	}
	return nil
}

func (a *adminServer) handleBOLACandidates(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.bolaCandidates.snapshot()
	a.audit.add(who(r).user, "api_bola.candidates_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}

func (a *adminServer) handleBOLAStatus(w http.ResponseWriter, r *http.Request) {
	status := a.srv.bolaCandidates.status()
	a.audit.add(who(r).user, "api_bola.status_view", fmt.Sprintf("count=%d", status.CandidateCount))
	writeJSON(w, status)
}
