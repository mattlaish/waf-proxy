package tlsfront

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ModeGo       = "go"
	ModeFrontend = "frontend"

	AccelOff      = "off"
	AccelAuto     = "auto"
	AccelRequired = "required"

	ClientIPHeader   = "X-WAF-TLS-Client-IP"
	ClientPortHeader = "X-WAF-TLS-Client-Port"
	ProtoHeader      = "X-WAF-TLS-Proto"

	defaultSocketDir = "/run/waf-proxy/tls"
)

// AccelerationConfig controls the optional OpenSSL-based TLS frontend. The
// default/zero value preserves the existing Go crypto/tls listener path.
type AccelerationConfig struct {
	Mode              string `json:"mode,omitempty"`             // go | frontend
	KTLS              string `json:"ktls,omitempty"`             // off | auto | required
	QAT               string `json:"qat,omitempty"`              // off | auto | required
	WorkerProcesses   int    `json:"worker_processes,omitempty"` // 0 = auto
	WorkerConnections int    `json:"worker_connections,omitempty"`
	HTTP2             *bool  `json:"http2,omitempty"`
}

func (c AccelerationConfig) Effective() AccelerationConfig {
	if c.Mode == "" {
		c.Mode = ModeGo
	}
	if c.KTLS == "" {
		c.KTLS = AccelOff
	}
	if c.QAT == "" {
		c.QAT = AccelOff
	}
	if c.WorkerConnections == 0 {
		c.WorkerConnections = 4096
	}
	if c.HTTP2 == nil {
		v := true
		c.HTTP2 = &v
	}
	return c
}

func FrontendEnabled(c AccelerationConfig) bool {
	return c.Effective().Mode == ModeFrontend
}

func Validate(c AccelerationConfig) error {
	e := c.Effective()
	if e.Mode != ModeGo && e.Mode != ModeFrontend {
		return errors.New("tls_acceleration.mode must be go or frontend")
	}
	for name, v := range map[string]string{"ktls": e.KTLS, "qat": e.QAT} {
		if v != AccelOff && v != AccelAuto && v != AccelRequired {
			return fmt.Errorf("tls_acceleration.%s must be off, auto, or required", name)
		}
	}
	if e.WorkerProcesses < 0 || e.WorkerProcesses > 256 {
		return errors.New("tls_acceleration.worker_processes must be 0-256")
	}
	if e.WorkerConnections < 0 || e.WorkerConnections > 1048576 {
		return errors.New("tls_acceleration.worker_connections must be 0-1048576")
	}
	return nil
}

type Site struct {
	Name      string   `json:"name"`
	Listen    string   `json:"listen"`
	Hostnames []string `json:"hostnames"`
	TLSCert   string   `json:"tls_cert,omitempty"`
	TLSKey    string   `json:"tls_key,omitempty"`
}

type LiveConfig struct {
	TLSAcceleration   AccelerationConfig `json:"tls_acceleration"`
	Sites             []Site             `json:"sites"`
	ReadTimeoutSec    int                `json:"read_timeout_sec,omitempty"`
	IdleTimeoutSec    int                `json:"idle_timeout_sec,omitempty"`
	BackendTimeoutSec int                `json:"backend_timeout_sec,omitempty"`
	PublishedAt       time.Time          `json:"published_at"`
}

func (c LiveConfig) Validate() error {
	if err := Validate(c.TLSAcceleration); err != nil {
		return err
	}
	frontend := FrontendEnabled(c.TLSAcceleration)
	for _, s := range c.Sites {
		if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.Listen) == "" {
			return errors.New("tls frontend site requires name and listen")
		}
		if (s.TLSCert == "") != (s.TLSKey == "") {
			return fmt.Errorf("site %q: tls_cert and tls_key must both be set or both be empty", s.Name)
		}
		if !frontend {
			continue
		}
		for _, h := range s.Hostnames {
			if err := validateNginxToken(h); err != nil {
				return fmt.Errorf("site %q hostname %q: %w", s.Name, h, err)
			}
		}
		if s.TLSCert != "" {
			if err := validatePathToken(s.TLSCert); err != nil {
				return fmt.Errorf("site %q tls_cert: %w", s.Name, err)
			}
			if err := validatePathToken(s.TLSKey); err != nil {
				return fmt.Errorf("site %q tls_key: %w", s.Name, err)
			}
		}
	}
	return nil
}

