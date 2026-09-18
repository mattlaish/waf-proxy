//go:build linux

package hsm

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func validateModuleOwnership(path string, allowed []string) error {
	base := ""
	for _, candidate := range allowed {
		rel, err := filepath.Rel(candidate, path)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			if len(candidate) > len(base) {
				base = candidate
			}
		}
	}
	if base == "" {
		return errors.New("PKCS#11 module path is outside approved directories")
	}
	cur := path
	for {
		st, err := os.Lstat(cur)
		if err != nil {
			return errors.New("PKCS#11 module ownership path is unavailable")
		}
		stat, ok := st.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			return errors.New("PKCS#11 module and approved parent path must be root-owned")
		}
		if cur != path {
			if !st.IsDir() {
				return errors.New("PKCS#11 module parent path must be a directory")
			}
			if st.Mode().Perm()&0o022 != 0 {
				return errors.New("PKCS#11 module parent path must not be group/world writable")
			}
		}
		if cur == base {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur || len(parent) < len(base) {
			return errors.New("PKCS#11 module ownership path escaped approved directory")
		}
		cur = parent
	}
	return nil
}
