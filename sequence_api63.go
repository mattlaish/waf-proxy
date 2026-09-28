package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	sequenceDetectionLearn  = "LEARN"
	sequenceDetectionDetect = "DETECT"

	sequenceAnomalyUnknownTransition    = "UNKNOWN_TRANSITION"
	sequenceAnomalyPrerequisiteSkipped  = "PREREQUISITE_SKIPPED"
	sequenceAnomalyUnexpectedEntryPoint = "UNEXPECTED_ENTRY_POINT"
	sequenceAnomalySequenceReversal     = "SEQUENCE_REVERSAL"
	sequenceAnomalyAbnormalRepetition   = "ABNORMAL_REPETITION"
	sequenceAnomalyWorkflowDivergence   = "WORKFLOW_DIVERGENCE"

	sequenceDefaultViolationTTL                           = 7 * 24 * time.Hour
	sequenceDefaultMaxViolations                          = 2048
	sequenceDefaultMaxExceptions                          = 256
	sequenceDefaultEntryConfidenceThreshold               = 0.50
	sequenceDefaultDivergenceExpectedMinConfidence        = 0.75
	sequenceDefaultDivergenceObservedMaxConfidence        = 0.01
	sequenceDefaultRepetitionTailProbability              = 0.01
	sequenceDefaultRepetitionMinimumRun            uint64 = 3
)

// SequenceViolation is bounded API-6.3 DETECT-only evidence. It contains only
// normalized operation IDs and keyed workflow/session fingerprints. It is not
// an enforcement decision and must never be translated into BLOCK/DENY/403 by
// the sequence engine.
type SequenceViolation struct {
	ID                   string    `json:"id"`
	Time                 time.Time `json:"time"`
	Type                 string    `json:"type"`
	Site                 string    `json:"site"`
	WorkflowID           string    `json:"workflow_id"`
	SessionFingerprint   string    `json:"session_fingerprint"`
	IdentityKind         string    `json:"identity_kind"`
	FromOperationID      string    `json:"from_operation_id,omitempty"`
	ToOperationID        string    `json:"to_operation_id,omitempty"`
	RelatedOperationID   string    `json:"related_operation_id,omitempty"`
	ModelMaturity        string    `json:"model_maturity"`
	TransitionConfidence float64   `json:"transition_confidence,omitempty"`
	ExpectedConfidence   float64   `json:"expected_confidence,omitempty"`
	Deviation            float64   `json:"deviation,omitempty"`
	ObservedRunLength    uint64    `json:"observed_run_length,omitempty"`
	EvidenceObservations uint64    `json:"evidence_observations,omitempty"`
	EvidenceSessions     uint64    `json:"evidence_sessions,omitempty"`
	ExpiresAt            time.Time `json:"expires_at"`
}

// SequenceException is the API-6.3 exception primitive consumed by detection.
// API-6.4 will own operator CRUD/RBAC workflows. Empty operation fields are
// wildcards; raw URLs, object IDs, tokens, cookies and claims are never valid
// exception selectors.
type SequenceException struct {
	ID                 string    `json:"id"`
	Site               string    `json:"site"`
	WorkflowID         string    `json:"workflow_id,omitempty"`
	Type               string    `json:"type,omitempty"`
	FromOperationID    string    `json:"from_operation_id,omitempty"`
	ToOperationID      string    `json:"to_operation_id,omitempty"`
	RelatedOperationID string    `json:"related_operation_id,omitempty"`
	Enabled            bool      `json:"enabled"`
	ExpiresAt          time.Time `json:"expires_at,omitempty"`
}

type sequenceDetectionContext struct {
	Workflow              *sequenceWorkflowRuntime
	WorkflowMaturity      string
	FromOperationID       string
	ToOperationID         string
	Direct                SequenceTransition
	DirectExists          bool
	Outgoing              []SequenceTransition
	SourceTotals          map[string]uint64
	MaxOutgoingConfidence float64
	OutgoingObservations  uint64
	OutgoingSessions      uint64
}

func sequenceDetectionState(maturity string) string {
	if maturity == sequenceMaturityMature {
		return sequenceDetectionDetect
	}
	return sequenceDetectionLearn
}

func validSequenceAnomalyType(v string) bool {
	switch v {
	case sequenceAnomalyUnknownTransition,
		sequenceAnomalyPrerequisiteSkipped,
		sequenceAnomalyUnexpectedEntryPoint,
		sequenceAnomalySequenceReversal,
		sequenceAnomalyAbnormalRepetition,
		sequenceAnomalyWorkflowDivergence:
		return true
	default:
		return false
	}
}

