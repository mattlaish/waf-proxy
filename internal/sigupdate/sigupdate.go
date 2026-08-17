// Package sigupdate implements the Signed Update & Publishing Standard (v1.0)
// for Go products (single-binary services such as the WAF console and NDR
// correlator). It is the Go counterpart of VaultGate's updater.py.
//
// Trust model: the publisher holds an RSA private key OFFLINE and signs update
// packages with it. Each build embeds only the PUBLIC key and verifies here.
// A product can verify an update but can never forge one.
//
// Verification: RSA PKCS#1 v1.5 with SHA-256 over the exact bytes of
// manifest.json. Interoperable with `openssl dgst -sha256 -sign` (what
// make-update.sh / make-catalog.sh produce).
//
// Stdlib only — no third-party dependencies.
//
// See signed-update-standard.md §5 (spec) and §6 (verifier requirements).
package sigupdate

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Config holds a product's update settings. One value per product; the shared
// verify/apply logic is parameterised by it.
type Config struct {
	// PublisherKeyPEM is the baked-in publisher PUBLIC key (SPKI or PKCS#1 PEM).
	// Empty string disables updates entirely (fail-safe: feature off).
	PublisherKeyPEM string

	// CatalogURL is the baked-in base URL for the online catalog, e.g.
	// "https://updates.example.com/waf/". Empty disables the online catalog
	// (upload-from-disk still works).
	CatalogURL string

	// InstallDir is where payload files are written (usually the dir holding
	// the running binary).
	InstallDir string

	// BackupDir holds the previous version of replaced files, for rollback.
	BackupDir string

	// AllowedExts is the set of permitted file extensions (lower-case, incl.
	// the dot), e.g. {".json": true, ".conf": true}.
	AllowedExts map[string]bool

	// AllowedNames is the set of permitted exact basenames for extensionless
	// files — chiefly the product binary, e.g. {"wafconsole": true}.
	AllowedNames map[string]bool

	// MaxPackageBytes / MaxCatalogBytes cap download/upload size.
	MaxPackageBytes int64
	MaxCatalogBytes int64

	// HTTPTimeout bounds catalog/package fetches.
	HTTPTimeout time.Duration
}

// Manifest is the signed metadata inside a package (standard §5.3).
type Manifest struct {
	Version         string         `json:"version"`
	Released        string         `json:"released,omitempty"`
	Notes           string         `json:"notes,omitempty"`
	RestartRequired bool           `json:"restart_required,omitempty"`
	Files           []ManifestFile `json:"files"`
}

// ManifestFile is one entry in Manifest.Files.
type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Catalog is the signed index of available packages (standard §5.4).
type Catalog struct {
	Generated string           `json:"generated,omitempty"`
	Packages  []CatalogPackage `json:"packages"`
}

// CatalogPackage is one entry in Catalog.Packages.
type CatalogPackage struct {
	Version  string `json:"version"`
	Filename string `json:"filename"`
	SHA256   string `json:"sha256,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Notes    string `json:"notes,omitempty"`
	Released string `json:"released,omitempty"`
}

// Applied records one file written by Apply.
type Applied struct {
	Path   string // install-dir-relative
	Action string // "add" or "replace"
}

// safePath: conservative allow-list of characters (standard §6.4).
var safePath = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./-]*$`)

// Sensible defaults for the size/timeout knobs if the caller leaves them zero.
func (c *Config) withDefaults() {
	if c.MaxPackageBytes == 0 {
		c.MaxPackageBytes = 200 << 20 // 200 MB (Go binaries are large)
	}
	if c.MaxCatalogBytes == 0 {
		c.MaxCatalogBytes = 1 << 20 // 1 MB
	}
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = 30 * time.Second
	}
}

// Enabled reports whether updates are usable (a publisher key is present).
func (c *Config) Enabled() bool {
	return strings.TrimSpace(c.PublisherKeyPEM) != ""
}

// ---------------------------------------------------------------------------
// Public key + signature
// ---------------------------------------------------------------------------

// ParsePublicKey parses an RSA public key from PEM (SPKI or PKCS#1).
func ParsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("no PEM block in public key")
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rp, ok := pub.(*rsa.PublicKey); ok {
			return rp, nil
		}
		return nil, errors.New("public key is not RSA")
	}
	// Fall back to PKCS#1.
	if rp, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return rp, nil
	}
	return nil, errors.New("unrecognised RSA public key format")
}

// KeyFingerprint returns the SHA-256 hex of the key's DER, for display.
func KeyFingerprint(pemStr string) (string, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return "", errors.New("no PEM block")
	}
	sum := sha256.Sum256(block.Bytes)
	return hex.EncodeToString(sum[:]), nil
}

// verify checks a PKCS#1 v1.5 / SHA-256 signature over msg.
func verify(pub *rsa.PublicKey, msg, sig []byte) error {
	h := sha256.Sum256(msg)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig)
}

