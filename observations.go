package main

// Bounded asynchronous observation plane.
//
// Host discovery, sitemap updates, learner accounting, and request-shape
// signals are visibility/learning telemetry. None of them participates in the
// blocking WAF verdict, so the data plane must never wait on their mutexes or
// allocations. Requests enqueue a small immutable event with a non-blocking
// send; a single background consumer updates the existing stores. When the
// queue is full, telemetry is dropped and counted rather than applying
// backpressure to traffic.

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

const observationQueueCapacity = 8192

type observationKind uint8

const (
	observationHost observationKind = iota + 1
	observationRequest
	observationMatch
)

type observationEvent struct {
	kind observationKind

	listener string
	host     string
	declared bool

	site        string
	method      string
	path        string
	rawQuery    string
	contentType string
	code        int
	fields      []DiscoveredField

	ruleID   int
	client   string
	severity string
}

type observationSnapshot struct {
	QueueDepth    int    `json:"queue_depth"`
	QueueCapacity int    `json:"queue_capacity"`
	Dropped       uint64 `json:"dropped"`
	Processed     uint64 `json:"processed"`
	Accepting     bool   `json:"accepting"`
}

type observationPlane struct {
	queue chan observationEvent
	stop  chan struct{}
	done  chan struct{}
	log   *slog.Logger

	hosts   *hostObserver
	maps    *siteMaps
	learn   *learnStore
	signals *signalStore

	accepting atomic.Bool
	dropped   atomic.Uint64
	processed atomic.Uint64
	started   atomic.Bool

	startOnce sync.Once
	stopOnce  sync.Once
}

func newObservationPlane(hosts *hostObserver, maps *siteMaps, learn *learnStore, signals *signalStore, log *slog.Logger, capacity int) *observationPlane {
	if capacity <= 0 {
		capacity = 1
	}
	return &observationPlane{
		queue:   make(chan observationEvent, capacity),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
		log:     log,
		hosts:   hosts,
		maps:    maps,
		learn:   learn,
		signals: signals,
	}
}

func (p *observationPlane) start() {
	if p == nil {
		return
	}
	p.startOnce.Do(func() {
		p.accepting.Store(true)
		p.started.Store(true)
		go p.run()
	})
}

// enqueue is deliberately non-blocking. A full observation queue is a
// telemetry-loss condition, not a request-path availability condition.
func (p *observationPlane) enqueue(ev observationEvent) bool {
	if p == nil || !p.accepting.Load() {
		return false
	}
	select {
	case p.queue <- ev:
		return true
	default:
		p.dropped.Add(1)
		return false
	}
}

func (p *observationPlane) noteHost(listener, host string, declared bool) {
	p.enqueue(observationEvent{kind: observationHost, listener: listener, host: host, declared: declared})
}

func (p *observationPlane) noteRequest(site, method, path, rawQuery, contentType string, code int, fields []DiscoveredField) {
	p.enqueue(observationEvent{
		kind: observationRequest, site: site, method: method, path: path,
		rawQuery: rawQuery, contentType: contentType, code: code, fields: fields,
	})
}

func (p *observationPlane) noteMatch(site, path string, ruleID int, client, severity string) {
	p.enqueue(observationEvent{
		kind: observationMatch, site: site, path: path, ruleID: ruleID,
		client: client, severity: severity,
	})
}

func (p *observationPlane) process(ev observationEvent) {
	switch ev.kind {
	case observationHost:
		if p.hosts != nil {
			p.hosts.note(ev.listener, ev.host, ev.declared)
		}
	case observationRequest:
		if p.maps != nil {
			p.maps.record(ev.site, ev.method, ev.path, ev.code, srcObserved)
		}
		if p.learn != nil {
			p.learn.noteRequest(ev.site, ev.path, ev.code)
		}
		if p.signals != nil {
			p.signals.noteRequestShape(ev.site, ev.path, ev.method, ev.rawQuery, ev.contentType, ev.fields)
		}
	case observationMatch:
		if p.learn != nil {
			p.learn.noteMatch(ev.site, ev.path, ev.ruleID, ev.client, ev.severity)
		}
	}
	p.processed.Add(1)
}

func (p *observationPlane) reportDrops(last *uint64) {
	if p == nil || last == nil {
		return
	}
	total := p.dropped.Load()
	if total == *last {
		return
	}
	if p.log != nil {
		p.log.Warn("observation telemetry dropped",
			"dropped", total-*last,
			"dropped_total", total,
			"queue_depth", len(p.queue),
			"queue_capacity", cap(p.queue))
	}
	*last = total
}

func (p *observationPlane) run() {
	defer close(p.done)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var reportedDrops uint64

	for {
		select {
		case ev := <-p.queue:
			p.process(ev)
		case <-ticker.C:
			p.reportDrops(&reportedDrops)
		case <-p.stop:
			// Requests have already been stopped before shutdown calls us. Drain
			// everything accepted so sitemap/learner state is not truncated.
			for {
				select {
				case ev := <-p.queue:
					p.process(ev)
				default:
					p.reportDrops(&reportedDrops)
					return
				}
			}
		}
	}
}

func (p *observationPlane) stopAndDrain() {
	if p == nil || !p.started.Load() {
		return
	}
	p.stopOnce.Do(func() {
		p.accepting.Store(false)
		close(p.stop)
	})
	<-p.done
}

func (p *observationPlane) snapshot() observationSnapshot {
	if p == nil {
		return observationSnapshot{}
	}
	return observationSnapshot{
		QueueDepth:    len(p.queue),
		QueueCapacity: cap(p.queue),
		Dropped:       p.dropped.Load(),
		Processed:     p.processed.Load(),
		Accepting:     p.accepting.Load(),
	}
}
