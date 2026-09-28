package main

import (
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	sequenceMaturityLearning = "LEARNING"
	sequenceMaturityMature   = "MATURE"
	sequenceMaturityStale    = "STALE"
)

// SequenceWorkflowOperation is a bounded aggregate over normalized API-1
// operation IDs. It never contains raw path/query/object values.
type SequenceWorkflowOperation struct {
	OperationID string `json:"operation_id"`
	Count       uint64 `json:"count"`
}

// SequenceWorkflowModel is the API-6.2 learned workflow cohort. WorkflowID is
// a keyed fingerprint over site plus anonymous/authenticated verified identity
// context. Verified subject is intentionally excluded from the cohort so users
// do not create unbounded per-subject models.
type SequenceWorkflowModel struct {
	ID                 string                      `json:"id"`
	Site               string                      `json:"site"`
	IdentityKind       string                      `json:"identity_kind"`
	ObservationCount   uint64                      `json:"observation_count"`
	SessionCount       uint64                      `json:"session_count"`
	FirstSeen          time.Time                   `json:"first_seen"`
	LastSeen           time.Time                   `json:"last_seen"`
	MaxObservedDepth   uint64                      `json:"max_observed_depth"`
	EntryOperations    []SequenceWorkflowOperation `json:"entry_operations"`
	TerminalOperations []SequenceWorkflowOperation `json:"terminal_operations"`
	Maturity           string                      `json:"maturity"`
	DetectionState     string                      `json:"detection_state"`
}

type sequenceWorkflowRuntime struct {
	ID                 string
	Site               string
	IdentityKind       string
	ObservationCount   uint64
	SessionCount       uint64
	FirstSeen          time.Time
	LastSeen           time.Time
	MaxObservedDepth   uint64
	EntryOperations    map[string]uint64
	TerminalOperations map[string]uint64
}

func verifiedWorkflowMaterial(id VerifiedAPIIdentity) string {
	roles := append([]string(nil), id.Roles...)
	scopes := append([]string(nil), id.Scopes...)
	for i := range roles {
		roles[i] = strings.TrimSpace(roles[i])
	}
	for i := range scopes {
		scopes[i] = strings.TrimSpace(scopes[i])
	}
	sort.Strings(roles)
	sort.Strings(scopes)
	return strings.Join([]string{
		strings.TrimSpace(id.Issuer),
		strings.TrimSpace(id.TenantID),
		strings.TrimSpace(id.ClientID),
		strings.Join(roles, "\x1f"),
		strings.Join(scopes, "\x1f"),
	}, "\x00")
}

func containsSequenceID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func (s *sequenceStore) ensureWorkflowCapacityLocked(now time.Time) {
	if len(s.workflows) < s.settings.MaxWorkflows {
		return
	}
	oldestKey := ""
	var oldest time.Time
	for k, v := range s.workflows {
		if v == nil {
			oldestKey = k
			break
		}
		if oldestKey == "" || v.LastSeen.Before(oldest) || (v.LastSeen.Equal(oldest) && k < oldestKey) {
			oldestKey, oldest = k, v.LastSeen
		}
	}
	if oldestKey == "" {
		return
	}
	delete(s.workflows, oldestKey)
	s.evictedWorkflows.Add(1)
	for id, edge := range s.transitions {
		if edge.WorkflowID == oldestKey {
			delete(s.transitions, id)
			s.evictedTransitions.Add(1)
		}
	}
	for id, session := range s.sessions {
		if session.WorkflowID == oldestKey {
			s.recordRecentSessionLocked(session, now)
			delete(s.sessions, id)
			s.evictedSessions.Add(1)
		}
	}
}

func (s *sequenceStore) workflowLocked(ev sequenceObservation) *sequenceWorkflowRuntime {
	if w := s.workflows[ev.WorkflowID]; w != nil {
		return w
	}
	s.ensureWorkflowCapacityLocked(ev.At)
	w := &sequenceWorkflowRuntime{
		ID:                 ev.WorkflowID,
		Site:               ev.Site,
		IdentityKind:       ev.IdentityKind,
		FirstSeen:          ev.At,
		LastSeen:           ev.At,
		EntryOperations:    make(map[string]uint64),
		TerminalOperations: make(map[string]uint64),
	}
	s.workflows[ev.WorkflowID] = w
	return w
}

func boundedIncrementOperation(m map[string]uint64, operationID string, max int) {
	if operationID == "" || max <= 0 {
		return
	}
	if _, ok := m[operationID]; !ok && len(m) >= max {
		return
	}
	if m[operationID] < ^uint64(0) {
		m[operationID]++
	}
}

func (s *sequenceStore) noteWorkflowSessionStartLocked(ev sequenceObservation) {
	w := s.workflowLocked(ev)
	if w.SessionCount < ^uint64(0) {
		w.SessionCount++
	}
	if w.FirstSeen.IsZero() || ev.At.Before(w.FirstSeen) {
		w.FirstSeen = ev.At
	}
	if ev.At.After(w.LastSeen) {
		w.LastSeen = ev.At
	}
	boundedIncrementOperation(w.EntryOperations, ev.OperationID, s.settings.MaxWorkflowOperations)
}

