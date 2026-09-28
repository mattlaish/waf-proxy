package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func api62TestSettings() sequenceSettings {
	s := defaultSequenceSettings()
	s.SessionTTL = time.Minute
	s.SessionAbsoluteTTL = time.Hour
	s.TransitionTTL = time.Hour
	s.WorkflowTTL = 24 * time.Hour
	s.StaleAfter = 30 * time.Minute
	s.MaxSessions = 64
	s.MaxTransitions = 64
	s.MaxWorkflows = 16
	s.MaxWorkflowOperations = 16
	s.MaxRecentOps = 16
	s.MaxSeenTransitions = 16
	s.MinObservations = 2
	s.MinSessions = 2
	s.MinLearningDuration = time.Second
	s.ConfidenceThreshold = 0.20
	s.QueueCapacity = 128
	return s
}

func api62Observation(s *sequenceStore, session, workflow, site, op string, at time.Time) sequenceObservation {
	return sequenceObservation{
		SessionKey:      s.digest("session\x00" + session),
		Site:            site,
		CorrelationKind: "anonymous",
		WorkflowID:      s.digest("workflow\x00" + workflow),
		IdentityKind:    "anonymous",
		OperationID:     op,
		At:              at,
	}
}

func transitionByOps(t *testing.T, model SequenceModel, from, to string) SequenceTransition {
	t.Helper()
	for _, edge := range model.Transitions {
		if edge.FromOperationID == from && edge.ToOperationID == to {
			return edge
		}
	}
	t.Fatalf("transition %s -> %s not found in %#v", from, to, model.Transitions)
	return SequenceTransition{}
}

func TestAPI62LearnsTransitionFrequencyUniqueSessionsAndMaturity(t *testing.T) {
	settings := api62TestSettings()
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC)
	a := apiOperationID("shop", http.MethodPost, "/login")
	b := apiOperationID("shop", http.MethodGet, "/profile")
	c := apiOperationID("shop", http.MethodPost, "/password-reset")

	for i := 0; i < 4; i++ {
		at := start.Add(time.Duration(i) * time.Second)
		s.process(api62Observation(s, fmt.Sprintf("s%d", i), "anon-shop", "shop", a, at))
		next := b
		if i == 3 {
			next = c
		}
		s.process(api62Observation(s, fmt.Sprintf("s%d", i), "anon-shop", "shop", next, at.Add(100*time.Millisecond)))
	}
	s.now = func() time.Time { return start.Add(10 * time.Second) }
	model := s.snapshot()
	if len(model.Workflows) != 1 {
		t.Fatalf("workflows=%d want 1", len(model.Workflows))
	}
	w := model.Workflows[0]
	if w.ObservationCount != 8 || w.SessionCount != 4 || w.MaxObservedDepth != 2 || w.Maturity != sequenceMaturityMature {
		t.Fatalf("unexpected workflow model: %#v", w)
	}
	if len(w.EntryOperations) != 1 || w.EntryOperations[0].OperationID != a || w.EntryOperations[0].Count != 4 {
		t.Fatalf("unexpected entry operations: %#v", w.EntryOperations)
	}

	ab := transitionByOps(t, model, a, b)
	if ab.ObservationCount != 3 || ab.Count != 3 || ab.SessionCount != 3 || ab.Confidence != 0.75 || ab.Maturity != sequenceMaturityMature {
		t.Fatalf("unexpected A->B learning: %#v", ab)
	}
	ac := transitionByOps(t, model, a, c)
	if ac.ObservationCount != 1 || ac.SessionCount != 1 || ac.Confidence != 0.25 || ac.Maturity != sequenceMaturityLearning {
		t.Fatalf("unexpected A->C learning: %#v", ac)
	}
}

func TestAPI62ColdStartAndStaleMaturityProtection(t *testing.T) {
	settings := api62TestSettings()
	settings.StaleAfter = 5 * time.Second
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	a := apiOperationID("cold", http.MethodGet, "/start")
	b := apiOperationID("cold", http.MethodPost, "/next")

	s.process(api62Observation(s, "one", "cold", "cold", a, start))
	s.process(api62Observation(s, "one", "cold", "cold", b, start.Add(time.Second)))
	s.now = func() time.Time { return start.Add(2 * time.Second) }
	cold := transitionByOps(t, s.snapshot(), a, b)
	if cold.Maturity != sequenceMaturityLearning {
		t.Fatalf("cold-start edge maturity=%s want LEARNING", cold.Maturity)
	}

	s.process(api62Observation(s, "two", "cold", "cold", a, start.Add(3*time.Second)))
	s.process(api62Observation(s, "two", "cold", "cold", b, start.Add(4*time.Second)))
	s.now = func() time.Time { return start.Add(4 * time.Second) }
	mature := transitionByOps(t, s.snapshot(), a, b)
	if mature.Maturity != sequenceMaturityMature {
		t.Fatalf("qualified edge maturity=%s want MATURE", mature.Maturity)
	}

	s.now = func() time.Time { return start.Add(10 * time.Second) }
	stale := transitionByOps(t, s.snapshot(), a, b)
	if stale.Maturity != sequenceMaturityStale {
		t.Fatalf("inactive edge maturity=%s want STALE", stale.Maturity)
	}
}

