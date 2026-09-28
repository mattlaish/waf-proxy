package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	sequenceDefaultExceptionTTL = 24 * time.Hour
	sequenceMaximumExceptionTTL = 30 * 24 * time.Hour
)

// SequenceSiteControl is the explicit API-6.4 operator control plane for one
// site. Only LEARN and DETECT are valid. There is deliberately no ENFORCE
// mode: learned sequence evidence can never become request-blocking authority.
type SequenceSiteControl struct {
	Site       string    `json:"site"`
	Mode       string    `json:"mode"`
	Generation uint64    `json:"generation"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SequenceRecentSession is a bounded, privacy-preserving summary retained
// after an active session expires or is capacity-evicted. It contains only
// keyed fingerprints and normalized API-1 operation IDs.
type SequenceRecentSession struct {
	SessionFingerprint  string    `json:"session_fingerprint"`
	Site                string    `json:"site"`
	WorkflowID          string    `json:"workflow_id"`
	CorrelationKind     string    `json:"correlation_kind"`
	EntryOperationID    string    `json:"entry_operation_id,omitempty"`
	TerminalOperationID string    `json:"terminal_operation_id,omitempty"`
	RequestCount        uint64    `json:"request_count"`
	StartedAt           time.Time `json:"started_at"`
	EndedAt             time.Time `json:"ended_at"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// SequenceLearningState is the bounded operational read model used by the
// API-6.4 console. Counts are derived from the immutable sequence snapshot.
type SequenceLearningState struct {
	Site              string    `json:"site"`
	Mode              string    `json:"mode"`
	Generation        uint64    `json:"generation"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
	WorkflowCount     int       `json:"workflow_count"`
	LearningWorkflows int       `json:"learning_workflows"`
	MatureWorkflows   int       `json:"mature_workflows"`
	StaleWorkflows    int       `json:"stale_workflows"`
	TransitionCount   int       `json:"transition_count"`
	ActiveSessions    int       `json:"active_sessions"`
	RecentSessions    int       `json:"recent_sessions"`
	ViolationCount    int       `json:"violation_count"`
	ExceptionCount    int       `json:"exception_count"`
}

// SequenceResetResult records the bounded scope of a reset/relearn action.
// Explicit exceptions are intentionally preserved because they are operator
// policy, not learned state. Reset/relearn always returns the site to LEARN.
type SequenceResetResult struct {
	Site                string    `json:"site"`
	Mode                string    `json:"mode"`
	Generation          uint64    `json:"generation"`
	ResetAt             time.Time `json:"reset_at"`
	RemovedSessions     int       `json:"removed_sessions"`
	RemovedRecent       int       `json:"removed_recent_sessions"`
	RemovedTransitions  int       `json:"removed_transitions"`
	RemovedWorkflows    int       `json:"removed_workflows"`
	RemovedViolations   int       `json:"removed_violations"`
	PreservedExceptions int       `json:"preserved_exceptions"`
}

func validSequenceSite(site string) bool {
	site = strings.TrimSpace(site)
	if site == "" || len(site) > 256 {
		return false
	}
	for _, r := range site {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func validSequenceMode(mode string) bool {
	return mode == sequenceDetectionLearn || mode == sequenceDetectionDetect
}

func (s *sequenceStore) sequenceModeLocked(site string) string {
	if c, ok := s.controls[site]; ok && validSequenceMode(c.Mode) {
		return c.Mode
	}
	return sequenceDetectionLearn
}

func (s *sequenceStore) siteControlLocked(site string) SequenceSiteControl {
	if c, ok := s.controls[site]; ok {
		return c
	}
	return SequenceSiteControl{Site: site, Mode: sequenceDetectionLearn}
}

func (s *sequenceStore) setSequenceMode(site, mode string, now time.Time) (SequenceSiteControl, error) {
	site = strings.TrimSpace(site)
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if !validSequenceSite(site) {
		return SequenceSiteControl{}, fmt.Errorf("site is required and must be bounded")
	}
	if !validSequenceMode(mode) {
		return SequenceSiteControl{}, fmt.Errorf("sequence mode must be LEARN or DETECT")
	}
	if now.IsZero() {
		now = s.now()
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if _, exists := s.controls[site]; !exists && len(s.controls) >= s.settings.MaxWorkflows {
		return SequenceSiteControl{}, fmt.Errorf("maximum sequence site controls reached")
	}
	if mode == sequenceDetectionDetect {
		mature := false
		for id, w := range s.workflows {
			if w != nil && w.Site == site && s.workflowMaturityLocked(id, now) == sequenceMaturityMature {
				mature = true
				break
			}
		}
		if !mature {
			return SequenceSiteControl{}, fmt.Errorf("DETECT requires at least one MATURE workflow for site")
		}
	}
	c := s.siteControlLocked(site)
	c.Site = site
	c.Mode = mode
	c.UpdatedAt = now.UTC()
	if c.Generation == 0 {
		c.Generation = 1
	}
	s.controls[site] = c
	return c, nil
}

func (s *sequenceStore) recordRecentSessionLocked(session SequenceSession, endedAt time.Time) {
	if session.Key == "" || !validSequenceSite(session.Site) || session.WorkflowID == "" || session.CurrentOperationID == "" {
		return
	}
	if endedAt.IsZero() {
		endedAt = s.now()
	}
	v := SequenceRecentSession{
		SessionFingerprint:  session.Key,
		Site:                session.Site,
		WorkflowID:          session.WorkflowID,
		CorrelationKind:     session.CorrelationKind,
		EntryOperationID:    session.EntryOperationID,
		TerminalOperationID: session.CurrentOperationID,
		RequestCount:        session.RequestCount,
		StartedAt:           session.StartedAt,
		EndedAt:             endedAt.UTC(),
		ExpiresAt:           endedAt.UTC().Add(s.settings.RecentSessionTTL),
	}
	if len(s.recentSessions) >= s.settings.MaxRecentSessions {
		copy(s.recentSessions, s.recentSessions[len(s.recentSessions)-s.settings.MaxRecentSessions+1:])
		s.recentSessions = s.recentSessions[:s.settings.MaxRecentSessions-1]
		s.evictedRecentSessions.Add(1)
	}
	s.recentSessions = append(s.recentSessions, v)
}

func validSequenceRecentSession(v SequenceRecentSession, now time.Time) bool {
	if len(v.SessionFingerprint) != 40 || len(v.WorkflowID) != 40 || !validSequenceSite(v.Site) ||
		(v.CorrelationKind != "anonymous" && v.CorrelationKind != "verified_identity") ||
		!validSequenceOperationID(v.TerminalOperationID) || !v.ExpiresAt.After(now) || v.EndedAt.IsZero() || v.StartedAt.IsZero() || v.EndedAt.Before(v.StartedAt) {
		return false
	}
	if v.EntryOperationID != "" && !validSequenceOperationID(v.EntryOperationID) {
		return false
	}
	return true
}

func (s *sequenceStore) pruneOperationsLocked(now time.Time) {
	if len(s.recentSessions) > 0 {
		out := s.recentSessions[:0]
		for _, v := range s.recentSessions {
			if v.ExpiresAt.After(now) {
				out = append(out, v)
			}
		}
		s.recentSessions = out
	}
}

func (s *sequenceStore) resetAndRelearn(site string, now time.Time) (SequenceResetResult, error) {
	site = strings.TrimSpace(site)
	if !validSequenceSite(site) {
		return SequenceResetResult{}, fmt.Errorf("site is required and must be bounded")
	}
	if now.IsZero() {
		now = s.now()
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if _, exists := s.controls[site]; !exists && len(s.controls) >= s.settings.MaxWorkflows {
		return SequenceResetResult{}, fmt.Errorf("maximum sequence site controls reached")
	}
	result := SequenceResetResult{Site: site, Mode: sequenceDetectionLearn, ResetAt: now.UTC()}
	for id, v := range s.sessions {
		if v.Site == site {
			delete(s.sessions, id)
			result.RemovedSessions++
		}
	}
	for id, v := range s.transitions {
		if v.Site == site {
			delete(s.transitions, id)
			result.RemovedTransitions++
		}
	}
	for id, v := range s.workflows {
		if v != nil && v.Site == site {
			delete(s.workflows, id)
			result.RemovedWorkflows++
		}
	}
	if len(s.recentSessions) > 0 {
		out := s.recentSessions[:0]
		for _, v := range s.recentSessions {
			if v.Site == site {
				result.RemovedRecent++
				continue
			}
			out = append(out, v)
		}
		s.recentSessions = out
	}
	if len(s.violations) > 0 {
		out := s.violations[:0]
		for _, v := range s.violations {
			if v.Site == site {
				result.RemovedViolations++
				continue
			}
			out = append(out, v)
		}
		s.violations = out
	}
	for _, v := range s.exceptions {
		if v.Site == site {
			result.PreservedExceptions++
		}
	}
	c := s.siteControlLocked(site)
	c.Site = site
	c.Mode = sequenceDetectionLearn
	c.UpdatedAt = now.UTC()
	if c.Generation < ^uint64(0) {
		c.Generation++
	}
	if c.Generation == 0 {
		c.Generation = 1
	}
	s.controls[site] = c
	result.Generation = c.Generation
	return result, nil
}

func (s *sequenceStore) createSequenceException(v SequenceException, ttl time.Duration, now time.Time) (SequenceException, error) {
	v.Site = strings.TrimSpace(v.Site)
	v.Type = strings.TrimSpace(v.Type)
	v.WorkflowID = strings.TrimSpace(v.WorkflowID)
	v.FromOperationID = strings.TrimSpace(v.FromOperationID)
	v.ToOperationID = strings.TrimSpace(v.ToOperationID)
	v.RelatedOperationID = strings.TrimSpace(v.RelatedOperationID)
	if !validSequenceSite(v.Site) || !validSequenceAnomalyType(v.Type) {
		return SequenceException{}, fmt.Errorf("site and a valid anomaly type are required")
	}
	if v.WorkflowID == "" && v.FromOperationID == "" && v.ToOperationID == "" && v.RelatedOperationID == "" {
		return SequenceException{}, fmt.Errorf("exception must include workflow or normalized operation scope")
	}
	if ttl <= 0 {
		ttl = sequenceDefaultExceptionTTL
	}
	if ttl > sequenceMaximumExceptionTTL {
		return SequenceException{}, fmt.Errorf("exception ttl exceeds maximum")
	}
	if now.IsZero() {
		now = s.now()
	}
	v.Enabled = true
	v.ExpiresAt = now.UTC().Add(ttl)
	v.ID = s.digest(strings.Join([]string{"sequence-exception", v.Site, v.WorkflowID, v.Type, v.FromOperationID, v.ToOperationID, v.RelatedOperationID, now.UTC().Format(time.RFC3339Nano)}, "\x00"))
	if !validSequenceException(v, now) {
		return SequenceException{}, fmt.Errorf("invalid sequence exception selector")
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if len(s.exceptions) >= s.settings.MaxExceptions {
		return SequenceException{}, fmt.Errorf("maximum sequence exceptions reached")
	}
	s.exceptions[v.ID] = v
	return v, nil
}

func (s *sequenceStore) deleteSequenceException(id string) (SequenceException, error) {
	id = strings.TrimSpace(id)
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	v, ok := s.exceptions[id]
	if !ok {
		return SequenceException{}, fmt.Errorf("sequence exception not found")
	}
	delete(s.exceptions, id)
	return v, nil
}

func sequenceLearningStates(model SequenceModel) []SequenceLearningState {
	bySite := make(map[string]*SequenceLearningState)
	ensure := func(site string) *SequenceLearningState {
		v := bySite[site]
		if v == nil {
			v = &SequenceLearningState{Site: site, Mode: sequenceDetectionLearn}
			bySite[site] = v
		}
		return v
	}
	for _, c := range model.Controls {
		v := ensure(c.Site)
		v.Mode, v.Generation, v.UpdatedAt = c.Mode, c.Generation, c.UpdatedAt
	}
	for _, w := range model.Workflows {
		v := ensure(w.Site)
		v.WorkflowCount++
		switch w.Maturity {
		case sequenceMaturityMature:
			v.MatureWorkflows++
		case sequenceMaturityStale:
			v.StaleWorkflows++
		default:
			v.LearningWorkflows++
		}
	}
	for _, t := range model.Transitions {
		ensure(t.Site).TransitionCount++
	}
	for _, s := range model.Sessions {
		ensure(s.Site).ActiveSessions++
	}
	for _, s := range model.RecentSessions {
		ensure(s.Site).RecentSessions++
	}
	for _, v := range model.Violations {
		ensure(v.Site).ViolationCount++
	}
	for _, e := range model.Exceptions {
		ensure(e.Site).ExceptionCount++
	}
	out := make([]SequenceLearningState, 0, len(bySite))
	for _, v := range bySite {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Site < out[j].Site })
	return out
}

func cloneSequenceControls(in []SequenceSiteControl) []SequenceSiteControl {
	return append([]SequenceSiteControl(nil), in...)
}
func cloneSequenceRecentSessions(in []SequenceRecentSession) []SequenceRecentSession {
	return append([]SequenceRecentSession(nil), in...)
}

func (a *adminServer) persistSequenceMutation(w http.ResponseWriter) bool {
	if err := a.srv.sequence.save(a.srv.configPath); err != nil {
		http.Error(w, "sequence state persistence failed: "+err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func (a *adminServer) handleSequenceLearningState(w http.ResponseWriter, r *http.Request) {
	states := sequenceLearningStates(a.srv.sequence.snapshot())
	a.audit.add(who(r).user, "api_sequence.learning_state_view", fmt.Sprintf("sites=%d", len(states)))
	writeJSON(w, states)
}

func (a *adminServer) handleSequenceRecentSessions(w http.ResponseWriter, r *http.Request) {
	model := a.srv.sequence.snapshot()
	a.audit.add(who(r).user, "api_sequence.recent_sessions_view", fmt.Sprintf("count=%d", len(model.RecentSessions)))
	writeJSON(w, model.RecentSessions)
}

func (a *adminServer) handleSequenceExceptions(w http.ResponseWriter, r *http.Request) {
	model := a.srv.sequence.snapshot()
	a.audit.add(who(r).user, "api_sequence.exceptions_view", fmt.Sprintf("count=%d", len(model.Exceptions)))
	writeJSON(w, model.Exceptions)
}

func (a *adminServer) handleSequenceMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site string `json:"site"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	control, err := a.srv.sequence.setSequenceMode(req.Site, req.Mode, time.Time{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if !a.persistSequenceMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_sequence.mode", control.Site+" -> "+control.Mode)
	a.srv.sequence.publishSnapshot()
	writeJSON(w, control)
}

func (a *adminServer) handleSequenceRelearn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site string `json:"site"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	result, err := a.srv.sequence.resetAndRelearn(req.Site, time.Time{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !a.persistSequenceMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_sequence.relearn", fmt.Sprintf("site=%s generation=%d", result.Site, result.Generation))
	a.srv.sequence.publishSnapshot()
	writeJSON(w, result)
}

func (a *adminServer) handleSequenceExceptionCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Site               string `json:"site"`
		WorkflowID         string `json:"workflow_id"`
		Type               string `json:"type"`
		FromOperationID    string `json:"from_operation_id"`
		ToOperationID      string `json:"to_operation_id"`
		RelatedOperationID string `json:"related_operation_id"`
		TTLSeconds         int64  `json:"ttl_seconds"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	ex, err := a.srv.sequence.createSequenceException(SequenceException{Site: req.Site, WorkflowID: req.WorkflowID, Type: req.Type, FromOperationID: req.FromOperationID, ToOperationID: req.ToOperationID, RelatedOperationID: req.RelatedOperationID}, time.Duration(req.TTLSeconds)*time.Second, time.Time{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if !a.persistSequenceMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_sequence.exception_create", ex.Site+" exception="+ex.ID)
	a.srv.sequence.publishSnapshot()
	writeJSON(w, ex)
}

func (a *adminServer) handleSequenceExceptionDelete(w http.ResponseWriter, r *http.Request) {
	ex, err := a.srv.sequence.deleteSequenceException(r.PathValue("exception_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !a.persistSequenceMutation(w) {
		return
	}
	a.audit.add(who(r).user, "api_sequence.exception_delete", ex.Site+" exception="+ex.ID)
	a.srv.sequence.publishSnapshot()
	writeJSON(w, map[string]any{"ok": true, "id": ex.ID})
}