func SocketDir() string {
	if v := strings.TrimSpace(os.Getenv("WAF_TLS_FRONTEND_SOCKET_DIR")); v != "" && filepath.IsAbs(v) {
		return filepath.Clean(v)
	}
	return defaultSocketDir
}

func InternalSocketPath(publicListen string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(publicListen)))
	return filepath.Join(SocketDir(), "listener-"+hex.EncodeToString(sum[:8])+".sock")
}

func InternalListenerKey(publicListen string) string {
	return "unix:" + InternalSocketPath(publicListen)
}
func IsInternalListenerKey(key string) bool { return strings.HasPrefix(key, "unix:") }

func SocketPathFromKey(key string) (string, bool) {
	if !IsInternalListenerKey(key) {
		return "", false
	}
	p := strings.TrimPrefix(key, "unix:")
	if !filepath.IsAbs(p) {
		return "", false
	}
	return filepath.Clean(p), true
}

func PublicTLSListeners(sites []Site) map[string]bool {
	out := map[string]bool{}
	for _, s := range sites {
		if _, ok := out[s.Listen]; !ok {
			out[s.Listen] = false
		}
		if s.TLSCert != "" && s.TLSKey != "" {
			out[s.Listen] = true
		}
	}
	return out
}

func ActualListenerKey(cfg AccelerationConfig, sites []Site, publicListen string) string {
	if FrontendEnabled(cfg) && PublicTLSListeners(sites)[publicListen] {
		return InternalListenerKey(publicListen)
	}
	return publicListen
}

func PublicListenForKey(cfg AccelerationConfig, sites []Site, key string) string {
	if !FrontendEnabled(cfg) || !IsInternalListenerKey(key) {
		return key
	}
	for public, isTLS := range PublicTLSListeners(sites) {
		if isTLS && InternalListenerKey(public) == key {
			return public
		}
	}
	return key
}

type Probe struct {
	NginxPath       string `json:"nginx_path,omitempty"`
	NginxVersion    string `json:"nginx_version,omitempty"`
	OpenSSLPath     string `json:"openssl_path,omitempty"`
	OpenSSLVersion  string `json:"openssl_version,omitempty"`
	KernelKTLS      bool   `json:"kernel_ktls"`
	OpenSSLKTLS     bool   `json:"openssl_ktls"`
	QATProvider     bool   `json:"qat_provider"`
	QATProviderInfo string `json:"qat_provider_info,omitempty"`
	Error           string `json:"error,omitempty"`
}

type Resolved struct {
	KTLS                 bool `json:"ktls"`
	QAT                  bool `json:"qat"`
	ModernHTTP2Directive bool `json:"modern_http2_directive"`
}

func ProbeHost(ctx context.Context) Probe {
	var p Probe
	p.KernelKTLS = kernelKTLSSupported()
	if path, err := exec.LookPath("nginx"); err == nil {
		p.NginxPath = path
		out, _ := exec.CommandContext(ctx, path, "-v").CombinedOutput()
		p.NginxVersion = strings.TrimSpace(string(out))
	}
	if path, err := exec.LookPath("openssl"); err == nil {
		p.OpenSSLPath = path
		out, _ := exec.CommandContext(ctx, path, "version", "-a").CombinedOutput()
		p.OpenSSLVersion = firstLine(string(out))
		help, _ := exec.CommandContext(ctx, path, "s_server", "-help").CombinedOutput()
		p.OpenSSLKTLS = bytes.Contains(bytes.ToLower(help), []byte("-ktls"))
		qat := exec.CommandContext(ctx, path, "list", "-providers", "-provider", "qatprovider", "-provider", "default")
		qout, qerr := qat.CombinedOutput()
		if qerr == nil && bytes.Contains(bytes.ToLower(qout), []byte("qatprovider")) {
			p.QATProvider = true
			p.QATProviderInfo = strings.TrimSpace(string(qout))
		}
	}
	if p.NginxPath == "" {
		p.Error = "nginx not found"
	} else if p.OpenSSLPath == "" {
		p.Error = "openssl not found"
	}
	return p
}

