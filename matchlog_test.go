package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testMatchRec(client string) matchRec {
	return matchRec{
		Site: "site-a", RuleID: 942100, Severity: "critical", Phase: 2,
		Client: client, URI: "/login", Msg: "SQL Injection Attack Detected", Data: "id=1'",
	}
}

func waitAtomic(t *testing.T, load func() uint64, want uint64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if load() >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("counter = %d, want >= %d", load(), want)
}

func TestMatchLogPlaneAggregatesSameKey(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	p := newMatchLogPlane(log, 16, 16, time.Hour)
	p.start()
	for i := 0; i < 3; i++ {
		if !p.enqueue(testMatchRec("192.0.2.10")) {
			t.Fatal("enqueue unexpectedly dropped")
		}
	}
	waitAtomic(t, p.processed.Load, 3)
	p.stopAndDrain()

	out := buf.String()
	if got := strings.Count(out, "msg=\"rule match summary\""); got != 1 {
		t.Fatalf("summary lines = %d, want 1; output=%q", got, out)
	}
	if !strings.Contains(out, "count=3") {
		t.Fatalf("missing aggregate count: %q", out)
	}
	if !strings.Contains(out, "rule_id=942100") || !strings.Contains(out, "client=192.0.2.10") {
		t.Fatalf("missing aggregation dimensions: %q", out)
	}
	if got := p.snapshot().Emitted; got != 1 {
		t.Fatalf("emitted = %d, want 1", got)
	}
}

func TestMatchLogPlaneSeparatesClients(t *testing.T) {
	var buf bytes.Buffer
	p := newMatchLogPlane(slog.New(slog.NewTextHandler(&buf, nil)), 16, 16, time.Hour)
	p.start()
	p.enqueue(testMatchRec("192.0.2.10"))
	p.enqueue(testMatchRec("192.0.2.11"))
	waitAtomic(t, p.processed.Load, 2)
	p.stopAndDrain()
	if got := strings.Count(buf.String(), "msg=\"rule match summary\""); got != 2 {
		t.Fatalf("summary lines = %d, want 2; output=%q", got, buf.String())
	}
}

// enqueue must not call slog synchronously. The background worker may aggregate
// immediately, but with a long flush window the logger remains untouched until
// stop/flush.
type countingHandler struct{ calls atomic.Uint64 }

func (h *countingHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *countingHandler) Handle(context.Context, slog.Record) error {
	h.calls.Add(1)
	return nil
}
func (h *countingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *countingHandler) WithGroup(string) slog.Handler      { return h }

func TestMatchLogEnqueueDoesNotLogSynchronously(t *testing.T) {
	h := &countingHandler{}
	p := newMatchLogPlane(slog.New(h), 16, 16, time.Hour)
	p.start()
	if !p.enqueue(testMatchRec("192.0.2.10")) {
		t.Fatal("enqueue unexpectedly dropped")
	}
	waitAtomic(t, p.processed.Load, 1)
	if got := h.calls.Load(); got != 0 {
		t.Fatalf("logger called before flush: %d", got)
	}
	p.stopAndDrain()
	if got := h.calls.Load(); got != 1 {
		t.Fatalf("logger calls after drain = %d, want 1", got)
	}
}

func TestMatchLogQueueSaturationDropsWithoutBlocking(t *testing.T) {
	p := newMatchLogPlane(nil, 1, 16, time.Hour)
	// Do not start a consumer; enable the producer gate so saturation is deterministic.
	p.accepting.Store(true)
	if !p.enqueue(testMatchRec("192.0.2.10")) {
		t.Fatal("first enqueue dropped")
	}
	if p.enqueue(testMatchRec("192.0.2.11")) {
		t.Fatal("second enqueue unexpectedly succeeded into full queue")
	}
	s := p.snapshot()
	if s.Dropped != 1 || s.QueueDropped != 1 || s.GroupDropped != 0 {
		t.Fatalf("unexpected drop counters: %+v", s)
	}
}

