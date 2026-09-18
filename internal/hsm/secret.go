package hsm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxSecretBytes = 4096

func ValidateSecretRef(ref string) error {
	if strings.HasPrefix(ref, "env:") {
		name := strings.TrimPrefix(ref, "env:")
		if name == "" || strings.ContainsAny(name, "=\x00") || strings.TrimSpace(name) != name {
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
	return errors.New("must use env:NAME or file:/absolute/path; inline PINs are not accepted")
}

func ResolveSecret(ref string) ([]byte, error) {
	if err := ValidateSecretRef(ref); err != nil {
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
	st, err := os.Lstat(p)
	if err != nil {
		return nil, errors.New("secret file is unavailable")
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
		return nil, errors.New("secret file must be a regular non-symlink file")
	}
	if st.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("secret file must not be group/world accessible")
	}
	if st.Size() <= 0 || st.Size() > maxSecretBytes {
		return nil, errors.New("secret file size is invalid")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, errors.New("secret file could not be read")
	}
	b = []byte(strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r"))
	if len(b) == 0 {
		return nil, errors.New("resolved secret is empty")
	}
	return b, nil
}

func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
