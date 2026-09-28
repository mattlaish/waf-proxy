package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testSequenceStore(settings sequenceSettings) *sequenceStore {
	return newSequenceStoreWithSettings(settings, []byte("0123456789abcdef0123456789abcdef"))
}

func sequenceRequest(method, target, cookie string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	r.RemoteAddr = "192.0.2.10:4321"
	r.Header.Set("User-Agent", "api-client/1")
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: "sessionid", Value: cookie})
	}
	return r
}

func TestAPI61UsesNormalizedOperationIDsAndVerifiedIdentityOnly(t *testing.T) {
	s := testSequenceStore(defaultSequenceSettings())
	at := time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)

	anon := sequenceRequest(http.MethodGet, "https://api.example.test/orders/123?expand=items", "raw-session-secret")
	anon.Header.Set("Authorization", "Bearer unverified.jwt.value")
	a := s.observation("payments", anon, at)
	if want := apiOperationID("payments", http.MethodGet, "/orders/{id}"); a.OperationID != want {
		t.Fatalf("operation id = %q, want normalized API-1 id %q", a.OperationID, want)
	}
	if a.CorrelationKind != "anonymous" {
		t.Fatalf("unverified Authorization data selected %q correlation", a.CorrelationKind)
	}
	if strings.Contains(a.SessionKey, "raw-session-secret") || strings.Contains(a.SessionKey, "192.0.2.10") {
		t.Fatal("anonymous session key retained raw correlation material")
	}

	id := VerifiedAPIIdentity{Issuer: "https://issuer.example/", Subject: "alice", TenantID: "acme", ClientID: "checkout"}
	verifiedA := anon.WithContext(context.WithValue(anon.Context(), apiIdentityContextKey{}, id))
	b := s.observation("payments", verifiedA, at)
	if b.CorrelationKind != "verified_identity" || b.SessionKey == a.SessionKey {
		t.Fatalf("verified identity correlation not selected: %#v", b)
	}
	verifiedB := sequenceRequest(http.MethodPost, "https://api.example.test/payments", "different-cookie")
	verifiedB.RemoteAddr = "198.51.100.22:99"
	verifiedB = verifiedB.WithContext(context.WithValue(verifiedB.Context(), apiIdentityContextKey{}, id))
	if got := s.observation("payments", verifiedB, at).SessionKey; got != b.SessionKey {
		t.Fatalf("same verified identity did not correlate: %q != %q", got, b.SessionKey)
	}
	id.Subject = "bob"
	verifiedC := verifiedB.WithContext(context.WithValue(verifiedB.Context(), apiIdentityContextKey{}, id))
	if got := s.observation("payments", verifiedC, at).SessionKey; got == b.SessionKey {
		t.Fatal("different verified subjects collapsed into one session")
	}
}

func TestAPI61DeterministicSessionAndTransitionFoundation(t *testing.T) {
	s := testSequenceStore(defaultSequenceSettings())
	at := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	r1 := sequenceRequest(http.MethodGet, "https://api.example.test/cart/42", "stable")
	r2 := sequenceRequest(http.MethodPost, "https://api.example.test/checkout", "stable")
	s.process(s.observation("shop", r1, at))
	s.process(s.observation("shop", r2, at.Add(time.Second)))

	s.now = func() time.Time { return at.Add(2 * time.Second) }
	model := s.snapshot()
	if len(model.Sessions) != 1 || len(model.Transitions) != 1 {
		t.Fatalf("sessions/transitions = %d/%d, want 1/1", len(model.Sessions), len(model.Transitions))
	}
	session := model.Sessions[0]
	if session.RequestCount != 2 || len(session.RecentOperationIDs) != 2 {
		t.Fatalf("unexpected session counters: %#v", session)
	}
	edge := model.Transitions[0]
	if edge.FromOperationID != session.RecentOperationIDs[0] || edge.ToOperationID != session.RecentOperationIDs[1] || edge.Count != 1 {
		t.Fatalf("unexpected transition: %#v", edge)
	}
}

