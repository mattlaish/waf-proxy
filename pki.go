package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const (
	maxPKIFileSize = 8 << 20
	maxCRLFiles    = 32
	crlClockSkew   = 5 * time.Minute
)

type crlSnapshot struct {
	Lists    []*x509.RevocationList
	LoadedAt time.Time
}

type crlStore struct {
	current atomic.Pointer[crlSnapshot]
	mode    string
}

type BackendTLSConfig struct {
	UseSystemCA    *bool    `json:"use_system_ca,omitempty"`
	CAFiles        []string `json:"ca_files,omitempty"`
	CRLFiles       []string `json:"crl_files,omitempty"`
	CRLURLs        []string `json:"crl_urls,omitempty"`
	ServerName     string   `json:"server_name,omitempty"`
	RevocationMode string   `json:"revocation_mode,omitempty"`
	RefreshSec     int      `json:"refresh_sec,omitempty"`
}

func (c BackendTLSConfig) configured() bool {
	return c.UseSystemCA != nil || len(c.CAFiles) > 0 || len(c.CRLFiles) > 0 || len(c.CRLURLs) > 0 ||
		c.ServerName != "" || c.RevocationMode != "" || c.RefreshSec != 0
}

func useSystemCA(c BackendTLSConfig) bool { return c.UseSystemCA == nil || *c.UseSystemCA }

func validateBackendTLSDraft(scheme string, c BackendTLSConfig) error {
	if scheme != "" && scheme != "https" && c.configured() {
		return errors.New("is only valid for an https pool")
	}
	if c.RevocationMode != "" && c.RevocationMode != "soft" && c.RevocationMode != "hard" {
		return errors.New("revocation_mode must be soft, hard, or empty")
	}
	if c.RefreshSec != 0 && (c.RefreshSec < 300 || c.RefreshSec > 604800) {
		return errors.New("refresh_sec must be 0 or between 300 and 604800")
	}
	if err := validateBackendServerName(c.ServerName); err != nil {
		return err
	}
	if err := validateUniquePaths("ca_files", c.CAFiles, 32); err != nil {
		return err
	}
	if err := validateUniquePaths("crl_files", c.CRLFiles, maxCRLFiles); err != nil {
		return err
	}
	if len(c.CRLURLs) > 0 {
		return errors.New("crl_urls are reserved for the URL refresh implementation slice")
	}
	return nil
}

func validateBackendTLS(scheme string, c BackendTLSConfig) error {
	if err := validateBackendTLSDraft(scheme, c); err != nil {
		return err
	}
	if scheme == "https" && !useSystemCA(c) && len(c.CAFiles) == 0 {
		return errors.New("use_system_ca=false requires at least one ca_file")
	}
	if c.RefreshSec != 0 {
		return errors.New("refresh_sec requires the URL refresh implementation slice")
	}
	if c.RevocationMode == "hard" && len(c.CRLFiles) == 0 {
		return errors.New("hard revocation mode requires at least one crl_file")
	}
	_, err := buildBackendTLSConfig(c, scheme)
	return err
}

func validateBackendServerName(name string) error {
	if name == "" {
		return nil
	}
	if len(name) > 253 || strings.TrimSpace(name) != name || strings.ContainsAny(name, "/\\:") ||
		strings.Contains(name, "://") || regexp.MustCompile(`[\x00-\x20\x7f]`).MatchString(name) {
		return errors.New("server_name must be a hostname without scheme, port, slash, or control characters")
	}
	return nil
}

func validateUniquePaths(label string, paths []string, max int) error {
	if len(paths) > max {
		return fmt.Errorf("%s supports at most %d files", label, max)
	}
	seen := map[string]bool{}
	for _, path := range paths {
		if strings.TrimSpace(path) == "" || path != strings.TrimSpace(path) {
			return fmt.Errorf("%s contains an empty or whitespace-padded path", label)
		}
		if seen[path] {
			return fmt.Errorf("%s contains duplicate path %q", label, path)
		}
		seen[path] = true
	}
	return nil
}

func readPKIFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxPKIFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("file is empty")
	}
	if len(b) > maxPKIFileSize {
		return nil, fmt.Errorf("file exceeds %d-byte limit", maxPKIFileSize)
	}
	return b, nil
}

func appendCABundle(pool *x509.CertPool, path string) ([]*x509.Certificate, error) {
	b, err := readPKIFile(path)
	if err != nil {
		return nil, err
	}
	rest, count := b, 0
	var certs []*x509.Certificate
	for len(strings.TrimSpace(string(rest))) > 0 {
		block, next := pem.Decode(rest)
		if block == nil {
			return nil, errors.New("invalid PEM or trailing data")
		}
		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("unexpected PEM block %q", block.Type)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid certificate: %w", err)
		}
		if !cert.BasicConstraintsValid || !cert.IsCA {
			return nil, errors.New("certificate is not a CA")
		}
		if cert.KeyUsage != 0 && cert.KeyUsage&x509.KeyUsageCertSign == 0 {
			return nil, errors.New("CA certificate does not permit certificate signing")
		}
		pool.AddCert(cert)
		certs = append(certs, cert)
		count++
		rest = next
	}
	if count == 0 {
		return nil, errors.New("no certificates found")
	}
	return certs, nil
}

