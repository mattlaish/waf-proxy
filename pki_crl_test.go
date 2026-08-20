package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type crlTestPKI struct {
	root    *x509.Certificate
	rootKey *rsa.PrivateKey
	rootPEM []byte
	leaf    *x509.Certificate
	server  tls.Certificate
}

func newCRLTestPKI(t *testing.T, rootName string) crlTestPKI {
	t.Helper()
	now := time.Now()
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rootTemplate := &x509.Certificate{SerialNumber: big.NewInt(101), Subject: pkix.Name{CommonName: rootName},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{SerialNumber: big.NewInt(202), Subject: pkix.Name{CommonName: "backend.internal"},
		DNSNames: []string{"backend.internal"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(12 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(leafKey)})
	server, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return crlTestPKI{root: root, rootKey: rootKey, rootPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}), leaf: leaf, server: server}
}

func (p crlTestPKI) crl(t *testing.T, thisUpdate, nextUpdate time.Time, revoked ...*big.Int) []byte {
	t.Helper()
	entries := make([]x509.RevocationListEntry, 0, len(revoked))
	for _, serial := range revoked {
		entries = append(entries, x509.RevocationListEntry{SerialNumber: serial, RevocationTime: thisUpdate})
	}
	der, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number: big.NewInt(1), ThisUpdate: thisUpdate, NextUpdate: nextUpdate, RevokedCertificateEntries: entries,
	}, p.root, p.rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: der})
}

func crlBackend(t *testing.T, cert tls.Certificate) *httptest.Server {
	t.Helper()
	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	s.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	s.StartTLS()
	return s
}

func crlClientResult(t *testing.T, backend *httptest.Server, pki crlTestPKI, crl []byte, mode string) error {
	t.Helper()
	cfg, err := buildBackendTLSConfig(BackendTLSConfig{
		UseSystemCA: boolPtr(false), CAFiles: []string{writeTestFile(t, "root.pem", pki.rootPEM)},
		CRLFiles: []string{writeTestFile(t, "issuer.crl", crl)}, ServerName: "backend.internal", RevocationMode: mode,
	}, "https")
	if err != nil {
		return err
	}
	_, err = (&http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}).Get(backend.URL)
	return err
}

func TestStaticCRLAllowsValidAndRejectsRevoked(t *testing.T) {
	pki := newCRLTestPKI(t, "root one")
	backend := crlBackend(t, pki.server)
	defer backend.Close()
	now := time.Now()
	if err := crlClientResult(t, backend, pki, pki.crl(t, now.Add(-time.Minute), now.Add(time.Hour)), "hard"); err != nil {
		t.Fatalf("valid unrevoked certificate rejected: %v", err)
	}
	if err := crlClientResult(t, backend, pki, pki.crl(t, now.Add(-time.Minute), now.Add(time.Hour), pki.leaf.SerialNumber), "soft"); err == nil {
		t.Fatal("revoked certificate accepted in soft mode")
	}
}

func TestStaticCRLSoftAndHardMissingCoverage(t *testing.T) {
	pki := newCRLTestPKI(t, "root one")
	other := newCRLTestPKI(t, "other root")
	backend := crlBackend(t, pki.server)
	defer backend.Close()
	now := time.Now()
	unrelated := other.crl(t, now.Add(-time.Minute), now.Add(time.Hour))
	if err := crlClientResult(t, backend, pki, unrelated, "soft"); err != nil {
		t.Fatalf("soft mode rejected missing CRL coverage: %v", err)
	}
	if err := crlClientResult(t, backend, pki, unrelated, "hard"); err == nil {
		t.Fatal("hard mode accepted missing CRL coverage")
	}
}

func TestStaticCRLRejectsWrongIssuerSignature(t *testing.T) {
	pki := newCRLTestPKI(t, "shared issuer name")
	wrongSigner := newCRLTestPKI(t, "shared issuer name")
	backend := crlBackend(t, pki.server)
	defer backend.Close()
	now := time.Now()
	if err := crlClientResult(t, backend, pki, wrongSigner.crl(t, now.Add(-time.Minute), now.Add(time.Hour)), "soft"); err == nil {
		t.Fatal("CRL with matching issuer name but invalid signature was accepted")
	}
}

