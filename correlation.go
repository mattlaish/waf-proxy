package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"
)

type requestCorrelationKey struct{}

func validRequestCorrelationID(id string) bool {
	if len(id) < 8 || len(id) > 128 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':' {
			continue
		}
		return false
	}
	return true
}

func newRequestCorrelationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	// Correlation IDs are not authentication tokens. A monotonic-ish local
	// fallback is safer than accepting attacker-controlled bytes when the CSPRNG
	// is unavailable.
	return "req-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func withRequestCorrelationID(r *http.Request) *http.Request {
	id := r.Header.Get("X-Request-ID")
	if !validRequestCorrelationID(id) {
		id = newRequestCorrelationID()
	}
	return r.WithContext(context.WithValue(r.Context(), requestCorrelationKey{}, id))
}

func requestCorrelationID(r *http.Request) string {
	if v, ok := r.Context().Value(requestCorrelationKey{}).(string); ok {
		return v
	}
	return ""
}

func correlationWrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = withRequestCorrelationID(r)
		w.Header().Set("X-Request-ID", requestCorrelationID(r))
		next.ServeHTTP(w, r)
	})
}
