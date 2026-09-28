package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func api64TrainWithoutMode(s *sequenceStore, workflow, site string, ops []string, sessions int, start time.Time) time.Time {
	at := start
	for i := 0; i < sessions; i++ {
		session := site + "-learn-" + itoa(i+1)
		for j, op := range ops {
			s.process(api62Observation(s, session, workflow, site, op, at.Add(time.Duration(j)*100*time.Millisecond)))
		}
		at = at.Add(2 * time.Second)
	}
	return at
}

func TestAPI64ExplicitLearnDetectModeNeverEnforces(t *testing.T) {
	settings := api63TestSettings()
	start := time.Date(2026, 9, 24, 11, 0, 0, 0, time.UTC)
	a := apiOperationID("mode64", http.MethodPost, "/login")
	b := apiOperationID("mode64", http.MethodGet, "/profile")
	unexpected := apiOperationID("mode64", http.MethodPost, "/admin-export")

	learn := testSequenceStore(settings)
	next := api64TrainWithoutMode(learn, "mode64", "mode64", []string{a, b}, 3, start)
	learn.process(api62Observation(learn, "learn-probe", "mode64", "mode64", a, next))
	learn.process(api62Observation(learn, "learn-probe", "mode64", "mode64", unexpected, next.Add(100*time.Millisecond)))
	if got := len(learn.snapshot().Violations); got != 0 {
		t.Fatalf("default LEARN generated %d violations", got)
	}

	detect := testSequenceStore(settings)
	next = api64TrainWithoutMode(detect, "mode64", "mode64", []string{a, b}, 3, start)
	control, err := detect.setSequenceMode("mode64", sequenceDetectionDetect, next)
	if err != nil || control.Mode != sequenceDetectionDetect {
		t.Fatalf("mature DETECT promotion failed: control=%#v err=%v", control, err)
	}
	detect.process(api62Observation(detect, "detect-probe", "mode64", "mode64", a, next))
	detect.process(api62Observation(detect, "detect-probe", "mode64", "mode64", unexpected, next.Add(100*time.Millisecond)))
	if got := len(api63ViolationsByType(detect.snapshot(), sequenceAnomalyUnknownTransition)); got != 1 {
		t.Fatalf("DETECT unknown-transition evidence=%d want 1", got)
	}
	if _, err := detect.setSequenceMode("mode64", "ENFORCE", next); err == nil {
		t.Fatal("sequence ENFORCE mode was accepted")
	}
}

func TestAPI64DetectPromotionRequiresMatureWorkflow(t *testing.T) {
	s := testSequenceStore(api63TestSettings())
	at := time.Date(2026, 9, 24, 11, 30, 0, 0, time.UTC)
	if _, err := s.setSequenceMode("cold64", sequenceDetectionDetect, at); err == nil {
		t.Fatal("cold site entered DETECT without a mature workflow")
	}
	control, err := s.setSequenceMode("cold64", sequenceDetectionLearn, at)
	if err != nil || control.Mode != sequenceDetectionLearn {
		t.Fatalf("LEARN should always be available: %#v %v", control, err)
	}
}

func TestAPI64SiteControlCardinalityIsBounded(t *testing.T) {
	settings := api63TestSettings()
	settings.MaxWorkflows = 2
	s := testSequenceStore(settings)
	at := time.Date(2026, 9, 24, 11, 45, 0, 0, time.UTC)
	for _, site := range []string{"control-a", "control-b"} {
		if _, err := s.setSequenceMode(site, sequenceDetectionLearn, at); err != nil {
			t.Fatalf("set LEARN for %s: %v", site, err)
		}
	}
	if _, err := s.setSequenceMode("control-c", sequenceDetectionLearn, at); err == nil {
		t.Fatal("site-control cardinality limit was not enforced")
	}
}