func sequenceViolationID(v SequenceViolation) string {
	material := strings.Join([]string{
		v.Time.UTC().Format(time.RFC3339Nano), v.Type, v.Site, v.WorkflowID,
		v.SessionFingerprint, v.FromOperationID, v.ToOperationID,
		v.RelatedOperationID, fmt.Sprintf("%d", v.ObservedRunLength),
	}, "\x00")
	h := sha256.Sum256([]byte(material))
	return hex.EncodeToString(h[:20])
}

func addSequenceCount(total, count uint64) uint64 {
	if ^uint64(0)-total < count {
		return ^uint64(0)
	}
	return total + count
}

func (s *sequenceStore) derivedTransitionWithTotalLocked(edge SequenceTransition, total uint64, now time.Time) SequenceTransition {
	if total > 0 {
		edge.Confidence = float64(edge.ObservationCount) / float64(total)
	} else {
		edge.Confidence = 0
	}
	edge.Maturity = maturityFor(now, edge.FirstSeen, edge.LastSeen, edge.ObservationCount, edge.SessionCount, edge.Confidence, s.settings)
	return edge
}

func (s *sequenceStore) derivedTransitionLocked(edge SequenceTransition, now time.Time) SequenceTransition {
	var total uint64
	for _, candidate := range s.transitions {
		if candidate.WorkflowID == edge.WorkflowID && candidate.FromOperationID == edge.FromOperationID {
			total = addSequenceCount(total, candidate.ObservationCount)
		}
	}
	return s.derivedTransitionWithTotalLocked(edge, total, now)
}

func (s *sequenceStore) workflowMaturityLocked(workflowID string, now time.Time) string {
	w := s.workflows[workflowID]
	if w == nil {
		return sequenceMaturityLearning
	}
	return maturityFor(now, w.FirstSeen, w.LastSeen, w.ObservationCount, w.SessionCount, 1, s.settings)
}

func (s *sequenceStore) detectionContextLocked(workflowID, from, to string, now time.Time) sequenceDetectionContext {
	ctx := sequenceDetectionContext{
		Workflow:         s.workflows[workflowID],
		WorkflowMaturity: s.workflowMaturityLocked(workflowID, now),
		FromOperationID:  from,
		ToOperationID:    to,
		SourceTotals:     make(map[string]uint64),
	}
	if from == "" {
		return ctx
	}
	for _, edge := range s.transitions {
		if edge.WorkflowID != workflowID {
			continue
		}
		ctx.SourceTotals[edge.FromOperationID] = addSequenceCount(ctx.SourceTotals[edge.FromOperationID], edge.ObservationCount)
	}
	for _, edge := range s.transitions {
		if edge.WorkflowID != workflowID || edge.FromOperationID != from {
			continue
		}
		derived := s.derivedTransitionWithTotalLocked(edge, ctx.SourceTotals[from], now)
		ctx.Outgoing = append(ctx.Outgoing, derived)
		if derived.Confidence > ctx.MaxOutgoingConfidence {
			ctx.MaxOutgoingConfidence = derived.Confidence
		}
		ctx.OutgoingObservations = addSequenceCount(ctx.OutgoingObservations, derived.ObservationCount)
		if derived.SessionCount > ctx.OutgoingSessions {
			ctx.OutgoingSessions = derived.SessionCount
		}
		if derived.ToOperationID == to {
			ctx.Direct = derived
			ctx.DirectExists = true
		}
	}
	sort.Slice(ctx.Outgoing, func(i, j int) bool {
		if ctx.Outgoing[i].Confidence != ctx.Outgoing[j].Confidence {
			return ctx.Outgoing[i].Confidence > ctx.Outgoing[j].Confidence
		}
		return ctx.Outgoing[i].ToOperationID < ctx.Outgoing[j].ToOperationID
	})
	return ctx
}

func (s *sequenceStore) transitionEvidenceReadyLocked(edge SequenceTransition, now time.Time, requireConfidence bool) bool {
	if edge.ObservationCount < s.settings.MinObservations || edge.SessionCount < s.settings.MinSessions || edge.FirstSeen.IsZero() {
		return false
	}
	if now.Sub(edge.FirstSeen) < s.settings.MinLearningDuration || (!edge.LastSeen.IsZero() && now.Sub(edge.LastSeen) >= s.settings.StaleAfter) {
		return false
	}
	if requireConfidence && edge.Confidence < s.settings.ConfidenceThreshold {
		return false
	}
	return true
}

