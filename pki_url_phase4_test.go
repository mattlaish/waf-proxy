package main

import (
	"context"
	"crypto/x509"
	"net"
	"testing"
	"time"
)

func TestPhase4CRLURLSyntaxAndSSRFGuards(t *testing.T) {
	bad := []string{
		"http://example.com/a.crl",
		"https://u:p@example.com/a.crl",
		"https://example.com:8443/a.crl",
		"https://example.com/a.crl#fragment",
		" https://example.com/a.crl",
	}
	for _, raw := range bad {
		if err := validateCRLURLSyntax(raw); err == nil {
			t.Fatalf("accepted unsafe URL %q", raw)
		}
	}
	if err := validateCRLURLSyntax("https://example.com/issuer.crl"); err != nil {
		t.Fatalf("valid URL rejected: %v", err)
	}
	for _, rawIP := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fc00::1"} {
		if !unsafeCRLAddress(net.ParseIP(rawIP)) {
			t.Fatalf("SSRF address %s was not rejected", rawIP)
		}
	}
	if _, err := fetchCRLURL(context.Background(), "https://127.0.0.1/issuer.crl", x509.NewCertPool()); err == nil {
		t.Fatal("loopback CRL URL was fetched")
	}
}

func TestPhase4CRLRefreshRetainsLastKnownGoodOnFailure(t *testing.T) {
	old := &crlSnapshot{Lists: []*x509.RevocationList{{Raw: []byte("old")}}, LoadedAt: time.Now().Add(-time.Minute)}
	store := &crlStore{mode: "soft", cfg: BackendTLSConfig{CRLURLs: []string{"https://127.0.0.1/fail.crl"}}, roots: x509.NewCertPool()}
	store.current.Store(old)
	if err := store.refresh(context.Background()); err == nil {
		t.Fatal("unsafe refresh unexpectedly succeeded")
	}
	if got := store.current.Load(); got != old {
		t.Fatal("failed refresh replaced last-known-good CRL snapshot")
	}
	if store.status().LastError == "" {
		t.Fatal("failed refresh did not expose status error")
	}
}

func TestPhase4CRLRefreshDeduplicatesConcurrentAttempt(t *testing.T) {
	store := &crlStore{}
	store.mu.Lock()
	store.refreshing = true
	store.mu.Unlock()
	if err := store.refresh(context.Background()); err == nil {
		t.Fatal("concurrent refresh was not deduplicated")
	}
}

func TestPhase4CRLCacheRoundTrip(t *testing.T) {
	pki := newCRLTestPKI(t, "cache root")
	t.Setenv("WAF_CRL_CACHE_DIR", t.TempDir())
	const sourceURL = "https://example.com/issuer.crl"
	crlPEM := pki.crl(t, time.Now().Add(-time.Minute), time.Now().Add(time.Hour))
	if err := writeCRLCache(sourceURL, crlPEM); err != nil {
		t.Fatalf("write CRL cache: %v", err)
	}
	got := loadCachedCRLs([]string{sourceURL}, time.Now())
	if len(got) != 1 {
		t.Fatalf("cached CRL count=%d, want 1", len(got))
	}
	if got[0].NextUpdate.Before(time.Now()) {
		t.Fatal("cached CRL unexpectedly expired")
	}
}
