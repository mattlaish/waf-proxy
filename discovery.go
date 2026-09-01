package main

// Slim, passive host observation.
//
// The WAF already sees the Host header (and SNI, via the same value used for
// cert selection) on every request. This just remembers which hostnames arrive
// on which listener, and whether a configured site claims them. It's a
// visibility flag for the rare case of an undeclared hostname reaching the box —
// not a discovery engine: no crawling, no proposals, no auto-creation. If you
// see an undeclared host and want it served, you create the site normally.

import (
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

type discoveredHost struct {
	Host      string `json:"host"`
	Listener  string `json:"listener"`
	Declared  bool   `json:"declared"`
	Hits      int64  `json:"hits"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

type hostObserver struct {
	mu   sync.Mutex
	seen map[string]*discoveredHost // key: listener|host
	cap  int
	log  *slog.Logger
}

func newHostObserver(l *slog.Logger) *hostObserver {
	return &hostObserver{seen: map[string]*discoveredHost{}, cap: 500, log: l}
}

// note records one request's host on a listener. declared = a site claimed it.
// New undeclared hosts are logged once (so they reach syslog/SIEM too).
func (o *hostObserver) note(listener, host string, declared bool) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return
	}
	if i := strings.IndexByte(host, ':'); i >= 0 { // strip any :port
		host = host[:i]
	}
	key := listener + "|" + host
	now := time.Now().Format("2006-01-02 15:04:05")

	o.mu.Lock()
	defer o.mu.Unlock()
	if h, ok := o.seen[key]; ok {
		h.Hits++
		h.LastSeen = now
		h.Declared = declared
		return
	}
	if len(o.seen) >= o.cap { // bounded: drop the oldest by last-seen
		var oldestKey string
		var oldest string
		for k, v := range o.seen {
			if oldestKey == "" || v.LastSeen < oldest {
				oldestKey, oldest = k, v.LastSeen
			}
		}
		delete(o.seen, oldestKey)
	}
	o.seen[key] = &discoveredHost{
		Host: host, Listener: listener, Declared: declared,
		Hits: 1, FirstSeen: now, LastSeen: now,
	}
	if !declared && o.log != nil {
		o.log.Warn("undeclared host observed", "host", host, "listener", listener)
	}
}

// snapshot returns discovered hosts, undeclared first, then by hits.
func (o *hostObserver) snapshot() []discoveredHost {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]discoveredHost, 0, len(o.seen))
	for _, v := range o.seen {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Declared != out[j].Declared {
			return !out[i].Declared // undeclared first
		}
		return out[i].Hits > out[j].Hits
	})
	return out
}
