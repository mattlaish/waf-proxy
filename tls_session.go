package main

// TLS session-ticket key management for the built-in Go TLS termination path.
//
// crypto/tls deliberately generates process-local ticket keys when operators do
// not configure them. That is safe for a single process, but it means all
// resumptions are lost on restart and cannot cross an HA pair. This manager
// derives a deterministic rotating key ring from one runtime-only secret. Nodes
// that receive the same secret derive the same keys for the same time epoch.
//
// The master secret is never persisted in Config, never returned by an API, and
// must be distributed independently (for example by the same secret manager
// used for the HA replication token). It must not reuse the HA token value.

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultTLSSessionTicketRotation = 24 * time.Hour
	defaultTLSSessionTicketRetained = 7
	tlsSessionTicketPurpose         = "waf-proxy/tls-session-ticket/v1"
)

type tlsSessionTicketManager struct {
	secret   []byte
	rotation time.Duration
	retained int

	mu      sync.Mutex
	configs map[*tls.Config]struct{}
	epoch   int64
}

func loadTLSSessionTicketSecret() (string, error) {
	direct := strings.TrimSpace(os.Getenv("WAF_TLS_SESSION_TICKET_SECRET"))
	path := strings.TrimSpace(os.Getenv("WAF_TLS_SESSION_TICKET_SECRET_FILE"))
	if direct != "" && path != "" {
		return "", errors.New("set only one of WAF_TLS_SESSION_TICKET_SECRET or WAF_TLS_SESSION_TICKET_SECRET_FILE")
	}
	if path == "" {
		return direct, nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat TLS session-ticket secret file: %w", err)
	}
	if !st.Mode().IsRegular() {
		return "", errors.New("TLS session-ticket secret file must be a regular file")
	}
	if st.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("TLS session-ticket secret file %s must not be group/world accessible", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read TLS session-ticket secret file: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

func newTLSSessionTicketManager(secretText string, rotation time.Duration, retained int) (*tlsSessionTicketManager, error) {
	m := &tlsSessionTicketManager{rotation: rotation, retained: retained, configs: map[*tls.Config]struct{}{}}
	secretText = strings.TrimSpace(secretText)
	if secretText == "" {
		return m, nil
	}
	secret, err := decodeTLSSessionTicketSecret(secretText)
	if err != nil {
		return nil, err
	}
	if rotation < time.Minute {
		return nil, errors.New("TLS session-ticket rotation must be at least one minute")
	}
	if retained < 2 || retained > 32 {
		return nil, errors.New("TLS session-ticket retained key count must be between 2 and 32")
	}
	m.secret = secret
	return m, nil
}

func decodeTLSSessionTicketSecret(v string) ([]byte, error) {
	var secret []byte
	var err error
	if strings.HasPrefix(v, "base64:") {
		secret, err = base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(v, "base64:")))
		if err != nil {
			return nil, fmt.Errorf("TLS session-ticket secret base64: %w", err)
		}
	} else {
		secret = []byte(v)
	}
	if len(secret) < 32 {
		return nil, errors.New("TLS session-ticket secret must be at least 32 bytes; use a high-entropy value")
	}
	return append([]byte(nil), secret...), nil
}

func (m *tlsSessionTicketManager) enabled() bool {
	return m != nil && len(m.secret) >= 32
}

func (m *tlsSessionTicketManager) epochAt(now time.Time) int64 {
	return now.UTC().Unix() / int64(m.rotation/time.Second)
}

func (m *tlsSessionTicketManager) keysAt(now time.Time) [][32]byte {
	if !m.enabled() {
		return nil
	}
	epoch := m.epochAt(now)
	keys := make([][32]byte, 0, m.retained)
	for i := 0; i < m.retained; i++ {
		mac := hmac.New(sha256.New, m.secret)
		_, _ = fmt.Fprintf(mac, "%s/%d", tlsSessionTicketPurpose, epoch-int64(i))
		sum := mac.Sum(nil)
		var key [32]byte
		copy(key[:], sum)
		keys = append(keys, key)
	}
	return keys
}

func (m *tlsSessionTicketManager) attach(cfg *tls.Config) {
	if !m.enabled() || cfg == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	keys := m.keysAt(now)
	cfg.SetSessionTicketKeys(keys)
	m.epoch = m.epochAt(now)
	m.configs[cfg] = struct{}{}
}

func (m *tlsSessionTicketManager) detach(cfg *tls.Config) {
	if m == nil || cfg == nil {
		return
	}
	m.mu.Lock()
	delete(m.configs, cfg)
	m.mu.Unlock()
}

func (m *tlsSessionTicketManager) refreshAt(now time.Time) bool {
	if !m.enabled() {
		return false
	}
	epoch := m.epochAt(now)
	m.mu.Lock()
	defer m.mu.Unlock()
	if epoch == m.epoch {
		return false
	}
	keys := m.keysAt(now)
	for cfg := range m.configs {
		cfg.SetSessionTicketKeys(keys)
	}
	m.epoch = epoch
	return true
}

func (m *tlsSessionTicketManager) start(stop <-chan struct{}) {
	if !m.enabled() {
		return
	}
	// Check much more often than the epoch size so rotation happens promptly,
	// while avoiding per-connection clock/key derivation work.
	interval := 5 * time.Minute
	if m.rotation/8 < interval {
		interval = m.rotation / 8
	}
	if interval < time.Minute {
		interval = time.Minute
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case now := <-t.C:
				m.refreshAt(now)
			}
		}
	}()
}

func (m *tlsSessionTicketManager) status() map[string]any {
	if !m.enabled() {
		return map[string]any{
			"mode":    "process_local_auto",
			"shared":  false,
			"enabled": false,
		}
	}
	m.mu.Lock()
	epoch := m.epoch
	listeners := len(m.configs)
	m.mu.Unlock()
	return map[string]any{
		"mode":                   "shared_derived_rotating",
		"enabled":                true,
		"shared":                 true,
		"rotation_seconds":       int64(m.rotation / time.Second),
		"retained_key_count":     m.retained,
		"current_epoch":          epoch,
		"attached_tls_listeners": listeners,
	}
}
