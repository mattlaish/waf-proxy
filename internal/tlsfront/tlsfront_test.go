package tlsfront

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func boolPtr(v bool) *bool { return &v }

func TestAccelerationDefaultsPreserveGoTLS(t *testing.T) {
	e := (AccelerationConfig{}).Effective()
	if e.Mode != ModeGo || e.KTLS != AccelOff || e.QAT != AccelOff || e.WorkerConnections != 4096 || e.HTTP2 == nil || !*e.HTTP2 {
		t.Fatalf("unexpected defaults: %#v", e)
	}
	if FrontendEnabled(AccelerationConfig{}) {
		t.Fatal("zero config unexpectedly enables frontend")
	}
}

func TestInternalSocketMappingDeterministic(t *testing.T) {
	t.Setenv("WAF_TLS_FRONTEND_SOCKET_DIR", t.TempDir())
	a := InternalListenerKey(":443")
	b := InternalListenerKey(":443")
	c := InternalListenerKey("192.0.2.10:443")
	if a != b || a == c || !strings.HasPrefix(a, "unix:") {
		t.Fatalf("bad mapping: a=%q b=%q c=%q", a, b, c)
	}
}

func TestSupportsModernHTTP2Directive(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"nginx version: nginx/1.24.0", false},
		{"nginx version: nginx/1.25.0", false},
		{"nginx version: nginx/1.25.1", true},
		{"nginx version: nginx/1.26.3", true},
		{"nginx version: nginx/2.0.0", true},
		{"unknown", false},
	}
	for _, tc := range cases {
		if got := SupportsModernHTTP2Directive(tc.version); got != tc.want {
			t.Errorf("SupportsModernHTTP2Directive(%q)=%v want %v", tc.version, got, tc.want)
		}
	}
}

func TestResolveAutoAndRequired(t *testing.T) {
	c := AccelerationConfig{Mode: ModeFrontend, KTLS: AccelAuto, QAT: AccelAuto}
	p := Probe{NginxPath: "/usr/sbin/nginx", NginxVersion: "nginx version: nginx/1.26.3", OpenSSLPath: "/usr/bin/openssl", KernelKTLS: true, OpenSSLKTLS: true, QATProvider: true}
	r, err := Resolve(c, p)
	if err != nil || !r.KTLS || !r.QAT || !r.ModernHTTP2Directive {
		t.Fatalf("resolve auto = %#v, %v", r, err)
	}
	_, err = Resolve(AccelerationConfig{Mode: ModeFrontend, KTLS: AccelRequired, QAT: AccelRequired}, Probe{NginxPath: "/n", OpenSSLPath: "/o"})
	if err == nil {
		t.Fatal("required accelerators unexpectedly accepted without capability")
	}
}

func testLive() LiveConfig {
	return LiveConfig{
		TLSAcceleration: AccelerationConfig{Mode: ModeFrontend, KTLS: AccelAuto, QAT: AccelOff, WorkerConnections: 8192, HTTP2: boolPtr(true)},
		Sites: []Site{
			{Name: "shop", Listen: ":443", Hostnames: []string{"shop.example.com"}, TLSCert: "/etc/waf/certs/shop.crt", TLSKey: "/etc/waf/certs/shop.key"},
			{Name: "fallback", Listen: ":443", Hostnames: []string{"*"}},
		},
		ReadTimeoutSec: 15, IdleTimeoutSec: 60, BackendTimeoutSec: 30,
	}
}

func TestRenderNginxModernHTTP2(t *testing.T) {
	t.Setenv("WAF_TLS_FRONTEND_SOCKET_DIR", t.TempDir())
	out, err := RenderNginx(testLive(), Resolved{KTLS: true, ModernHTTP2Directive: true}, "/run/waf-tls-frontend")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		"ssl_conf_command Options KTLS;",
		"server unix:" + InternalSocketPath(":443") + ";",
		"proxy_set_header " + ClientIPHeader + " $remote_addr;",
		"proxy_set_header " + ProtoHeader + " https;",
		"proxy_set_header X-Forwarded-For \"\";",
		"listen 443 ssl",
		"http2 on;",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("nginx config missing %q\n%s", want, text)
		}
	}
	if strings.Contains(text, "listen 443 ssl http2") {
		t.Fatalf("modern nginx config still uses deprecated listen http2 parameter:\n%s", text)
	}
}

