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
	"encoding/json"
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
	ln         net.Listener
	done       chan struct{}
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
			// During an atomic Go-TLS <-> external-frontend ownership handoff,
			// the newly-bound socket can become reachable a fraction before the
			// runtime pointer swaps. Fall back to the equivalent logical listener
			// in the previous runtime so the transition never exposes a spurious
			// 421 solely because the ownership key changed.
			if frontendListener {
				lr = rt.listeners[logicalAddr]
			} else {
				lr = rt.listeners[tlsfront.InternalListenerKey(addr)]
			}
		}
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
		}
		installTLSHandshakeObserver(srv.TLSConfig, s.metrics, func(hello *tls.ClientHelloInfo) error {
			rt := s.rt.Load()
			if rt == nil || rt.l7Abuse == nil || hello == nil || hello.Conn == nil {
				return nil
			}
			peer := hello.Conn.RemoteAddr().String()
			if host, _, err := net.SplitHostPort(peer); err == nil {
				peer = host
			}
			if peer != "" && !rt.l7Abuse.allowTLSHandshakeAt(logicalAddr, peer, time.Now()) {
				return errors.New("tls handshake rate limit exceeded")
			}
			return nil
		})
	}
	return srv
}

// start launches a server in the background. Bind failures are retried while
// the socket remains desired. This matters during a Go-TLS <-> external-TLS
// mode transition, where the old owner may hold the public port briefly.
func (m *listenerManager) prepare(addr string, isTLS bool, cfg Config) (*managedListener, error) {
	srv := m.buildServer(addr, isTLS, cfg)
	ml := &managedListener{srv: srv, isTLS: isTLS, done: make(chan struct{})}
	if p, ok := tlsfront.SocketPathFromKey(addr); ok {
		ml.socketPath = p
	}
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
		if ln != nil {
			_ = ln.Close()
		}
		if ml.socketPath != "" {
			_ = os.Remove(ml.socketPath)
		}
		return nil, err
	}
	ml.ln = ln
	if ml.isTLS && m.srv.tlsTickets != nil {
		m.srv.tlsTickets.attach(ml.srv.TLSConfig)
	}
	return ml, nil
}

func (m *listenerManager) serveBound(addr string, ml *managedListener, cfg Config) {
	defer close(ml.done)
	if ml.isTLS && m.srv.tlsTickets != nil {
		defer m.srv.tlsTickets.detach(ml.srv.TLSConfig)
	}
	m.log.Info("listener up", "addr", addr, "public_addr", tlsfront.PublicListenForKey(cfg.TLSAcceleration, tlsFrontendSites(cfg), addr), "tls", ml.isTLS, "tls_frontend_internal", ml.socketPath != "")
	defer func() {
		if ml.ln != nil {
			_ = ml.ln.Close()
		}
		if ml.socketPath != "" {
			_ = os.Remove(ml.socketPath)
		}
	}()
	var serveErr error
	if ml.isTLS {
		serveErr = ml.srv.ServeTLS(ml.ln, "", "")
	} else {
		serveErr = ml.srv.Serve(ml.ln)
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		m.listenerFailed(addr, ml, serveErr)
	}
}

func (m *listenerManager) startLocked(addr string, isTLS bool, cfg Config) error {
	ml, err := m.prepare(addr, isTLS, cfg)
	if err != nil {
		return err
	}
	m.live[addr] = ml
	go m.serveBound(addr, ml, cfg)
	return nil
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
	if err := m.startLocked(addr, isTLS, rt.cfg); err != nil {
		m.log.Error("listener retry bind failed", "addr", addr, "err", err)
		time.AfterFunc(time.Second, func() { m.retry(addr, isTLS) })
	}
}

func waitListenerStopped(ml *managedListener) {
	if ml == nil || ml.done == nil {
		return
	}
	select {
	case <-ml.done:
	case <-time.After(2 * time.Second):
	}
}

func closePrepared(ml *managedListener) {
	if ml == nil {
		return
	}
	if ml.ln != nil {
		_ = ml.ln.Close()
	}
	if ml.socketPath != "" {
		_ = os.Remove(ml.socketPath)
	}
}