func TestAPI61BoundedStateAndTTL(t *testing.T) {
	settings := sequenceSettings{SessionTTL: time.Second, TransitionTTL: time.Second, MaxSessions: 3, MaxTransitions: 2, MaxRecentOps: 2, QueueCapacity: 8}
	s := testSequenceStore(settings)
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 8; i++ {
		key := s.digest(fmt.Sprintf("session-%d", i))
		s.process(sequenceObservation{SessionKey: key, Site: "bounded", CorrelationKind: "anonymous", OperationID: apiOperationID("bounded", http.MethodGet, "/start"), At: at.Add(time.Duration(i) * time.Millisecond)})
		s.process(sequenceObservation{SessionKey: key, Site: "bounded", CorrelationKind: "anonymous", OperationID: apiOperationID("bounded", http.MethodPost, fmt.Sprintf("/step/%d", i)), At: at.Add(time.Duration(i)*time.Millisecond + time.Microsecond)})
	}
	s.now = func() time.Time { return at.Add(100 * time.Millisecond) }
	model := s.snapshot()
	if len(model.Sessions) > settings.MaxSessions || len(model.Transitions) > settings.MaxTransitions {
		t.Fatalf("state exceeded caps: sessions=%d transitions=%d", len(model.Sessions), len(model.Transitions))
	}
	if model.EvictedSessions == 0 || model.EvictedTransitions == 0 {
		t.Fatalf("expected bounded eviction evidence: %#v", model)
	}
	for _, session := range model.Sessions {
		if len(session.RecentOperationIDs) > settings.MaxRecentOps {
			t.Fatalf("recent operations exceeded cap: %#v", session)
		}
	}

	s.now = func() time.Time { return at.Add(2 * time.Second) }
	model = s.snapshot()
	if len(model.Sessions) != 0 || len(model.Transitions) != 0 {
		t.Fatalf("expired state retained: sessions=%d transitions=%d", len(model.Sessions), len(model.Transitions))
	}
}

func TestAPI61RestartPersistenceExcludesRawIdentityAndAnonymousInputs(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	s := testSequenceStore(defaultSequenceSettings())
	at := time.Date(2026, 9, 23, 11, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return at }

	r := sequenceRequest(http.MethodGet, "https://api.example.test/accounts/987", "cookie-super-secret")
	id := VerifiedAPIIdentity{Issuer: "https://issuer.example/", Subject: "identity-super-secret", ClientID: "client-super-secret"}
	r = r.WithContext(context.WithValue(r.Context(), apiIdentityContextKey{}, id))
	s.process(s.observation("accounts", r, at))
	s.process(s.observation("accounts", sequenceRequest(http.MethodPost, "https://api.example.test/accounts/987/pay", "anonymous-super-secret"), at.Add(time.Second)))
	s.now = func() time.Time { return at.Add(2 * time.Second) }
	if err := s.save(configPath); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "api-sequence.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"cookie-super-secret", "anonymous-super-secret", "identity-super-secret", "client-super-secret", "192.0.2.10", "api-client/1"} {
		if strings.Contains(string(b), forbidden) {
			t.Fatalf("persisted sequence state contains raw sensitive input %q", forbidden)
		}
	}
	var raw sequenceStateFile
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}

	restored := newSequenceStoreWithSettings(defaultSequenceSettings(), []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	restored.now = func() time.Time { return at.Add(2 * time.Second) }
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	model := restored.snapshot()
	if len(model.Sessions) != len(raw.Sessions) || len(model.Sessions) != 2 {
		t.Fatalf("restart restored %d sessions, want %d", len(model.Sessions), len(raw.Sessions))
	}
	if got, want := restored.observation("accounts", r, at).SessionKey, s.observation("accounts", r, at).SessionKey; got != want {
		t.Fatalf("correlation key did not survive restart: %q != %q", got, want)
	}
}

func TestAPI61ConcurrentNonBlockingRuntimeSnapshot(t *testing.T) {
	settings := sequenceSettings{SessionTTL: time.Hour, TransitionTTL: time.Hour, MaxSessions: 16, MaxTransitions: 16, MaxRecentOps: 8, QueueCapacity: 1024}
	s := testSequenceStore(settings)
	s.start()
	var served atomic.Uint64
	h := s.wrap("race", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	const workers = 32
	const each = 8
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < each; j++ {
				r := sequenceRequest(http.MethodGet, fmt.Sprintf("https://api.example.test/items/%d", (worker+j)%8), "shared")
				h.ServeHTTP(httptest.NewRecorder(), r)
				_ = s.snapshot()
			}
		}(i)
	}
	wg.Wait()
	s.stopAndDrain()
	model := s.snapshot()
	if got, want := served.Load(), uint64(workers*each); got != want {
		t.Fatalf("downstream served %d, want %d", got, want)
	}
	if len(model.Sessions) != 1 || model.Sessions[0].RequestCount != uint64(workers*each) {
		t.Fatalf("concurrent session state lost updates: %#v", model.Sessions)
	}
	if len(model.Sessions) > settings.MaxSessions || len(model.Transitions) > settings.MaxTransitions {
		t.Fatalf("concurrent state exceeded caps: %#v", model)
	}
}
