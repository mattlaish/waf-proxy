package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type crlStatus struct {
	Mode        string    `json:"mode"`
	Sources     int       `json:"sources"`
	Lists       int       `json:"lists"`
	LoadedAt    time.Time `json:"loaded_at,omitempty"`
	LastAttempt time.Time `json:"last_attempt,omitempty"`
	LastSuccess time.Time `json:"last_success,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
	Refreshing  bool      `json:"refreshing"`
}

func validateCRLURLSyntax(raw string) error {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return errors.New("URL is empty or whitespace-padded")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("CRL URL must use https")
	}
	if u.User != nil {
		return errors.New("CRL URL must not contain userinfo")
	}
	if u.Hostname() == "" {
		return errors.New("CRL URL requires a hostname")
	}
	if u.Fragment != "" {
		return errors.New("CRL URL must not contain a fragment")
	}
	if p := u.Port(); p != "" && p != "443" {
		return errors.New("CRL URL port must be 443")
	}
	return nil
}

func unsafeCRLAddress(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap()
	return !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified()
}

func fetchCRLURL(ctx context.Context, raw string, roots *x509.CertPool) ([]byte, error) {
	if err := validateCRLURLSyntax(raw); err != nil {
		return nil, err
	}
	u, _ := url.Parse(raw)
	host := u.Hostname()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve CRL host %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("resolve CRL host %q: no addresses", host)
	}
	var chosen net.IP
	for _, entry := range ips {
		if unsafeCRLAddress(entry.IP) {
			return nil, fmt.Errorf("CRL host %q resolves to disallowed address %s", host, entry.IP)
		}
		if chosen == nil {
			chosen = entry.IP
		}
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	dialAddr := net.JoinHostPort(chosen.String(), port)
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 15 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, dialAddr)
		},
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: host},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		DisableKeepAlives:     true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/pkix-crl, application/x-pem-file, application/octet-stream")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("CRL endpoint returned HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxPKIFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("CRL response is empty")
	}
	if len(b) > maxPKIFileSize {
		return nil, fmt.Errorf("CRL response exceeds %d-byte limit", maxPKIFileSize)
	}
	return b, nil
}

func parseCRLBytes(b []byte, now time.Time) ([]*x509.RevocationList, error) {
	var lists []*x509.RevocationList
	if block, _ := pem.Decode(b); block != nil {
		rest := b
		for len(bytes.TrimSpace(rest)) > 0 {
			block, next := pem.Decode(rest)
			if block == nil {
				return nil, errors.New("invalid PEM or trailing data")
			}
			if block.Type != "X509 CRL" {
				return nil, fmt.Errorf("unexpected PEM block %q", block.Type)
			}
			list, err := x509.ParseRevocationList(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("invalid CRL: %w", err)
			}
			lists = append(lists, list)
			rest = next
		}
	} else {
		list, err := x509.ParseRevocationList(b)
		if err != nil {
			return nil, fmt.Errorf("invalid DER CRL: %w", err)
		}
		lists = append(lists, list)
	}
	for _, list := range lists {
		if list.ThisUpdate.After(now.Add(crlClockSkew)) {
			return nil, fmt.Errorf("CRL thisUpdate is in the future: %s", list.ThisUpdate.UTC().Format(time.RFC3339))
		}
		if list.NextUpdate.IsZero() || list.NextUpdate.Before(now.Add(-crlClockSkew)) {
			return nil, fmt.Errorf("CRL is expired or has no nextUpdate: %s", list.NextUpdate.UTC().Format(time.RFC3339))
		}
	}
	return lists, nil
}

func dedupeCRLs(lists []*x509.RevocationList) []*x509.RevocationList {
	seen := map[[32]byte]struct{}{}
	out := make([]*x509.RevocationList, 0, len(lists))
	for _, list := range lists {
		if list == nil {
			continue
		}
		sum := sha256.Sum256(list.Raw)
		if _, ok := seen[sum]; ok {
			continue
		}
		seen[sum] = struct{}{}
		out = append(out, list)
	}
	return out
}

func validateCRLSignatures(lists []*x509.RevocationList, issuers []*x509.Certificate) error {
	for _, list := range lists {
		for _, issuer := range issuers {
			if !bytes.Equal(list.RawIssuer, issuer.RawSubject) {
				continue
			}
			if err := list.CheckSignatureFrom(issuer); err != nil {
				return fmt.Errorf("CRL signature verification failed for custom issuer %q", issuer.Subject.CommonName)
			}
			break
		}
	}
	return nil
}

func crlCacheDir() string {
	if v := strings.TrimSpace(os.Getenv("WAF_CRL_CACHE_DIR")); v != "" {
		return v
	}
	return "/var/lib/waf-proxy/crl-cache"
}

func crlCachePath(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return filepath.Join(crlCacheDir(), hex.EncodeToString(sum[:])+".crl")
}

func loadCachedCRLs(urls []string, now time.Time) []*x509.RevocationList {
	var out []*x509.RevocationList
	for _, raw := range urls {
		b, err := os.ReadFile(crlCachePath(raw))
		if err != nil || len(b) == 0 || len(b) > maxPKIFileSize {
			continue
		}
		lists, err := parseCRLBytes(b, now)
		if err != nil {
			continue
		}
		out = append(out, lists...)
	}
	return out
}

func writeCRLCache(raw string, b []byte) error {
	dir := crlCacheDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".crl-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, crlCachePath(raw)); err != nil {
		return err
	}
	ok = true
	return nil
}

func prepareCRLStore(ctx context.Context, c BackendTLSConfig, issuers []*x509.Certificate, roots *x509.CertPool, now time.Time) (*crlStore, error) {
	store, err := loadCRLStore(c, issuers, now)
	if err != nil {
		return nil, err
	}
	store.cfg = c
	store.issuers = append([]*x509.Certificate(nil), issuers...)
	store.roots = roots
	if len(c.CRLURLs) > 0 {
		cached := loadCachedCRLs(c.CRLURLs, now)
		if len(cached) > 0 {
			base := store.current.Load()
			lists := append([]*x509.RevocationList(nil), cached...)
			if base != nil {
				lists = append(lists, base.Lists...)
			}
			lists = dedupeCRLs(lists)
			if validateCRLSignatures(lists, issuers) == nil {
				store.current.Store(&crlSnapshot{Lists: lists, LoadedAt: now})
			}
		}
	}
	if len(c.CRLURLs) == 0 {
		store.lastSuccess = now
		return store, nil
	}
	if err := store.refresh(ctx); err != nil {
		snap := store.current.Load()
		if store.mode == "hard" && (snap == nil || len(snap.Lists) == 0) {
			return nil, fmt.Errorf("initial CRL URL refresh: %w", err)
		}
	}
	return store, nil
}

func (s *crlStore) refresh(ctx context.Context) error {
	if s == nil {
		return errors.New("CRL store unavailable")
	}
	s.mu.Lock()
	if s.refreshing {
		s.mu.Unlock()
		return errors.New("CRL refresh already in progress")
	}
	s.refreshing = true
	s.lastAttempt = time.Now().UTC()
	cfg := s.cfg
	issuers := append([]*x509.Certificate(nil), s.issuers...)
	roots := s.roots
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.refreshing = false
		s.mu.Unlock()
	}()

	now := time.Now()
	var lists []*x509.RevocationList
	for _, path := range cfg.CRLFiles {
		parsed, err := parseCRLFile(path, now)
		if err != nil {
			s.recordRefreshError(err)
			return err
		}
		lists = append(lists, parsed...)
	}
	fetched := make(map[string][]byte, len(cfg.CRLURLs))
	for _, raw := range cfg.CRLURLs {
		b, err := fetchCRLURL(ctx, raw, roots)
		if err != nil {
			s.recordRefreshError(err)
			return err
		}
		parsed, err := parseCRLBytes(b, now)
		if err != nil {
			s.recordRefreshError(err)
			return fmt.Errorf("CRL URL %q: %w", raw, err)
		}
		fetched[raw] = b
		lists = append(lists, parsed...)
	}
	lists = dedupeCRLs(lists)
	if len(lists) > maxCRLFiles {
		err := fmt.Errorf("CRL sources contain more than %d unique lists", maxCRLFiles)
		s.recordRefreshError(err)
		return err
	}
	if err := validateCRLSignatures(lists, issuers); err != nil {
		s.recordRefreshError(err)
		return err
	}
	for raw, b := range fetched {
		if err := writeCRLCache(raw, b); err != nil {
			s.recordRefreshError(err)
			return fmt.Errorf("persist CRL cache: %w", err)
		}
	}
	s.current.Store(&crlSnapshot{Lists: lists, LoadedAt: now})
	s.mu.Lock()
	s.lastSuccess = time.Now().UTC()
	s.lastError = ""
	s.mu.Unlock()
	return nil
}

func (s *crlStore) recordRefreshError(err error) {
	s.mu.Lock()
	if err != nil {
		s.lastError = err.Error()
	}
	s.mu.Unlock()
}

func (s *crlStore) startRefresher(ctx context.Context, interval time.Duration) {
	if s == nil || interval <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = s.refresh(ctx)
			}
		}
	}()
}

func (s *crlStore) status() crlStatus {
	if s == nil {
		return crlStatus{}
	}
	s.mu.Lock()
	st := crlStatus{
		Mode:        s.mode,
		Sources:     len(s.cfg.CRLFiles) + len(s.cfg.CRLURLs),
		LastAttempt: s.lastAttempt,
		LastSuccess: s.lastSuccess,
		LastError:   s.lastError,
		Refreshing:  s.refreshing,
	}
	s.mu.Unlock()
	if snap := s.current.Load(); snap != nil {
		st.Lists = len(snap.Lists)
		st.LoadedAt = snap.LoadedAt
	}
	return st
}
