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
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

type managedListener struct {
	srv   *http.Server
	isTLS bool
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
		s.hosts.note(addr, r.Host, site != nil)
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

// start launches a server in the background. A bind failure is logged but does
// not crash the process — the rest of the WAF keeps running.
func (m *listenerManager) start(addr string, isTLS bool, cfg Config) {
	srv := m.buildServer(addr, isTLS, cfg)
	m.live[addr] = &managedListener{srv: srv, isTLS: isTLS}
	go func() {
		m.log.Info("listener up", "addr", addr, "tls", isTLS)
		// Create the listening socket with IP_FREEBIND so a site may bind an IP
		// that isn't assigned to a local interface (floating/VIP, or one of
		// several managed service IPs). Falls back to a normal bind off Linux.
		lc := net.ListenConfig{Control: freebindControl}
		ln, err := lc.Listen(context.Background(), "tcp", addr)
		if err != nil {
			m.log.Error("listener error", "addr", addr, "err", err)
			m.mu.Lock()
			if ml, ok := m.live[addr]; ok && ml.srv == srv {
				delete(m.live, addr)
			}
			m.mu.Unlock()
			return
		}
		var e error
		if isTLS {
			e = srv.ServeTLS(ln, "", "")
		} else {
			e = srv.Serve(ln)
		}
		if e != nil && !errors.Is(e, http.ErrServerClosed) {
			m.log.Error("listener error", "addr", addr, "err", e)
			m.mu.Lock()
			if ml, ok := m.live[addr]; ok && ml.srv == srv {
				delete(m.live, addr)
			}
			m.mu.Unlock()
		}
	}()
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
