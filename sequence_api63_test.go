package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func api63TestSettings() sequenceSettings {
	s := api62TestSettings()
	s.MinObservations = 2
	s.MinSessions = 2
	s.MinLearningDuration = time.Second
	s.ConfidenceThreshold = 0.20
	s.MaxViolations = 32
	s.MaxExceptions = 8
	s.ViolationTTL = time.Hour
	s.EntryConfidenceThreshold = 0.50
	s.DivergenceExpectedMinConfidence = 0.75
	s.DivergenceObservedMaxConfidence = 0.01
	s.RepetitionTailProbability = 0.01
	s.RepetitionMinimumRun = 3
	return s
}

func api63TrainPath(s *sequenceStore, workflow, site string, ops []string, sessions int, start time.Time) time.Time {
	at := start
	for i := 0; i < sessions; i++ {
		session := fmt.Sprintf("train-%s-%d", workflow, i)
		for j, op := range ops {
			s.process(api62Observation(s, session, workflow, site, op, at.Add(time.Duration(j)*100*time.Millisecond)))
		}
		at = at.Add(2 * time.Second)
	}
	if _, err := s.setSequenceMode(site, sequenceDetectionDetect, at); err != nil {
		panic("API-6.3 test baseline did not mature before DETECT: " + err.Error())
	}
	return at
}

func api63ViolationsByType(model SequenceModel, anomalyType string) []SequenceViolation {
	var out []SequenceViolation
	for _, v := range model.Violations {
		if v.Type == anomalyType {
			out = append(out, v)
		}
	}
	return out
}

func TestAPI63UnknownTransitionRequiresMatureBaselineAndNeverEnforces(t *testing.T) {
	settings := api63TestSettings()
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 4, 0, 0, 0, time.UTC)
	a := apiOperationID("unknown", http.MethodPost, "/login")
	b := apiOperationID("unknown", http.MethodGet, "/profile")
	c := apiOperationID("unknown", http.MethodPost, "/admin-export")

	// Cold start: an unseen transition is still learning evidence, not anomaly evidence.
	s.process(api62Observation(s, "cold", "unknown", "unknown", a, start))
	s.process(api62Observation(s, "cold", "unknown", "unknown", c, start.Add(100*time.Millisecond)))
	if got := len(s.snapshot().Violations); got != 0 {
		t.Fatalf("cold-start request produced %d violations", got)
	}

	// Mature only the intended A->B path, then exercise a new A->C transition.
	s = testSequenceStore(settings)
	next := api63TrainPath(s, "unknown", "unknown", []string{a, b}, 3, start)
	s.process(api62Observation(s, "probe", "unknown", "unknown", a, next))
	s.process(api62Observation(s, "probe", "unknown", "unknown", c, next.Add(100*time.Millisecond)))
	model := s.snapshot()
	unknown := api63ViolationsByType(model, sequenceAnomalyUnknownTransition)
	if len(unknown) != 1 {
		t.Fatalf("UNKNOWN_TRANSITION violations=%d want 1: %#v", len(unknown), model.Violations)
	}
	if unknown[0].ModelMaturity != sequenceMaturityMature || unknown[0].ExpectedConfidence < settings.ConfidenceThreshold {
		t.Fatalf("unknown-transition evidence not maturity/confidence guarded: %#v", unknown[0])
	}
	// The sequence engine has no response writer or enforcement action in its detector.
	if strings.Contains(fmt.Sprintf("%#v", unknown[0]), "403") {
		t.Fatal("sequence violation unexpectedly carried enforcement status")
	}
}

func TestAPI63PrerequisiteSkipAndReversalUseMaturePaths(t *testing.T) {
	settings := api63TestSettings()
	start := time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC)
	a := apiOperationID("flow", http.MethodPost, "/create-order")
	b := apiOperationID("flow", http.MethodPost, "/confirm-order")
	c := apiOperationID("flow", http.MethodPost, "/payment")

	skip := testSequenceStore(settings)
	next := api63TrainPath(skip, "flow", "flow", []string{a, b, c}, 3, start)
	skip.process(api62Observation(skip, "skip", "flow", "flow", a, next))
	skip.process(api62Observation(skip, "skip", "flow", "flow", c, next.Add(100*time.Millisecond)))
	model := skip.snapshot()
	skipped := api63ViolationsByType(model, sequenceAnomalyPrerequisiteSkipped)
	if len(skipped) != 1 || skipped[0].RelatedOperationID != b {
		t.Fatalf("prerequisite skip evidence unexpected: %#v", model.Violations)
	}
	if len(api63ViolationsByType(model, sequenceAnomalyUnknownTransition)) != 0 {
		t.Fatal("specific prerequisite skip was duplicated as generic unknown transition")
	}

	reverse := testSequenceStore(settings)
	next = api63TrainPath(reverse, "reverse", "flow", []string{a, b}, 3, start)
	reverse.process(api62Observation(reverse, "reverse-probe", "reverse", "flow", b, next))
	reverse.process(api62Observation(reverse, "reverse-probe", "reverse", "flow", a, next.Add(100*time.Millisecond)))
	model = reverse.snapshot()
	if got := len(api63ViolationsByType(model, sequenceAnomalySequenceReversal)); got != 1 {
		t.Fatalf("SEQUENCE_REVERSAL violations=%d want 1: %#v", got, model.Violations)
	}
	if len(api63ViolationsByType(model, sequenceAnomalyUnknownTransition)) != 0 {
		t.Fatal("specific reversal was duplicated as generic unknown transition")
	}
}