func (s *sequenceStore) noteWorkflowObservationLocked(ev sequenceObservation, depth uint64) {
	w := s.workflowLocked(ev)
	if w.ObservationCount < ^uint64(0) {
		w.ObservationCount++
	}
	if depth > w.MaxObservedDepth {
		w.MaxObservedDepth = depth
	}
	if w.FirstSeen.IsZero() || ev.At.Before(w.FirstSeen) {
		w.FirstSeen = ev.At
	}
	if ev.At.After(w.LastSeen) {
		w.LastSeen = ev.At
	}
}

func (s *sequenceStore) finalizeSessionLocked(session SequenceSession, at time.Time) {
	if session.WorkflowID == "" || session.CurrentOperationID == "" {
		return
	}
	s.recordRecentSessionLocked(session, at)
	w := s.workflows[session.WorkflowID]
	if w == nil {
		return
	}
	boundedIncrementOperation(w.TerminalOperations, session.CurrentOperationID, s.settings.MaxWorkflowOperations)
	if at.After(w.LastSeen) {
		w.LastSeen = at
	}
}

func maturityFor(now, firstSeen, lastSeen time.Time, observations, sessions uint64, confidence float64, settings sequenceSettings) string {
	if !lastSeen.IsZero() && now.Sub(lastSeen) >= settings.StaleAfter {
		return sequenceMaturityStale
	}
	if observations >= settings.MinObservations && sessions >= settings.MinSessions &&
		!firstSeen.IsZero() && now.Sub(firstSeen) >= settings.MinLearningDuration && confidence >= settings.ConfidenceThreshold {
		return sequenceMaturityMature
	}
	return sequenceMaturityLearning
}

func (s *sequenceStore) refreshLearningDerivedLocked(now time.Time) {
	outgoing := make(map[string]uint64, len(s.transitions))
	for _, edge := range s.transitions {
		key := edge.WorkflowID + "\x00" + edge.FromOperationID
		outgoing[key] += edge.ObservationCount
	}
	for id, edge := range s.transitions {
		key := edge.WorkflowID + "\x00" + edge.FromOperationID
		if total := outgoing[key]; total > 0 {
			edge.Confidence = float64(edge.ObservationCount) / float64(total)
		} else {
			edge.Confidence = 0
		}
		edge.Maturity = maturityFor(now, edge.FirstSeen, edge.LastSeen, edge.ObservationCount, edge.SessionCount, edge.Confidence, s.settings)
		edge.Count = edge.ObservationCount
		s.transitions[id] = edge
	}
}

func operationMapSnapshot(m map[string]uint64) []SequenceWorkflowOperation {
	out := make([]SequenceWorkflowOperation, 0, len(m))
	for id, count := range m {
		out = append(out, SequenceWorkflowOperation{OperationID: id, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].OperationID < out[j].OperationID
	})
	return out
}

func (w *sequenceWorkflowRuntime) snapshot(settings sequenceSettings, now time.Time) SequenceWorkflowModel {
	maturity := maturityFor(now, w.FirstSeen, w.LastSeen, w.ObservationCount, w.SessionCount, 1, settings)
	return SequenceWorkflowModel{
		ID:                 w.ID,
		Site:               w.Site,
		IdentityKind:       w.IdentityKind,
		ObservationCount:   w.ObservationCount,
		SessionCount:       w.SessionCount,
		FirstSeen:          w.FirstSeen,
		LastSeen:           w.LastSeen,
		MaxObservedDepth:   w.MaxObservedDepth,
		EntryOperations:    operationMapSnapshot(w.EntryOperations),
		TerminalOperations: operationMapSnapshot(w.TerminalOperations),
		Maturity:           maturity,
		DetectionState:     sequenceDetectionState(maturity),
	}
}

func cloneSequenceWorkflowModels(in []SequenceWorkflowModel) []SequenceWorkflowModel {
	out := append([]SequenceWorkflowModel(nil), in...)
	for i := range out {
		out[i].EntryOperations = append([]SequenceWorkflowOperation(nil), in[i].EntryOperations...)
		out[i].TerminalOperations = append([]SequenceWorkflowOperation(nil), in[i].TerminalOperations...)
	}
	return out
}

func validSequenceWorkflow(v SequenceWorkflowModel, now time.Time, settings sequenceSettings) bool {
	if len(v.ID) != 40 || strings.TrimSpace(v.Site) == "" || (v.IdentityKind != "anonymous" && v.IdentityKind != "authenticated" && v.IdentityKind != "legacy") {
		return false
	}
	if v.FirstSeen.IsZero() || v.LastSeen.IsZero() || v.LastSeen.Before(v.FirstSeen) {
		return false
	}
	if now.Sub(v.LastSeen) >= settings.WorkflowTTL {
		return false
	}
	for _, list := range [][]SequenceWorkflowOperation{v.EntryOperations, v.TerminalOperations} {
		for _, op := range list {
			if !validSequenceOperationID(op.OperationID) {
				return false
			}
		}
	}
	return true
}

