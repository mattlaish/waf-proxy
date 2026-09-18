package hsm

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockSigner struct {
	key    *rsa.PrivateKey
	slot   uint64
	ref    string
	closed bool
}

func (m *mockSigner) Public() crypto.PublicKey { return &m.key.PublicKey }
func (m *mockSigner) Sign(r io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return m.key.Sign(r, digest, opts)
}
func (m *mockSigner) Close() error { m.closed = true; return nil }
func (m *mockSigner) HealthCheck(context.Context) HealthStatus {
	return HealthStatus{Provider: "pkcs11", Slot: "7", KeyReference: m.ref, Status: "READY"}
}
func (m *mockSigner) SlotID() uint64       { return m.slot }
func (m *mockSigner) KeyReference() string { return m.ref }

type mockProvider struct{ signer Signer }

func (m mockProvider) OpenSigner(context.Context, RuntimeConfig, KeyConfig, crypto.PublicKey, AuditSink) (Signer, error) {
	return m.signer, nil
}

type failingSigner struct {
	pub    crypto.PublicKey
	slot   uint64
	ref    string
	closed bool
}

func (f *failingSigner) Public() crypto.PublicKey { return f.pub }
func (f *failingSigner) Sign(io.Reader, []byte, crypto.SignerOpts) ([]byte, error) {
	return nil, errors.New("forced HSM signing failure")
}
func (f *failingSigner) Close() error { f.closed = true; return nil }
func (f *failingSigner) HealthCheck(context.Context) HealthStatus {
	return HealthStatus{Provider: ProviderPKCS11, Slot: "7", KeyReference: f.ref, Status: "UNAVAILABLE"}
}
func (f *failingSigner) SlotID() uint64       { return f.slot }
func (f *failingSigner) KeyReference() string { return f.ref }