func (s *sequenceStore) sourceBaselineReadyLocked(ctx sequenceDetectionContext, now time.Time) bool {
	if ctx.Workflow == nil || ctx.WorkflowMaturity != sequenceMaturityMature || ctx.OutgoingObservations < s.settings.MinObservations || ctx.OutgoingSessions < s.settings.MinSessions {
		return false
	}
	for _, edge := range ctx.Outgoing {
		if s.transitionEvidenceReadyLocked(edge, now, true) {
			return true
		}
	}
	return false
}

func consecutiveOperationRun(recent []string, operationID string) uint64 {
	var run uint64 = 1
	for i := len(recent) - 1; i >= 0; i-- {
		if recent[i] != operationID {
			break
		}
		if run < ^uint64(0) {
			run++
		}
	}
	return run
}

func (s *sequenceStore) exceptionMatchesLocked(v SequenceException, candidate SequenceViolation, now time.Time) bool {
	if !v.Enabled || strings.TrimSpace(v.Site) == "" || v.Site != candidate.Site {
		return false
	}
	if !v.ExpiresAt.IsZero() && !v.ExpiresAt.After(now) {
		return false
	}
	if v.WorkflowID != "" && v.WorkflowID != candidate.WorkflowID {
		return false
	}
	if v.Type != "" && v.Type != candidate.Type {
		return false
	}
	if v.FromOperationID != "" && v.FromOperationID != candidate.FromOperationID {
		return false
	}
	if v.ToOperationID != "" && v.ToOperationID != candidate.ToOperationID {
		return false
	}
	if v.RelatedOperationID != "" && v.RelatedOperationID != candidate.RelatedOperationID {
		return false
	}
	return true
}

func (s *sequenceStore) exceptedLocked(candidate SequenceViolation, now time.Time) bool {
	for _, v := range s.exceptions {
		if s.exceptionMatchesLocked(v, candidate, now) {
			return true
		}
	}
	return false
}

func (s *sequenceStore) appendViolationLocked(v SequenceViolation) {
	if v.Time.IsZero() || !validSequenceAnomalyType(v.Type) || v.Site == "" || len(v.WorkflowID) != 40 || len(v.SessionFingerprint) != 40 {
		return
	}
	v.ModelMaturity = sequenceMaturityMature
	v.ExpiresAt = v.Time.Add(s.settings.ViolationTTL)
	v.ID = sequenceViolationID(v)
	if s.exceptedLocked(v, v.Time) {
		return
	}
	if len(s.violations) >= s.settings.MaxViolations {
		copy(s.violations, s.violations[len(s.violations)-s.settings.MaxViolations+1:])
		s.violations = s.violations[:s.settings.MaxViolations-1]
		s.evictedViolations.Add(1)
	}
	s.violations = append(s.violations, v)
}

func (s *sequenceStore) detectUnexpectedEntryLocked(ev sequenceObservation) {
	if s.sequenceModeLocked(ev.Site) != sequenceDetectionDetect {
		return
	}
	w := s.workflows[ev.WorkflowID]
	if w == nil || s.workflowMaturityLocked(ev.WorkflowID, ev.At) != sequenceMaturityMature || w.SessionCount < s.settings.MinSessions {
		return
	}
	var known uint64
	for op, count := range w.EntryOperations {
		if op == ev.OperationID {
			return
		}
		if ^uint64(0)-known < count {
			known = ^uint64(0)
		} else {
			known += count
		}
	}
	if known < s.settings.MinSessions || w.SessionCount == 0 {
		return
	}
	maxConfidence := 0.0
	for _, count := range w.EntryOperations {
		confidence := float64(count) / float64(w.SessionCount)
		if confidence > maxConfidence {
			maxConfidence = confidence
		}
	}
	if maxConfidence < s.settings.EntryConfidenceThreshold {
		return
	}
	s.appendViolationLocked(SequenceViolation{
		Time: ev.At, Type: sequenceAnomalyUnexpectedEntryPoint, Site: ev.Site,
		WorkflowID: ev.WorkflowID, SessionFingerprint: ev.SessionKey, IdentityKind: ev.IdentityKind,
		ToOperationID: ev.OperationID, ExpectedConfidence: maxConfidence,
		EvidenceObservations: w.ObservationCount, EvidenceSessions: w.SessionCount,
	})
}