func TestAPI64ResetRelearnIsSiteScopedPreservesExceptionsAndReturnsLearn(t *testing.T) {
	settings := api63TestSettings()
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	a1 := apiOperationID("site-a", http.MethodGet, "/start")
	a2 := apiOperationID("site-a", http.MethodGet, "/next")
	b1 := apiOperationID("site-b", http.MethodGet, "/start")
	b2 := apiOperationID("site-b", http.MethodGet, "/next")
	nextA := api63TrainPath(s, "wf-a", "site-a", []string{a1, a2}, 3, start)
	_ = api63TrainPath(s, "wf-b", "site-b", []string{b1, b2}, 3, start.Add(time.Minute))
	ex, err := s.createSequenceException(SequenceException{Site: "site-a", Type: sequenceAnomalyUnknownTransition, FromOperationID: a1}, time.Hour, nextA)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.resetAndRelearn("site-a", nextA.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != sequenceDetectionLearn || result.Generation == 0 || result.PreservedExceptions != 1 {
		t.Fatalf("unexpected relearn result: %#v", result)
	}
	model := s.snapshot()
	for _, w := range model.Workflows {
		if w.Site == "site-a" {
			t.Fatal("site-a workflow survived reset/relearn")
		}
	}
	foundB := false
	for _, w := range model.Workflows {
		if w.Site == "site-b" {
			foundB = true
		}
	}
	if !foundB {
		t.Fatal("site-b workflow was removed by site-a relearn")
	}
	if len(model.Exceptions) != 1 || model.Exceptions[0].ID != ex.ID {
		t.Fatalf("explicit exception was not preserved: %#v", model.Exceptions)
	}
	states := sequenceLearningStates(model)
	foundA := false
	for _, st := range states {
		if st.Site == "site-a" {
			foundA = true
			if st.Mode != sequenceDetectionLearn {
				t.Fatalf("site-a mode=%s", st.Mode)
			}
		}
	}
	if !foundA {
		t.Fatal("site-a learning state missing after relearn")
	}
}

func TestAPI64RecentSessionsAreBoundedTTLAndPrivacyPreserving(t *testing.T) {
	settings := api63TestSettings()
	settings.SessionTTL = time.Second
	settings.MaxRecentSessions = 2
	settings.RecentSessionTTL = time.Minute
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 13, 0, 0, 0, time.UTC)
	op := apiOperationID("recent64", http.MethodGet, "/orders/{id}")
	for i := 0; i < 4; i++ {
		at := start.Add(time.Duration(i) * 2 * time.Second)
		s.process(api62Observation(s, "raw-cookie-secret-"+itoa(i+1), "recent64", "recent64", op, at))
		s.stateMu.Lock()
		s.pruneLocked(at.Add(1500 * time.Millisecond))
		s.stateMu.Unlock()
	}
	model := s.snapshot()
	if len(model.RecentSessions) != 2 || model.EvictedRecentSessions == 0 {
		t.Fatalf("recent sessions not bounded: len=%d evicted=%d", len(model.RecentSessions), model.EvictedRecentSessions)
	}
	b, _ := json.Marshal(model.RecentSessions)
	if strings.Contains(string(b), "raw-cookie-secret") || strings.Contains(string(b), "/orders/") {
		t.Fatalf("recent session evidence leaked raw session/path material: %s", b)
	}
	s.now = func() time.Time { return start.Add(2 * time.Hour) }
	if got := len(s.snapshot().RecentSessions); got != 0 {
		t.Fatalf("expired recent sessions retained=%d", got)
	}
}