func TestAPI62AbsoluteSessionLifetimeAndTerminalLearning(t *testing.T) {
	settings := api62TestSettings()
	settings.SessionTTL = 10 * time.Minute
	settings.SessionAbsoluteTTL = 2 * time.Second
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	a := apiOperationID("life", http.MethodGet, "/create")
	b := apiOperationID("life", http.MethodPost, "/approve")
	c := apiOperationID("life", http.MethodPost, "/execute")

	s.process(api62Observation(s, "same", "life", "life", a, start))
	s.process(api62Observation(s, "same", "life", "life", b, start.Add(time.Second)))
	s.process(api62Observation(s, "same", "life", "life", c, start.Add(3*time.Second)))
	s.now = func() time.Time { return start.Add(3500 * time.Millisecond) }
	model := s.snapshot()
	if len(model.Sessions) != 1 || model.Sessions[0].RequestCount != 1 || model.Sessions[0].EntryOperationID != c {
		t.Fatalf("absolute lifetime did not create a fresh session: %#v", model.Sessions)
	}
	if len(model.Workflows) != 1 || model.Workflows[0].SessionCount != 2 {
		t.Fatalf("workflow session count did not reflect absolute expiry: %#v", model.Workflows)
	}
	terminal := model.Workflows[0].TerminalOperations
	if len(terminal) != 1 || terminal[0].OperationID != b || terminal[0].Count != 1 {
		t.Fatalf("expired-session terminal learning incorrect: %#v", terminal)
	}
}

func TestAPI62IdentityCohortsUseVerifiedClaimsOnlyAndExcludeSubject(t *testing.T) {
	s := testSequenceStore(api62TestSettings())
	at := time.Date(2026, 9, 24, 4, 0, 0, 0, time.UTC)
	base := sequenceRequest(http.MethodGet, "https://api.example.test/users/123", "raw-cookie")
	base.Header.Set("Authorization", "Bearer unverified.secret.jwt")
	anon := s.observation("identity", base, at)
	if anon.IdentityKind != "anonymous" || anon.CorrelationKind != "anonymous" {
		t.Fatalf("unverified token influenced workflow identity: %#v", anon)
	}

	alice := VerifiedAPIIdentity{Issuer: "https://issuer.example/", Subject: "alice-secret", TenantID: "tenant-secret", ClientID: "mobile-secret", Roles: []string{"customer", "member"}, Scopes: []string{"orders:read", "profile:read"}}
	bob := alice
	bob.Subject = "bob-secret"
	bob.Roles = []string{"member", "customer"}
	bob.Scopes = []string{"profile:read", "orders:read"}
	ra := base.WithContext(context.WithValue(base.Context(), apiIdentityContextKey{}, alice))
	rb := base.WithContext(context.WithValue(base.Context(), apiIdentityContextKey{}, bob))
	a := s.observation("identity", ra, at)
	b := s.observation("identity", rb, at)
	if a.WorkflowID != b.WorkflowID {
		t.Fatalf("subject split a shared verified cohort: %q != %q", a.WorkflowID, b.WorkflowID)
	}
	if a.SessionKey == b.SessionKey {
		t.Fatal("different verified subjects collapsed into one session")
	}

	other := bob
	other.Roles = []string{"admin"}
	ro := base.WithContext(context.WithValue(base.Context(), apiIdentityContextKey{}, other))
	if got := s.observation("identity", ro, at).WorkflowID; got == a.WorkflowID {
		t.Fatal("different verified role context collapsed into the same workflow cohort")
	}
}

