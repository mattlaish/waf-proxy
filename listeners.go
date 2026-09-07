package main

// Dynamic data-plane listeners.
//
// Listeners used to be bound once at startup, so changing a site's listen
// address (or flipping HTTP<->TLS) needed a full process restart — which drops
// every other listener too. That's wrong for a reverse proxy: adding a site
// should not interrupt unrelated traffic.
//
// The listenerManager owns one *http.Server per distinct address and reconciles
// them on every config apply: it opens newly-added addresses, closes removed
// ones, and leaves unchanged addresses (and their live connections) completely
// alone. Only the specific socket you changed is affected; everything else
// keeps serving without a blip.
//
// A TLS-ness change on the same address (http<->https) is handled as
// close-then-reopen of just that one address.

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"waf-proxy/internal/tlsfront"
)

type managedListener struct {
	srv        *http.Server
	isTLS      bool
	socketPath string
}

type originalSchemeContextKey struct{}

func originalRequestScheme(r *http.Request) string {
	if r == nil {
		return ""
	}
	v, _ := r.Context().Value(originalSchemeContextKey{}).(string)
	return v
}

// prepareTLSFrontendRequest accepts the private metadata inserted by the local
// TLS frontend only on Unix-domain listeners. External clients cannot reach the
// socket through the public listener, and nginx overwrites (rather than appends)
// this metadata before forwarding.
func prepareTLSFrontendRequest(r *http.Request) (*http.Request, error) {
	ipText := strings.TrimSpace(r.Header.Get(tlsfront.ClientIPHeader))
	ip := net.ParseIP(ipText)
	if ip == nil {
		return nil, errors.New("missing or invalid TLS frontend client IP")
	}
	port := 0
	if p := strings.TrimSpace(r.Header.Get(tlsfront.ClientPortHeader)); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nil, errors.New("invalid TLS frontend client port")
		}
		port = n
	}
	proto := strings.ToLower(strings.TrimSpace(r.Header.Get(tlsfront.ProtoHeader)))
	if proto != "https" {
		return nil, errors.New("invalid TLS frontend protocol marker")
	}
	r.RemoteAddr = net.JoinHostPort(ip.String(), strconv.Itoa(port))
	// Coraza's net/http connector derives the URI from r.URL. Restore the public
	// HTTPS origin so scheme-sensitive rules behave like the built-in Go TLS path.
	r.URL.Scheme = "https"
	r.URL.Host = r.Host
	// Never preserve forwarding material supplied by the Internet-facing side.
	// The reverse proxy reconstructs one authoritative chain from RemoteAddr.
	r.Header.Del("X-Forwarded-For")
	r.Header.Del("Forwarded")
	r.Header.Del("X-Real-IP")
	r.Header.Del(tlsfront.ClientIPHeader)
	r.Header.Del(tlsfront.ClientPortHeader)
	r.Header.Del(tlsfront.ProtoHeader)
	return r.WithContext(context.WithValue(r.Context(), originalSchemeContextKey{}, "https")), nil
}

type listenerManager struct {
	mu   sync.Mutex
	srv  *server
	log  *slog.Logger
	live map[string]*managedListener // addr -> running server
}

func newListenerManager(s *server, log *slog.Logger) *listenerManager {
	return &listenerManager{srv: s, log: log, live: map[string]*managedListener{}}
}

// buildServer constructs (but does not start) the *http.Server for one address.
// The handler routes by Host via the current runtime, so it always reflects the
// latest applied config without rebinding.
func (m *listenerManager) buildServer(addr string, isTLS bool, cfg Config) *http.Server {
	s := m.srv
	frontendListener := tlsfront.IsInternalListenerKey(addr)
	logicalAddr := tlsfront.PublicListenForKey(cfg.TLSAcceleration, tlsFrontendSites(cfg), addr)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		rt := s.rt.Load()
		healthy := rt != nil && !s.draining.Load()
		role := "solo"
		if h := s.ha.status(); h.Role != "" {
			role = h.Role
		}
		w.Header().Set("Content-Type", "application/json")
		if !healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			writeHealth(w, false, s.draining.Load(), role)
			return
		}
		w.WriteHeader(http.StatusOK)
		writeHealth(w, true, false, role)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if frontendListener {
			var err error
			r, err = prepareTLSFrontendRequest(r)
			if err != nil {
				http.Error(w, "invalid TLS frontend request", http.StatusBadRequest)
				return
			}
		}
		rt := s.rt.Load()
		if rt == nil {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		lr := rt.listeners[addr]
		if lr == nil {
			http.Error(w, "listener not configured", http.StatusMisdirectedRequest)
			return
		}
		site := lr.lookup(r.Host)
		s.observeHost(logicalAddr, r.Host, site != nil)
		if site == nil {
			http.Error(w, "unknown host", http.StatusMisdirectedRequest)
			return
		}
		site.handler.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: time.Duration(cfg.ReadTimeoutSec) * time.Second,
		ReadTimeout:       time.Duration(cfg.ReadTimeoutSec) * time.Second,
		WriteTimeout:      time.Duration(cfg.BackendTimeoutSec+10) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeoutSec) * time.Second,
		MaxHeaderBytes:    64 << 10,
		ErrorLog:          slog.NewLogLogger(m.log.Handler(), slog.LevelWarn),
	}
	if isTLS {
		srv.TLSConfig = &tls.Config{
			MinVersion:       tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
			GetCertificate:   s.getCertificate(addr),
			GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
				s.metrics.addTLSHandshake() // fires once per handshake attempt
				return nil, nil
			},
		}
	}
	return srv
}

