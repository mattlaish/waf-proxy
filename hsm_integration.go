package main

import (
	"errors"
	"fmt"

	"waf-proxy/internal/hsm"
	"waf-proxy/internal/tlsfront"
)

func siteTLSEnabled(s SiteConfig) bool {
	return s.TLSCert != "" && (s.TLSKey != "" || s.TLSKeyProvider.Configured())
}

func validateSiteTLSKeyProvider(s SiteConfig, accel tlsfront.AccelerationConfig, full bool) error {
	provider := s.TLSKeyProvider.Configured()
	if provider {
		if err := s.TLSKeyProvider.ValidateStatic(); err != nil {
			return err
		}
		if s.TLSCert == "" {
			return errors.New("tls_cert is required with tls_key_provider")
		}
		if s.TLSKey != "" {
			return errors.New("tls_key must be empty when tls_key_provider is configured; filesystem fallback is forbidden")
		}
		if tlsfront.FrontendEnabled(accel) {
			return errors.New("HSM-backed TLS requires tls_acceleration.mode=go; external TLS frontend key fallback is forbidden")
		}
		return nil
	}
	if (s.TLSCert == "") != (s.TLSKey == "") {
		return errors.New("tls_cert and tls_key must both be set or both be empty")
	}
	if full && s.TLSCert != "" && s.TLSKey == "" {
		return errors.New("TLS private key is required when no HSM provider is configured")
	}
	return nil
}

func (s *server) recordHSMAudit(e hsm.AuditEvent) {
	if s == nil {
		return
	}
	if s.hsmAudit != nil {
		s.hsmAudit.Add(e)
	}
	if s.syslog != nil {
		s.syslog.forwardHSMAudit(e)
	}
	if s.log != nil {
		// Intentionally restricted fields: never log module paths, token labels,
		// PINs, PIN references, certificate paths, or raw provider errors.
		s.log.Info("HSM operation", "provider", e.Provider, "slot", e.Slot, "key_reference", e.KeyReference, "operation", e.Operation, "result", e.Result)
	}
}

func hsmStatusLabel(s SiteConfig) string {
	if !s.TLSKeyProvider.Configured() {
		return "filesystem"
	}
	return fmt.Sprintf("%s:%s", s.TLSKeyProvider.Provider, s.TLSKeyProvider.KeyReference())
}

func redactHSMSecretRefs(c Config) Config {
	sites := append([]SiteConfig(nil), c.Sites...)
	for i := range sites {
		sites[i].TLSKeyProvider.PINSecretRef = ""
	}
	c.Sites = sites
	return c
}

func preserveHSMSecretRefs(current Config, next *Config) {
	if next == nil {
		return
	}
	byName := make(map[string]SiteConfig, len(current.Sites))
	for _, site := range current.Sites {
		byName[site.Name] = site
	}
	for i := range next.Sites {
		n := &next.Sites[i]
		if !n.TLSKeyProvider.Configured() || n.TLSKeyProvider.PINSecretRef != "" {
			continue
		}
		old, ok := byName[n.Name]
		if !ok || !sameHSMKeyIdentity(old.TLSKeyProvider, n.TLSKeyProvider) {
			continue
		}
		n.TLSKeyProvider.PINSecretRef = old.TLSKeyProvider.PINSecretRef
	}
}

func sameHSMKeyIdentity(a, b hsm.KeyConfig) bool {
	if a.Provider != b.Provider || a.ModulePath != b.ModulePath || a.TokenLabel != b.TokenLabel || a.KeyLabel != b.KeyLabel || a.KeyID != b.KeyID {
		return false
	}
	if (a.SlotID == nil) != (b.SlotID == nil) {
		return false
	}
	return a.SlotID == nil || *a.SlotID == *b.SlotID
}