func Resolve(c AccelerationConfig, p Probe) (Resolved, error) {
	e := c.Effective()
	if e.Mode != ModeFrontend {
		return Resolved{}, nil
	}
	if p.NginxPath == "" {
		return Resolved{}, errors.New("nginx is required for tls_acceleration.mode=frontend")
	}
	if p.OpenSSLPath == "" {
		return Resolved{}, errors.New("openssl is required for tls_acceleration.mode=frontend")
	}
	r := Resolved{ModernHTTP2Directive: SupportsModernHTTP2Directive(p.NginxVersion)}
	ktlsAvailable := p.KernelKTLS && p.OpenSSLKTLS
	switch e.KTLS {
	case AccelAuto:
		r.KTLS = ktlsAvailable
	case AccelRequired:
		if !ktlsAvailable {
			return Resolved{}, errors.New("kTLS is required but kernel/OpenSSL capability was not detected")
		}
		r.KTLS = true
	}
	switch e.QAT {
	case AccelAuto:
		r.QAT = p.QATProvider
	case AccelRequired:
		if !p.QATProvider {
			return Resolved{}, errors.New("Intel QAT provider is required but qatprovider could not be loaded")
		}
		r.QAT = true
	}
	return r, nil
}

// SupportsModernHTTP2Directive reports whether nginx supports the standalone
// `http2 on;` directive introduced in 1.25.1. Unknown/unparseable versions use
// the legacy listen parameter because it is compatible with older nginx.
func SupportsModernHTTP2Directive(raw string) bool {
	const marker = "nginx/"
	i := strings.Index(strings.ToLower(raw), marker)
	if i < 0 {
		return false
	}
	v := raw[i+len(marker):]
	end := len(v)
	for i, r := range v {
		if (r < '0' || r > '9') && r != '.' {
			end = i
			break
		}
	}
	parts := strings.Split(v[:end], ".")
	if len(parts) < 2 {
		return false
	}
	nums := [3]int{}
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return false
		}
		nums[i] = n
	}
	if nums[0] != 1 {
		return nums[0] > 1
	}
	if nums[1] != 25 {
		return nums[1] > 25
	}
	return nums[2] >= 1
}

func kernelKTLSSupported() bool {
	if b, err := os.ReadFile("/proc/sys/net/ipv4/tcp_available_ulp"); err == nil {
		for _, f := range strings.Fields(string(b)) {
			if f == "tls" {
				return true
			}
		}
	}
	if _, err := os.Stat("/proc/net/tls_stat"); err == nil {
		return true
	}
	return false
}

func RenderOpenSSLQAT() []byte {
	return []byte(`openssl_conf = openssl_init

[openssl_init]
providers = provider_section

[provider_section]
qatprovider = qat_prov_section
default = default_sect

[qat_prov_section]
activate = 1

[default_sect]
activate = 1
`)
}

