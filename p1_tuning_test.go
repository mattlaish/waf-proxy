package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestPolicyDirectivesResponseBodyInspection(t *testing.T) {
	base := PolicyConfig{}
	got := policyDirectives(base, "DetectionOnly")
	if strings.Contains(got, "SecResponseBodyAccess") || strings.Contains(got, "SecResponseBodyLimit") {
		t.Fatalf("inherit policy unexpectedly overrides response body directives: %q", got)
	}

	off := policyDirectives(PolicyConfig{ResponseBodyInspection: "off"}, "DetectionOnly")
	if !strings.Contains(off, "SecResponseBodyAccess Off\n") {
		t.Fatalf("off policy missing response-body disable directive: %q", off)
	}

	on := policyDirectives(PolicyConfig{ResponseBodyInspection: "on", ResponseBodyLimit: 262144}, "DetectionOnly")
	if !strings.Contains(on, "SecResponseBodyAccess On\n") || !strings.Contains(on, "SecResponseBodyLimit 262144\n") {
		t.Fatalf("on policy missing response-body directives: %q", on)
	}

	inherit := policyDirectives(PolicyConfig{ResponseBodyInspection: "inherit"}, "DetectionOnly")
	if strings.Contains(inherit, "SecResponseBodyAccess") {
		t.Fatalf("explicit inherit unexpectedly overrides response body access: %q", inherit)
	}
}

func TestResponseBodyPolicyValidation(t *testing.T) {
	c := validTestConfig(t)
	c.Policies[0].ResponseBodyInspection = "sometimes"
	if err := c.validate(); err == nil || !strings.Contains(err.Error(), "response_body_inspection") {
		t.Fatalf("validate invalid response_body_inspection error = %v", err)
	}

	c = validTestConfig(t)
	c.Policies[0].ResponseBodyLimit = -1
	if err := c.validate(); err == nil || !strings.Contains(err.Error(), "response_body_limit") {
		t.Fatalf("validate negative response_body_limit error = %v", err)
	}
}

func TestEffectiveBackendTransportDefaults(t *testing.T) {
	got := effectiveBackendTransport(BackendTransportConfig{})
	want := defaultBackendTransportConfig()
	if got != want {
		t.Fatalf("effective default transport = %#v, want %#v", got, want)
	}
	if got.MaxConnsPerHost != 0 {
		t.Fatalf("default max_conns_per_host = %d, want unlimited (0)", got.MaxConnsPerHost)
	}
}

func TestEffectiveBackendTransportPreservesOverrides(t *testing.T) {
	in := BackendTransportConfig{
		MaxIdleConns:        5000,
		MaxIdleConnsPerHost: 700,
		MaxConnsPerHost:     900,
		IdleConnTimeoutSec:  120,
		DialTimeoutSec:      2,
		KeepAliveSec:        45,
	}
	if got := effectiveBackendTransport(in); got != in {
		t.Fatalf("explicit transport changed: got %#v want %#v", got, in)
	}
}

func TestBackendTransportValidation(t *testing.T) {
	cases := []BackendTransportConfig{
		{MaxIdleConns: -1},
		{MaxIdleConnsPerHost: 20000},
		{MaxConnsPerHost: 70000},
		{IdleConnTimeoutSec: 3601},
		{DialTimeoutSec: 301},
		{KeepAliveSec: 3601},
	}
	for _, tc := range cases {
		if err := validateBackendTransport(tc); err == nil {
			t.Fatalf("validateBackendTransport(%#v) unexpectedly passed", tc)
		}
	}
}

func TestBuildBackendTransportUsesPoolTuning(t *testing.T) {
	pool := &poolRuntime{transport: BackendTransportConfig{
		MaxIdleConns:        4096,
		MaxIdleConnsPerHost: 512,
		MaxConnsPerHost:     1024,
		IdleConnTimeoutSec:  150,
		DialTimeoutSec:      3,
		KeepAliveSec:        40,
	}}
	tr := buildBackendTransport(pool, Config{BackendTimeoutSec: 17})
	if tr.MaxIdleConns != 4096 || tr.MaxIdleConnsPerHost != 512 || tr.MaxConnsPerHost != 1024 {
		t.Fatalf("connection pool tuning not applied: %#v", tr)
	}
	if tr.IdleConnTimeout != 150*time.Second {
		t.Fatalf("IdleConnTimeout = %v", tr.IdleConnTimeout)
	}
	if tr.ResponseHeaderTimeout != 17*time.Second {
		t.Fatalf("ResponseHeaderTimeout = %v", tr.ResponseHeaderTimeout)
	}
	if !tr.ForceAttemptHTTP2 {
		t.Fatal("ForceAttemptHTTP2 unexpectedly disabled")
	}
}

func TestDefaultConfigUsesTunedBackendPool(t *testing.T) {
	c := defaultConfig()
	if len(c.Pools) == 0 {
		t.Fatal("default config has no pool")
	}
	got := c.Pools[0].Transport
	want := defaultBackendTransportConfig()
	if got != want {
		t.Fatalf("default pool transport = %#v, want %#v", got, want)
	}
}

func TestBuildProxyUsesPoolSharedTransport(t *testing.T) {
	shared := buildBackendTransport(nil, Config{BackendTimeoutSec: 9})
	pool := &poolRuntime{name: "shared", httpTransport: shared}
	proxy := buildProxy(pool, SiteConfig{Name: "site"}, Config{BackendTimeoutSec: 3}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	lb, ok := proxy.Transport.(lbTransport)
	if !ok {
		t.Fatalf("proxy transport type = %T, want lbTransport", proxy.Transport)
	}
	if lb.base != shared {
		t.Fatal("buildProxy did not reuse the pool-scoped backend transport")
	}
}

func TestAdminUIExposesP1Tuning(t *testing.T) {
	html := string(adminHTML)
	for _, want := range []string{
		`data-ftr="max_idle_conns"`,
		`data-ftr="max_idle_conns_per_host"`,
		`data-ftr="max_conns_per_host"`,
		`data-f="response_body_inspection"`,
		`data-f="response_body_limit"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("admin UI missing %q", want)
		}
	}
}

func TestPoolStatusExposesEffectiveTransport(t *testing.T) {
	p := &poolRuntime{name: "p", transport: BackendTransportConfig{MaxIdleConns: 1234}}
	st := p.status()
	if st.Transport.MaxIdleConns != 1234 {
		t.Fatalf("status max_idle_conns = %d", st.Transport.MaxIdleConns)
	}
	if st.Transport.MaxIdleConnsPerHost != defaultBackendMaxIdleConnsPerHost {
		t.Fatalf("status max_idle_conns_per_host = %d, want default %d", st.Transport.MaxIdleConnsPerHost, defaultBackendMaxIdleConnsPerHost)
	}
}