func TestAPI62NormalizedCardinalityAndRandomSessionBounds(t *testing.T) {
	settings := api62TestSettings()
	settings.MaxSessions = 8
	settings.MaxTransitions = 4
	settings.MaxWorkflows = 2
	settings.MaxWorkflowOperations = 2
	settings.MaxRecentOps = 4
	settings.MaxSeenTransitions = 4
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC)

	for i := 0; i < 2000; i++ {
		r1 := sequenceRequest(http.MethodGet, fmt.Sprintf("https://api.example.test/users/%d", i+1), fmt.Sprintf("attacker-%d", i))
		r2 := sequenceRequest(http.MethodPost, fmt.Sprintf("https://api.example.test/users/%d/checkout", i+1), fmt.Sprintf("attacker-%d", i))
		s.process(s.observation("bounded", r1, start.Add(time.Duration(i)*time.Microsecond)))
		s.process(s.observation("bounded", r2, start.Add(time.Duration(i)*time.Microsecond+time.Nanosecond)))
	}
	s.now = func() time.Time { return start.Add(time.Second) }
	model := s.snapshot()
	if len(model.Sessions) > settings.MaxSessions || len(model.Transitions) > settings.MaxTransitions || len(model.Workflows) > settings.MaxWorkflows {
		t.Fatalf("bounded state exceeded: sessions=%d transitions=%d workflows=%d", len(model.Sessions), len(model.Transitions), len(model.Workflows))
	}
	if len(model.Transitions) != 1 {
		t.Fatalf("raw object IDs caused transition cardinality growth: %d edges", len(model.Transitions))
	}
	edge := model.Transitions[0]
	if edge.FromOperationID != apiOperationID("bounded", http.MethodGet, "/users/{id}") || edge.ToOperationID != apiOperationID("bounded", http.MethodPost, "/users/{id}/checkout") {
		t.Fatalf("unexpected normalized edge: %#v", edge)
	}
	if model.EvictedSessions == 0 {
		t.Fatal("random session flood did not exercise bounded session eviction")
	}
}

func TestAPI62RestartPersistenceRestoresLearningWithoutRawVerifiedClaims(t *testing.T) {
	settings := api62TestSettings()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 6, 0, 0, 0, time.UTC)
	id := VerifiedAPIIdentity{Issuer: "https://secret-issuer.example/", Subject: "secret-subject", TenantID: "secret-tenant", ClientID: "secret-client", Roles: []string{"secret-role"}, Scopes: []string{"secret:scope"}}

	for i := 0; i < 2; i++ {
		r1 := sequenceRequest(http.MethodPost, "https://api.example.test/login", fmt.Sprintf("secret-cookie-%d", i))
		r1 = r1.WithContext(context.WithValue(r1.Context(), apiIdentityContextKey{}, id))
		r2 := sequenceRequest(http.MethodGet, "https://api.example.test/profile", fmt.Sprintf("other-cookie-%d", i))
		r2 = r2.WithContext(context.WithValue(r2.Context(), apiIdentityContextKey{}, id))
		s.process(s.observation("persist", r1, start.Add(time.Duration(i)*2*time.Second)))
		s.process(s.observation("persist", r2, start.Add(time.Duration(i)*2*time.Second+time.Second)))
		id.Subject = fmt.Sprintf("secret-subject-%d", i+1)
	}
	s.now = func() time.Time { return start.Add(5 * time.Second) }
	before := s.snapshot()
	if err := s.save(configPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "api-sequence.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"secret-issuer", "secret-subject", "secret-tenant", "secret-client", "secret-role", "secret:scope", "secret-cookie"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("durable learning state leaked verified/session material %q", forbidden)
		}
	}

	restored := newSequenceStoreWithSettings(settings, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	restored.now = func() time.Time { return start.Add(5 * time.Second) }
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	after := restored.snapshot()
	if len(after.Workflows) != len(before.Workflows) || len(after.Transitions) != len(before.Transitions) {
		t.Fatalf("learning state not restored: before workflows/edges=%d/%d after=%d/%d", len(before.Workflows), len(before.Transitions), len(after.Workflows), len(after.Transitions))
	}
	if len(after.Transitions) != 1 || after.Transitions[0].ObservationCount != 2 || after.Transitions[0].SessionCount != 2 {
		t.Fatalf("transition learning counters not restored: %#v", after.Transitions)
	}
}

func TestAPI62ConcurrentWorkflowLearningAndSnapshotsRemainBounded(t *testing.T) {
	settings := api62TestSettings()
	settings.MaxSessions = 32
	settings.MaxTransitions = 8
	settings.MaxWorkflows = 4
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 7, 0, 0, 0, time.UTC)
	a := apiOperationID("race62", http.MethodGet, "/items/{id}")
	b := apiOperationID("race62", http.MethodPost, "/checkout")
	const workers = 32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			at := start.Add(time.Duration(i) * time.Millisecond)
			s.process(api62Observation(s, fmt.Sprintf("race-%d", i), "race-cohort", "race62", a, at))
			s.process(api62Observation(s, fmt.Sprintf("race-%d", i), "race-cohort", "race62", b, at.Add(time.Microsecond)))
			_ = s.snapshot()
		}(i)
	}
	wg.Wait()
	s.now = func() time.Time { return start.Add(time.Second) }
	model := s.snapshot()
	if len(model.Sessions) > settings.MaxSessions || len(model.Transitions) > settings.MaxTransitions || len(model.Workflows) > settings.MaxWorkflows {
		t.Fatalf("concurrent learning exceeded bounds: %#v", model)
	}
	if len(model.Transitions) != 1 || model.Transitions[0].ObservationCount != workers || model.Transitions[0].SessionCount != workers {
		t.Fatalf("concurrent learning lost counts: %#v", model.Transitions)
	}
}
