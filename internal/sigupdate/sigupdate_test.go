package sigupdate

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// These are the five required test vectors from the Signed Update & Publishing
// Standard §12. They are self-contained: an RSA keypair is generated here and
// packages are built + signed in memory, so no openssl or network is needed.
//
//   go test ./...

func pubPEM(t *testing.T, priv *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func sign(t *testing.T, priv *rsa.PrivateKey, msg []byte) []byte {
	t.Helper()
	h := sha256.Sum256(msg)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		t.Fatal(err)
	}
	return []byte(base64.StdEncoding.EncodeToString(sig))
}

// buildPackage builds a .update zip. If tamperManifest/tamperPayload/tamperSig
// are set, it corrupts the corresponding part AFTER signing.
func buildPackage(t *testing.T, priv *rsa.PrivateKey, files map[string][]byte,
	tamperManifest, tamperPayload bool, resignWith *rsa.PrivateKey) []byte {
	t.Helper()

	type mf struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	var entries []mf
	for name, data := range files {
		sum := sha256.Sum256(data)
		entries = append(entries, mf{Path: name, SHA256: hex.EncodeToString(sum[:])})
	}
	manifest := map[string]any{"version": "1.1.0", "notes": "test", "files": entries}
	manifestBytes, _ := json.Marshal(manifest)

	signBytes := manifestBytes
	sigKey := priv
	if resignWith != nil {
		sigKey = resignWith
	}
	sig := sign(t, sigKey, signBytes)

	// tamper AFTER signing
	if tamperManifest {
		manifestBytes = bytes.Replace(manifestBytes, []byte("1.1.0"), []byte("9.9.9"), 1)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("manifest.json")
	w.Write(manifestBytes)
	w, _ = zw.Create("manifest.sig")
	w.Write(sig)
	for name, data := range files {
		w, _ = zw.Create("files/" + name)
		if tamperPayload {
			w.Write([]byte("MALICIOUS"))
		} else {
			w.Write(data)
		}
	}
	zw.Close()
	return buf.Bytes()
}

func testConfig(t *testing.T, pubKey string) *Config {
	dir := t.TempDir()
	return &Config{
		PublisherKeyPEM: pubKey,
		InstallDir:      dir,
		BackupDir:       filepath.Join(dir, ".backup"),
		AllowedExts:     map[string]bool{".json": true, ".conf": true, ".html": true},
		AllowedNames:    map[string]bool{"wafconsole": true},
	}
}

func TestVector1_PristineAccepted(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cfg := testConfig(t, pubPEM(t, priv))
	pkg := buildPackage(t, priv, map[string][]byte{"app.conf": []byte("ok")}, false, false, nil)

	m, payloads, err := cfg.Inspect(pkg)
	if err != nil {
		t.Fatalf("pristine package rejected: %v", err)
	}
	if m.Version != "1.1.0" || len(payloads) != 1 {
		t.Fatalf("unexpected manifest: %+v", m)
	}
}

func TestVector2_SwappedPayloadRejected(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cfg := testConfig(t, pubPEM(t, priv))
	pkg := buildPackage(t, priv, map[string][]byte{"app.conf": []byte("ok")}, false, true, nil)

	if _, _, err := cfg.Inspect(pkg); err == nil {
		t.Fatal("swapped payload was ACCEPTED — must be rejected")
	}
}

func TestVector3_AlteredManifestRejected(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cfg := testConfig(t, pubPEM(t, priv))
	pkg := buildPackage(t, priv, map[string][]byte{"app.conf": []byte("ok")}, true, false, nil)

	if _, _, err := cfg.Inspect(pkg); err == nil {
		t.Fatal("altered manifest was ACCEPTED — must be rejected")
	}
}

func TestVector4_ForeignKeyRejected(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	attacker, _ := rsa.GenerateKey(rand.Reader, 2048)
	cfg := testConfig(t, pubPEM(t, priv)) // trusts priv
	pkg := buildPackage(t, priv, map[string][]byte{"app.conf": []byte("ok")}, false, false, attacker)

	if _, _, err := cfg.Inspect(pkg); err == nil {
		t.Fatal("foreign-key signature was ACCEPTED — must be rejected")
	}
}

func TestVector5_TamperedCatalogRejected(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := pubPEM(t, priv)
	pubParsed, err := ParsePublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	catalog := []byte(`{"packages":[{"version":"1.1.0","filename":"x.wafupdate"}]}`)
	sig := sign(t, priv, catalog)
	tampered := bytes.Replace(catalog, []byte("1.1.0"), []byte("9.9.9"), 1)

	if err := verify(pubParsed, tampered, decodeSig(sig)); err == nil {
		t.Fatal("tampered catalog signature was ACCEPTED — must be rejected")
	}
	// sanity: the untampered catalog verifies
	if err := verify(pubParsed, catalog, decodeSig(sig)); err != nil {
		t.Fatalf("pristine catalog rejected: %v", err)
	}
}

// Bonus: path-traversal and disallowed types are rejected.
func TestPathSafety(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cfg := testConfig(t, pubPEM(t, priv))
	for _, bad := range []string{"../etc/passwd", "/abs/path", "evil.py", `back\slash`} {
		pkg := buildPackage(t, priv, map[string][]byte{bad: []byte("x")}, false, false, nil)
		if _, _, err := cfg.Inspect(pkg); err == nil {
			t.Fatalf("unsafe path %q was ACCEPTED", bad)
		}
	}
}

// Bonus: end-to-end apply + rollback.
func TestApplyAndRollback(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	cfg := testConfig(t, pubPEM(t, priv))

	// seed an existing file so we get a "replace" + backup
	orig := filepath.Join(cfg.InstallDir, "app.conf")
	os.WriteFile(orig, []byte("ORIGINAL"), 0o644)

	pkg := buildPackage(t, priv, map[string][]byte{"app.conf": []byte("NEW")}, false, false, nil)
	_, payloads, err := cfg.Inspect(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.Apply(payloads); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(orig); string(b) != "NEW" {
		t.Fatalf("apply did not write new content: %q", b)
	}
	if !cfg.HasBackup() {
		t.Fatal("no backup recorded")
	}
	if _, err := cfg.Rollback(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(orig); string(b) != "ORIGINAL" {
		t.Fatalf("rollback did not restore: %q", b)
	}
}