func parseCRLFile(path string, now time.Time) ([]*x509.RevocationList, error) {
	b, err := readPKIFile(path)
	if err != nil {
		return nil, err
	}
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

func loadCRLStore(c BackendTLSConfig, issuers []*x509.Certificate, now time.Time) (*crlStore, error) {
	mode := c.RevocationMode
	if mode == "" {
		mode = "soft"
	}
	store := &crlStore{mode: mode}
	snapshot := &crlSnapshot{LoadedAt: now}
	for _, path := range c.CRLFiles {
		lists, err := parseCRLFile(path, now)
		if err != nil {
			return nil, fmt.Errorf("CRL file %q: %w", path, err)
		}
		snapshot.Lists = append(snapshot.Lists, lists...)
		if len(snapshot.Lists) > maxCRLFiles {
			return nil, fmt.Errorf("CRL files contain more than %d lists", maxCRLFiles)
		}
	}
	for _, list := range snapshot.Lists {
		for _, issuer := range issuers {
			if !bytes.Equal(list.RawIssuer, issuer.RawSubject) {
				continue
			}
			if err := list.CheckSignatureFrom(issuer); err != nil {
				return nil, fmt.Errorf("CRL signature verification failed for custom issuer %q", issuer.Subject.CommonName)
			}
			break
		}
	}
	store.current.Store(snapshot)
	return store, nil
}

func crlRevokes(list *x509.RevocationList, serial *big.Int) bool {
	for _, entry := range list.RevokedCertificateEntries {
		if entry.SerialNumber.Cmp(serial) == 0 {
			return true
		}
	}
	for _, entry := range list.RevokedCertificates {
		if entry.SerialNumber.Cmp(serial) == 0 {
			return true
		}
	}
	return false
}

func (s *crlStore) verify(cs tls.ConnectionState) error {
	snapshot := s.current.Load()
	if snapshot == nil {
		return errors.New("CRL snapshot is unavailable")
	}
	now := time.Now()
	for _, chain := range cs.VerifiedChains {
		for i := 0; i+1 < len(chain); i++ {
			cert, issuer := chain[i], chain[i+1]
			matched := false
			for _, list := range snapshot.Lists {
				if !bytes.Equal(list.RawIssuer, cert.RawIssuer) {
					continue
				}
				if err := list.CheckSignatureFrom(issuer); err != nil {
					return fmt.Errorf("CRL signature verification failed for issuer %q", issuer.Subject.CommonName)
				}
				if list.ThisUpdate.After(now.Add(crlClockSkew)) || list.NextUpdate.IsZero() || list.NextUpdate.Before(now.Add(-crlClockSkew)) {
					return fmt.Errorf("CRL is not currently valid for issuer %q", issuer.Subject.CommonName)
				}
				matched = true
				if crlRevokes(list, cert.SerialNumber) {
					return fmt.Errorf("certificate serial %s is revoked by issuer %q", cert.SerialNumber.Text(16), issuer.Subject.CommonName)
				}
			}
			if !matched && s.mode == "hard" {
				return fmt.Errorf("no valid CRL for issuer %q", issuer.Subject.CommonName)
			}
		}
		return nil
	}
	return errors.New("no verified certificate chain available for CRL checking")
}

func buildBackendTLSConfig(c BackendTLSConfig, scheme string) (*tls.Config, error) {
	if scheme != "https" {
		if c.configured() {
			return nil, errors.New("backend TLS settings require scheme https")
		}
		return nil, nil
	}
	var roots *x509.CertPool
	var err error
	if useSystemCA(c) {
		roots, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("load system CA pool: %w", err)
		}
		if roots == nil {
			return nil, errors.New("system CA pool is unavailable")
		}
	} else {
		roots = x509.NewCertPool()
	}
	var customIssuers []*x509.Certificate
	for _, path := range c.CAFiles {
		certs, err := appendCABundle(roots, path)
		if err != nil {
			return nil, fmt.Errorf("CA file %q: %w", path, err)
		}
		customIssuers = append(customIssuers, certs...)
	}
	store, err := loadCRLStore(c, customIssuers, time.Now())
	if err != nil {
		return nil, err
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: c.ServerName}
	if len(c.CRLFiles) > 0 || c.RevocationMode == "hard" {
		config.VerifyConnection = store.verify
	}
	return config, nil
}
