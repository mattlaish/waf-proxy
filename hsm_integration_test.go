package main

import (
	"encoding/json"
	"strings"
	"testing"

	"waf-proxy/internal/hsm"
	"waf-proxy/internal/tlsfront"
)

func hsmSiteFixture() SiteConfig {
	slot := uint64(7)
	return SiteConfig{Name: "hsm", Listen: ":443", Hostnames: []string{"example.test"}, TLSCert: "/etc/waf/tls/cert.pem", TLSKeyProvider: hsm.KeyConfig{Provider: hsm.ProviderPKCS11, ModulePath: "/usr/lib/pkcs11/vendor.so", SlotID: &slot, TokenLabel: "prod-token", KeyLabel: "waf-tls", KeyID: "01", PINSecretRef: "file:/run/secrets/hsm-pin"}}
}

func TestHSMConfigForbidsFilesystemFallback(t *testing.T) {
	site := hsmSiteFixture()
	site.TLSKey = "/etc/waf/tls/key.pem"
	err := validateSiteTLSKeyProvider(site, tlsfront.AccelerationConfig{Mode: tlsfront.ModeGo}, true)
	if err == nil || !strings.Contains(err.Error(), "fallback is forbidden") {
		t.Fatalf("expected fallback rejection, got %v", err)
	}
}

func TestHSMConfigForbidsExternalTLSFrontend(t *testing.T) {
	site := hsmSiteFixture()
	err := validateSiteTLSKeyProvider(site, tlsfront.AccelerationConfig{Mode: tlsfront.ModeFrontend}, true)
	if err == nil || !strings.Contains(err.Error(), "mode=go") {
		t.Fatalf("expected frontend rejection, got %v", err)
	}
}

func TestHSMSecretReferenceRedactionAndPreservation(t *testing.T) {
	cur := Config{Sites: []SiteConfig{hsmSiteFixture()}}
	next := cur
	next.Sites = append([]SiteConfig(nil), cur.Sites...)
	next.Sites[0].TLSKeyProvider.PINSecretRef = ""
	preserveHSMSecretRefs(cur, &next)
	if next.Sites[0].TLSKeyProvider.PINSecretRef == "" {
		t.Fatal("matching key identity did not preserve secret reference")
	}
	next.Sites[0].TLSKeyProvider.KeyLabel = "other"
	next.Sites[0].TLSKeyProvider.PINSecretRef = ""
	preserveHSMSecretRefs(cur, &next)
	if next.Sites[0].TLSKeyProvider.PINSecretRef != "" {
		t.Fatal("secret reference transferred across key identity change")
	}
}

func TestHSMAuditJSONContainsNoSecretReference(t *testing.T) {
	slot := uint64(7)
	cfg := hsm.KeyConfig{Provider: hsm.ProviderPKCS11, SlotID: &slot, KeyLabel: "web", PINSecretRef: "file:/super/secret/pin"}
	e := hsm.AuditEvent{Provider: cfg.Provider, Slot: "7", KeyReference: cfg.KeyReference(), Operation: "SIGN", Result: "SUCCESS"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret") || strings.Contains(string(b), "pin") {
		t.Fatalf("audit leaked secret reference: %s", b)
	}
}

func TestHSMSecretReferenceRedactionDoesNotMutateSourceConfig(t *testing.T) {
	original := Config{Sites: []SiteConfig{hsmSiteFixture()}}
	redacted := redactHSMSecretRefs(original)
	if redacted.Sites[0].TLSKeyProvider.PINSecretRef != "" {
		t.Fatal("redacted config still contains secret reference")
	}
	if original.Sites[0].TLSKeyProvider.PINSecretRef == "" {
		t.Fatal("redaction mutated live/source config")
	}
}