// start launches a server in the background. Bind failures are retried while
// the socket remains desired. This matters during a Go-TLS <-> external-TLS
// mode transition, where the old owner may hold the public port briefly.
func (m *listenerManager) start(addr string, isTLS bool, cfg Config) {
	srv := m.buildServer(addr, isTLS, cfg)
	ml := &managedListener{srv: srv, isTLS: isTLS}
	if p, ok := tlsfront.SocketPathFromKey(addr); ok {
		ml.socketPath = p
	}
	m.live[addr] = ml
	go m.serve(addr, ml, cfg)
}

func (m *listenerManager) serve(addr string, ml *managedListener, cfg Config) {
	m.log.Info("listener up", "addr", addr, "public_addr", tlsfront.PublicListenForKey(cfg.TLSAcceleration, tlsFrontendSites(cfg), addr), "tls", ml.isTLS, "tls_frontend_internal", ml.socketPath != "")
	var ln net.Listener
	var err error
	if ml.socketPath != "" {
		if err = os.MkdirAll(filepath.Dir(ml.socketPath), 0o750); err == nil {
			if st, statErr := os.Lstat(ml.socketPath); statErr == nil {
				if st.Mode()&os.ModeSocket == 0 {
					err = fmt.Errorf("refusing to replace non-socket path %s", ml.socketPath)
				} else {
					err = os.Remove(ml.socketPath)
				}
			} else if !errors.Is(statErr, os.ErrNotExist) {
				err = statErr
			}
		}
		if err == nil {
			ln, err = net.Listen("unix", ml.socketPath)
		}
		if err == nil {
			err = os.Chmod(ml.socketPath, 0o660)
		}
	} else {
		lc := net.ListenConfig{Control: freebindControl}
		ln, err = lc.Listen(context.Background(), "tcp", addr)
	}
	if err != nil {
		m.listenerFailed(addr, ml, err)
		return
	}
	defer func() {
		_ = ln.Close()
		if ml.socketPath != "" {
			_ = os.Remove(ml.socketPath)
		}
	}()
	var serveErr error
	if ml.isTLS {
		serveErr = ml.srv.ServeTLS(ln, "", "")
	} else {
		serveErr = ml.srv.Serve(ln)
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		m.listenerFailed(addr, ml, serveErr)
	}
}

func (m *listenerManager) listenerFailed(addr string, ml *managedListener, err error) {
	m.log.Error("listener error", "addr", addr, "err", err)
	m.mu.Lock()
	if cur, ok := m.live[addr]; ok && cur == ml {
		delete(m.live, addr)
	}
	m.mu.Unlock()
	time.AfterFunc(time.Second, func() { m.retry(addr, ml.isTLS) })
}

func (m *listenerManager) retry(addr string, isTLS bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, running := m.live[addr]; running {
		return
	}
	rt := m.srv.rt.Load()
	if rt == nil {
		return
	}
	desired, ok := listenerSet(rt.cfg)[addr]
	if !ok || desired != isTLS {
		return
	}
	m.start(addr, isTLS, rt.cfg)
}

// reconcile brings the running listeners in line with the desired set from cfg.
// Added addresses are opened, removed ones gracefully closed, TLS flips
// reopened; unchanged addresses are untouched (their connections persist).
func (m *listenerManager) reconcile(cfg Config) {
	desired := listenerSet(cfg) // addr -> isTLS
	m.mu.Lock()
	defer m.mu.Unlock()

	// close removed or TLS-changed
	for addr, ml := range m.live {
		wantTLS, keep := desired[addr]
		if !keep {
			m.log.Info("listener closing", "addr", addr, "reason", "removed")
			go gracefulClose(ml.srv) // drain removed address
			delete(m.live, addr)
		} else if wantTLS != ml.isTLS {
			// Same address, http<->tls flip: free the port synchronously so the
			// reopen below can rebind immediately (graceful drain would hold it).
			m.log.Info("listener reopening", "addr", addr, "reason", "tls-change", "tls", wantTLS)
			_ = ml.srv.Close()
			delete(m.live, addr)
		}
	}
	// open added (or reopen after a TLS flip)
	for addr, isTLS := range desired {
		if _, running := m.live[addr]; !running {
			m.start(addr, isTLS, cfg)
		}
	}
}

// startAll is the initial bind at boot.
func (m *listenerManager) startAll(cfg Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for addr, isTLS := range listenerSet(cfg) {
		m.start(addr, isTLS, cfg)
	}
}

// shutdown closes every listener (process exit).
func (m *listenerManager) shutdown(ctx context.Context) {
	m.mu.Lock()
	srvs := make([]*http.Server, 0, len(m.live))
	for _, ml := range m.live {
		srvs = append(srvs, ml.srv)
	}
	m.live = map[string]*managedListener{}
	m.mu.Unlock()
	for _, srv := range srvs {
		_ = srv.Shutdown(ctx)
	}
}

func (m *listenerManager) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.live)
}

func gracefulClose(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func writeHealth(w http.ResponseWriter, ok bool, draining bool, role string) {
	if ok {
		_, _ = w.Write([]byte(`{"status":"ok","role":"` + role + `"}` + "\n"))
		return
	}
	status := "unavailable"
	_, _ = w.Write([]byte(`{"status":"` + status + `","draining":` + boolStr(draining) + `,"role":"` + role + `"}` + "\n"))
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
