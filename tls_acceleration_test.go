package main

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"waf-proxy/internal/tlsfront"
)

func TestTLSFrontendListenerRemap(t *testing.T) {
	t.Setenv("WAF_TLS_FRONTEND_SOCKET_DIR", t.TempDir())
	cfg := defaultConfig()
	cfg.Sites[0].Listen = "127.0.0.1:18443"
	cfg.Sites[0].TLSCert, cfg.Sites[0].TLSKey = "/tmp/cert.pem", "/tmp/key.pem"
	cfg.TLSAcceleration.Mode = tlsfront.ModeFrontend
	got := listenerSet(cfg)
	key := tlsfront.InternalListenerKey(cfg.Sites[0].Listen)
	if isTLS, ok := got[key]; !ok || isTLS {
		t.Fatalf("listenerSet=%v, want private HTTP unix listener", got)
	}
	if _, ok := got[cfg.Sites[0].Listen]; ok {
		t.Fatalf("public TLS listener still owned by waf-proxy: %v", got)
	}
}

func TestPrepareTLSFrontendRequestSanitizesForwarding(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://internal/test?q=1", nil)
	r.Host = "app.example:443"
	r.Header.Set(tlsfront.ClientIPHeader, "198.51.100.9")
	r.Header.Set(tlsfront.ClientPortHeader, "43210")
	r.Header.Set(tlsfront.ProtoHeader, "https")
	r.Header.Set("X-Forwarded-For", "203.0.113.66")
	r.Header.Set("X-Real-IP", "203.0.113.67")
	got, err := prepareTLSFrontendRequest(r)
	if err != nil {
		t.Fatal(err)
	}
	if got.RemoteAddr != "198.51.100.9:43210" {
		t.Fatalf("RemoteAddr=%q", got.RemoteAddr)
	}
	if got.URL.Scheme != "https" || got.URL.Host != r.Host {
		t.Fatalf("URL=%s", got.URL.String())
	}
	if originalRequestScheme(got) != "https" {
		t.Fatalf("scheme marker missing")
	}
	for _, h := range []string{"X-Forwarded-For", "X-Real-IP", tlsfront.ClientIPHeader, tlsfront.ClientPortHeader, tlsfront.ProtoHeader} {
		if got.Header.Get(h) != "" {
			t.Fatalf("%s not stripped", h)
		}
	}
}

func TestProxyRewriteRestoresHTTPSForwarding(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://internal/", nil)
	r.Host = "app.example"
	r.RemoteAddr = "198.51.100.9:43210"
	r.Header.Set(tlsfront.ClientIPHeader, "198.51.100.9")
	r.Header.Set(tlsfront.ClientPortHeader, "43210")
	r.Header.Set(tlsfront.ProtoHeader, "https")
	var err error
	r, err = prepareTLSFrontendRequest(r)
	if err != nil {
		t.Fatal(err)
	}
	site := SiteConfig{Name: "s", PreserveHost: true}
	pool := &poolRuntime{name: "p"}
	p := buildProxy(pool, site, defaultConfig(), discardLogger(), nil)
	pr := &httputil.ProxyRequest{In: r, Out: r.Clone(r.Context())}
	p.Rewrite(pr)
	if pr.Out.Header.Get("X-Forwarded-Proto") != "https" {
		t.Fatalf("xfp=%q", pr.Out.Header.Get("X-Forwarded-Proto"))
	}
}

func TestTLSFrontendPublisherWritesControl(t *testing.T) {
	d := t.TempDir()
	path := filepath.Join(d, "live.json")
	p := &tlsFrontendPublisher{path: path, statusPath: filepath.Join(d, "status.json")}
	cfg := defaultConfig()
	cfg.TLSAcceleration.Mode = tlsfront.ModeGo
	if err := p.publish(cfg); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"tls_acceleration"`) {
		t.Fatalf("missing tls acceleration: %s", b)
	}
}
