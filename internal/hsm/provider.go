package hsm

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"
)

type AuditEvent struct {
	Provider     string `json:"provider"`
	Slot         string `json:"slot"`
	KeyReference string `json:"key_reference"`
	Operation    string `json:"operation"`
	Result       string `json:"result"`
}

type AuditSink func(AuditEvent)

func audit(sink AuditSink, cfg KeyConfig, slot uint64, operation, result string) {
	if sink == nil {
		return
	}
	sink(AuditEvent{Provider: cfg.Provider, Slot: fmt.Sprintf("%d", slot), KeyReference: cfg.KeyReference(), Operation: operation, Result: result})
}

type HealthCheck struct {
	Component string `json:"component"`
	Status    string `json:"status"`
}

type HealthStatus struct {
	Provider     string        `json:"provider"`
	Slot         string        `json:"slot"`
	KeyReference string        `json:"key_reference"`
	Status       string        `json:"status"`
	Checks       []HealthCheck `json:"checks"`
}

type Signer interface {
	crypto.Signer
	Close() error
	HealthCheck(context.Context) HealthStatus
	SlotID() uint64
	KeyReference() string
}

type Provider interface {
	OpenSigner(context.Context, RuntimeConfig, KeyConfig, crypto.PublicKey, AuditSink) (Signer, error)
}

type PKCS11Provider struct{}

func (PKCS11Provider) OpenSigner(ctx context.Context, runtime RuntimeConfig, cfg KeyConfig, public crypto.PublicKey, sink AuditSink) (Signer, error) {
	return openPKCS11Signer(ctx, runtime, cfg, public, sink)
}

func LoadTLSCertificate(ctx context.Context, provider Provider, runtime RuntimeConfig, cfg KeyConfig, certPath string, sink AuditSink) (*tls.Certificate, Signer, error) {
	chain, leaf, err := readCertificateChain(certPath)
	if err != nil {
		return nil, nil, err
	}
	signer, err := provider.OpenSigner(ctx, runtime, cfg, leaf.PublicKey, sink)
	if err != nil {
		return nil, nil, err
	}
	cert := &tls.Certificate{Certificate: chain, PrivateKey: signer, Leaf: leaf}
	if err := VerifySignerAssociation(signer, leaf.PublicKey); err != nil {
		audit(sink, cfg, signer.SlotID(), "ASSOCIATION_VERIFY", "FAILED")
		_ = signer.Close()
		return nil, nil, errors.New("HSM private key does not match TLS certificate public key")
	}
	audit(sink, cfg, signer.SlotID(), "ASSOCIATION_VERIFY", "SUCCESS")
	return cert, signer, nil
}

func readCertificateChain(path string) ([][]byte, *x509.Certificate, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read TLS certificate: %w", err)
	}
	var chain [][]byte
	rest := b
	for len(rest) > 0 {
		block, next := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = next
		if block.Type != "CERTIFICATE" {
			continue
		}
		chain = append(chain, append([]byte(nil), block.Bytes...))
	}
	if len(chain) == 0 {
		return nil, nil, errors.New("TLS certificate file contains no CERTIFICATE PEM blocks")
	}
	leaf, err := x509.ParseCertificate(chain[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parse TLS leaf certificate: %w", err)
	}
	return chain, leaf, nil
}

func VerifySignerAssociation(signer crypto.Signer, public crypto.PublicKey) error {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return err
	}
	digest := sha256.Sum256(challenge)
	sig, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return err
	}
	switch pub := public.(type) {
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig)
	case *ecdsa.PublicKey:
		var rs struct{ R, S *big.Int }
		if _, err := asn1.Unmarshal(sig, &rs); err != nil || rs.R == nil || rs.S == nil || !ecdsa.Verify(pub, digest[:], rs.R, rs.S) {
			return errors.New("ECDSA signature verification failed")
		}
		return nil
	default:
		return fmt.Errorf("unsupported HSM TLS public key type %T", public)
	}
}

type AuditRing struct {
	mu     sync.Mutex
	cap    int
	events []AuditEvent
}

func NewAuditRing(capacity int) *AuditRing {
	if capacity <= 0 {
		capacity = 500
	}
	return &AuditRing{cap: capacity}
}

func (r *AuditRing) Add(e AuditEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	if len(r.events) > r.cap {
		r.events = r.events[len(r.events)-r.cap:]
	}
}

func (r *AuditRing) List() []AuditEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]AuditEvent, len(r.events))
	copy(out, r.events)
	return out
}
