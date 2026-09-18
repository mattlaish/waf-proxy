package secretref

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxSecretBytes = 4096

// Validate accepts references to environment variables or root/operator managed
// files. Inline secret values are deliberately not accepted.
func Validate(ref string) error {
	if strings.HasPrefix(ref, "env:") {
		name := strings.TrimPrefix(ref, "env:")
		if !validEnvName(name) {
			return errors.New("invalid env secret reference")
		}
		return nil
	}
	if strings.HasPrefix(ref, "file:") {
		p := strings.TrimPrefix(ref, "file:")
		if !filepath.IsAbs(p) || filepath.Clean(p) != p {
			return errors.New("file secret reference must use an absolute clean path")
		}
		return nil
	}
	return errors.New("must use env:NAME or file:/absolute/path; inline secrets are not accepted")
}

func validEnvName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

// Resolve resolves one validated secret reference. File secrets must be regular,
// non-symlink files and no path component may be a symlink. The post-open
// SameFile check detects replacement between metadata validation and Open.
func Resolve(ref string) ([]byte, error) {
	if err := Validate(ref); err != nil {
		return nil, err
	}
	if strings.HasPrefix(ref, "env:") {
		name := strings.TrimPrefix(ref, "env:")
		value, ok := os.LookupEnv(name)
		if !ok || value == "" {
			return nil, fmt.Errorf("environment secret %q is unavailable", name)
		}
		if len(value) > maxSecretBytes {
			return nil, errors.New("environment secret exceeds size limit")
		}
		return []byte(value), nil
	}

	p := strings.TrimPrefix(ref, "file:")
	realPath, err := filepath.EvalSymlinks(p)
	if err != nil || realPath != p {
		return nil, errors.New("secret file path must not contain symlinks")
	}
	before, err := os.Lstat(p)
	if err != nil {
		return nil, errors.New("secret file is unavailable")
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, errors.New("secret file must be a regular non-symlink file")
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, errors.New("secret file could not be read")
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, errors.New("secret file changed while opening")
	}
	if after.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("secret file must not be group/world accessible")
	}
	if after.Size() <= 0 || after.Size() > maxSecretBytes {
		return nil, errors.New("secret file size is invalid")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSecretBytes+1))
	if err != nil || len(b) == 0 || len(b) > maxSecretBytes {
		Zero(b)
		return nil, errors.New("secret file could not be read")
	}
	if n := len(b); n > 0 && b[n-1] == '\n' {
		b = b[:n-1]
		if n = len(b); n > 0 && b[n-1] == '\r' {
			b = b[:n-1]
		}
	}
	if len(b) == 0 {
		Zero(b)
		return nil, errors.New("resolved secret is empty")
	}
	return b, nil
}

func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
