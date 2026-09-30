package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

// debugRequestContext carries the per-request debug-evidence identity through
// the request context so downstream stages (Coraza observer, proxy transport)
// can merge evidence into the same bundle. It is only present when debug
// evidence capture is active for the tenant.
type debugRequestContext struct {
	Tenant        string
	TransactionID string
	Started       time.Time
}

type debugRequestContextKey struct{}

// debugContextFrom returns the debug request context stored on ctx, if any.
func debugContextFrom(ctx context.Context) (debugRequestContext, bool) {
	if ctx == nil {
		return debugRequestContext{}, false
	}
	dc, ok := ctx.Value(debugRequestContextKey{}).(debugRequestContext)
	return dc, ok
}

// newDebugTransactionID returns a random 128-bit hex identifier for one captured
// request. It is a correlation key, not an authentication token; on CSPRNG
// failure it falls back to a monotonic-ish local value rather than blocking.
func newDebugTransactionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return "dbg-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}