func workflowRuntimeFromModel(v SequenceWorkflowModel, settings sequenceSettings) *sequenceWorkflowRuntime {
	w := &sequenceWorkflowRuntime{
		ID:                 v.ID,
		Site:               v.Site,
		IdentityKind:       v.IdentityKind,
		ObservationCount:   v.ObservationCount,
		SessionCount:       v.SessionCount,
		FirstSeen:          v.FirstSeen,
		LastSeen:           v.LastSeen,
		MaxObservedDepth:   v.MaxObservedDepth,
		EntryOperations:    make(map[string]uint64),
		TerminalOperations: make(map[string]uint64),
	}
	for _, op := range v.EntryOperations {
		if len(w.EntryOperations) >= settings.MaxWorkflowOperations {
			break
		}
		w.EntryOperations[op.OperationID] = op.Count
	}
	for _, op := range v.TerminalOperations {
		if len(w.TerminalOperations) >= settings.MaxWorkflowOperations {
			break
		}
		w.TerminalOperations[op.OperationID] = op.Count
	}
	return w
}

func (s *sequenceStore) migrateV1StateLocked(state *sequenceStateFile, now time.Time) {
	if state == nil {
		return
	}
	workflowBySiteKind := make(map[string]string)
	for i := range state.Sessions {
		v := &state.Sessions[i]
		kind := "anonymous"
		if v.CorrelationKind == "verified_identity" {
			kind = "legacy"
		}
		key := v.Site + "\x00" + kind
		workflowID := workflowBySiteKind[key]
		if workflowID == "" {
			workflowID = s.digest("workflow\x00" + v.Site + "\x00legacy\x00" + kind)
			workflowBySiteKind[key] = workflowID
		}
		v.WorkflowID = workflowID
		if len(v.RecentOperationIDs) > 0 {
			v.EntryOperationID = v.RecentOperationIDs[0]
		}
		v.WorkflowDepth = v.RequestCount
		v.AbsoluteExpiresAt = v.StartedAt.Add(s.settings.SessionAbsoluteTTL)
		if v.ExpiresAt.After(v.AbsoluteExpiresAt) {
			v.ExpiresAt = v.AbsoluteExpiresAt
		}
		if v.ExpiresAt.After(now) {
			w := s.workflows[workflowID]
			if w == nil && len(s.workflows) < s.settings.MaxWorkflows {
				w = &sequenceWorkflowRuntime{ID: workflowID, Site: v.Site, IdentityKind: kind, FirstSeen: v.StartedAt, LastSeen: v.LastSeen, EntryOperations: make(map[string]uint64), TerminalOperations: make(map[string]uint64)}
				s.workflows[workflowID] = w
			}
			if w != nil {
				w.SessionCount++
				w.ObservationCount += v.RequestCount
				if v.WorkflowDepth > w.MaxObservedDepth {
					w.MaxObservedDepth = v.WorkflowDepth
				}
				boundedIncrementOperation(w.EntryOperations, v.EntryOperationID, s.settings.MaxWorkflowOperations)
			}
		}
	}
	legacyTransitionWorkflows := make(map[string]string)
	for i := range state.Transitions {
		v := &state.Transitions[i]
		legacyWorkflowID := legacyTransitionWorkflows[v.Site]
		if legacyWorkflowID == "" {
			legacyWorkflowID = s.digest("workflow\x00" + v.Site + "\x00legacy\x00transition")
			legacyTransitionWorkflows[v.Site] = legacyWorkflowID
		}
		v.WorkflowID = legacyWorkflowID
		v.ObservationCount = v.Count
		v.SessionCount = 0
		v.Maturity = sequenceMaturityLearning
		v.ID = sequenceWorkflowTransitionID(v.Site, v.WorkflowID, v.FromOperationID, v.ToOperationID)
		w := s.workflows[legacyWorkflowID]
		if w == nil && len(s.workflows) < s.settings.MaxWorkflows {
			w = &sequenceWorkflowRuntime{ID: legacyWorkflowID, Site: v.Site, IdentityKind: "legacy", FirstSeen: v.FirstSeen, LastSeen: v.LastSeen, EntryOperations: make(map[string]uint64), TerminalOperations: make(map[string]uint64)}
			s.workflows[legacyWorkflowID] = w
		}
		if w != nil {
			w.ObservationCount += v.Count
			if w.FirstSeen.IsZero() || (!v.FirstSeen.IsZero() && v.FirstSeen.Before(w.FirstSeen)) {
				w.FirstSeen = v.FirstSeen
			}
			if v.LastSeen.After(w.LastSeen) {
				w.LastSeen = v.LastSeen
			}
		}
	}
	if len(state.Workflows) == 0 {
		for _, w := range s.workflows {
			state.Workflows = append(state.Workflows, w.snapshot(s.settings, now))
		}
	}
}

func (a *adminServer) handleSequenceWorkflows(w http.ResponseWriter, r *http.Request) {
	model := a.srv.sequence.snapshot()
	a.audit.add(who(r).user, "api_sequence.workflows_view", "count="+itoa(len(model.Workflows)))
	writeJSON(w, model.Workflows)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
