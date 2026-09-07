package main

// Bounded asynchronous aggregation for Coraza rule-match structured logs.
//
// WAF enforcement, match-ring recording, learner telemetry, syslog forwarding,
// and AI match handling stay on their existing paths. Only the human-oriented
// slog warning is moved off the Coraza callback so attack floods cannot turn
// stdout/log serialization into request-path backpressure.

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

const (
	matchLogQueueCapacity = 8192
	matchLogMaxGroups     = 1024
	matchLogFlushInterval = 5 * time.Second
)

type matchLogKey struct {
	site   string
	ruleID int
	client string
}

type matchLogBucket struct {
	sample    matchRec
	count     uint64
	firstSeen time.Time
	lastSeen  time.Time
}

type matchLogSnapshot struct {
	QueueDepth    int    `json:"queue_depth"`
	QueueCapacity int    `json:"queue_capacity"`
	Dropped       uint64 `json:"dropped"`
	QueueDropped  uint64 `json:"queue_dropped"`
	GroupDropped  uint64 `json:"group_dropped"`
	Processed     uint64 `json:"processed"`
	Emitted       uint64 `json:"emitted"`
	ActiveGroups  int    `json:"active_groups"`
	MaxGroups     int    `json:"max_groups"`
	Accepting     bool   `json:"accepting"`
}

type matchLogPlane struct {
	queue chan matchRec
	stop  chan struct{}
	done  chan struct{}
	log   *slog.Logger

	flushEvery time.Duration
	maxGroups  int

	accepting    atomic.Bool
	dropped      atomic.Uint64
	queueDropped atomic.Uint64
	groupDropped atomic.Uint64
	processed    atomic.Uint64
	emitted      atomic.Uint64
	groups       atomic.Int64
	started      atomic.Bool

	startOnce sync.Once
	stopOnce  sync.Once
}

func newMatchLogPlane(log *slog.Logger, capacity, maxGroups int, flushEvery time.Duration) *matchLogPlane {
	if capacity <= 0 {
		capacity = 1
	}
	if maxGroups <= 0 {
		maxGroups = 1
	}
	if flushEvery <= 0 {
		flushEvery = matchLogFlushInterval
	}
	return &matchLogPlane{
		queue:      make(chan matchRec, capacity),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		log:        log,
		flushEvery: flushEvery,
		maxGroups:  maxGroups,
	}
}

func (p *matchLogPlane) start() {
	if p == nil {
		return
	}
	p.startOnce.Do(func() {
		p.accepting.Store(true)
		p.started.Store(true)
		go p.run()
	})
}

// enqueue is deliberately non-blocking. Logging is visibility only and must
// never delay or fail a WAF transaction.
func (p *matchLogPlane) enqueue(rec matchRec) bool {
	if p == nil || !p.accepting.Load() {
		return false
	}
	select {
	case p.queue <- rec:
		return true
	default:
		p.dropped.Add(1)
		p.queueDropped.Add(1)
		return false
	}
}

func (p *matchLogPlane) add(groups map[matchLogKey]*matchLogBucket, rec matchRec, now time.Time) {
	key := matchLogKey{site: rec.Site, ruleID: rec.RuleID, client: rec.Client}
	if b := groups[key]; b != nil {
		b.count++
		b.lastSeen = now
		p.processed.Add(1)
		return
	}
	if len(groups) >= p.maxGroups {
		p.dropped.Add(1)
		p.groupDropped.Add(1)
		return
	}
	groups[key] = &matchLogBucket{sample: rec, count: 1, firstSeen: now, lastSeen: now}
	p.groups.Store(int64(len(groups)))
	p.processed.Add(1)
}

func matchLogField(s string, limit int) (string, bool) {
	if limit <= 0 || len(s) <= limit {
		return s, false
	}
	return s[:limit], true
}

func (p *matchLogPlane) flush(groups map[matchLogKey]*matchLogBucket) {
	if len(groups) == 0 {
		return
	}
	for _, b := range groups {
		if p.log != nil {
			uri, uriTruncated := matchLogField(b.sample.URI, 1024)
			msg, msgTruncated := matchLogField(b.sample.Msg, 1024)
			data, dataTruncated := matchLogField(b.sample.Data, 2048)
			p.log.Warn("rule match summary",
				"site", b.sample.Site,
				"rule_id", b.sample.RuleID,
				"severity", b.sample.Severity,
				"phase", b.sample.Phase,
				"client", b.sample.Client,
				"count", b.count,
				"first_seen", b.firstSeen.Format(time.RFC3339Nano),
				"last_seen", b.lastSeen.Format(time.RFC3339Nano),
				"uri", uri, "uri_truncated", uriTruncated,
				"msg", msg, "msg_truncated", msgTruncated,
				"data", data, "data_truncated", dataTruncated)
			p.emitted.Add(1)
		}
	}
	clear(groups)
	p.groups.Store(0)
}

func (p *matchLogPlane) reportDrops(last *uint64) {
	if p == nil || last == nil {
		return
	}
	total := p.dropped.Load()
	if total == *last {
		return
	}
	if p.log != nil {
		p.log.Warn("rule match logging telemetry dropped",
			"dropped", total-*last,
			"dropped_total", total,
			"queue_dropped_total", p.queueDropped.Load(),
			"group_dropped_total", p.groupDropped.Load(),
			"queue_depth", len(p.queue),
			"queue_capacity", cap(p.queue),
			"active_groups", p.groups.Load(),
			"max_groups", p.maxGroups)
	}
	*last = total
}

func (p *matchLogPlane) run() {
	defer close(p.done)
	ticker := time.NewTicker(p.flushEvery)
	defer ticker.Stop()
	groups := make(map[matchLogKey]*matchLogBucket)
	var reportedDrops uint64

	for {
		select {
		case rec := <-p.queue:
			p.add(groups, rec, time.Now())
		case <-ticker.C:
			p.flush(groups)
			p.reportDrops(&reportedDrops)
		case <-p.stop:
			for {
				select {
				case rec := <-p.queue:
					p.add(groups, rec, time.Now())
				default:
					p.flush(groups)
					p.reportDrops(&reportedDrops)
					return
				}
			}
		}
	}
}

func (p *matchLogPlane) stopAndDrain() {
	if p == nil || !p.started.Load() {
		return
	}
	p.stopOnce.Do(func() {
		p.accepting.Store(false)
		close(p.stop)
	})
	<-p.done
}

func (p *matchLogPlane) snapshot() matchLogSnapshot {
	if p == nil {
		return matchLogSnapshot{}
	}
	return matchLogSnapshot{
		QueueDepth:    len(p.queue),
		QueueCapacity: cap(p.queue),
		Dropped:       p.dropped.Load(),
		QueueDropped:  p.queueDropped.Load(),
		GroupDropped:  p.groupDropped.Load(),
		Processed:     p.processed.Load(),
		Emitted:       p.emitted.Load(),
		ActiveGroups:  int(p.groups.Load()),
		MaxGroups:     p.maxGroups,
		Accepting:     p.accepting.Load(),
	}
}