func TestMatchLogGroupCapDropsNewCardinality(t *testing.T) {
	p := newMatchLogPlane(nil, 16, 1, time.Hour)
	groups := make(map[matchLogKey]*matchLogBucket)
	now := time.Now()
	p.add(groups, testMatchRec("192.0.2.10"), now)
	p.add(groups, testMatchRec("192.0.2.11"), now)
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	s := p.snapshot()
	if s.GroupDropped != 1 || s.Dropped != 1 || s.Processed != 1 {
		t.Fatalf("unexpected counters: %+v", s)
	}
}

func TestMatchLogStopDrainsAcceptedEvents(t *testing.T) {
	p := newMatchLogPlane(nil, 64, 16, time.Hour)
	p.start()
	for i := 0; i < 32; i++ {
		if !p.enqueue(testMatchRec("192.0.2.10")) {
			t.Fatal("unexpected drop")
		}
	}
	p.stopAndDrain()
	s := p.snapshot()
	if s.Processed != 32 || s.Dropped != 0 || s.Accepting {
		t.Fatalf("unexpected drained snapshot: %+v", s)
	}
}

func BenchmarkMatchLogEnqueue(b *testing.B) {
	p := newMatchLogPlane(nil, 8192, 1, time.Hour)
	p.start()
	defer p.stopAndDrain()
	rec := testMatchRec("192.0.2.10")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.enqueue(rec)
	}
}

func BenchmarkMatchLogEnqueueSuccessful(b *testing.B) {
	p := newMatchLogPlane(nil, 1024, 1, time.Hour)
	p.accepting.Store(true)
	rec := testMatchRec("192.0.2.10")
	done := make(chan struct{})
	go func() {
		for i := 0; i < b.N; i++ {
			<-p.queue
		}
		close(done)
	}()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for !p.enqueue(rec) {
			runtime.Gosched()
		}
	}
	b.StopTimer()
	<-done
}

func BenchmarkSlogWarnPerMatch(b *testing.B) {
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	rec := testMatchRec("192.0.2.10")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Warn("rule match",
			"site", rec.Site, "rule_id", rec.RuleID, "severity", rec.Severity,
			"phase", rec.Phase, "client", rec.Client, "uri", rec.URI,
			"msg", rec.Msg, "data", rec.Data)
	}
}

func TestMatchLogSummaryBoundsSampleFields(t *testing.T) {
	var buf bytes.Buffer
	p := newMatchLogPlane(slog.New(slog.NewTextHandler(&buf, nil)), 8, 8, time.Hour)
	p.start()
	rec := testMatchRec("192.0.2.10")
	rec.URI = strings.Repeat("u", 1100)
	rec.Msg = strings.Repeat("m", 1100)
	rec.Data = strings.Repeat("d", 2200)
	p.enqueue(rec)
	waitAtomic(t, p.processed.Load, 1)
	p.stopAndDrain()
	out := buf.String()
	for _, marker := range []string{"uri_truncated=true", "msg_truncated=true", "data_truncated=true"} {
		if !strings.Contains(out, marker) {
			t.Fatalf("missing %s in %q", marker, out)
		}
	}
}

func TestMatchLogDropReportIsAggregated(t *testing.T) {
	var buf bytes.Buffer
	p := newMatchLogPlane(slog.New(slog.NewTextHandler(&buf, nil)), 1, 1, time.Hour)
	p.accepting.Store(true)
	p.enqueue(testMatchRec("192.0.2.10"))
	p.enqueue(testMatchRec("192.0.2.11")) // queue drop
	var last uint64
	p.reportDrops(&last)
	p.reportDrops(&last)
	if got := strings.Count(buf.String(), "msg=\"rule match logging telemetry dropped\""); got != 1 {
		t.Fatalf("drop summary lines = %d, want 1; output=%q", got, buf.String())
	}
}