func (s *sequenceStore) findSkippedPrerequisiteLocked(ctx sequenceDetectionContext, now time.Time) (string, SequenceTransition, bool) {
	for _, first := range ctx.Outgoing {
		if first.ToOperationID == ctx.ToOperationID || !s.transitionEvidenceReadyLocked(first, now, true) {
			continue
		}
		id := sequenceWorkflowTransitionID(first.Site, first.WorkflowID, first.ToOperationID, ctx.ToOperationID)
		second, ok := s.transitions[id]
		if !ok {
			continue
		}
		second = s.derivedTransitionWithTotalLocked(second, ctx.SourceTotals[first.ToOperationID], now)
		if s.transitionEvidenceReadyLocked(second, now, true) {
			return first.ToOperationID, second, true
		}
	}
	return "", SequenceTransition{}, false
}

func (s *sequenceStore) detectTransitionAnomaliesLocked(ev sequenceObservation, session SequenceSession, from string) {
	if s.sequenceModeLocked(ev.Site) != sequenceDetectionDetect {
		return
	}
	if from == "" {
		return
	}
	ctx := s.detectionContextLocked(ev.WorkflowID, from, ev.OperationID, ev.At)
	if ctx.Workflow == nil || ctx.WorkflowMaturity != sequenceMaturityMature {
		return
	}
	sourceReady := s.sourceBaselineReadyLocked(ctx, ev.At)

	if sourceReady {
		run := consecutiveOperationRun(session.RecentOperationIDs, ev.OperationID)
		if from == ev.OperationID && run >= s.settings.RepetitionMinimumRun {
			selfConfidence := 0.0
			selfReady := false
			if ctx.DirectExists {
				selfConfidence = ctx.Direct.Confidence
				selfReady = s.transitionEvidenceReadyLocked(ctx.Direct, ev.At, false)
			}
			tail := 0.0
			if selfReady && selfConfidence > 0 {
				tail = math.Pow(selfConfidence, float64(run-1))
			}
			if ((!ctx.DirectExists || !selfReady) || tail < s.settings.RepetitionTailProbability) &&
				ctx.MaxOutgoingConfidence >= s.settings.ConfidenceThreshold {
				s.appendViolationLocked(SequenceViolation{
					Time: ev.At, Type: sequenceAnomalyAbnormalRepetition, Site: ev.Site,
					WorkflowID: ev.WorkflowID, SessionFingerprint: ev.SessionKey, IdentityKind: ev.IdentityKind,
					FromOperationID: from, ToOperationID: ev.OperationID, TransitionConfidence: selfConfidence,
					ExpectedConfidence: ctx.MaxOutgoingConfidence, ObservedRunLength: run,
					EvidenceObservations: ctx.OutgoingObservations, EvidenceSessions: ctx.OutgoingSessions,
				})
			}
		}

		if ctx.DirectExists {
			if s.transitionEvidenceReadyLocked(ctx.Direct, ev.At, false) &&
				ctx.MaxOutgoingConfidence >= s.settings.DivergenceExpectedMinConfidence &&
				ctx.Direct.Confidence <= s.settings.DivergenceObservedMaxConfidence &&
				ctx.Direct.Confidence < ctx.MaxOutgoingConfidence {
				s.appendViolationLocked(SequenceViolation{
					Time: ev.At, Type: sequenceAnomalyWorkflowDivergence, Site: ev.Site,
					WorkflowID: ev.WorkflowID, SessionFingerprint: ev.SessionKey, IdentityKind: ev.IdentityKind,
					FromOperationID: from, ToOperationID: ev.OperationID,
					TransitionConfidence: ctx.Direct.Confidence, ExpectedConfidence: ctx.MaxOutgoingConfidence,
					Deviation:            ctx.MaxOutgoingConfidence - ctx.Direct.Confidence,
					EvidenceObservations: ctx.OutgoingObservations, EvidenceSessions: ctx.OutgoingSessions,
				})
			}
			return
		}
	} else if ctx.DirectExists {
		return
	}

	// Reversal is supported by the mature reverse edge itself and does not
	// require the current source operation to have an outgoing baseline.
	reverseID := sequenceWorkflowTransitionID(ev.Site, ev.WorkflowID, ev.OperationID, from)
	if reverse, ok := s.transitions[reverseID]; ok {
		reverse = s.derivedTransitionWithTotalLocked(reverse, ctx.SourceTotals[ev.OperationID], ev.At)
		if s.transitionEvidenceReadyLocked(reverse, ev.At, true) {
			s.appendViolationLocked(SequenceViolation{
				Time: ev.At, Type: sequenceAnomalySequenceReversal, Site: ev.Site,
				WorkflowID: ev.WorkflowID, SessionFingerprint: ev.SessionKey, IdentityKind: ev.IdentityKind,
				FromOperationID: from, ToOperationID: ev.OperationID, RelatedOperationID: reverse.FromOperationID,
				ExpectedConfidence:   reverse.Confidence,
				EvidenceObservations: reverse.ObservationCount, EvidenceSessions: reverse.SessionCount,
			})
			return
		}
	}

	if !sourceReady {
		return
	}
	if prerequisite, second, ok := s.findSkippedPrerequisiteLocked(ctx, ev.At); ok {
		s.appendViolationLocked(SequenceViolation{
			Time: ev.At, Type: sequenceAnomalyPrerequisiteSkipped, Site: ev.Site,
			WorkflowID: ev.WorkflowID, SessionFingerprint: ev.SessionKey, IdentityKind: ev.IdentityKind,
			FromOperationID: from, ToOperationID: ev.OperationID, RelatedOperationID: prerequisite,
			ExpectedConfidence:   math.Min(ctx.MaxOutgoingConfidence, second.Confidence),
			EvidenceObservations: ctx.OutgoingObservations, EvidenceSessions: ctx.OutgoingSessions,
		})
		return
	}

	s.appendViolationLocked(SequenceViolation{
		Time: ev.At, Type: sequenceAnomalyUnknownTransition, Site: ev.Site,
		WorkflowID: ev.WorkflowID, SessionFingerprint: ev.SessionKey, IdentityKind: ev.IdentityKind,
		FromOperationID: from, ToOperationID: ev.OperationID, ExpectedConfidence: ctx.MaxOutgoingConfidence,
		EvidenceObservations: ctx.OutgoingObservations, EvidenceSessions: ctx.OutgoingSessions,
	})
}

