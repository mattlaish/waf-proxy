package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	loginAttemptWindow = 5 * time.Minute
	loginBlockDuration = 5 * time.Minute
	loginAttemptLimit  = 8
	loginAttemptMaxKey = 4096
)

type loginAttempt struct {
	Count        int
	WindowStart  time.Time
	BlockedUntil time.Time
}

type loginAttemptLimiter struct {
	mu       sync.Mutex
	entries  map[string]loginAttempt
	overflow loginAttempt
}

func newLoginAttemptLimiter() *loginAttemptLimiter {
	return &loginAttemptLimiter{entries: map[string]loginAttempt{}}
}

func adminRemoteHost(r *http.Request) string {
	host := strings.TrimSpace(r.RemoteAddr)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if len(host) > 128 {
		host = host[:128]
	}
	return host
}

func loginAttemptKeys(r *http.Request, username string) []string {
	u := strings.ToLower(strings.TrimSpace(username))
	if len(u) > 128 {
		u = u[:128]
	}
	return []string{"ip:" + adminRemoteHost(r), "user:" + u}
}

func (l *loginAttemptLimiter) pruneLocked(now time.Time) {
	for k, v := range l.entries {
		if now.After(v.BlockedUntil) && now.Sub(v.WindowStart) > loginAttemptWindow {
			delete(l.entries, k)
		}
	}
	if !l.overflow.WindowStart.IsZero() && now.After(l.overflow.BlockedUntil) && now.Sub(l.overflow.WindowStart) > loginAttemptWindow {
		l.overflow = loginAttempt{}
	}
}

func (l *loginAttemptLimiter) stateLocked(key string) (loginAttempt, bool) {
	if v, ok := l.entries[key]; ok {
		return v, false
	}
	if len(l.entries) >= loginAttemptMaxKey {
		return l.overflow, true
	}
	return loginAttempt{}, false
}

func (l *loginAttemptLimiter) storeLocked(key string, v loginAttempt, overflow bool) {
	if overflow {
		l.overflow = v
		return
	}
	l.entries[key] = v
}

func (l *loginAttemptLimiter) allow(keys []string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)
	var retry time.Duration
	for _, k := range keys {
		v, _ := l.stateLocked(k)
		if now.Before(v.BlockedUntil) {
			r := v.BlockedUntil.Sub(now)
			if r > retry {
				retry = r
			}
		}
	}
	return retry <= 0, retry
}

func (l *loginAttemptLimiter) failure(keys []string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)
	for _, k := range keys {
		v, overflow := l.stateLocked(k)
		if v.WindowStart.IsZero() || now.Sub(v.WindowStart) > loginAttemptWindow {
			v = loginAttempt{WindowStart: now}
		}
		v.Count++
		if v.Count >= loginAttemptLimit {
			v.BlockedUntil = now.Add(loginBlockDuration)
		}
		l.storeLocked(k, v, overflow)
	}
}

func (l *loginAttemptLimiter) success(keys []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		delete(l.entries, k)
	}
}

// mutationAuditWriter captures status without changing Admin API response bodies.
type mutationAuditWriter struct {
	http.ResponseWriter
	status int
}

func (w *mutationAuditWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *mutationAuditWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