func TestStaticCRLRejectsExpiredFutureAndMalformed(t *testing.T) {
	pki := newCRLTestPKI(t, "root one")
	now := time.Now()
	for name, body := range map[string][]byte{
		"expired":   pki.crl(t, now.Add(-2*time.Hour), now.Add(-time.Hour)),
		"future":    pki.crl(t, now.Add(time.Hour), now.Add(2*time.Hour)),
		"malformed": []byte("not a CRL"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := buildBackendTLSConfig(BackendTLSConfig{UseSystemCA: boolPtr(false), CAFiles: []string{writeTestFile(t, "root.pem", pki.rootPEM)}, CRLFiles: []string{writeTestFile(t, "bad.crl", body)}}, "https")
			if err == nil {
				t.Fatal("invalid CRL accepted")
			}
		})
	}
}

func TestStaticCRLChecksLeafAndIntermediate(t *testing.T) {
	now := time.Now()
	root := newCRLTestPKI(t, "chain root")
	interKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	interTemplate := &x509.Certificate{SerialNumber: big.NewInt(303), Subject: pkix.Name{CommonName: "chain intermediate"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(12 * time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	interDER, err := x509.CreateCertificate(rand.Reader, interTemplate, root.root, &interKey.PublicKey, root.rootKey)
	if err != nil {
		t.Fatal(err)
	}
	inter, err := x509.ParseCertificate(interDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	leafTemplate := &x509.Certificate{SerialNumber: big.NewInt(404), Subject: pkix.Name{CommonName: "backend.internal"}, DNSNames: []string{"backend.internal"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(6 * time.Hour), ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, KeyUsage: x509.KeyUsageDigitalSignature}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, inter, &leafKey.PublicKey, interKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(leafDER)
	makeList := func(issuer *x509.Certificate, key *rsa.PrivateKey, revoked ...*big.Int) *x509.RevocationList {
		entries := make([]x509.RevocationListEntry, 0, len(revoked))
		for _, serial := range revoked {
			entries = append(entries, x509.RevocationListEntry{SerialNumber: serial, RevocationTime: now})
		}
		der, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{Number: big.NewInt(9), ThisUpdate: now.Add(-time.Minute), NextUpdate: now.Add(time.Hour), RevokedCertificateEntries: entries}, issuer, key)
		if err != nil {
			t.Fatal(err)
		}
		list, err := x509.ParseRevocationList(der)
		if err != nil {
			t.Fatal(err)
		}
		return list
	}
	state := tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{leaf, inter, root.root}}}
	verify := func(lists ...*x509.RevocationList) error {
		store := &crlStore{mode: "hard"}
		store.current.Store(&crlSnapshot{Lists: lists, LoadedAt: now})
		return store.verify(state)
	}
	if err := verify(makeList(inter, interKey), makeList(root.root, root.rootKey)); err != nil {
		t.Fatalf("covered chain rejected: %v", err)
	}
	if err := verify(makeList(inter, interKey, leaf.SerialNumber), makeList(root.root, root.rootKey)); err == nil {
		t.Fatal("revoked leaf accepted")
	}
	if err := verify(makeList(inter, interKey), makeList(root.root, root.rootKey, inter.SerialNumber)); err == nil {
		t.Fatal("revoked intermediate accepted")
	}
}

func TestStaticCRLParsesDER(t *testing.T) {
	pki := newCRLTestPKI(t, "DER root")
	pemCRL := pki.crl(t, time.Now().Add(-time.Minute), time.Now().Add(time.Hour))
	block, _ := pem.Decode(pemCRL)
	path := writeTestFile(t, "issuer.crl", block.Bytes)
	lists, err := parseCRLFile(path, time.Now())
	if err != nil || len(lists) != 1 {
		t.Fatalf("DER CRL parse: lists=%d err=%v", len(lists), err)
	}
}