func TestAPI64ExceptionCRUDIsScopedBoundedAndPersistent(t *testing.T) {
	settings := api63TestSettings()
	settings.MaxExceptions = 2
	s := testSequenceStore(settings)
	at := time.Date(2026, 9, 24, 14, 0, 0, 0, time.UTC)
	op := apiOperationID("exception64", http.MethodGet, "/account")
	if _, err := s.createSequenceException(SequenceException{Site: "exception64", Type: sequenceAnomalyUnknownTransition}, time.Hour, at); err == nil {
		t.Fatal("site-wide unscoped exception was accepted")
	}
	if _, err := s.createSequenceException(SequenceException{Site: "exception64", Type: sequenceAnomalyUnknownTransition, FromOperationID: op}, sequenceMaximumExceptionTTL+time.Second, at); err == nil {
		t.Fatal("overlong exception TTL was accepted")
	}
	ex1, err := s.createSequenceException(SequenceException{Site: "exception64", Type: sequenceAnomalyUnknownTransition, FromOperationID: op}, time.Hour, at)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.createSequenceException(SequenceException{Site: "exception64", Type: sequenceAnomalySequenceReversal, ToOperationID: op}, 2*time.Hour, at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.createSequenceException(SequenceException{Site: "exception64", Type: sequenceAnomalyUnexpectedEntryPoint, ToOperationID: op}, time.Hour, at.Add(2*time.Second)); err == nil {
		t.Fatal("exception cap was not enforced")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	s.now = func() time.Time { return at.Add(10 * time.Second) }
	if err := s.save(configPath); err != nil {
		t.Fatal(err)
	}
	restored := newSequenceStoreWithSettings(settings, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	restored.now = s.now
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	if len(restored.snapshot().Exceptions) != 2 {
		t.Fatalf("exceptions not restored: %#v", restored.snapshot().Exceptions)
	}
	if _, err := restored.deleteSequenceException(ex1.ID); err != nil {
		t.Fatal(err)
	}
	if len(restored.snapshot().Exceptions) != 1 {
		t.Fatal("exception delete did not remove exactly one exception")
	}
}

func TestAPI64V3MigrationDefaultsToLearnAndV4PersistsControls(t *testing.T) {
	settings := api63TestSettings()
	s := testSequenceStore(settings)
	at := time.Date(2026, 9, 24, 15, 0, 0, 0, time.UTC)
	a := apiOperationID("persist64", http.MethodGet, "/a")
	b := apiOperationID("persist64", http.MethodGet, "/b")
	next := api64TrainWithoutMode(s, "persist64", "persist64", []string{a, b}, 3, at)
	if _, err := s.setSequenceMode("persist64", sequenceDetectionDetect, next); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	s.now = func() time.Time { return next.Add(time.Second) }
	if err := s.save(configPath); err != nil {
		t.Fatal(err)
	}
	restored := newSequenceStoreWithSettings(settings, []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
	restored.now = s.now
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	model := restored.snapshot()
	if model.Version != 4 || len(model.Controls) != 1 || model.Controls[0].Mode != sequenceDetectionDetect {
		t.Fatalf("v4 controls did not restore: %#v", model.Controls)
	}

	var state map[string]any
	bdata, err := os.ReadFile(filepath.Join(dir, "api-sequence.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bdata, &state); err != nil {
		t.Fatal(err)
	}
	state["version"] = float64(3)
	delete(state, "controls")
	delete(state, "recent_sessions")
	legacy, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "api-sequence.json"), legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	migrated := newSequenceStoreWithSettings(settings, []byte("cccccccccccccccccccccccccccccccc"))
	migrated.now = s.now
	if err := migrated.load(configPath); err != nil {
		t.Fatal(err)
	}
	for _, st := range sequenceLearningStates(migrated.snapshot()) {
		if st.Site == "persist64" && st.Mode != sequenceDetectionLearn {
			t.Fatalf("v3 migration mode=%s want LEARN", st.Mode)
		}
	}
}

func TestAPI64AdminControlsRequireReviewerPersistAndAudit(t *testing.T) {
	settings := api63TestSettings()
	s := testSequenceStore(settings)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	a := &adminServer{srv: &server{sequence: s, configPath: configPath}, token: "break-glass", sessions: newSessionStore(), audit: newAuditLog()}
	h := a.handler()
	viewer := a.sessions.create("viewer-user", roleViewer)
	req := httptest.NewRequest(http.MethodPost, "/api/security/sequence/mode", bytes.NewBufferString(`{"site":"admin64","mode":"LEARN"}`))
	req.Header.Set("Authorization", "Bearer "+viewer)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer status=%d want 403", rr.Code)
	}

	reviewer := a.sessions.create("review-user", roleReviewer)
	req = httptest.NewRequest(http.MethodPost, "/api/security/sequence/mode", bytes.NewBufferString(`{"site":"admin64","mode":"LEARN"}`))
	req.Header.Set("Authorization", "Bearer "+reviewer)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("reviewer status=%d body=%s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "api-sequence.json")); err != nil {
		t.Fatalf("mode mutation not persisted: %v", err)
	}
	found := false
	for _, e := range a.audit.list(20) {
		if e.Action == "api_sequence.mode" && e.User == "review-user" {
			found = true
		}
	}
	if !found {
		t.Fatal("sequence mode mutation missing audit evidence")
	}
}

func TestAPI64ConcurrentModeSnapshotAndRelearnStayBounded(t *testing.T) {
	settings := api63TestSettings()
	settings.MaxSessions = 64
	settings.MaxTransitions = 64
	settings.MaxRecentSessions = 16
	s := testSequenceStore(settings)
	start := time.Date(2026, 9, 24, 16, 0, 0, 0, time.UTC)
	a := apiOperationID("race64", http.MethodGet, "/a")
	b := apiOperationID("race64", http.MethodGet, "/b")
	next := api64TrainWithoutMode(s, "race64", "race64", []string{a, b}, 3, start)
	if _, err := s.setSequenceMode("race64", sequenceDetectionDetect, next); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.snapshot()
			if i%3 == 0 {
				_, _ = s.setSequenceMode("race64", sequenceDetectionLearn, next.Add(time.Duration(i)*time.Millisecond))
				_, _ = s.setSequenceMode("race64", sequenceDetectionDetect, next.Add(time.Duration(i+1)*time.Millisecond))
			}
		}(i)
	}
	wg.Wait()
	if _, err := s.resetAndRelearn("race64", next.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	model := s.snapshot()
	if len(model.Sessions) > settings.MaxSessions || len(model.Transitions) > settings.MaxTransitions || len(model.RecentSessions) > settings.MaxRecentSessions {
		t.Fatalf("API-6.4 bounded state exceeded: %#v", model)
	}
	states := sequenceLearningStates(model)
	if len(states) != 1 || states[0].Mode != sequenceDetectionLearn {
		t.Fatalf("relearn final state=%#v", states)
	}
}