func RenderNginx(live LiveConfig, resolved Resolved, runtimeDir string) ([]byte, error) {
	if err := live.Validate(); err != nil {
		return nil, err
	}
	e := live.TLSAcceleration.Effective()
	if e.Mode != ModeFrontend {
		return nil, errors.New("tls frontend rendering requires mode=frontend")
	}
	if runtimeDir == "" || !filepath.IsAbs(runtimeDir) {
		return nil, errors.New("runtime directory must be absolute")
	}

	groups := map[string][]Site{}
	tlsMap := PublicTLSListeners(live.Sites)
	for _, s := range live.Sites {
		if tlsMap[s.Listen] {
			groups[s.Listen] = append(groups[s.Listen], s)
		}
	}
	listens := make([]string, 0, len(groups))
	for l := range groups {
		listens = append(listens, l)
	}
	sort.Strings(listens)

	var b strings.Builder
	b.WriteString("# Generated by waf-tlsfront. Do not edit.\n")
	b.WriteString("pid " + nginxQuote(filepath.Join(runtimeDir, "nginx.pid")) + ";\n")
	b.WriteString("error_log stderr warn;\n")
	if e.WorkerProcesses == 0 {
		b.WriteString("worker_processes auto;\n")
	} else {
		fmt.Fprintf(&b, "worker_processes %d;\n", e.WorkerProcesses)
	}
	b.WriteString("events {\n")
	fmt.Fprintf(&b, "  worker_connections %d;\n", e.WorkerConnections)
	b.WriteString("  multi_accept on;\n}\n\n")
	b.WriteString("http {\n")
	b.WriteString("  access_log off;\n")
	b.WriteString("  map $http_upgrade $connection_upgrade { default upgrade; '' ''; }\n")
	b.WriteString("  server_tokens off;\n")
	b.WriteString("  ssl_protocols TLSv1.2 TLSv1.3;\n")
	b.WriteString("  ssl_session_cache shared:WAFSSL:50m;\n")
	b.WriteString("  ssl_session_timeout 1d;\n")
	if resolved.KTLS {
		b.WriteString("  ssl_conf_command Options KTLS;\n")
	}
	readTO := live.ReadTimeoutSec
	if readTO <= 0 {
		readTO = 15
	}
	backendTO := live.BackendTimeoutSec
	if backendTO <= 0 {
		backendTO = 30
	}
	idleTO := live.IdleTimeoutSec
	if idleTO <= 0 {
		idleTO = 60
	}
	fmt.Fprintf(&b, "  client_header_timeout %ds;\n", readTO)
	fmt.Fprintf(&b, "  client_body_timeout %ds;\n", readTO)
	fmt.Fprintf(&b, "  keepalive_timeout %ds;\n", idleTO)
	b.WriteString("  client_max_body_size 0;\n\n")

	for _, listen := range listens {
		ss := append([]Site(nil), groups[listen]...)
		sort.SliceStable(ss, func(i, j int) bool { return ss[i].Name < ss[j].Name })
		defCert, defKey := "", ""
		for _, s := range ss {
			if s.TLSCert != "" && s.TLSKey != "" {
				defCert, defKey = s.TLSCert, s.TLSKey
				break
			}
		}
		if defCert == "" {
			continue
		}
		defaultIndex := 0
		for i, s := range ss {
			for _, h := range s.Hostnames {
				if h == "*" {
					defaultIndex = i
					break
				}
			}
		}
		upstreamName := "waf_" + socketID(listen)
		sock := InternalSocketPath(listen)
		fmt.Fprintf(&b, "  upstream %s {\n", upstreamName)
		b.WriteString("    server unix:" + sock + ";\n")
		b.WriteString("    keepalive 256;\n")
		b.WriteString("  }\n\n")
		for i, s := range ss {
			cert, key := s.TLSCert, s.TLSKey
			if cert == "" {
				cert, key = defCert, defKey
			}
			b.WriteString("  server {\n")
			listenToken, err := nginxListen(listen)
			if err != nil {
				return nil, err
			}
			defaultFlag := ""
			if i == defaultIndex {
				defaultFlag = " default_server"
			}
			h2 := e.HTTP2 != nil && *e.HTTP2
			if h2 && resolved.ModernHTTP2Directive {
				fmt.Fprintf(&b, "    listen %s ssl%s;\n", listenToken, defaultFlag)
				b.WriteString("    http2 on;\n")
			} else {
				legacyH2 := ""
				if h2 {
					legacyH2 = " http2"
				}
				fmt.Fprintf(&b, "    listen %s ssl%s%s;\n", listenToken, legacyH2, defaultFlag)
			}
			fmt.Fprintf(&b, "    ssl_certificate %s;\n", nginxQuote(cert))
			fmt.Fprintf(&b, "    ssl_certificate_key %s;\n", nginxQuote(key))
			names := make([]string, 0, len(s.Hostnames))
			for _, h := range s.Hostnames {
				if h == "*" {
					names = append(names, "_")
				} else {
					names = append(names, nginxQuote(h))
				}
			}
			if len(names) == 0 {
				names = []string{"_"}
			}
			b.WriteString("    server_name " + strings.Join(names, " ") + ";\n")
			b.WriteString("    location / {\n")
			b.WriteString("      proxy_http_version 1.1;\n")
			b.WriteString("      proxy_request_buffering off;\n")
			b.WriteString("      proxy_buffering off;\n")
			b.WriteString("      proxy_set_header Host $http_host;\n")
			b.WriteString("      proxy_set_header X-Forwarded-For \"\";\n")
			b.WriteString("      proxy_set_header Forwarded \"\";\n")
			b.WriteString("      proxy_set_header X-Real-IP \"\";\n")
			b.WriteString("      proxy_set_header " + ClientIPHeader + " $remote_addr;\n")
			b.WriteString("      proxy_set_header " + ClientPortHeader + " $remote_port;\n")
			b.WriteString("      proxy_set_header " + ProtoHeader + " https;\n")
			b.WriteString("      proxy_set_header Upgrade $http_upgrade;\n")
			b.WriteString("      proxy_set_header Connection $connection_upgrade;\n")
			fmt.Fprintf(&b, "      proxy_connect_timeout %ds;\n", backendTO)
			fmt.Fprintf(&b, "      proxy_read_timeout %ds;\n", backendTO+10)
			fmt.Fprintf(&b, "      proxy_send_timeout %ds;\n", backendTO+10)
			fmt.Fprintf(&b, "      proxy_pass http://%s;\n", upstreamName)
			b.WriteString("    }\n")
			b.WriteString("  }\n\n")
		}
	}
	b.WriteString("}\n")
	return []byte(b.String()), nil
}