func TestRenderNginxLegacyHTTP2(t *testing.T) {
	t.Setenv("WAF_TLS_FRONTEND_SOCKET_DIR", t.TempDir())
	out, err := RenderNginx(testLive(), Resolved{ModernHTTP2Directive: false}, "/run/waf-tls-frontend")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "listen 443 ssl http2") {
		t.Fatalf("legacy nginx config missing listen http2 parameter:\n%s", text)
	}
	if strings.Contains(text, "http2 on;") {
		t.Fatalf("legacy nginx config contains unsupported standalone http2 directive:\n%s", text)
	}
}

func TestRenderNginxHTTP2Disabled(t *testing.T) {
	t.Setenv("WAF_TLS_FRONTEND_SOCKET_DIR", t.TempDir())
	live := testLive()
	live.TLSAcceleration.HTTP2 = boolPtr(false)
	out, err := RenderNginx(live, Resolved{ModernHTTP2Directive: true}, "/run/waf-tls-frontend")
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, "http2") {
		t.Fatalf("http2 disabled but generated config contains http2:\n%s", text)
	}
}

func TestLiveConfigRejectsNginxInjectionInFrontendMode(t *testing.T) {
	live := LiveConfig{TLSAcceleration: AccelerationConfig{Mode: ModeFrontend}, Sites: []Site{{Name: "x", Listen: ":443", Hostnames: []string{"bad;$host"}, TLSCert: "/a.crt", TLSKey: "/a.key"}}}
	if err := live.Validate(); err == nil {
		t.Fatal("injection-like hostname unexpectedly accepted")
	}
}

func TestProbeHostWithFakeBinaries(t *testing.T) {
	d := t.TempDir()
	nginx := filepath.Join(d, "nginx")
	openssl := filepath.Join(d, "openssl")
	if err := os.WriteFile(nginx, []byte("#!/bin/sh\necho 'nginx version: nginx/1.26.0' >&2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	sslScript := `#!/bin/sh
if [ "$1" = "version" ]; then echo 'OpenSSL 3.6.3'; exit 0; fi
if [ "$1" = "s_server" ]; then echo 'usage: -ktls'; exit 0; fi
if [ "$1" = "list" ]; then echo 'Providers: qatprovider default'; exit 0; fi
exit 0
`
	if err := os.WriteFile(openssl, []byte(sslScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", d)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	p := ProbeHost(ctx)
	if p.NginxPath == "" || p.OpenSSLPath == "" || !p.OpenSSLKTLS || !p.QATProvider || !SupportsModernHTTP2Directive(p.NginxVersion) {
		t.Fatalf("probe = %#v", p)
	}
}

func TestPreflightWithFakeNginxOpenSSL(t *testing.T) {
	d := t.TempDir()
	nginx := filepath.Join(d, "nginx")
	openssl := filepath.Join(d, "openssl")
	if err := os.WriteFile(nginx, []byte("#!/bin/sh\nif [ \"$1\" = \"-v\" ]; then echo 'nginx version: nginx/1.26.0' >&2; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(openssl, []byte("#!/bin/sh\nif [ \"$1\" = \"version\" ]; then echo 'OpenSSL 3.6.3'; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", d)
	t.Setenv("WAF_TLS_FRONTEND_SOCKET_DIR", filepath.Join(d, "sockets"))
	live := LiveConfig{TLSAcceleration: AccelerationConfig{Mode: ModeFrontend, KTLS: AccelOff, QAT: AccelOff}, Sites: []Site{{Name: "s", Listen: ":443", Hostnames: []string{"s.example"}, TLSCert: "/etc/waf/certs/s.crt", TLSKey: "/etc/waf/certs/s.key"}}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	probe, resolved, err := Preflight(ctx, live, "/run/waf-tls-frontend")
	if err != nil {
		t.Fatal(err)
	}
	if probe.NginxPath == "" || probe.OpenSSLPath == "" || resolved.KTLS || resolved.QAT || !resolved.ModernHTTP2Directive {
		t.Fatalf("unexpected preflight: probe=%#v resolved=%#v", probe, resolved)
	}
}
