package main

import (
	"crypto/tls"
	"crypto/x509"
	"time"
)

// installTLSHandshakeObserver preserves the existing ClientHello policy hook
// while adding a per-connection VerifyConnection callback. VerifyConnection is
// documented by crypto/tls to run for successful full and resumed handshakes,
// so DidResume and negotiated protocol details are measured from authoritative
// TLS state instead of inferred from requests.
func installTLSHandshakeObserver(base *tls.Config, m *metrics, policy func(*tls.ClientHelloInfo) error) {
	if base == nil || m == nil {
		return
	}
	baseGetCertificate := base.GetCertificate
	baseVerifyConnection := base.VerifyConnection
	base.GetConfigForClient = func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
		started := time.Now()
		m.addTLSHandshakeAttempt()
		if policy != nil {
			if err := policy(hello); err != nil {
				m.addTLSPolicyRejected()
				return nil, err
			}
		}

		// The returned config is connection-local and is never mutated after
		// return. Session-ticket keys remain inherited from the parent config as
		// specified by crypto/tls, so manager rotation stays centralized.
		cfg := base.Clone()
		cfg.GetConfigForClient = nil
		selectedKeyAlgorithm := "unknown"
		if baseGetCertificate != nil {
			cfg.GetCertificate = func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
				cert, err := baseGetCertificate(chi)
				if err == nil && cert != nil {
					selectedKeyAlgorithm = tlsCertificateKeyAlgorithm(cert)
				}
				return cert, err
			}
		}
		cfg.VerifyConnection = func(cs tls.ConnectionState) error {
			if baseVerifyConnection != nil {
				if err := baseVerifyConnection(cs); err != nil {
					return err
				}
			}
			m.observeTLSHandshake(cs, selectedKeyAlgorithm, time.Since(started))
			return nil
		}
		return cfg, nil
	}
}

func tlsCertificateKeyAlgorithm(cert *tls.Certificate) string {
	if cert == nil {
		return "unknown"
	}
	leaf := cert.Leaf
	if leaf == nil && len(cert.Certificate) > 0 {
		parsed, err := x509.ParseCertificate(cert.Certificate[0])
		if err == nil {
			leaf = parsed
		}
	}
	if leaf == nil {
		return "unknown"
	}
	switch leaf.PublicKeyAlgorithm {
	case x509.RSA:
		return "rsa"
	case x509.ECDSA:
		return "ecdsa"
	case x509.Ed25519:
		return "ed25519"
	default:
		return "other"
	}
}
