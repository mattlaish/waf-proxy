package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// recentDropWindow is how long a dropped message keeps the delivery flagged
// for review. A brief outage stops warning on its own once it is over.
const recentDropWindow = time.Hour

// Transport observations, not a promise that a SIEM received/indexed a message.
type deliveryHealth struct {
	mu          sync.Mutex
	sent        uint64
	failed      uint64
	lastSuccess time.Time
	lastFailure time.Time
	lastError   string
	lastDrop    time.Time
	generation  uint64
	epoch       string
}

// deliveryErrorReason turns a delivery error into a short reason that is safe
// to show and store: URL wrappers are removed because a webhook URL or its
// query can carry credentials.
func deliveryErrorReason(err error) string {
	if err == nil {
		return ""
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	reason := err.Error()
	if strings.Contains(reason, "://") {
		reason = "request failed"
	}
	if len(reason) > 160 {
		reason = reason[:160]
	}
	return reason
}

// fail records a failed delivery attempt with its reason.
func (h *deliveryHealth) fail(generation uint64, err error) {
	h.recordAt(generation, false)
	h.mu.Lock()
	defer h.mu.Unlock()
	if generation == h.generation {
		h.lastError = deliveryErrorReason(err)
	}
}

// noteDrop records that a message was dropped (queue full, backoff, failure).
func (h *deliveryHealth) noteDrop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastDrop = time.Now().UTC()
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
	h.lastError = ""
	h.lastDrop = time.Time{}
}

func (h *deliveryHealth) recordAt(generation uint64, success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if success {
		h.sent++
		if generation == h.generation {
			h.lastSuccess = time.Now().UTC()
			h.lastError = ""
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
		// No age-based "stale" state: a healthy integration with nothing to
		// send is not a fault. The last_success timestamp shows its age.
		state = "last_send_succeeded"
	}
	stamp := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339Nano)
	}
	recentDrops := !h.lastDrop.IsZero() && time.Since(h.lastDrop) < recentDropWindow
	return map[string]any{"enabled": enabled, "state": state, "sent": h.sent, "failures": h.failed,
		"last_success": stamp(h.lastSuccess), "last_failure": stamp(h.lastFailure), "last_error": h.lastError,
		"last_drop": stamp(h.lastDrop), "recent_drops": recentDrops,
		"queue_depth": depth, "queue_capacity": capacity, "dropped": dropped}
}
