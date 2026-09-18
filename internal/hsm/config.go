package hsm

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ProviderPKCS11 = "pkcs11"

type RuntimeConfig struct {
	AllowedModuleDirs []string `json:"allowed_module_dirs,omitempty"`
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{AllowedModuleDirs: []string{"/usr/lib", "/usr/lib64", "/usr/local/lib", "/usr/local/lib64", "/opt"}}
}

func (c RuntimeConfig) EffectiveAllowedModuleDirs() []string {
	dirs := c.AllowedModuleDirs
	if len(dirs) == 0 {
		dirs = DefaultRuntimeConfig().AllowedModuleDirs
	}
	out := make([]string, 0, len(dirs))
	seen := map[string]bool{}
	for _, d := range dirs {
		if !filepath.IsAbs(d) {
			continue
		}
		d = filepath.Clean(d)
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

func (c RuntimeConfig) Validate() error {
	if len(c.EffectiveAllowedModuleDirs()) == 0 {
		return errors.New("hsm.allowed_module_dirs must contain at least one absolute directory")
	}
	for _, d := range c.AllowedModuleDirs {
		if !filepath.IsAbs(d) || filepath.Clean(d) != d {
			return fmt.Errorf("hsm.allowed_module_dirs entry %q must be an absolute clean path", d)
		}
	}
	return nil
}

type KeyConfig struct {
	Provider     string  `json:"provider,omitempty"`
	ModulePath   string  `json:"module_path,omitempty"`
	SlotID       *uint64 `json:"slot_id,omitempty"`
	TokenLabel   string  `json:"token_label,omitempty"`
	KeyLabel     string  `json:"key_label,omitempty"`
	KeyID        string  `json:"key_id,omitempty"` // hex, exact CKA_ID match
	PINSecretRef string  `json:"pin_secret_ref,omitempty"`
}

func (c KeyConfig) Configured() bool { return strings.TrimSpace(c.Provider) != "" }

func (c KeyConfig) KeyReference() string {
	switch {
	case c.KeyLabel != "" && c.KeyID != "":
		return "label:" + c.KeyLabel + "+id:" + strings.ToLower(c.KeyID)
	case c.KeyLabel != "":
		return "label:" + c.KeyLabel
	default:
		return "id:" + strings.ToLower(c.KeyID)
	}
}

func (c KeyConfig) ValidateStatic() error {
	if !c.Configured() {
		return nil
	}
	if c.Provider != ProviderPKCS11 {
		return fmt.Errorf("tls_key_provider.provider must be %q", ProviderPKCS11)
	}
	if !filepath.IsAbs(c.ModulePath) || filepath.Clean(c.ModulePath) != c.ModulePath {
		return errors.New("tls_key_provider.module_path must be an absolute clean path")
	}
	if c.SlotID == nil && c.TokenLabel == "" {
		return errors.New("tls_key_provider requires exact slot_id and/or token_label")
	}
	if strings.TrimSpace(c.TokenLabel) != c.TokenLabel || strings.ContainsRune(c.TokenLabel, '\x00') {
		return errors.New("tls_key_provider.token_label must not contain surrounding whitespace or NUL")
	}
	if c.KeyLabel == "" && c.KeyID == "" {
		return errors.New("tls_key_provider requires exact key_label and/or key_id")
	}
	if strings.TrimSpace(c.KeyLabel) != c.KeyLabel || strings.ContainsRune(c.KeyLabel, '\x00') {
		return errors.New("tls_key_provider.key_label must not contain surrounding whitespace or NUL")
	}
	if c.KeyID != "" {
		b, err := hex.DecodeString(c.KeyID)
		if err != nil || len(b) == 0 || len(b) > 128 {
			return errors.New("tls_key_provider.key_id must be 1-128 bytes of hexadecimal")
		}
	}
	if err := ValidateSecretRef(c.PINSecretRef); err != nil {
		return fmt.Errorf("tls_key_provider.pin_secret_ref: %w", err)
	}
	return nil
}

func ValidateModulePath(runtime RuntimeConfig, path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("PKCS#11 module path must be absolute and clean")
	}
	allowed := false
	for _, base := range runtime.EffectiveAllowedModuleDirs() {
		rel, err := filepath.Rel(base, path)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			allowed = true
			break
		}
	}
	if !allowed {
		return errors.New("PKCS#11 module path is outside hsm.allowed_module_dirs")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("PKCS#11 module unavailable: %w", err)
	}
	if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
		return errors.New("PKCS#11 module must be a regular non-symlink file")
	}
	if st.Mode().Perm()&0o022 != 0 {
		return errors.New("PKCS#11 module must not be group/world writable")
	}
	if err := validateModuleOwnership(path, runtime.EffectiveAllowedModuleDirs()); err != nil {
		return err
	}
	return nil
}
