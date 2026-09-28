package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	bolaPolicyStateVersion = 1
	bolaPolicyMaxRules     = 1024
	bolaReviewMaxRecords   = 4096
	bolaPolicyDefaultTTL   = 30 * 24 * time.Hour
	bolaPolicyMaxTTL       = 180 * 24 * time.Hour
	bolaReviewDefaultTTL   = 30 * 24 * time.Hour

	bolaPolicyActionReview   = "REVIEW"
	bolaPolicyActionSuppress = "SUPPRESS"

	bolaWorkflowOpen         = "OPEN"
	bolaWorkflowAcknowledged = "ACKNOWLEDGED"
	bolaWorkflowDismissed    = "DISMISSED"
	bolaWorkflowResolved     = "RESOLVED"
)

var bolaWorkflowReasonCodes = map[string]struct{}{
	"":                         {},
	"INVESTIGATING":            {},
	"EXPECTED_SHARED_RESOURCE": {},
	"AUTHORIZED_CROSS_TENANT":  {},
	"TEST_TRAFFIC":             {},
	"FALSE_POSITIVE":           {},
	"FIX_DEPLOYED":             {},
	"OTHER_REVIEWED":           {},
}

// BOLAPolicy is an evidence-handling policy, not a request authorization or
// enforcement policy. It can make API-7.3 candidates visible for review or
// suppress them from the actionable queue, but it never writes BLOCK/DENY/403
// decisions into the request path.
type BOLAPolicy struct {
	ID            string    `json:"id"`
	OperationID   string    `json:"operation_id"`
	LocatorID     string    `json:"locator_id,omitempty"`
	CandidateType string    `json:"candidate_type,omitempty"`
	MinConfidence string    `json:"min_confidence"`
	Action        string    `json:"action"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// BOLAEvidenceReview is bounded operator workflow metadata. It references only
// an API-7.3 candidate ID and enumerated reason code; it does not accept free-
// text notes or raw identity/object selectors.
type BOLAEvidenceReview struct {
	CandidateID           string    `json:"candidate_id"`
	State                 string    `json:"state"`
	ReasonCode            string    `json:"reason_code,omitempty"`
	EvidenceCountAtReview uint64    `json:"evidence_count_at_review"`
	CandidateLastSeen     time.Time `json:"candidate_last_seen"`
	UpdatedAt             time.Time `json:"updated_at"`
	ExpiresAt             time.Time `json:"expires_at"`
}

type BOLAEvidenceView struct {
	Candidate       BOLACandidate `json:"candidate"`
	Policy          *BOLAPolicy   `json:"policy,omitempty"`
	EffectiveAction string        `json:"effective_action"`
	WorkflowState   string        `json:"workflow_state"`
	ReasonCode      string        `json:"reason_code,omitempty"`
	Reopened        bool          `json:"reopened"`
	Suppressed      bool          `json:"suppressed"`
	ReviewedAt      time.Time     `json:"reviewed_at,omitempty"`
}

type BOLAPolicyStatus struct {
	PolicyCount       int  `json:"policy_count"`
	ReviewCount       int  `json:"review_count"`
	EvidenceCount     int  `json:"evidence_count"`
	SuppressedCount   int  `json:"suppressed_count"`
	ReopenedCount     int  `json:"reopened_count"`
	MaxPolicies       int  `json:"max_policies"`
	MaxReviews        int  `json:"max_reviews"`
	InferenceBlocking bool `json:"inference_blocking"`
}

type bolaPolicyStateFile struct {
	Version  int                  `json:"version"`
	Saved    time.Time            `json:"saved"`
	Policies []BOLAPolicy         `json:"policies"`
	Reviews  []BOLAEvidenceReview `json:"reviews"`
}

type bolaPolicyStore struct {
	mu       sync.RWMutex
	policies map[string]*BOLAPolicy
	reviews  map[string]*BOLAEvidenceReview
	now      func() time.Time
}

func newBOLAPolicyStore() *bolaPolicyStore {
	return &bolaPolicyStore{
		policies: map[string]*BOLAPolicy{},
		reviews:  map[string]*BOLAEvidenceReview{},
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func bolaPolicyID(operationID, locatorID, candidateType, minConfidence string) string {
	h := sha256.Sum256([]byte("api74-bola-policy\x00" + operationID + "\x00" + locatorID + "\x00" + candidateType + "\x00" + minConfidence))
	return hex.EncodeToString(h[:20])
}

func validBOLACandidateType(v string) bool {
	return v == "" || v == bolaCandidateIdentityDivergence || v == bolaCandidateTenantDivergence || v == bolaCandidateEnumeration
}

func validBOLAConfidence(v string) bool {
	return v == bolaConfidenceMedium || v == bolaConfidenceHigh
}

func confidenceRank(v string) int {
	switch v {
	case bolaConfidenceHigh:
		return 2
	case bolaConfidenceMedium:
		return 1
	default:
		return 0
	}
}

func validBOLAPolicy(p BOLAPolicy, now time.Time) bool {
	if len(p.ID) != 40 || !validSequenceOperationID(p.OperationID) || (p.LocatorID != "" && len(p.LocatorID) != 32) ||
		!validBOLACandidateType(p.CandidateType) || !validBOLAConfidence(p.MinConfidence) ||
		(p.Action != bolaPolicyActionReview && p.Action != bolaPolicyActionSuppress) || p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() || !p.ExpiresAt.After(now) {
		return false
	}
	for _, digest := range []string{p.ID, p.OperationID, p.LocatorID} {
		if digest == "" {
			continue
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	return !p.ExpiresAt.After(p.UpdatedAt.Add(bolaPolicyMaxTTL + time.Second))
}

func validBOLAWorkflowState(v string) bool {
	return v == bolaWorkflowOpen || v == bolaWorkflowAcknowledged || v == bolaWorkflowDismissed || v == bolaWorkflowResolved
}

func validBOLAEvidenceReview(v BOLAEvidenceReview, now time.Time) bool {
	if len(v.CandidateID) != 40 || !validBOLAWorkflowState(v.State) || v.EvidenceCountAtReview == 0 || v.CandidateLastSeen.IsZero() || v.UpdatedAt.IsZero() || !v.ExpiresAt.After(now) {
		return false
	}
	if _, ok := bolaWorkflowReasonCodes[v.ReasonCode]; !ok {
		return false
	}
	_, err := hex.DecodeString(v.CandidateID)
	return err == nil
}

func (s *bolaPolicyStore) pruneLocked(now time.Time) {
	for id, p := range s.policies {
		if p == nil || !p.ExpiresAt.After(now) {
			delete(s.policies, id)
		}
	}
	for id, r := range s.reviews {
		if r == nil || !r.ExpiresAt.After(now) {
			delete(s.reviews, id)
		}
	}
}

func (s *bolaPolicyStore) locatorBelongsToOperation(locatorID, operationID string, locators *objectLocatorStore) bool {
	if locatorID == "" {
		return true
	}
	if locators == nil {
		return false
	}
	for _, locator := range locators.snapshot() {
		if locator.ID == locatorID && locator.OperationID == operationID {
			return true
		}
	}
	return false
}

func (s *bolaPolicyStore) upsertPolicy(req BOLAPolicy, ttl time.Duration, locators *objectLocatorStore) (BOLAPolicy, error) {
	if s == nil {
		return BOLAPolicy{}, errors.New("BOLA policy store unavailable")
	}
	req.OperationID = strings.TrimSpace(req.OperationID)
	req.LocatorID = strings.TrimSpace(req.LocatorID)
	req.CandidateType = strings.TrimSpace(req.CandidateType)
	req.MinConfidence = strings.TrimSpace(req.MinConfidence)
	req.Action = strings.TrimSpace(req.Action)
	if !validSequenceOperationID(req.OperationID) {
		return BOLAPolicy{}, errors.New("operation_id must be a normalized API operation ID")
	}
	if req.LocatorID != "" {
		if len(req.LocatorID) != 32 {
			return BOLAPolicy{}, errors.New("locator_id must be an API-7.1 locator ID")
		}
		if _, err := hex.DecodeString(req.LocatorID); err != nil || !s.locatorBelongsToOperation(req.LocatorID, req.OperationID, locators) {
			return BOLAPolicy{}, errors.New("locator_id must belong to operation_id")
		}
	}
	if !validBOLACandidateType(req.CandidateType) {
		return BOLAPolicy{}, errors.New("unsupported candidate_type")
	}
	if req.MinConfidence == "" {
		req.MinConfidence = bolaConfidenceMedium
	}
	if !validBOLAConfidence(req.MinConfidence) {
		return BOLAPolicy{}, errors.New("min_confidence must be MEDIUM or HIGH")
	}
	if req.Action != bolaPolicyActionReview && req.Action != bolaPolicyActionSuppress {
		return BOLAPolicy{}, errors.New("action must be REVIEW or SUPPRESS")
	}
	if ttl == 0 {
		ttl = bolaPolicyDefaultTTL
	}
	if ttl < time.Minute || ttl > bolaPolicyMaxTTL {
		return BOLAPolicy{}, fmt.Errorf("ttl must be between 60 seconds and %d seconds", int64(bolaPolicyMaxTTL/time.Second))
	}
	now := s.now().UTC()
	id := bolaPolicyID(req.OperationID, req.LocatorID, req.CandidateType, req.MinConfidence)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	p := s.policies[id]
	if p == nil {
		if len(s.policies) >= bolaPolicyMaxRules {
			return BOLAPolicy{}, errors.New("BOLA policy capacity reached")
		}
		p = &BOLAPolicy{ID: id, OperationID: req.OperationID, LocatorID: req.LocatorID, CandidateType: req.CandidateType, MinConfidence: req.MinConfidence, CreatedAt: now}
		s.policies[id] = p
	}
	p.Action = req.Action
	p.Enabled = req.Enabled
	p.UpdatedAt = now
	p.ExpiresAt = now.Add(ttl)
	return *p, nil
}

func (s *bolaPolicyStore) deletePolicy(id string) (BOLAPolicy, error) {
	if s == nil {
		return BOLAPolicy{}, errors.New("BOLA policy store unavailable")
	}
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.policies[id]
	if p == nil {
		return BOLAPolicy{}, errors.New("BOLA policy not found")
	}
	out := *p
	delete(s.policies, id)
	return out, nil
}

func (s *bolaPolicyStore) policiesSnapshot() []BOLAPolicy {
	if s == nil {
		return []BOLAPolicy{}
	}
	now := s.now().UTC()
	s.mu.Lock()
	s.pruneLocked(now)
	out := make([]BOLAPolicy, 0, len(s.policies))
	for _, p := range s.policies {
		out = append(out, *p)
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].OperationID != out[j].OperationID {
			return out[i].OperationID < out[j].OperationID
		}
		if out[i].LocatorID != out[j].LocatorID {
			return out[i].LocatorID < out[j].LocatorID
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func policySpecificity(p BOLAPolicy) int {
	n := 0
	if p.LocatorID != "" {
		n += 2
	}
	if p.CandidateType != "" {
		n++
	}
	return n
}

func policyMatchesCandidate(p BOLAPolicy, c BOLACandidate, now time.Time) bool {
	if !p.Enabled || !p.ExpiresAt.After(now) || p.OperationID != c.OperationID || confidenceRank(c.Confidence) < confidenceRank(p.MinConfidence) {
		return false
	}
	if p.LocatorID != "" && p.LocatorID != c.LocatorID {
		return false
	}
	return p.CandidateType == "" || p.CandidateType == c.Type
}

func (s *bolaPolicyStore) matchingPolicyLocked(c BOLACandidate, now time.Time) *BOLAPolicy {
	var best *BOLAPolicy
	for _, p := range s.policies {
		if p == nil || !policyMatchesCandidate(*p, c, now) {
			continue
		}
		if best == nil || policySpecificity(*p) > policySpecificity(*best) ||
			(policySpecificity(*p) == policySpecificity(*best) && p.UpdatedAt.After(best.UpdatedAt)) ||
			(policySpecificity(*p) == policySpecificity(*best) && p.UpdatedAt.Equal(best.UpdatedAt) && p.ID < best.ID) {
			cp := *p
			best = &cp
		}
	}
	return best
}

func (s *bolaPolicyStore) reviewCandidate(candidate BOLACandidate, state, reasonCode string) (BOLAEvidenceReview, error) {
	if s == nil {
		return BOLAEvidenceReview{}, errors.New("BOLA policy store unavailable")
	}
	state = strings.TrimSpace(state)
	reasonCode = strings.TrimSpace(reasonCode)
	if !validBOLAWorkflowState(state) {
		return BOLAEvidenceReview{}, errors.New("unsupported workflow state")
	}
	if _, ok := bolaWorkflowReasonCodes[reasonCode]; !ok {
		return BOLAEvidenceReview{}, errors.New("unsupported reason_code")
	}
	if state != bolaWorkflowOpen && reasonCode == "" {
		return BOLAEvidenceReview{}, errors.New("reason_code is required for reviewed workflow states")
	}
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	if _, exists := s.reviews[candidate.ID]; !exists && len(s.reviews) >= bolaReviewMaxRecords {
		return BOLAEvidenceReview{}, errors.New("BOLA review capacity reached")
	}
	r := &BOLAEvidenceReview{
		CandidateID: candidate.ID, State: state, ReasonCode: reasonCode,
		EvidenceCountAtReview: candidate.EvidenceCount, CandidateLastSeen: candidate.LastSeen,
		UpdatedAt: now, ExpiresAt: now.Add(bolaReviewDefaultTTL),
	}
	s.reviews[candidate.ID] = r
	return *r, nil
}

func candidateByID(store *bolaDetectionStore, id string) (BOLACandidate, bool) {
	if store == nil {
		return BOLACandidate{}, false
	}
	for _, c := range store.snapshot() {
		if c.ID == id {
			return c, true
		}
	}
	return BOLACandidate{}, false
}

func (s *bolaPolicyStore) evidenceSnapshot(candidates *bolaDetectionStore) []BOLAEvidenceView {
	if s == nil || candidates == nil {
		return []BOLAEvidenceView{}
	}
	now := s.now().UTC()
	rows := candidates.snapshot()
	s.mu.Lock()
	s.pruneLocked(now)
	out := make([]BOLAEvidenceView, 0, len(rows))
	for _, c := range rows {
		view := BOLAEvidenceView{Candidate: c, EffectiveAction: bolaPolicyActionReview, WorkflowState: bolaWorkflowOpen}
		if p := s.matchingPolicyLocked(c, now); p != nil {
			view.Policy = p
			view.EffectiveAction = p.Action
			view.Suppressed = p.Action == bolaPolicyActionSuppress
		}
		if r := s.reviews[c.ID]; r != nil {
			view.WorkflowState = r.State
			view.ReasonCode = r.ReasonCode
			view.ReviewedAt = r.UpdatedAt
			if (r.State == bolaWorkflowDismissed || r.State == bolaWorkflowResolved) && c.LastSeen.After(r.CandidateLastSeen) {
				view.WorkflowState = bolaWorkflowOpen
				view.Reopened = true
			}
		}
		out = append(out, view)
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Suppressed != out[j].Suppressed {
			return !out[i].Suppressed
		}
		if !out[i].Candidate.LastSeen.Equal(out[j].Candidate.LastSeen) {
			return out[i].Candidate.LastSeen.After(out[j].Candidate.LastSeen)
		}
		return out[i].Candidate.ID < out[j].Candidate.ID
	})
	return out
}

func (s *bolaPolicyStore) status(candidates *bolaDetectionStore) BOLAPolicyStatus {
	evidence := s.evidenceSnapshot(candidates)
	status := BOLAPolicyStatus{EvidenceCount: len(evidence), MaxPolicies: bolaPolicyMaxRules, MaxReviews: bolaReviewMaxRecords, InferenceBlocking: false}
	for _, row := range evidence {
		if row.Suppressed {
			status.SuppressedCount++
		}
		if row.Reopened {
			status.ReopenedCount++
		}
	}
	s.mu.RLock()
	status.PolicyCount = len(s.policies)
	status.ReviewCount = len(s.reviews)
	s.mu.RUnlock()
	return status
}

func (s *bolaPolicyStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	now := s.now().UTC()
	s.mu.Lock()
	s.pruneLocked(now)
	state := bolaPolicyStateFile{Version: bolaPolicyStateVersion, Saved: time.Now().UTC(), Policies: make([]BOLAPolicy, 0, len(s.policies)), Reviews: make([]BOLAEvidenceReview, 0, len(s.reviews))}
	for _, p := range s.policies {
		state.Policies = append(state.Policies, *p)
	}
	for _, r := range s.reviews {
		state.Reviews = append(state.Reviews, *r)
	}
	s.mu.Unlock()
	sort.Slice(state.Policies, func(i, j int) bool { return state.Policies[i].ID < state.Policies[j].ID })
	sort.Slice(state.Reviews, func(i, j int) bool { return state.Reviews[i].CandidateID < state.Reviews[j].CandidateID })
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-bola-policy.json"), state)
}

func (s *bolaPolicyStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state bolaPolicyStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-bola-policy.json"), &state); err != nil {
		return err
	}
	if state.Version != 0 && state.Version != bolaPolicyStateVersion {
		return fmt.Errorf("unsupported BOLA policy state version %d", state.Version)
	}
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range state.Policies {
		if len(s.policies) >= bolaPolicyMaxRules || !validBOLAPolicy(p, now) {
			continue
		}
		cp := p
		s.policies[p.ID] = &cp
	}
	for _, r := range state.Reviews {
		if len(s.reviews) >= bolaReviewMaxRecords || !validBOLAEvidenceReview(r, now) {
			continue
		}
		cp := r
		s.reviews[r.CandidateID] = &cp
	}
	return nil
}

func (a *adminServer) persistBOLAPolicyMutation(w http.ResponseWriter) bool {
	if a == nil || a.srv == nil || a.srv.bolaPolicy == nil {
		http.Error(w, "BOLA policy store unavailable", http.StatusServiceUnavailable)
		return false
	}
	if err := a.srv.bolaPolicy.save(a.srv.configPath); err != nil {
		http.Error(w, "BOLA policy persistence failed: "+err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func decodeAPI74JSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func (a *adminServer) handleBOLAPolicies(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.bolaPolicy.policiesSnapshot()
	a.audit.add(who(r).user, "api_bola.policies_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}

func (a *adminServer) handleBOLAPolicyUpsert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OperationID   string `json:"operation_id"`
		LocatorID     string `json:"locator_id"`
		CandidateType string `json:"candidate_type"`
		MinConfidence string `json:"min_confidence"`
		Action        string `json:"action"`
		Enabled       bool   `json:"enabled"`
		TTLSeconds    int64  `json:"ttl_seconds"`
	}
	if err := decodeAPI74JSON(w, r, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	p, err := a.srv.bolaPolicy.upsertPolicy(BOLAPolicy{OperationID: req.OperationID, LocatorID: req.LocatorID, CandidateType: req.CandidateType, MinConfidence: req.MinConfidence, Action: req.Action, Enabled: req.Enabled}, time.Duration(req.TTLSeconds)*time.Second, a.srv.objectLocators)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if !a.persistBOLAPolicyMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_bola.policy_upsert", p.OperationID+" "+p.Action+" policy="+p.ID)
	writeJSON(w, p)
}

func (a *adminServer) handleBOLAPolicyDelete(w http.ResponseWriter, r *http.Request) {
	p, err := a.srv.bolaPolicy.deletePolicy(r.PathValue("policy_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !a.persistBOLAPolicyMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_bola.policy_delete", p.OperationID+" policy="+p.ID)
	writeJSON(w, map[string]any{"ok": true, "id": p.ID})
}

func (a *adminServer) handleBOLAEvidence(w http.ResponseWriter, r *http.Request) {
	rows := a.srv.bolaPolicy.evidenceSnapshot(a.srv.bolaCandidates)
	a.audit.add(who(r).user, "api_bola.evidence_view", fmt.Sprintf("count=%d", len(rows)))
	writeJSON(w, rows)
}

func (a *adminServer) handleBOLAEvidenceWorkflow(w http.ResponseWriter, r *http.Request) {
	candidateID := strings.TrimSpace(r.PathValue("candidate_id"))
	candidate, ok := candidateByID(a.srv.bolaCandidates, candidateID)
	if !ok {
		http.Error(w, "BOLA candidate not found", http.StatusNotFound)
		return
	}
	var req struct {
		State      string `json:"state"`
		ReasonCode string `json:"reason_code"`
	}
	if err := decodeAPI74JSON(w, r, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	review, err := a.srv.bolaPolicy.reviewCandidate(candidate, req.State, req.ReasonCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if !a.persistBOLAPolicyMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_bola.evidence_workflow", candidate.ID+" -> "+review.State+" reason="+review.ReasonCode)
	writeJSON(w, review)
}

func (a *adminServer) handleBOLAPolicyStatus(w http.ResponseWriter, r *http.Request) {
	status := a.srv.bolaPolicy.status(a.srv.bolaCandidates)
	a.audit.add(who(r).user, "api_bola.policy_status_view", fmt.Sprintf("policies=%d evidence=%d", status.PolicyCount, status.EvidenceCount))
	writeJSON(w, status)
}