func TestAPI63UnexpectedEntryAndExceptionSuppression(t *testing.T) {
	settings := api63TestSettings()
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 6, 0, 0, 0, time.UTC)
	a := apiOperationID("entry", http.MethodPost, "/login")
	b := apiOperationID("entry", http.MethodGet, "/profile")
	payment := apiOperationID("entry", http.MethodPost, "/payment/confirm")
	next := api63TrainPath(s, "entry", "entry", []string{a, b}, 3, start)

	exceptionID := s.digest("exception-entry-payment")
	s.stateMu.Lock()
	workflowID := s.digest("workflow\x00entry")
	s.exceptions[exceptionID] = SequenceException{
		ID: exceptionID, Site: "entry", WorkflowID: workflowID,
		Type: sequenceAnomalyUnexpectedEntryPoint, ToOperationID: payment,
		Enabled: true, ExpiresAt: next.Add(time.Hour),
	}
	s.stateMu.Unlock()

	s.process(api62Observation(s, "exception-session", "entry", "entry", payment, next))
	if got := len(api63ViolationsByType(s.snapshot(), sequenceAnomalyUnexpectedEntryPoint)); got != 0 {
		t.Fatalf("exception did not suppress unexpected entry: %d", got)
	}

	s.stateMu.Lock()
	delete(s.exceptions, exceptionID)
	s.stateMu.Unlock()
	otherEntry := apiOperationID("entry", http.MethodPost, "/admin/export")
	s.process(api62Observation(s, "no-exception-session", "entry", "entry", otherEntry, next.Add(2*time.Second)))
	if got := len(api63ViolationsByType(s.snapshot(), sequenceAnomalyUnexpectedEntryPoint)); got != 1 {
		t.Fatalf("unexpected entry violations=%d want 1", got)
	}
}

func TestAPI63AbnormalRepetitionUsesSessionHistoryAndLearnedProbability(t *testing.T) {
	settings := api63TestSettings()
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC)
	otp := apiOperationID("repeat", http.MethodPost, "/otp/verify")
	done := apiOperationID("repeat", http.MethodGet, "/account")
	next := api63TrainPath(s, "repeat", "repeat", []string{otp, done}, 3, start)

	s.process(api62Observation(s, "attacker", "repeat", "repeat", otp, next))
	s.process(api62Observation(s, "attacker", "repeat", "repeat", otp, next.Add(100*time.Millisecond)))
	s.process(api62Observation(s, "attacker", "repeat", "repeat", otp, next.Add(200*time.Millisecond)))
	model := s.snapshot()
	repeated := api63ViolationsByType(model, sequenceAnomalyAbnormalRepetition)
	if len(repeated) != 1 || repeated[0].ObservedRunLength != 3 {
		t.Fatalf("abnormal repetition evidence unexpected: %#v", model.Violations)
	}
	if repeated[0].ExpectedConfidence < settings.ConfidenceThreshold {
		t.Fatalf("repetition anomaly lacks learned baseline confidence: %#v", repeated[0])
	}
}

