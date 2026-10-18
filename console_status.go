package main

import (
	"crypto/tls"
	"crypto/x509"
	"time"
)

// runtimeCertificateExpiry reports only the certificate held by the active Go
// listener. It never reads key files, refreshes certificates or guesses an
// external TLS frontend's certificate. Empty means unavailable, not healthy.
func runtimeCertificateExpiry(rt *runtimeState, site string) string {
	if rt == nil {
		return ""
	}
	var earliest time.Time
	visit := func(sr *siteRuntime) {
		if sr == nil || sr.cfg.Name != site {
			return
		}
		expires := certificateExpiry(sr.cert)
		if !expires.IsZero() && (earliest.IsZero() || expires.Before(earliest)) {
			earliest = expires
		}
	}
	for _, lr := range rt.listeners {
		if lr == nil {
			continue
		}
		visit(lr.catchAll)
		for _, sr := range lr.exact {
			visit(sr)
		}
		for _, sr := range lr.wildcard {
			visit(sr)
		}
	}
	if earliest.IsZero() {
		return ""
	}
	return earliest.UTC().Format(time.RFC3339)
}

func certificateExpiry(cert *tls.Certificate) time.Time {
	if cert == nil {
		return time.Time{}
	}
	if cert.Leaf != nil {
		return cert.Leaf.NotAfter
	}
	if len(cert.Certificate) > 0 {
		if leaf, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {
			return leaf.NotAfter
		}
	}
	return time.Time{}
}
