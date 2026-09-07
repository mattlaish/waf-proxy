package main

import (
	"encoding/json"
	"testing"
)

func TestSizedJSONTarget(t *testing.T) {
	for _, n := range []int{1024, 16 * 1024, 64 * 1024} {
		if got := len(sizedJSON(n, "x")); got != n {
			t.Fatalf("size %d got %d", n, got)
		}
	}
}

func TestScenarioCorpus(t *testing.T) {
	s := scenarios(0)
	for _, n := range []string{"clean-get", "json-1k", "json-16k", "json-64k", "json-256k", "sqli-64k", "xss-64k", "traversal"} {
		if _, ok := s[n]; !ok {
			t.Fatalf("missing %s", n)
		}
	}
	if len(s["sqli-64k"].Body) != 64*1024 {
		t.Fatalf("sqli body=%d", len(s["sqli-64k"].Body))
	}
	for _, n := range []string{"json-1k", "json-64k", "sqli-64k", "xss-64k"} {
		if !json.Valid(s[n].Body) {
			t.Fatalf("%s body is not valid JSON", n)
		}
	}
}

func TestLatencyHistogramQuantiles(t *testing.T) {
	var h latencyHist
	for i := 1; i <= 1000; i++ {
		h.addNs(int64(i) * 1e6)
	}
	p50, p95 := h.quantile(.5), h.quantile(.95)
	if p50 < 480 || p50 > 520 {
		t.Fatalf("p50=%f", p50)
	}
	if p95 < 900 || p95 > 1000 {
		t.Fatalf("p95=%f", p95)
	}
}

func TestCompareResults(t *testing.T) {
	h := []Result{{Mode: "http", Scenario: "json-64k", CPUUSPerRequest: 100, HostSoftIRQPercent: 1}}
	c := []Result{{Mode: "coraza", Scenario: "json-64k", CPUUSPerRequest: 45}}
	got := compareResults(h, c)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].CorazaShare < .44 || got[0].CorazaShare > .46 {
		t.Fatalf("share=%f", got[0].CorazaShare)
	}
}