func (m *listenerManager) restoreOldLocked(oldCfg Config) error {
	oldDesired := listenerSet(oldCfg)
	var errs []string
	for addr, ml := range m.live {
		wantTLS, keep := oldDesired[addr]
		if !keep || wantTLS != ml.isTLS {
			_ = ml.srv.Close()
			waitListenerStopped(ml)
			delete(m.live, addr)
		}
	}
	for addr, isTLS := range oldDesired {
		if _, ok := m.live[addr]; ok {
			continue
		}
		if err := m.startLocked(addr, isTLS, oldCfg); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", addr, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("listener rollback failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// reconcile synchronously binds every newly-required socket before reporting
// success. Destructive TLS-mode flips are rolled back to the old listener set
// on bind failure; removed sockets are closed only after all additions/flips
// are known-good. This prevents Apply from claiming success when the data plane
// cannot actually own the configured addresses.
func (m *listenerManager) reconcile(cfg Config) error {
	desired := listenerSet(cfg)
	m.mu.Lock()
	defer m.mu.Unlock()

	oldCfg := cfg
	if rt := m.srv.rt.Load(); rt != nil {
		oldCfg = rt.cfg
	}

	prepared := map[string]*managedListener{}
	for addr, isTLS := range desired {
		if cur, ok := m.live[addr]; ok {
			if cur.isTLS == isTLS {
				continue
			}
			continue // protocol flips are handled below after closing the old socket
		}
		ml, err := m.prepare(addr, isTLS, cfg)
		if err != nil {
			for _, p := range prepared {
				closePrepared(p)
			}
			return fmt.Errorf("bind listener %s: %w", addr, err)
		}
		prepared[addr] = ml
	}

	// TLS-mode flips require releasing the old socket. If any replacement bind
	// fails, restore the complete old listener topology before returning.
	for addr, cur := range m.live {
		wantTLS, keep := desired[addr]
		if !keep || wantTLS == cur.isTLS {
			continue
		}
		m.log.Info("listener reopening", "addr", addr, "reason", "tls-change", "tls", wantTLS)
		_ = cur.srv.Close()
		waitListenerStopped(cur)
		delete(m.live, addr)
		ml, err := m.prepare(addr, wantTLS, cfg)
		if err != nil {
			for _, p := range prepared {
				closePrepared(p)
			}
			rbErr := m.restoreOldLocked(oldCfg)
			if rbErr != nil {
				return fmt.Errorf("bind listener %s after TLS change: %v; %v", addr, err, rbErr)
			}
			return fmt.Errorf("bind listener %s after TLS change: %w", addr, err)
		}
		m.live[addr] = ml
		go m.serveBound(addr, ml, cfg)
	}

	for addr, ml := range prepared {
		m.live[addr] = ml
		go m.serveBound(addr, ml, cfg)
	}

	frontendOwnedPublic := map[string]struct{}{}
	if tlsfront.FrontendEnabled(cfg.TLSAcceleration) {
		for addr, isTLS := range publicListenerSet(cfg) {
			if isTLS {
				frontendOwnedPublic[addr] = struct{}{}
			}
		}
	}
	for addr, ml := range m.live {
		if _, keep := desired[addr]; keep {
			continue
		}
		if _, handoff := frontendOwnedPublic[addr]; handoff {
			m.log.Info("listener releasing", "addr", addr, "reason", "tls-frontend-ownership")
			releaseForFrontendOwnership(ml)
		} else {
			m.log.Info("listener closing", "addr", addr, "reason", "removed")
			go gracefulClose(ml.srv)
		}
		delete(m.live, addr)
	}
	return nil
}

// startAll is the initial bind at boot. Startup keeps the historical retry
// behavior, but each initial bind attempt is synchronous and visible in logs.
func (m *listenerManager) startAll(cfg Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for addr, isTLS := range listenerSet(cfg) {
		if err := m.startLocked(addr, isTLS, cfg); err != nil {
			m.log.Error("initial listener bind failed", "addr", addr, "err", err)
			time.AfterFunc(time.Second, func() { m.retry(addr, isTLS) })
		}
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

func releaseForFrontendOwnership(ml *managedListener) {
	if ml == nil || ml.srv == nil {
		return
	}
	// Shutdown closes the listening socket before waiting for active requests,
	// which is the property the external TLS frontend needs before it can bind
	// the public address. Keep the drain bounded; Close is the fail-safe that
	// guarantees ownership is released even if a handler ignores cancellation.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	err := ml.srv.Shutdown(ctx)
	cancel()
	if err != nil {
		_ = ml.srv.Close()
	}
	waitListenerStopped(ml)
}

func gracefulClose(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func writeHealth(w http.ResponseWriter, ok bool, draining bool, role string) {
	status := "unavailable"
	if ok {
		status = "ok"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   status,
		"draining": draining,
		"role":     role,
	})
}
