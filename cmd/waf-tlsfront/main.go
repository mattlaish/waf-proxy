// waf-tlsfront manages the optional OpenSSL/nginx TLS termination plane used
// by waf-proxy. It never handles WAF policy itself; decrypted HTTP is forwarded
// over private Unix-domain sockets owned by the waf service account.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"waf-proxy/internal/tlsfront"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
)

const (
	defaultControl = "/run/waf-proxy/tls-frontend-live.json"
	defaultOutDir  = "/run/waf-tls-frontend"
)

type runtimeStatus struct {
	Mode      string            `json:"mode,omitempty"`
	Active    bool              `json:"active"`
	NginxPID  int               `json:"nginx_pid,omitempty"`
	Resolved  tlsfront.Resolved `json:"resolved"`
	Probe     tlsfront.Probe    `json:"probe"`
	LastError string            `json:"last_error,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	Version   string            `json:"version,omitempty"`
	Commit    string            `json:"commit,omitempty"`
}

type rendered struct {
	nginx    []byte
	openssl  []byte
	resolved tlsfront.Resolved
	probe    tlsfront.Probe
	hash     string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "preflight":
		os.Exit(runPreflight(os.Args[2:]))
	case "render":
		os.Exit(runRender(os.Args[2:]))
	case "run":
		os.Exit(runManager(os.Args[2:]))
	case "version":
		fmt.Printf("waf-tlsfront %s (%s)\n", buildVersion, buildCommit)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: waf-tlsfront <preflight|render|run|version> [options]")
}

func runPreflight(args []string) int {
	fs := flag.NewFlagSet("preflight", flag.ContinueOnError)
	config := fs.String("config", "/etc/waf/config.json", "waf-proxy JSON config or live TLS frontend control file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	live, err := loadLive(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	probeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	probe := tlsfront.ProbeHost(probeCtx)
	resolved, err := tlsfront.Resolve(live.TLSAcceleration, probe)
	out := map[string]any{"config": live.TLSAcceleration.Effective(), "probe": probe, "resolved": resolved, "ok": err == nil}
	if err != nil {
		out["error"] = err.Error()
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
	if err != nil {
		return 1
	}
	return 0
}

func runRender(args []string) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	config := fs.String("config", "/etc/waf/config.json", "waf-proxy JSON config or live TLS frontend control file")
	outDir := fs.String("out-dir", defaultOutDir, "directory for generated nginx/OpenSSL files")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	live, err := loadLive(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if live.TLSAcceleration.Effective().Mode != tlsfront.ModeFrontend {
		fmt.Fprintln(os.Stderr, "tls_acceleration.mode is not frontend")
		return 1
	}
	r, err := prepare(live, *outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := installRendered(r, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("rendered %s (kTLS=%v QAT=%v modernHTTP2=%v)\n", filepath.Join(*outDir, "nginx.conf"), r.resolved.KTLS, r.resolved.QAT, r.resolved.ModernHTTP2Directive)
	return 0
}

func runManager(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	control := fs.String("control", defaultControl, "live TLS frontend control file published by waf-proxy")
	outDir := fs.String("out-dir", defaultOutDir, "runtime directory")
	poll := fs.Duration("poll", time.Second, "control-file polling interval")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !filepath.IsAbs(*control) || !filepath.IsAbs(*outDir) {
		fmt.Fprintln(os.Stderr, "control and out-dir must be absolute")
		return 2
	}
	if *poll < 50*time.Millisecond {
		fmt.Fprintln(os.Stderr, "poll must be at least 50ms")
		return 2
	}
	if err := os.MkdirAll(*outDir, 0o750); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	m := &manager{control: *control, outDir: *outDir, poll: *poll}
	return m.run(ctx)
}

type manager struct {
	control string
	outDir  string
	poll    time.Duration

	mu          sync.Mutex
	child       *exec.Cmd
	waitCh      chan error
	currentHash string
	currentQAT  bool
	lastControl string
	lastStatus  runtimeStatus
}

func (m *manager) run(ctx context.Context) int {
	ticker := time.NewTicker(m.poll)
	defer ticker.Stop()
	m.reconcile(ctx)
	for {
		var wait <-chan error
		m.mu.Lock()
		if m.waitCh != nil {
			wait = m.waitCh
		}
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			m.stopChild(10 * time.Second)
			m.writeStatus(runtimeStatus{Mode: tlsfront.ModeGo, Active: false})
			return 0
		case err, ok := <-wait:
			if !ok {
				continue
			}
			m.mu.Lock()
			m.child, m.waitCh = nil, nil
			m.mu.Unlock()
			st := m.lastStatus
			st.Active = false
			st.NginxPID = 0
			if err != nil {
				st.LastError = "nginx exited: " + err.Error()
			}
			m.writeStatus(st)
		case <-ticker.C:
			m.reconcile(ctx)
		}
	}
}

func (m *manager) reconcile(ctx context.Context) {
	b, err := os.ReadFile(m.control)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			m.writeStatus(runtimeStatus{LastError: "read control: " + err.Error()})
		}
		return
	}
	controlHash := sha256Hex(b)
	m.mu.Lock()
	unchanged := controlHash == m.lastControl
	running := m.child != nil
	m.mu.Unlock()
	if unchanged && running {
		return
	}
	var live tlsfront.LiveConfig
	if err := json.Unmarshal(b, &live); err != nil {
		m.writeStatus(runtimeStatus{LastError: "decode control: " + err.Error()})
		return
	}
	if err := live.Validate(); err != nil {
		m.writeStatus(runtimeStatus{Mode: live.TLSAcceleration.Effective().Mode, LastError: err.Error()})
		return
	}
	mode := live.TLSAcceleration.Effective().Mode
	if mode != tlsfront.ModeFrontend {
		m.stopChild(10 * time.Second)
		m.mu.Lock()
		m.lastControl = controlHash
		m.currentHash = ""
		m.currentQAT = false
		m.mu.Unlock()
		m.writeStatus(runtimeStatus{Mode: mode, Active: false})
		return
	}

	r, err := prepare(live, m.outDir)
	if err != nil {
		m.writeStatus(runtimeStatus{Mode: mode, Probe: r.probe, Resolved: r.resolved, LastError: err.Error()})
		return
	}
	if err := installRendered(r, m.outDir); err != nil {
		m.writeStatus(runtimeStatus{Mode: mode, Probe: r.probe, Resolved: r.resolved, LastError: err.Error()})
		return
	}

	m.mu.Lock()
	child := m.child
	oldHash := m.currentHash
	oldQAT := m.currentQAT
	m.mu.Unlock()
	if child == nil {
		if err := m.startChild(r); err != nil {
			m.writeStatus(runtimeStatus{Mode: mode, Probe: r.probe, Resolved: r.resolved, LastError: err.Error()})
			return
		}
	} else if r.hash != oldHash {
		if oldQAT != r.resolved.QAT {
			m.stopChild(10 * time.Second)
			if err := m.startChild(r); err != nil {
				m.writeStatus(runtimeStatus{Mode: mode, Probe: r.probe, Resolved: r.resolved, LastError: err.Error()})
				return
			}
		} else if err := child.Process.Signal(syscall.SIGHUP); err != nil {
			m.stopChild(5 * time.Second)
			if err := m.startChild(r); err != nil {
				m.writeStatus(runtimeStatus{Mode: mode, Probe: r.probe, Resolved: r.resolved, LastError: err.Error()})
				return
			}
		}
	}
	m.mu.Lock()
	m.lastControl = controlHash
	m.currentHash = r.hash
	m.currentQAT = r.resolved.QAT
	pid := 0
	if m.child != nil && m.child.Process != nil {
		pid = m.child.Process.Pid
	}
	m.mu.Unlock()
	m.writeStatus(runtimeStatus{Mode: mode, Active: pid != 0, NginxPID: pid, Probe: r.probe, Resolved: r.resolved})
	_ = ctx
}

func prepare(live tlsfront.LiveConfig, outDir string) (rendered, error) {
	probeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p := tlsfront.ProbeHost(probeCtx)
	resolved, err := tlsfront.Resolve(live.TLSAcceleration, p)
	r := rendered{probe: p, resolved: resolved}
	if err != nil {
		return r, err
	}
	nginx, err := tlsfront.RenderNginx(live, resolved, outDir)
	if err != nil {
		return r, err
	}
	r.nginx = nginx
	if resolved.QAT {
		r.openssl = tlsfront.RenderOpenSSLQAT()
	}

	stage, err := os.MkdirTemp(outDir, ".preflight-*")
	if err != nil {
		return r, err
	}
	defer os.RemoveAll(stage)
	nginxPath := filepath.Join(stage, "nginx.conf")
	if err := os.WriteFile(nginxPath, nginx, 0o640); err != nil {
		return r, err
	}
	opensslPath := filepath.Join(stage, "openssl.cnf")
	if resolved.QAT {
		if err := os.WriteFile(opensslPath, r.openssl, 0o640); err != nil {
			return r, err
		}
	}
	cmd := exec.Command(p.NginxPath, "-t", "-c", nginxPath)
	cmd.Env = childEnv(resolved.QAT, opensslPath)
	out, testErr := cmd.CombinedOutput()
	if testErr != nil {
		return r, fmt.Errorf("nginx preflight failed: %w: %s", testErr, strings.TrimSpace(string(out)))
	}
	h := sha256.New()
	_, _ = h.Write(nginx)
	_, _ = h.Write(r.openssl)
	if resolved.QAT {
		_, _ = h.Write([]byte("qat=1"))
	}
	if resolved.KTLS {
		_, _ = h.Write([]byte("ktls=1"))
	}
	r.hash = hex.EncodeToString(h.Sum(nil))
	return r, nil
}

func installRendered(r rendered, outDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(outDir, "nginx.conf"), r.nginx, 0o640); err != nil {
		return err
	}
	opensslPath := filepath.Join(outDir, "openssl.cnf")
	if r.resolved.QAT {
		if err := atomicWrite(opensslPath, r.openssl, 0o640); err != nil {
			return err
		}
	} else {
		_ = os.Remove(opensslPath)
	}
	return nil
}

func (m *manager) startChild(r rendered) error {
	nginxPath := r.probe.NginxPath
	if nginxPath == "" {
		return errors.New("nginx not found")
	}
	cmd := exec.Command(nginxPath, "-c", filepath.Join(m.outDir, "nginx.conf"), "-g", "daemon off;")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = childEnv(r.resolved.QAT, filepath.Join(m.outDir, "openssl.cnf"))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start nginx: %w", err)
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
		close(waitCh)
	}()
	m.mu.Lock()
	m.child = cmd
	m.waitCh = waitCh
	m.currentHash = r.hash
	m.currentQAT = r.resolved.QAT
	m.mu.Unlock()
	return nil
}

func (m *manager) stopChild(timeout time.Duration) {
	m.mu.Lock()
	cmd := m.child
	wait := m.waitCh
	m.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGTERM)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	if wait != nil {
		select {
		case <-wait:
		case <-timer.C:
			_ = cmd.Process.Kill()
			<-wait
		}
	}
	m.mu.Lock()
	if m.child == cmd {
		m.child, m.waitCh = nil, nil
	}
	m.mu.Unlock()
}

func childEnv(qat bool, opensslConf string) []string {
	env := append([]string(nil), os.Environ()...)
	env = dropEnv(env, "OPENSSL_CONF")
	if qat {
		env = append(env, "OPENSSL_CONF="+opensslConf)
	}
	return env
}

func dropEnv(env []string, key string) []string {
	prefix := key + "="
	out := env[:0]
	for _, v := range env {
		if !strings.HasPrefix(v, prefix) {
			out = append(out, v)
		}
	}
	return out
}

func loadLive(path string) (tlsfront.LiveConfig, error) {
	var live tlsfront.LiveConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return live, err
	}
	if err := json.Unmarshal(b, &live); err != nil {
		return live, err
	}
	if live.PublishedAt.IsZero() {
		live.PublishedAt = time.Now().UTC()
	}
	if err := live.Validate(); err != nil {
		return live, err
	}
	return live, nil
}

func (m *manager) writeStatus(st runtimeStatus) {
	st.UpdatedAt = time.Now().UTC()
	st.Version, st.Commit = buildVersion, buildCommit
	m.lastStatus = st
	_ = atomicWrite(filepath.Join(m.outDir, "status.json"), tlsfront.MarshalStatus(st), 0o640)
}

func atomicWrite(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := io.Copy(f, bytes.NewReader(b)); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
