package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestGenerateOWIReturnFixture drives the shipping connector handler and native
// mappers with isolated synthetic product state. It is skipped during normal
// tests. Release/return tooling sets WAF_OWI_RETURN_FIXTURE_DIR to capture the
// exact HTTP response bodies and cursor chains for the Dashboard handoff.
func TestGenerateOWIReturnFixture(t *testing.T) {
	outDir := os.Getenv("WAF_OWI_RETURN_FIXTURE_DIR")
	if outDir == "" {
		t.Skip("fixture generation is opt-in")
	}
	cfg, token, s := testOWIConfig(t, 10000, 10000)
	cfg.SourceInstanceID = "waf-proxy-fixture-01"
	c, err := newOWIConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.close()
	s.dashboard = c
	for i := 0; i < 5; i++ {
		c.noteWAFMatch(matchRec{Site: "site-a", RuleID: 942100 + i, Severity: "CRITICAL", URI: "/api/users/123456?token=fixture-secret"}, time.Now().UTC().Add(time.Duration(i)*time.Millisecond))
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, rows, _ := c.store.stats()
		if rows >= 5 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	h := c.handler()
	responses := map[string][]json.RawMessage{}
	requests := []map[string]any{}
	control := func(path string) {
		code, _, body := owiRequest(t, h, token, http.MethodGet, path)
		if code != 200 {
			t.Fatalf("control %s=%d %s", path, code, body)
		}
		responses[path] = append(responses[path], json.RawMessage(append([]byte(nil), body...)))
	}
	control(owiBasePath + "/health")
	control(owiBasePath + "/capabilities")
	type lane struct {
		path, capability, kind string
		page                   int
	}
	lanes := []lane{{owiBasePath + "/assets", "LIST_ASSETS", "ASSET", 2}, {owiBasePath + "/detections", "LIST_DETECTIONS", "DETECTION", 2}, {owiBasePath + "/policies", "OBSERVE_POLICY", "POLICY", 1}, {owiBasePath + "/health-observations", "HEALTH", "HEALTH", 1}}
	finalCursor := map[string]string{}
	appendCall := func(l lane, cursor *string) owiPage {
		path := l.path + "?limit=" + strconv.Itoa(l.page)
		if cursor != nil {
			path += "&cursor=" + *cursor
		}
		code, _, body := owiRequest(t, h, token, http.MethodGet, path)
		if code != 200 {
			t.Fatalf("%s=%d %s", path, code, body)
		}
		var page owiPage
		if err := json.Unmarshal(body, &page); err != nil {
			t.Fatal(err)
		}
		responses[l.path] = append(responses[l.path], json.RawMessage(append([]byte(nil), body...)))
		var reqCursor any = nil
		if cursor != nil {
			reqCursor = *cursor
		}
		requests = append(requests, map[string]any{"tenant_id": "fixture-tenant-01", "product_id": "dashboard-waf-proxy-fixture-01", "capability": l.capability, "resource_kind": l.kind, "cursor": reqCursor, "page_size": l.page})
		return page
	}
	for _, l := range lanes {
		var cursor *string
		for {
			p := appendCall(l, cursor)
			if !p.HasMore {
				finalCursor[l.kind] = p.NextCursor
				break
			}
			next := p.NextCursor
			cursor = &next
		}
	}
	// Native changes after the stable bootstrap snapshots: state lanes get
	// real config/health revisions, HISTORY gets one additional WAF match.
	rt := s.rt.Load()
	next := rt.cfg
	next.Nodes = append(next.Nodes, NodeConfig{Name: "app-3", Host: "192.0.2.12"})
	next.Policies[0].ParanoiaLevel = 2
	s.rt.Store(&runtimeState{cfg: next})
	s.draining.Store(true)
	c.noteWAFMatch(matchRec{Site: "site-a", RuleID: 949110, Severity: "HIGH", URI: "/admin/abcdef0123456789?authorization=secret"}, time.Now().UTC())
	time.Sleep(30 * time.Millisecond)
	for _, l := range lanes {
		cur := finalCursor[l.kind]
		totalDelta := 0
		for {
			p := appendCall(l, &cur)
			if p.SyncMode != "INCREMENTAL" {
				t.Fatalf("lane %s did not enter incremental mode", l.kind)
			}
			totalDelta += len(p.Records)
			cur = p.NextCursor
			if !p.HasMore {
				break
			}
		}
		if totalDelta == 0 {
			t.Fatalf("lane %s missing non-empty delta", l.kind)
		}
		finalCursor[l.kind] = cur
	}
	for _, l := range lanes {
		cur := finalCursor[l.kind]
		p := appendCall(l, &cur)
		if p.SyncMode != "INCREMENTAL" || len(p.Records) != 0 || p.HasMore {
			t.Fatalf("lane %s missing empty delta", l.kind)
		}
	}
	// Derive decoder negatives from actual positive pages rather than copying
	// specification examples.
	var negatives []map[string]any
	for _, route := range []string{owiBasePath + "/assets", owiBasePath + "/detections"} {
		var v map[string]any
		if err := json.Unmarshal(responses[route][0], &v); err != nil {
			t.Fatal(err)
		}
		if route == owiBasePath+"/assets" {
			delete(v, "contract")
		} else {
			v["unexpected_secret_field"] = "must-be-rejected"
		}
		negatives = append(negatives, map[string]any{"route_name": filepath.Base(route), "page": v})
	}
	// Attention examples are produced by the same mappers. They are evidence
	// fixtures only and are not injected into the resource page replay.
	now := time.Now().UTC().Truncate(time.Minute)
	highAttack := owiDetectionRecord(cfg.SourceInstanceID, owiNativeDetection{At: now, Site: "site-a", RuleID: 942100, Severity: "CRITICAL", Path: "/login"})
	lowSignal := owiDetectionRecord(cfg.SourceInstanceID, owiNativeDetection{At: now, Site: "site-a", RuleID: 920100, Severity: "INFO", Path: "/status"})
	s.draining.Store(true)
	degraded := c.healthRecord(s.rt.Load().cfg, now)
	s.draining.Store(false)
	for i := 0; i < 7; i++ {
		s.observations.queue <- observationEvent{kind: observationHost}
	}
	capacity := c.healthRecord(s.rt.Load().cfg, now)
	for len(s.observations.queue) > 0 {
		<-s.observations.queue
	}
	normal := c.healthRecord(s.rt.Load().cfg, now)
	attentionExamples := map[string]any{"WAF.HIGH_SIGNAL_ATTACK": map[string]any{"positive": highAttack, "counterexample": lowSignal}, "WAF.PROTECTION_DEGRADED": map[string]any{"positive": degraded, "counterexample": normal}, "WAF.CAPACITY_PRESSURE": map[string]any{"positive": capacity, "counterexample": normal}}
	out := map[string]any{"responses": responses, "requests": requests, "negative_pages": negatives, "attention_examples": attentionExamples, "token_digest": owiTokenDigest(token), "source_instance_id": cfg.SourceInstanceID}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "generated-fixture-data.json"), append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
