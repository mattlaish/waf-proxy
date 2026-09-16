package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestCorrelationKey struct{}

func withRequestCorrelationID(r *http.Request) *http.Request {
	id := r.Header.Get("X-Request-ID")
	if len(id) < 8 || len(id) > 128 {
		var b [16]byte
		_, _ = rand.Read(b[:])
		id = hex.EncodeToString(b[:])
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
