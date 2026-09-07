package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestObservationPlaneDrainAppliesTelemetry(t *testing.T) {
	log := discardLogger()
	hosts := newHostObserver(log)
	maps := newSiteMaps(128)
	learn := newLearnStore()
	signals := newSignalStore()
	p := newObservationPlane(hosts, maps, learn, signals, log, 16)
	p.start()

	fields := []DiscoveredField{{
		Name: "password", Type: "password", Method: "POST", Action: "/api/login",
		DiscoverySource: fieldSourcePassive,
	}}
	p.noteHost(":8443", "Example.COM:8443", false)
	p.noteMatch("site-a", "/api/login", 942100, "192.0.2.10", "critical")
	p.noteRequest("site-a", "POST", "/api/login", "next=%2F&debug=1", "application/json", 200, fields)
	p.stopAndDrain()

	stats := p.snapshot()
	if stats.Processed != 3 || stats.Dropped != 0 || stats.QueueDepth != 0 || stats.Accepting {
		t.Fatalf("observation stats = %+v, want processed=3 dropped=0 depth=0 accepting=false", stats)
	}

	gotHosts := hosts.snapshot()
	if len(gotHosts) != 1 || gotHosts[0].Host != "example.com" || gotHosts[0].Hits != 1 || gotHosts[0].Declared {
		t.Fatalf("host observation = %#v", gotHosts)
	}

	sm := maps.forSite("site-a")
	sm.mu.Lock()
	api := sm.root.children["api"]
	var login *pathNode
	if api != nil {
		login = api.children["login"]
	}
	if login == nil || login.hits != 1 || login.lastCode != 200 {
		sm.mu.Unlock()
		t.Fatalf("sitemap login node = %#v", login)
	}
	sm.mu.Unlock()

	sl := learn.forSite("site-a")
	sl.mu.Lock()
	pa := sl.paths["/api/login"]
	if pa == nil || pa.hits != 1 || pa.ok2xx != 1 {
		sl.mu.Unlock()
		t.Fatalf("learn request aggregate = %#v", pa)
	}
	ra := pa.rules[942100]
	if ra == nil || ra.count != 1 {
		sl.mu.Unlock()
		t.Fatalf("learn rule aggregate = %#v", ra)
	}
	sl.mu.Unlock()

	summary := signals.summary("site-a", "/api/login")
	for _, want := range []string{"password field", "JSON content-type", "query params", "observed POST/PUT"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("signal summary %q missing %q", summary, want)
		}
	}
}

func TestObservationPlaneFullQueueDropsWithoutBlocking(t *testing.T) {
	p := newObservationPlane(nil, nil, nil, nil, nil, 1)
	p.accepting.Store(true)
	ev := observationEvent{kind: observationHost, listener: ":443", host: "example.com"}
	if !p.enqueue(ev) {
		t.Fatal("first enqueue unexpectedly failed")
	}
	if p.enqueue(ev) {
		t.Fatal("second enqueue unexpectedly succeeded into a full queue")
	}
	stats := p.snapshot()
	if stats.QueueDepth != 1 || stats.Dropped != 1 {
		t.Fatalf("stats after full queue = %+v", stats)
	}
}

func TestObservationEnqueueZeroAllocs(t *testing.T) {
	p := newObservationPlane(nil, nil, nil, nil, nil, 1)
	p.accepting.Store(true)
	ev := observationEvent{
		kind: observationRequest, site: "site-a", method: "GET", path: "/health",
		contentType: "application/json", code: 200,
	}
	allocs := testing.AllocsPerRun(1000, func() {
		if !p.enqueue(ev) {
			panic("observation queue unexpectedly full")
		}
		<-p.queue
	})
	if allocs != 0 {
		t.Fatalf("observation enqueue allocated %.2f objects/run, want 0", allocs)
	}
}

func BenchmarkObservationRequestSynchronous(b *testing.B) {
	maps := newSiteMaps(128)
	learn := newLearnStore()
	signals := newSignalStore()
	fields := []DiscoveredField{{Name: "user", Method: "POST", Action: "/api/login", DiscoverySource: fieldSourcePassive}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		maps.record("site-a", "POST", "/api/login", 200, srcObserved)
		learn.noteRequest("site-a", "/api/login", 200)
		signals.noteRequestShape("site-a", "/api/login", "POST", "next=%2F", "application/json", fields)
	}
}

func BenchmarkObservationRequestEnqueue(b *testing.B) {
	p := newObservationPlane(nil, nil, nil, nil, nil, 1)
	p.accepting.Store(true)
	ev := observationEvent{
		kind: observationRequest, site: "site-a", method: "POST", path: "/api/login",
		rawQuery: "next=%2F", contentType: "application/json", code: 200,
		fields: []DiscoveredField{{Name: "user", Method: "POST", Action: "/api/login", DiscoverySource: fieldSourcePassive}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !p.enqueue(ev) {
			b.Fatal("observation queue unexpectedly full")
		}
		<-p.queue
	}
}

func BenchmarkHostObservationSynchronous(b *testing.B) {
	hosts := newHostObserver(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hosts.note(":443", "www.example.com", true)
	}
}

func BenchmarkHostObservationEnqueue(b *testing.B) {
	p := newObservationPlane(nil, nil, nil, nil, nil, 1)
	p.accepting.Store(true)
	ev := observationEvent{kind: observationHost, listener: ":443", host: "www.example.com", declared: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !p.enqueue(ev) {
			b.Fatal("observation queue unexpectedly full")
		}
		<-p.queue
	}
}

func BenchmarkObservationRequestSynchronousParallel(b *testing.B) {
	maps := newSiteMaps(128)
	learn := newLearnStore()
	signals := newSignalStore()
	fields := []DiscoveredField{{Name: "user", Method: "POST", Action: "/api/login", DiscoverySource: fieldSourcePassive}}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			maps.record("site-a", "POST", "/api/login", 200, srcObserved)
			learn.noteRequest("site-a", "/api/login", 200)
			signals.noteRequestShape("site-a", "/api/login", "POST", "next=%2F", "application/json", fields)
		}
	})
}

func BenchmarkObservationRequestEnqueueParallel(b *testing.B) {
	p := newObservationPlane(nil, nil, nil, nil, nil, 1024)
	p.accepting.Store(true)
	ev := observationEvent{
		kind: observationRequest, site: "site-a", method: "POST", path: "/api/login",
		rawQuery: "next=%2F", contentType: "application/json", code: 200,
		fields: []DiscoveredField{{Name: "user", Method: "POST", Action: "/api/login", DiscoverySource: fieldSourcePassive}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if !p.enqueue(ev) {
				panic("observation queue unexpectedly full")
			}
			<-p.queue
		}
	})
}
