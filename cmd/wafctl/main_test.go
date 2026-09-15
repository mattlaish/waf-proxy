package main

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestSanitizeTextMasksCredentials(t *testing.T) {
	got := string(sanitizeText([]byte(`Authorization: Bearer abc123 password=secret api_key="xyz"`)))
	for _, secret := range []string{"abc123", "secret", "xyz"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q remained in %q", secret, got)
		}
	}
}

func TestBuildSupportZipHasManifestAndChecksums(t *testing.T) {
	b, err := buildSupportZip(map[string][]byte{"doctor.json": []byte(`{"checks":[]}`)})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range zr.File {
		seen[f.Name] = true
		if f.Name == "manifest.json" {
			r, _ := f.Open()
			data, _ := io.ReadAll(r)
			_ = r.Close()
			if !strings.Contains(string(data), "waf-support-bundle-v1") {
				t.Fatalf("unexpected manifest: %s", data)
			}
		}
	}
	if !seen["manifest.json"] || !seen["SHA256SUMS.txt"] || !seen["doctor.json"] {
		t.Fatalf("bundle entries: %#v", seen)
	}
}

func TestRejectSecrets(t *testing.T) {
	if err := rejectSecrets("x", []byte("-----BEGIN PRIVATE KEY-----")); err == nil {
		t.Fatal("expected private key rejection")
	}
}
