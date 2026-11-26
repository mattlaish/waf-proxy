package main

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"sync"
	"time"
)

// Transport observations, not a promise that a SIEM received/indexed a message.
type deliveryHealth struct {
	mu          sync.Mutex
	sent        uint64
	failed      uint64
	lastSuccess time.Time
	lastFailure time.Time
	generation  uint64
	epoch       string
}

// A process-specific nonce prevents a stale confirmation from matching another
// node/restart that happens to have the same numeric configuration generation.
func (h *deliveryHealth) configToken() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.epoch == "" {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return ""
		}
		h.epoch = hex.EncodeToString(b[:])
	}
	return h.epoch + ":" + strconv.FormatUint(h.generation, 10)
}

func (h *deliveryHealth) record(success bool) {
	h.recordAt(h.current(), success)
}

func (h *deliveryHealth) current() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.generation
}

func (h *deliveryHealth) invalidate() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.generation++
	h.lastSuccess = time.Time{}
	h.lastFailure = time.Time{}
}

func (h *deliveryHealth) recordAt(generation uint64, success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if success {
		h.sent++
		if generation == h.generation {
			h.lastSuccess = time.Now().UTC()
		}
	} else {
		h.failed++
		if generation == h.generation {
			h.lastFailure = time.Now().UTC()
		}
	}
}

func (h *deliveryHealth) snapshot(enabled bool, depth, capacity int, dropped uint64) map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	state := "unverified"
	if !enabled {
		state = "disabled"
	} else if h.lastFailure.After(h.lastSuccess) {
		state = "degraded"
	} else if !h.lastSuccess.IsZero() {
		state = "last_send_succeeded"
		if time.Since(h.lastSuccess) > 5*time.Minute {
			state = "stale"
		}
	}
	stamp := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339Nano)
	}
	return map[string]any{"enabled": enabled, "state": state, "sent": h.sent, "failures": h.failed,
		"last_success": stamp(h.lastSuccess), "last_failure": stamp(h.lastFailure),
		"queue_depth": depth, "queue_capacity": capacity, "dropped": dropped}
}
