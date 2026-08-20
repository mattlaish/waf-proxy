package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func makeBackendPKI(t *testing.T, dnsName string) ([]byte, tls.Certificate) {
	t.Helper()
	now := time.Now()
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	root := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test root"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	rootDER, err := x509.CreateCertificate(rand.Reader, root, root, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: dnsName}, DNSNames: []string{dnsName},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(leafKey)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}), cert
}

func writeTestFile(t *testing.T, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBackendTLSCustomCAAndServerNameThroughProxy(t *testing.T) {
	rootPEM, cert := makeBackendPKI(t, "backend.internal")
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "trusted") }))
	backend.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	backend.StartTLS()
	defer backend.Close()
	target, _ := url.Parse(backend.URL)
	tlsCfg, err := buildBackendTLSConfig(BackendTLSConfig{UseSystemCA: boolPtr(false), CAFiles: []string{writeTestFile(t, "root.pem", rootPEM)}, ServerName: "backend.internal"}, "https")
	if err != nil {
		t.Fatal(err)
	}
	m := &memberRuntime{target: target, weight: 1}
	atomic.StoreInt32(&m.healthy, 1)
	p := &poolRuntime{name: "secure", method: "round_robin", members: []*memberRuntime{m}, backendTLS: tlsCfg}
	proxy := buildProxy(p, SiteConfig{Name: "site"}, Config{BackendTimeoutSec: 5}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://waf.local/", nil))
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != "trusted" {
		t.Fatalf("proxy status=%d body=%q", w.Code, w.Body.String())
	}
}

func boolPtr(v bool) *bool { return &v }

func TestBackendTLSRejectsUntrustedAndHostnameMismatch(t *testing.T) {
	rootPEM, cert := makeBackendPKI(t, "backend.internal")
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	backend.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	backend.StartTLS()
	defer backend.Close()
	rootPath := writeTestFile(t, "root.pem", rootPEM)
	wrongName, err := buildBackendTLSConfig(BackendTLSConfig{UseSystemCA: boolPtr(false), CAFiles: []string{rootPath}, ServerName: "wrong.internal"}, "https")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&http.Client{Transport: &http.Transport{TLSClientConfig: wrongName}}).Get(backend.URL); err == nil {
		t.Fatal("hostname mismatch was accepted")
	}
	untrusted, err := buildBackendTLSConfig(BackendTLSConfig{UseSystemCA: boolPtr(false), ServerName: "backend.internal"}, "https")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&http.Client{Transport: &http.Transport{TLSClientConfig: untrusted}}).Get(backend.URL); err == nil {
		t.Fatal("untrusted backend was accepted")
	}
}

func TestBackendTLSRejectsTLS11(t *testing.T) {
	rootPEM, cert := makeBackendPKI(t, "backend.internal")
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	backend.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS11, MaxVersion: tls.VersionTLS11}
	backend.StartTLS()
	defer backend.Close()
	cfg, err := buildBackendTLSConfig(BackendTLSConfig{UseSystemCA: boolPtr(false), CAFiles: []string{writeTestFile(t, "root.pem", rootPEM)}, ServerName: "backend.internal"}, "https")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}).Get(backend.URL); err == nil {
		t.Fatal("TLS 1.1 backend was accepted")
	}
}

func TestBackendTLSValidationAndMalformedCA(t *testing.T) {
	if err := validateBackendTLSDraft("http", BackendTLSConfig{ServerName: "backend.internal"}); err == nil {
		t.Fatal("HTTP pool accepted backend TLS settings")
	}
	if err := validateBackendTLS("https", BackendTLSConfig{UseSystemCA: boolPtr(false)}); err == nil {
		t.Fatal("empty isolated trust store accepted at Apply validation")
	}
	if err := validateBackendTLS("https", BackendTLSConfig{RevocationMode: "hard"}); err == nil {
		t.Fatal("unimplemented CRL enforcement setting accepted")
	}
	for _, name := range []string{"https://backend", "backend:443", "bad/name", "bad\nname"} {
		if err := validateBackendServerName(name); err == nil {
			t.Fatalf("unsafe server_name %q accepted", name)
		}
	}
	bad := writeTestFile(t, "bad.pem", []byte("not a certificate"))
	if _, err := buildBackendTLSConfig(BackendTLSConfig{UseSystemCA: boolPtr(false), CAFiles: []string{bad}}, "https"); err == nil {
		t.Fatal("malformed CA accepted")
	}
	large := writeTestFile(t, "large.pem", make([]byte, maxPKIFileSize+1))
	if _, err := buildBackendTLSConfig(BackendTLSConfig{UseSystemCA: boolPtr(false), CAFiles: []string{large}}, "https"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized CA error = %v", err)
	}
}

func TestBackendTLSDefaultsToSystemCAAndTLS12(t *testing.T) {
	cfg, err := buildBackendTLSConfig(BackendTLSConfig{}, "https")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RootCAs == nil || cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("unexpected TLS defaults: %#v", cfg)
	}
	if cfg, err := buildBackendTLSConfig(BackendTLSConfig{}, "http"); err != nil || cfg != nil {
		t.Fatalf("HTTP compatibility changed: cfg=%#v err=%v", cfg, err)
	}
}