func TestTLSHandshakeFailsClosedWhenHSMSigningFails(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	certPath := writeTestCert(t, key)
	chain, leaf, err := readCertificateChain(certPath)
	if err != nil {
		t.Fatal(err)
	}
	signer := &failingSigner{pub: leaf.PublicKey, slot: 7, ref: "label:web"}
	cert := tls.Certificate{Certificate: chain, PrivateKey: signer, Leaf: leaf}
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	deadline := time.Now().Add(2 * time.Second)
	_ = serverConn.SetDeadline(deadline)
	_ = clientConn.SetDeadline(deadline)
	srv := tls.Server(serverConn, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	cli := tls.Client(clientConn, &tls.Config{ServerName: "localhost", InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}) // local fixture
	errs := make(chan error, 2)
	go func() { errs <- srv.Handshake() }()
	go func() { errs <- cli.Handshake() }()
	first, second := <-errs, <-errs
	if first == nil && second == nil {
		t.Fatal("TLS handshake unexpectedly succeeded after HSM signing failure")
	}
}

func writeTestCert(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	now := time.Now()
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(p, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAuditEventContainsOnlyApprovedFields(t *testing.T) {
	e := AuditEvent{Provider: ProviderPKCS11, Slot: "7", KeyReference: "label:web", Operation: "SIGN", Result: "SUCCESS"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"provider": true, "slot": true, "key_reference": true, "operation": true, "result": true}
	if len(got) != len(want) {
		t.Fatalf("audit fields = %v, want only %v", got, want)
	}
	for k := range got {
		if !want[k] {
			t.Fatalf("unexpected audit field %q in %s", k, b)
		}
	}
}

func TestLoadTLSCertificateUsesCryptoSignerAndHandshake(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	certPath := writeTestCert(t, key)
	signer := &mockSigner{key: key, slot: 7, ref: "label:web"}
	cfg := KeyConfig{Provider: ProviderPKCS11, ModulePath: "/unused", SlotID: ptrU64(7), KeyLabel: "web", PINSecretRef: "env:UNUSED"}
	cert, got, err := LoadTLSCertificate(context.Background(), mockProvider{signer}, DefaultRuntimeConfig(), cfg, certPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != signer {
		t.Fatal("unexpected signer")
	}
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	srv := tls.Server(serverConn, &tls.Config{Certificates: []tls.Certificate{*cert}, MinVersion: tls.VersionTLS12})
	cli := tls.Client(clientConn, &tls.Config{ServerName: "localhost", InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}) // local fixture; signer proof is the handshake signature
	errs := make(chan error, 2)
	go func() { errs <- srv.Handshake() }()
	go func() { errs <- cli.Handshake() }()
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
}

func TestLoadTLSCertificateRejectsMismatchedSigner(t *testing.T) {
	certKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	certPath := writeTestCert(t, certKey)
	signer := &mockSigner{key: other, slot: 7, ref: "label:web"}
	cfg := KeyConfig{Provider: ProviderPKCS11, ModulePath: "/unused", SlotID: ptrU64(7), KeyLabel: "web", PINSecretRef: "env:UNUSED"}
	if _, _, err := LoadTLSCertificate(context.Background(), mockProvider{signer}, DefaultRuntimeConfig(), cfg, certPath, nil); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("expected mismatch, got %v", err)
	}
	if !signer.closed {
		t.Fatal("mismatched signer was not closed")
	}
}

func TestKeyConfigRejectsInlinePINAndAmbiguousSelectors(t *testing.T) {
	bad := KeyConfig{Provider: ProviderPKCS11, ModulePath: "/usr/lib/p11.so", KeyLabel: "web", PINSecretRef: "1234"}
	if err := bad.ValidateStatic(); err == nil {
		t.Fatal("inline PIN accepted")
	}
	bad.PINSecretRef = "env:HSM_PIN"
	if err := bad.ValidateStatic(); err == nil {
		t.Fatal("missing slot/token selector accepted")
	}
}

func TestResolveSecretFileRequiresPrivateMode(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pin")
	if err := os.WriteFile(p, []byte("1234\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSecret("file:" + p); err == nil {
		t.Fatal("world-readable PIN file accepted")
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveSecret("file:" + p)
	if err != nil {
		t.Fatal(err)
	}
	defer ZeroBytes(got)
	if string(got) != "1234" {
		t.Fatalf("unexpected secret length/content normalization")
	}
}

func ptrU64(v uint64) *uint64 { return &v }

type fakeNativeModule struct {
	key     *rsa.PrivateKey
	session *fakeNativeSession
	closed  bool
}
type fakeNativeSession struct {
	key      *rsa.PrivateKey
	loggedIn bool
	found    bool
	closed   bool
	signs    int
}

func (m *fakeNativeModule) Slots() ([]uint64, error) { return []uint64{7}, nil }
func (m *fakeNativeModule) TokenLabel(slot uint64) (string, error) {
	if slot != 7 {
		return "", os.ErrNotExist
	}
	return "prod-token", nil
}
func (m *fakeNativeModule) OpenSession(slot uint64) (nativeSession, error) {
	m.session = &fakeNativeSession{key: m.key}
	return m.session, nil
}
func (m *fakeNativeModule) Close() error { m.closed = true; return nil }
func (s *fakeNativeSession) Login(pin []byte) error {
	if string(pin) != "1234" {
		return os.ErrPermission
	}
	s.loggedIn = true
	return nil
}
func (s *fakeNativeSession) FindPrivateKey(label string, id []byte) (uint64, error) {
	if !s.loggedIn || label != "waf-tls" || string(id) != "\x01" {
		return 0, os.ErrNotExist
	}
	s.found = true
	return 99, nil
}
func (s *fakeNativeSession) Sign(key, mechanism uint64, param, data []byte) ([]byte, error) {
	if key != 99 || !s.found {
		return nil, os.ErrPermission
	}
	s.signs++
	switch mechanism {
	case ckmRSAPKCS:
		if len(data) < 32 {
			return nil, os.ErrInvalid
		}
		return rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, data[len(data)-32:])
	case ckmRSAPSS:
		if len(data) != 32 {
			return nil, os.ErrInvalid
		}
		return rsa.SignPSS(rand.Reader, s.key, crypto.SHA256, data, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	default:
		return nil, os.ErrInvalid
	}
}
func (s *fakeNativeSession) Close() error { s.closed = true; return nil }

func TestPKCS11ProviderSessionLoginLookupAndAuditLifecycle(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	certPath := writeTestCert(t, key)
	oldOpen := openNativePKCS11Module
	oldValidate := validateModulePathForOpen
	fake := &fakeNativeModule{key: key}
	openNativePKCS11Module = func(string) (nativeModule, error) { return fake, nil }
	validateModulePathForOpen = func(RuntimeConfig, string) error { return nil }
	t.Cleanup(func() {
		openNativePKCS11Module = oldOpen
		validateModulePathForOpen = oldValidate
		moduleRegistry.Lock()
		moduleRegistry.m = map[string]*moduleEntry{}
		moduleRegistry.Unlock()
	})
	t.Setenv("TEST_HSM_PIN", "1234")
	slot := uint64(7)
	cfg := KeyConfig{Provider: ProviderPKCS11, ModulePath: "/approved/vendor.so", SlotID: &slot, TokenLabel: "prod-token", KeyLabel: "waf-tls", KeyID: "01", PINSecretRef: "env:TEST_HSM_PIN"}
	var audits []AuditEvent
	_, signer, err := LoadTLSCertificate(context.Background(), PKCS11Provider{}, RuntimeConfig{AllowedModuleDirs: []string{"/approved"}}, cfg, certPath, func(e AuditEvent) { audits = append(audits, e) })
	if err != nil {
		t.Fatal(err)
	}
	if !fake.session.loggedIn || !fake.session.found || fake.session.signs == 0 {
		t.Fatalf("lifecycle missing: %+v", fake.session)
	}
	health := signer.HealthCheck(context.Background())
	if health.Status != "READY" {
		t.Fatalf("health status = %q, want READY: %+v", health.Status, health)
	}
	wantChecks := map[string]bool{"provider": false, "slot": false, "token": false, "key": false}
	for _, check := range health.Checks {
		if _, ok := wantChecks[check.Component]; ok && check.Status == "READY" {
			wantChecks[check.Component] = true
		}
	}
	for component, ok := range wantChecks {
		if !ok {
			t.Fatalf("missing READY health check for %s: %+v", component, health.Checks)
		}
	}
	if err := signer.Close(); err != nil {
		t.Fatal(err)
	}
	if !fake.session.closed || !fake.closed {
		t.Fatal("session/module did not close")
	}
	want := []string{"SESSION_OPEN", "LOGIN", "KEY_LOOKUP", "SIGN", "ASSOCIATION_VERIFY", "SESSION_CLOSE"}
	got := make([]string, 0, len(audits))
	for _, e := range audits {
		got = append(got, e.Operation)
		b, _ := json.Marshal(e)
		if strings.Contains(string(b), "TEST_HSM_PIN") || strings.Contains(string(b), "1234") {
			t.Fatalf("audit leaked secret: %s", b)
		}
	}
	for _, op := range want {
		found := false
		for _, g := range got {
			if g == op {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing audit op %s in %v", op, got)
		}
	}
}
