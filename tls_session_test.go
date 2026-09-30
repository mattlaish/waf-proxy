package main

import (
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestTLSSessionTicketKeysDeterministicAcrossNodesAndOverlapRotation(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	a, err := newTLSSessionTicketManager(secret, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	b, err := newTLSSessionTicketManager(secret, time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Unix(1_800_000_000, 0).UTC()
	ak := a.keysAt(t0)
	bk := b.keysAt(t0)
	if !reflect.DeepEqual(ak, bk) {
		t.Fatal("same shared secret did not derive the same HA ticket-key ring")
	}
	next := a.keysAt(t0.Add(time.Hour))
	if next[1] != ak[0] {
		t.Fatal("rotation did not retain the previous signing key for overlap decryption")
	}
	other, err := newTLSSessionTicketManager("fedcba9876543210fedcba9876543210", time.Hour, 3)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(ak, other.keysAt(t0)) {
		t.Fatal("different master secrets derived an identical ticket-key ring")
	}
}

func TestTLSSessionTicketSecretValidationAndPrivateFile(t *testing.T) {
	if _, err := newTLSSessionTicketManager("too-short", time.Hour, 3); err == nil {
		t.Fatal("short ticket secret accepted")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ticket-secret")
	if err := os.WriteFile(path, []byte("0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WAF_TLS_SESSION_TICKET_SECRET", "")
	t.Setenv("WAF_TLS_SESSION_TICKET_SECRET_FILE", path)
	got, err := loadTLSSessionTicketSecret()
	if err != nil {
		t.Fatal(err)
	}
	if got != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("secret-file read mismatch: %q", got)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadTLSSessionTicketSecret(); err == nil {
		t.Fatal("group/world-readable ticket secret file accepted")
	}
}

func tlsPipeHandshake(t *testing.T, serverCfg, clientCfg *tls.Config) tls.ConnectionState {
	t.Helper()
	serverRaw, clientRaw := net.Pipe()
	server := tls.Server(serverRaw, serverCfg)
	client := tls.Client(clientRaw, clientCfg)
	errCh := make(chan error, 1)
	go func() {
		if err := server.Handshake(); err != nil {
			errCh <- err
			return
		}
		_, err := server.Write([]byte{1}) // delivers TLS 1.3 NewSessionTicket before close
		errCh <- err
	}()
	if err := client.Handshake(); err != nil {
		_ = clientRaw.Close()
		_ = serverRaw.Close()
		t.Fatalf("client handshake: %v", err)
	}
	var one [1]byte
	if _, err := client.Read(one[:]); err != nil {
		_ = clientRaw.Close()
		_ = serverRaw.Close()
		t.Fatalf("client post-handshake read: %v", err)
	}
	if err := <-errCh; err != nil {
		_ = clientRaw.Close()
		_ = serverRaw.Close()
		t.Fatalf("server handshake/write: %v", err)
	}
	state := client.ConnectionState()
	_ = clientRaw.Close()
	_ = serverRaw.Close()
	return state
}

func TestTLSSessionTicketSharedSecretResumesAcrossServerConfigs(t *testing.T) {
	_, cert := makeBackendPKI(t, "waf.example")
	secret := "0123456789abcdef0123456789abcdef"
	for _, version := range []uint16{tls.VersionTLS12, tls.VersionTLS13} {
		t.Run(tls.VersionName(version), func(t *testing.T) {
			cache := tls.NewLRUClientSessionCache(4)
			clientCfg := &tls.Config{
				InsecureSkipVerify: true, // test-only self-signed fixture
				ServerName:         "waf.example",
				ClientSessionCache: cache,
				MinVersion:         version,
				MaxVersion:         version,
			}

			firstMgr, err := newTLSSessionTicketManager(secret, 24*time.Hour, 7)
			if err != nil {
				t.Fatal(err)
			}
			firstCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: version, MaxVersion: version}
			firstMgr.attach(firstCfg)
			if st := tlsPipeHandshake(t, firstCfg, clientCfg); st.DidResume {
				t.Fatal("first handshake unexpectedly resumed")
			}

			secondMgr, err := newTLSSessionTicketManager(secret, 24*time.Hour, 7)
			if err != nil {
				t.Fatal(err)
			}
			secondCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: version, MaxVersion: version}
			secondMgr.attach(secondCfg)
			if st := tlsPipeHandshake(t, secondCfg, clientCfg); !st.DidResume {
				t.Fatal("session did not resume across independent server configs with the same shared ticket secret")
			}
		})
	}
}

func TestTLSHandshakeObserverRecordsFullResumeVersionCertificateAndLatency(t *testing.T) {
	_, cert := makeBackendPKI(t, "waf.example")
	m := newMetrics()
	manager, err := newTLSSessionTicketManager("0123456789abcdef0123456789abcdef", 24*time.Hour, 7)
	if err != nil {
		t.Fatal(err)
	}
	serverCfg := &tls.Config{
		MinVersion:     tls.VersionTLS12,
		MaxVersion:     tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return &cert, nil },
	}
	manager.attach(serverCfg)
	installTLSHandshakeObserver(serverCfg, m, nil)
	clientCfg := &tls.Config{
		InsecureSkipVerify: true, // test-only self-signed fixture
		ServerName:         "waf.example",
		ClientSessionCache: tls.NewLRUClientSessionCache(4),
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS12,
	}
	if st := tlsPipeHandshake(t, serverCfg, clientCfg); st.DidResume {
		t.Fatal("first handshake unexpectedly resumed")
	}
	if st := tlsPipeHandshake(t, serverCfg, clientCfg); !st.DidResume {
		t.Fatal("second handshake did not resume")
	}
	status := m.tlsStatus()
	if status["attempts"].(int64) != 2 || status["successful"].(int64) != 2 || status["full"].(int64) != 1 || status["resumed"].(int64) != 1 {
		t.Fatalf("unexpected TLS counters: %#v", status)
	}
	versions := status["versions"].(map[string]int64)
	if versions["tls_1_2"] != 2 {
		t.Fatalf("TLS 1.2 count mismatch: %#v", versions)
	}
	algs := status["full_handshake_certificate_key_algorithms"].(map[string]int64)
	if algs["rsa"] != 1 {
		t.Fatalf("RSA full-handshake certificate count mismatch: %#v", algs)
	}
	latency := status["handshake_processing_latency_ms"].(map[string]float64)
	if latency["avg"] < 0 || latency["p95_bucket_upper_bound"] <= 0 {
		t.Fatalf("invalid handshake latency metrics: %#v", latency)
	}
}

func TestTLSHandshakeObserverCountsPolicyRejectWithoutSuccess(t *testing.T) {
	_, cert := makeBackendPKI(t, "waf.example")
	m := newMetrics()
	serverCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12}
	installTLSHandshakeObserver(serverCfg, m, func(*tls.ClientHelloInfo) error { return errTLSHandshakeTestReject })
	clientCfg := &tls.Config{InsecureSkipVerify: true, ServerName: "waf.example", MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12}
	serverRaw, clientRaw := net.Pipe()
	server := tls.Server(serverRaw, serverCfg)
	client := tls.Client(clientRaw, clientCfg)
	errCh := make(chan error, 1)
	go func() { errCh <- server.Handshake() }()
	_ = client.Handshake()
	_ = <-errCh
	_ = clientRaw.Close()
	_ = serverRaw.Close()
	status := m.tlsStatus()
	if status["attempts"].(int64) != 1 || status["policy_rejected"].(int64) != 1 || status["successful"].(int64) != 0 {
		t.Fatalf("policy-reject counters mismatch: %#v", status)
	}
}

var errTLSHandshakeTestReject = &tlsHandshakeTestError{}

type tlsHandshakeTestError struct{}

func (*tlsHandshakeTestError) Error() string { return "test reject" }
