package secretref

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvReference(t *testing.T) {
	t.Setenv("OPENAI_TEST_SECRET", "abc123")
	b, err := Resolve("env:OPENAI_TEST_SECRET")
	if err != nil || string(b) != "abc123" {
		t.Fatalf("Resolve env = %q, %v", b, err)
	}
	Zero(b)
	for _, v := range b {
		if v != 0 {
			t.Fatal("Zero did not clear secret bytes")
		}
	}
}

func TestFileReferencePermissionsAndSymlink(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "key")
	if err := os.WriteFile(p, []byte("abc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := Resolve("file:" + p)
	if err != nil || string(b) != "abc" {
		t.Fatalf("Resolve file = %q, %v", b, err)
	}
	Zero(b)
	if err := os.Chmod(p, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("file:" + p); err == nil {
		t.Fatal("group-readable secret accepted")
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("file:" + link); err == nil {
		t.Fatal("symlink secret file accepted")
	}

	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0o700); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(realDir, "nested-key")
	if err := os.WriteFile(nested, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	dirLink := filepath.Join(dir, "linked-dir")
	if err := os.Symlink(realDir, dirLink); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("file:" + filepath.Join(dirLink, "nested-key")); err == nil {
		t.Fatal("secret file beneath symlinked directory accepted")
	}
}

func TestValidateRejectsInlineAndRelative(t *testing.T) {
	for _, ref := range []string{"sekrit", "file:relative", "env:", "env: BAD", "env:BAD NAME", "env:9BAD", "env:BAD-NAME"} {
		if err := Validate(ref); err == nil {
			t.Fatalf("Validate(%q) unexpectedly succeeded", ref)
		}
	}
}

func TestFileReferenceCRLFAndOversize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "key")
	if err := os.WriteFile(p, []byte("abc\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve("file:" + p)
	if err != nil || string(got) != "abc" {
		t.Fatalf("Resolve CRLF file = %q, %v", got, err)
	}
	Zero(got)

	tooLarge := make([]byte, maxSecretBytes+1)
	for i := range tooLarge {
		tooLarge[i] = 'x'
	}
	if err := os.WriteFile(p, tooLarge, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("file:" + p); err == nil {
		t.Fatal("oversized secret file accepted")
	}
}