func nginxListen(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q: %w", addr, err)
	}
	if host == "" {
		return port, nil
	}
	if strings.Contains(host, ":") {
		return "[" + host + "]:" + port, nil
	}
	return host + ":" + port, nil
}

func socketID(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:6])
}

func nginxQuote(v string) string { return strconv.Quote(v) }

func validateNginxToken(v string) error {
	if strings.TrimSpace(v) == "" {
		return errors.New("value is empty")
	}
	if strings.ContainsAny(v, "\r\n\x00;$ {}\t") {
		return errors.New("contains characters not allowed in generated nginx configuration")
	}
	return nil
}

func validatePathToken(v string) error {
	if !filepath.IsAbs(v) {
		return errors.New("path must be absolute")
	}
	if strings.ContainsAny(v, "\r\n\x00") {
		return errors.New("path contains control characters")
	}
	return nil
}

func firstLine(v string) string {
	if i := strings.IndexByte(v, '\n'); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

func MarshalStatus(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return append(b, '\n')
}

// Preflight validates the generated nginx configuration against the installed
// nginx/OpenSSL stack. It performs no bind/listen and is intended for config
// Apply paths and deployment diagnostics, not request processing.
func Preflight(ctx context.Context, live LiveConfig, runtimeDir string) (Probe, Resolved, error) {
	probe := ProbeHost(ctx)
	resolved, err := Resolve(live.TLSAcceleration, probe)
	if err != nil {
		return probe, resolved, err
	}
	if live.TLSAcceleration.Effective().Mode != ModeFrontend {
		return probe, resolved, nil
	}
	stage, err := os.MkdirTemp("", "waf-tlsfront-preflight-*")
	if err != nil {
		return probe, resolved, err
	}
	defer os.RemoveAll(stage)
	nginx, err := RenderNginx(live, resolved, stage)
	if err != nil {
		return probe, resolved, err
	}
	_ = runtimeDir
	nginxPath := filepath.Join(stage, "nginx.conf")
	if err := os.WriteFile(nginxPath, nginx, 0o600); err != nil {
		return probe, resolved, err
	}
	env := append([]string(nil), os.Environ()...)
	env = removeEnvKey(env, "OPENSSL_CONF")
	if resolved.QAT {
		opensslPath := filepath.Join(stage, "openssl.cnf")
		if err := os.WriteFile(opensslPath, RenderOpenSSLQAT(), 0o600); err != nil {
			return probe, resolved, err
		}
		env = append(env, "OPENSSL_CONF="+opensslPath)
	}
	cmd := exec.CommandContext(ctx, probe.NginxPath, "-t", "-c", nginxPath)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return probe, resolved, fmt.Errorf("nginx configuration preflight failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return probe, resolved, nil
}

func removeEnvKey(env []string, key string) []string {
	prefix := key + "="
	out := env[:0]
	for _, v := range env {
		if !strings.HasPrefix(v, prefix) {
			out = append(out, v)
		}
	}
	return out
}
