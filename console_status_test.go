package main

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"
)

func TestConsoleRuntimeCertificateExpiry(t *testing.T) {
	expiry := time.Date(2026, 11, 1, 12, 0, 0, 0, time.FixedZone("test", 8*3600))
	makeSite := func(name string, at time.Time) *siteRuntime {
		return &siteRuntime{cfg: SiteConfig{Name: name}, cert: &tls.Certificate{Leaf: &x509.Certificate{NotAfter: at}}}
	}
	lr := &listenerRuntime{}
	lr.exact = map[string]*siteRuntime{"shop.test": makeSite("shop", expiry.Add(time.Hour))}
	lr.wildcard = map[string]*siteRuntime{".shop.test": makeSite("shop", expiry)}
	lr.catchAll = makeSite("other", expiry.Add(-time.Hour))
	rt := &runtimeState{listeners: map[string]*listenerRuntime{":443": lr, ":80": nil}}
	if got := runtimeCertificateExpiry(rt, "shop"); got != "2026-11-01T04:00:00Z" {
		t.Fatalf("earliest active shop certificate: %q", got)
	}
	for _, name := range []string{"missing", "external-tls"} {
		if got := runtimeCertificateExpiry(rt, name); got != "" {
			t.Fatalf("unavailable certificate must not look healthy: %q", got)
		}
	}
	if runtimeCertificateExpiry(nil, "shop") != "" {
		t.Fatal("nil runtime must be unavailable")
	}
	for _, cert := range []*tls.Certificate{nil, {}, {Certificate: [][]byte{[]byte("invalid DER")}}} {
		if !certificateExpiry(cert).IsZero() {
			t.Fatal("unreadable certificate must be unavailable")
		}
	}
}