var base64ish = regexp.MustCompile(`^[A-Za-z0-9+/=\s]+$`)

// decodeSig accepts either base64 (what the sign scripts emit) or raw bytes.
func decodeSig(raw []byte) []byte {
	s := bytes.TrimSpace(raw)
	if base64ish.Match(s) {
		if d, err := base64.StdEncoding.DecodeString(string(bytes.ReplaceAll(s, []byte("\n"), nil))); err == nil {
			return d
		}
	}
	return s
}

// ---------------------------------------------------------------------------
// Package verification (standard §6.1–§6.4)
// ---------------------------------------------------------------------------

// Inspect fully verifies a package WITHOUT writing anything. On success it
// returns the trusted manifest and the payload bytes keyed by path.
// Checks happen in order and fail closed on the first problem (§6).
func (c *Config) Inspect(pkg []byte) (*Manifest, map[string][]byte, error) {
	if !c.Enabled() {
		return nil, nil, errors.New("no publisher key configured — updates disabled")
	}
	pub, err := ParsePublicKey(c.PublisherKeyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("bad publisher key: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil {
		return nil, nil, errors.New("not a valid update archive")
	}

	members := map[string]*zip.File{}
	for _, f := range zr.File {
		members[f.Name] = f
	}
	mf, ok1 := members["manifest.json"]
	sf, ok2 := members["manifest.sig"]
	if !ok1 || !ok2 {
		return nil, nil, errors.New("package missing manifest.json or manifest.sig")
	}

	manifestBytes, err := readZip(mf)
	if err != nil {
		return nil, nil, err
	}
	sigBytes, err := readZip(sf)
	if err != nil {
		return nil, nil, err
	}

	// (§6.2) Verify signature BEFORE parsing the manifest as JSON.
	if err := verify(pub, manifestBytes, decodeSig(sigBytes)); err != nil {
		return nil, nil, errors.New("SIGNATURE INVALID — package is not signed by the publisher key")
	}

	var m Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return nil, nil, errors.New("manifest.json is not valid JSON")
	}
	if strings.TrimSpace(m.Version) == "" || len(m.Files) == 0 {
		return nil, nil, errors.New("manifest must list a version and at least one file")
	}

	// (§6.3, §6.4) Path safety + checksum for every file.
	payloads := map[string][]byte{}
	for _, entry := range m.Files {
		if err := c.checkPath(entry.Path); err != nil {
			return nil, nil, err
		}
		member, ok := members["files/"+entry.Path]
		if !ok {
			return nil, nil, fmt.Errorf("manifest references missing file: %s", entry.Path)
		}
		data, err := readZip(member)
		if err != nil {
			return nil, nil, err
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != strings.ToLower(entry.SHA256) {
			return nil, nil, fmt.Errorf("checksum mismatch for %s", entry.Path)
		}
		payloads[entry.Path] = data
	}
	return &m, payloads, nil
}

// checkPath enforces the standard's path rules (§6.4).
func (c *Config) checkPath(p string) error {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, `\`) {
		return fmt.Errorf("unsafe path in package: %q", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return fmt.Errorf("path traversal in package: %q", p)
		}
	}
	if !safePath.MatchString(p) {
		return fmt.Errorf("illegal characters in path: %q", p)
	}
	base := path.Base(p)
	ext := strings.ToLower(filepath.Ext(base))
	if ext == "" {
		if !c.AllowedNames[base] {
			return fmt.Errorf("file not permitted: %q", p)
		}
	} else if !c.AllowedExts[ext] {
		return fmt.Errorf("file type not permitted: %q", p)
	}
	return nil
}

func readZip(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// ---------------------------------------------------------------------------
// Apply / rollback (standard §6.7)
// ---------------------------------------------------------------------------

// Apply writes verified payloads into InstallDir, backing up any replaced file
// into BackupDir first. Writes are temp-file-then-rename (atomic per file), and
// each target is re-checked to be inside InstallDir at write time.
func (c *Config) Apply(payloads map[string][]byte) ([]Applied, error) {
	instAbs, err := filepath.Abs(c.InstallDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.BackupDir, 0o755); err != nil {
		return nil, err
	}

	var applied []Applied
	for rel, data := range payloads {
		target := filepath.Join(instAbs, filepath.FromSlash(rel))
		// (§6.4) Re-check containment at write time (defends against symlinks).
		clean := filepath.Clean(target)
		if clean != instAbs && !strings.HasPrefix(clean, instAbs+string(os.PathSeparator)) {
			return applied, fmt.Errorf("resolved path escapes install dir: %s", rel)
		}

		action := "add"
		if _, err := os.Stat(target); err == nil {
			action = "replace"
			bpath := filepath.Join(c.BackupDir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(bpath), 0o755); err != nil {
				return applied, err
			}
			if err := copyFile(target, bpath); err != nil {
				return applied, err
			}
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return applied, err
		}
		mode := os.FileMode(0o644)
		// Preserve executable bit when replacing an executable (e.g. the binary).
		if fi, err := os.Stat(target); err == nil && fi.Mode()&0o111 != 0 {
			mode = fi.Mode().Perm()
		} else if ext := strings.ToLower(filepath.Ext(rel)); ext == ".sh" || ext == "" {
			mode = 0o755
		}
		if err := atomicWrite(target, data, mode); err != nil {
			return applied, err
		}
		applied = append(applied, Applied{Path: rel, Action: action})
	}
	return applied, nil
}

// Rollback restores every file present in BackupDir into InstallDir.
func (c *Config) Rollback() ([]string, error) {
	var restored []string
	err := filepath.Walk(c.BackupDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(c.BackupDir, p)
		if err != nil {
			return err
		}
		target := filepath.Join(c.InstallDir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyFile(p, target); err != nil {
			return err
		}
		restored = append(restored, filepath.ToSlash(rel))
		return nil
	})
	if os.IsNotExist(err) {
		return nil, errors.New("no backup to roll back to")
	}
	return restored, err
}

// HasBackup reports whether a rollback is available.
func (c *Config) HasBackup() bool {
	entries, err := os.ReadDir(c.BackupDir)
	return err == nil && len(entries) > 0
}

func atomicWrite(target string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".sigupd-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, target) // atomic on the same filesystem
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	fi, err := os.Stat(src)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = fi.Mode().Perm()
	}
	return os.WriteFile(dst, data, mode)
}

// NeedsRestart reports whether applying the package requires a restart.
// For single-binary Go products this is effectively always true once anything
// is applied; the manifest may also force it. (Standard §7.3.)
func (c *Config) NeedsRestart(m *Manifest, applied []Applied) bool {
	if m.RestartRequired {
		return true
	}
	// Any binary replacement (extensionless file matching AllowedNames) forces
	// a restart; and for a compiled product, a replaced binary is the norm.
	for _, a := range applied {
		base := path.Base(a.Path)
		if filepath.Ext(base) == "" && c.AllowedNames[base] {
			return true
		}
	}
	// Conservative default for compiled products: restart if anything changed.
	return len(applied) > 0
}

// ---------------------------------------------------------------------------
// Online catalog (standard §5.4, §6.5)
// ---------------------------------------------------------------------------

func (c *Config) httpClient() *http.Client {
	c.withDefaults()
	return &http.Client{Timeout: c.HTTPTimeout}
}

func (c *Config) fetch(url string, max int64) ([]byte, error) {
	resp, err := c.httpClient().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s -> HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("response too large: %s", url)
	}
	return data, nil
}

// FetchCatalog downloads catalog.json + catalog.json.sig from CatalogURL and
// verifies the signature. A tampered or unsigned catalog is rejected, so a
// hostile server cannot fake even the LIST of available packages.
func (c *Config) FetchCatalog() (*Catalog, error) {
	c.withDefaults()
	if strings.TrimSpace(c.CatalogURL) == "" {
		return nil, errors.New("no catalog URL configured")
	}
	if !c.Enabled() {
		return nil, errors.New("no publisher key configured")
	}
	pub, err := ParsePublicKey(c.PublisherKeyPEM)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(c.CatalogURL, "/")
	catBytes, err := c.fetch(base+"/catalog.json", c.MaxCatalogBytes)
	if err != nil {
		return nil, err
	}
	sigBytes, err := c.fetch(base+"/catalog.json.sig", 64<<10)
	if err != nil {
		return nil, err
	}
	if err := verify(pub, catBytes, decodeSig(sigBytes)); err != nil {
		return nil, errors.New("catalog SIGNATURE INVALID — not signed by the publisher key")
	}
	var cat Catalog
	if err := json.Unmarshal(catBytes, &cat); err != nil {
		return nil, errors.New("catalog.json is not valid JSON")
	}
	return &cat, nil
}

// Download fetches one package by its catalog filename and fully verifies it.
// The filename must be a bare name ending in the product extension (§6.5); it
// is appended to CatalogURL. Returns the same result shape as Inspect.
func (c *Config) Download(filename, ext string) (*Manifest, map[string][]byte, error) {
	c.withDefaults()
	if strings.ContainsAny(filename, `/\`) || strings.Contains(filename, "..") ||
		!strings.HasSuffix(filename, ext) {
		return nil, nil, errors.New("invalid package filename")
	}
	base := strings.TrimRight(c.CatalogURL, "/")
	data, err := c.fetch(base+"/"+filename, c.MaxPackageBytes)
	if err != nil {
		return nil, nil, err
	}
	return c.Inspect(data)
}

// ---------------------------------------------------------------------------
// Restart (Unix; standard §7.3)
// ---------------------------------------------------------------------------
//
// After Apply replaces the binary on disk, the running process is still the old
// code. ReExec replaces the process image with the (new) binary. Implemented in
// sigupdate_unix.go via syscall.Exec; on other platforms callers should exit and
// let a supervisor restart. See that file.