func TestAPI63WorkflowDivergenceRequiresLearnedRareEdgeNotZeroProbability(t *testing.T) {
	settings := api63TestSettings()
	settings.MinObservations = 2
	settings.MinSessions = 2
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	a := apiOperationID("diverge", http.MethodGet, "/account")
	common := apiOperationID("diverge", http.MethodGet, "/orders")
	rare := apiOperationID("diverge", http.MethodPost, "/password-reset")

	at := api63TrainPath(s, "diverge", "diverge", []string{a, common}, 198, start)
	for i := 0; i < 2; i++ {
		session := fmt.Sprintf("rare-learn-%d", i)
		s.process(api62Observation(s, session, "diverge", "diverge", a, at))
		s.process(api62Observation(s, session, "diverge", "diverge", rare, at.Add(100*time.Millisecond)))
		at = at.Add(2 * time.Second)
	}
	// Discard detection evidence produced while intentionally seeding the rare edge;
	// its learned counts remain and the next observation evaluates non-zero probability.
	s.stateMu.Lock()
	s.violations = nil
	s.stateMu.Unlock()

	s.process(api62Observation(s, "rare-probe", "diverge", "diverge", a, at))
	s.process(api62Observation(s, "rare-probe", "diverge", "diverge", rare, at.Add(100*time.Millisecond)))
	model := s.snapshot()
	divergence := api63ViolationsByType(model, sequenceAnomalyWorkflowDivergence)
	if len(divergence) != 1 {
		t.Fatalf("WORKFLOW_DIVERGENCE violations=%d want 1: %#v", len(divergence), model.Violations)
	}
	if divergence[0].TransitionConfidence <= 0 || divergence[0].ExpectedConfidence < settings.DivergenceExpectedMinConfidence || divergence[0].Deviation <= 0 {
		t.Fatalf("divergence did not require learned non-zero edge evidence: %#v", divergence[0])
	}
}

func TestAPI63BoundedEvidenceRestartPersistenceAndPrivacy(t *testing.T) {
	settings := api63TestSettings()
	settings.MaxViolations = 3
	settings.MaxExceptions = 2
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	a := apiOperationID("bounded63", http.MethodGet, "/start")
	b := apiOperationID("bounded63", http.MethodGet, "/expected")
	next := api63TrainPath(s, "bounded63", "bounded63", []string{a, b}, 3, start)

	for i := 0; i < 10; i++ {
		session := fmt.Sprintf("private-cookie-%d", i)
		unexpected := apiOperationID("bounded63", http.MethodPost, fmt.Sprintf("/unexpected/op-x%x", i))
		s.process(api62Observation(s, session, "bounded63", "bounded63", a, next.Add(time.Duration(i)*time.Second)))
		s.process(api62Observation(s, session, "bounded63", "bounded63", unexpected, next.Add(time.Duration(i)*time.Second+100*time.Millisecond)))
	}
	model := s.snapshot()
	if len(model.Violations) != settings.MaxViolations || model.EvictedViolations == 0 {
		t.Fatalf("violation evidence not bounded: len=%d evicted=%d", len(model.Violations), model.EvictedViolations)
	}

	exceptionID := s.digest("persisted-exception")
	s.stateMu.Lock()
	s.exceptions[exceptionID] = SequenceException{ID: exceptionID, Site: "bounded63", Type: sequenceAnomalyUnknownTransition, Enabled: true, ExpiresAt: next.Add(time.Hour)}
	s.stateMu.Unlock()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	s.now = func() time.Time { return next.Add(20 * time.Second) }
	if err := s.save(configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "api-sequence.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-cookie", "Authorization", "Bearer ", "raw JWT", "subject"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("durable API-6.3 evidence leaked forbidden raw material %q", forbidden)
		}
	}

	restored := newSequenceStoreWithSettings(settings, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	restored.now = func() time.Time { return next.Add(20 * time.Second) }
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	after := restored.snapshot()
	if len(after.Violations) != settings.MaxViolations || len(after.Exceptions) != 1 || after.Version != sequenceStateVersion {
		t.Fatalf("API-6.3 durable state did not restore: %#v", after)
	}
}

func TestAPI63ConcurrentDetectionSnapshotsRemainBounded(t *testing.T) {
	settings := api63TestSettings()
	settings.MaxViolations = 16
	settings.MaxSessions = 64
	settings.MaxTransitions = 128
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	a := apiOperationID("race63", http.MethodGet, "/start")
	b := apiOperationID("race63", http.MethodGet, "/expected")
	next := api63TrainPath(s, "race63", "race63", []string{a, b}, 3, start)

	const workers = 32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			at := next.Add(time.Duration(i) * time.Millisecond)
			unexpected := apiOperationID("race63", http.MethodPost, fmt.Sprintf("/unexpected/op-x%x", i))
			s.process(api62Observation(s, fmt.Sprintf("race63-%d", i), "race63", "race63", a, at))
			s.process(api62Observation(s, fmt.Sprintf("race63-%d", i), "race63", "race63", unexpected, at.Add(time.Microsecond)))
			_ = s.snapshot()
		}(i)
	}
	wg.Wait()
	model := s.snapshot()
	if len(model.Violations) > settings.MaxViolations || len(model.Sessions) > settings.MaxSessions || len(model.Transitions) > settings.MaxTransitions {
		t.Fatalf("concurrent API-6.3 state exceeded bounds: violations=%d sessions=%d transitions=%d", len(model.Violations), len(model.Sessions), len(model.Transitions))
	}
	if len(model.Violations) == 0 {
		t.Fatal("concurrent detector produced no evidence")
	}
}
