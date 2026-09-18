package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"waf-proxy/internal/hsm"
)

type report struct {
	Format       string           `json:"format"`
	GeneratedAt  time.Time        `json:"generated_at"`
	Status       string           `json:"status"`
	Provider     string           `json:"provider"`
	Slot         string           `json:"slot"`
	KeyReference string           `json:"key_reference"`
	Health       hsm.HealthStatus `json:"health"`
	TLSHandshake string           `json:"tls_handshake"`
	Lifecycle    string           `json:"session_lifecycle"`
	VendorClass  string           `json:"vendor_class"`
	Note         string           `json:"note,omitempty"`
}

func main() {
	var module, token, keyLabel, keyID, certPath, allowedDir, outPath, qualificationClass string
	var slotText string
	flag.StringVar(&module, "module", "", "absolute PKCS#11 module path")
	flag.StringVar(&slotText, "slot", "", "exact numeric PKCS#11 slot ID")
	flag.StringVar(&token, "token-label", "", "exact token label")
	flag.StringVar(&keyLabel, "key-label", "", "exact private-key label")
	flag.StringVar(&keyID, "key-id", "", "exact private-key CKA_ID as hex")
	flag.StringVar(&certPath, "cert", "", "PEM TLS certificate chain matching the HSM key")
	flag.StringVar(&allowedDir, "allow-module-dir", "", "approved module directory")
	flag.StringVar(&outPath, "out", "", "qualification report path")
	flag.StringVar(&qualificationClass, "qualification-class", "pkcs11_unspecified", "evidence class: softhsm, real_vendor_hsm, or pkcs11_unspecified")
	flag.Parse()

	switch qualificationClass {
	case "softhsm", "real_vendor_hsm", "pkcs11_unspecified":
	default:
		fatal("--qualification-class must be softhsm, real_vendor_hsm, or pkcs11_unspecified")
	}

	secretRef := os.Getenv("WAF_HSM_PIN_SECRET_REF")
	if secretRef == "" {
		fatal("WAF_HSM_PIN_SECRET_REF is required; the PIN itself must never be passed on the command line")
	}
	if module == "" || certPath == "" || outPath == "" {
		fatal("--module, --cert and --out are required")
	}
	if allowedDir == "" {
		allowedDir = filepath.Dir(module)
	}
	cfg := hsm.KeyConfig{Provider: hsm.ProviderPKCS11, ModulePath: module, TokenLabel: token, KeyLabel: keyLabel, KeyID: keyID, PINSecretRef: secretRef}
	if slotText != "" {
		n, err := strconv.ParseUint(slotText, 10, 64)
		if err != nil {
			fatal("invalid --slot")
		}
		cfg.SlotID = &n
	}
	runtimeCfg := hsm.RuntimeConfig{AllowedModuleDirs: []string{filepath.Clean(allowedDir)}}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var events []hsm.AuditEvent
	cert, signer, err := hsm.LoadTLSCertificate(ctx, hsm.PKCS11Provider{}, runtimeCfg, cfg, certPath, func(e hsm.AuditEvent) { events = append(events, e) })
	if err != nil {
		writeFailure(outPath, cfg, qualificationClass, "provider/key/certificate qualification failed")
		fatal("HSM qualification failed")
	}
	defer signer.Close()
	health := signer.HealthCheck(ctx)
	hs := "PASS"
	if health.Status != "READY" {
		hs = "FAIL"
	}
	handshake := "NOT_RUN"
	if hs == "PASS" {
		handshake = "PASS"
		if err := tlsHandshake(cert); err != nil {
			handshake = "FAIL"
			hs = "FAIL"
		}
	}
	lifecycle := "PASS"
	if err := signer.Close(); err != nil {
		lifecycle = "FAIL"
		hs = "FAIL"
	}
	r := report{Format: "waf-proxy-hsm-qualification-v1", GeneratedAt: time.Now().UTC(), Status: hs, Provider: cfg.Provider, Slot: fmt.Sprintf("%d", signer.SlotID()), KeyReference: signer.KeyReference(), Health: health, TLSHandshake: handshake, Lifecycle: lifecycle, VendorClass: qualificationClass, Note: "Report intentionally excludes PIN, secret reference, module path, and token label."}
	if err := writeReport(outPath, r); err != nil {
		fatal("write qualification report: " + err.Error())
	}
	_ = events // events are deliberately not embedded; runtime audit carries only approved fields.
	if hs != "PASS" {
		os.Exit(1)
	}
}

func tlsHandshake(cert *tls.Certificate) error {
	if cert == nil {
		return errors.New("nil certificate")
	}
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	srv := tls.Server(serverConn, &tls.Config{Certificates: []tls.Certificate{*cert}, MinVersion: tls.VersionTLS12})
	cli := tls.Client(clientConn, &tls.Config{ServerName: "hsm-qualification.invalid", InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}) // in-memory signer exercise only; certificate association was already verified explicitly
	errs := make(chan error, 2)
	go func() { errs <- srv.Handshake() }()
	go func() { errs <- cli.Handshake() }()
	if err := <-errs; err != nil {
		return err
	}
	return <-errs
}

func writeFailure(path string, cfg hsm.KeyConfig, qualificationClass, note string) {
	if path == "" {
		return
	}
	slot := ""
	if cfg.SlotID != nil {
		slot = fmt.Sprintf("%d", *cfg.SlotID)
	}
	_ = writeReport(path, report{Format: "waf-proxy-hsm-qualification-v1", GeneratedAt: time.Now().UTC(), Status: "FAIL", Provider: cfg.Provider, Slot: slot, KeyReference: cfg.KeyReference(), TLSHandshake: "NOT_RUN", Lifecycle: "NOT_RUN", VendorClass: qualificationClass, Note: note + "; secret-bearing fields are omitted."})
}
func writeReport(path string, r report) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}
func fatal(msg string) { fmt.Fprintln(os.Stderr, "hsmqualify:", strings.TrimSpace(msg)); os.Exit(2) }