func validSequenceViolation(v SequenceViolation, now time.Time) bool {
	if len(v.ID) != 40 || len(v.WorkflowID) != 40 || len(v.SessionFingerprint) != 40 || strings.TrimSpace(v.Site) == "" ||
		!validSequenceAnomalyType(v.Type) || v.Time.IsZero() || !v.ExpiresAt.After(now) || v.ModelMaturity != sequenceMaturityMature {
		return false
	}
	for _, op := range []string{v.FromOperationID, v.ToOperationID, v.RelatedOperationID} {
		if op != "" && !validSequenceOperationID(op) {
			return false
		}
	}
	for _, confidence := range []float64{v.TransitionConfidence, v.ExpectedConfidence} {
		if confidence < 0 || confidence > 1 {
			return false
		}
	}
	_, idErr := hex.DecodeString(v.ID)
	_, workflowErr := hex.DecodeString(v.WorkflowID)
	_, sessionErr := hex.DecodeString(v.SessionFingerprint)
	return idErr == nil && workflowErr == nil && sessionErr == nil
}

func validSequenceException(v SequenceException, now time.Time) bool {
	if len(v.ID) != 40 || strings.TrimSpace(v.Site) == "" || (!v.ExpiresAt.IsZero() && !v.ExpiresAt.After(now)) {
		return false
	}
	if v.WorkflowID != "" {
		if len(v.WorkflowID) != 40 {
			return false
		}
		if _, err := hex.DecodeString(v.WorkflowID); err != nil {
			return false
		}
	}
	if v.Type != "" && !validSequenceAnomalyType(v.Type) {
		return false
	}
	for _, op := range []string{v.FromOperationID, v.ToOperationID, v.RelatedOperationID} {
		if op != "" && !validSequenceOperationID(op) {
			return false
		}
	}
	_, err := hex.DecodeString(v.ID)
	return err == nil
}

func (s *sequenceStore) pruneDetectionLocked(now time.Time) {
	if len(s.violations) > 0 {
		out := s.violations[:0]
		for _, v := range s.violations {
			if v.ExpiresAt.After(now) {
				out = append(out, v)
			}
		}
		s.violations = out
	}
	for id, v := range s.exceptions {
		if !v.ExpiresAt.IsZero() && !v.ExpiresAt.After(now) {
			delete(s.exceptions, id)
		}
	}
}

func cloneSequenceViolations(in []SequenceViolation) []SequenceViolation {
	return append([]SequenceViolation(nil), in...)
}

func cloneSequenceExceptions(in []SequenceException) []SequenceException {
	return append([]SequenceException(nil), in...)
}

func (a *adminServer) handleSequenceViolations(w http.ResponseWriter, r *http.Request) {
	model := a.srv.sequence.snapshot()
	a.audit.add(who(r).user, "api_sequence.violations_view", "count="+itoa(len(model.Violations)))
	writeJSON(w, model.Violations)
}
