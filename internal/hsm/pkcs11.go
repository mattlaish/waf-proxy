package hsm

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
)

type nativeModule interface {
	Slots() ([]uint64, error)
	TokenLabel(uint64) (string, error)
	OpenSession(uint64) (nativeSession, error)
	Close() error
}

type nativeSession interface {
	Login([]byte) error
	FindPrivateKey(label string, id []byte) (uint64, error)
	Sign(key uint64, mechanism uint64, parameter []byte, data []byte) ([]byte, error)
	Close() error
}

var openNativePKCS11Module = openPlatformPKCS11Module
var validateModulePathForOpen = ValidateModulePath

type moduleEntry struct {
	module nativeModule
	refs   int
}

var moduleRegistry = struct {
	sync.Mutex
	m map[string]*moduleEntry
}{m: map[string]*moduleEntry{}}

func acquireModule(path string) (nativeModule, func(), error) {
	moduleRegistry.Lock()
	if e := moduleRegistry.m[path]; e != nil {
		e.refs++
		moduleRegistry.Unlock()
		return e.module, func() { releaseModule(path) }, nil
	}
	moduleRegistry.Unlock()
	m, err := openNativePKCS11Module(path)
	if err != nil {
		return nil, nil, err
	}
	moduleRegistry.Lock()
	if e := moduleRegistry.m[path]; e != nil {
		e.refs++
		moduleRegistry.Unlock()
		_ = m.Close()
		return e.module, func() { releaseModule(path) }, nil
	}
	moduleRegistry.m[path] = &moduleEntry{module: m, refs: 1}
	moduleRegistry.Unlock()
	return m, func() { releaseModule(path) }, nil
}

func releaseModule(path string) {
	moduleRegistry.Lock()
	e := moduleRegistry.m[path]
	if e == nil {
		moduleRegistry.Unlock()
		return
	}
	e.refs--
	if e.refs > 0 {
		moduleRegistry.Unlock()
		return
	}
	delete(moduleRegistry.m, path)
	moduleRegistry.Unlock()
	_ = e.module.Close()
}

type pkcs11Signer struct {
	mu      sync.Mutex
	public  crypto.PublicKey
	cfg     KeyConfig
	sink    AuditSink
	slot    uint64
	key     uint64
	session nativeSession
	release func()
	closed  bool
}

func openPKCS11Signer(ctx context.Context, runtime RuntimeConfig, cfg KeyConfig, public crypto.PublicKey, sink AuditSink) (Signer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cfg.ValidateStatic(); err != nil {
		return nil, err
	}
	if err := runtime.Validate(); err != nil {
		return nil, err
	}
	if err := validateModulePathForOpen(runtime, cfg.ModulePath); err != nil {
		return nil, err
	}
	m, release, err := acquireModule(cfg.ModulePath)
	if err != nil {
		return nil, errors.New("PKCS#11 provider initialization failed")
	}
	fail := func() { release() }
	slots, err := m.Slots()
	if err != nil {
		fail()
		return nil, errors.New("PKCS#11 slot enumeration failed")
	}
	slot, err := selectSlot(m, slots, cfg)
	if err != nil {
		fail()
		return nil, err
	}
	session, err := m.OpenSession(slot)
	if err != nil {
		fail()
		audit(sink, cfg, slot, "SESSION_OPEN", "FAILED")
		return nil, errors.New("PKCS#11 session open failed")
	}
	audit(sink, cfg, slot, "SESSION_OPEN", "SUCCESS")
	pin, err := ResolveSecret(cfg.PINSecretRef)
	if err != nil {
		_ = session.Close()
		fail()
		audit(sink, cfg, slot, "LOGIN", "FAILED")
		return nil, errors.New("PKCS#11 login secret unavailable")
	}
	err = session.Login(pin)
	ZeroBytes(pin)
	if err != nil {
		_ = session.Close()
		fail()
		audit(sink, cfg, slot, "LOGIN", "FAILED")
		return nil, errors.New("PKCS#11 login failed")
	}
	audit(sink, cfg, slot, "LOGIN", "SUCCESS")
	keyID, _ := hex.DecodeString(cfg.KeyID)
	key, err := session.FindPrivateKey(cfg.KeyLabel, keyID)
	if err != nil {
		_ = session.Close()
		fail()
		audit(sink, cfg, slot, "KEY_LOOKUP", "FAILED")
		return nil, errors.New("PKCS#11 private key lookup failed")
	}
	audit(sink, cfg, slot, "KEY_LOOKUP", "SUCCESS")
	return &pkcs11Signer{public: public, cfg: cfg, sink: sink, slot: slot, key: key, session: session, release: release}, nil
}

func selectSlot(m nativeModule, slots []uint64, cfg KeyConfig) (uint64, error) {
	var matches []uint64
	for _, slot := range slots {
		if cfg.SlotID != nil && slot != *cfg.SlotID {
			continue
		}
		if cfg.TokenLabel != "" {
			label, err := m.TokenLabel(slot)
			if err != nil {
				continue
			}
			if label != cfg.TokenLabel {
				continue
			}
		}
		matches = append(matches, slot)
	}
	if len(matches) != 1 {
		if len(matches) == 0 {
			return 0, errors.New("PKCS#11 slot/token selection matched no token")
		}
		return 0, errors.New("PKCS#11 slot/token selection is ambiguous")
	}
	return matches[0], nil
}

func (s *pkcs11Signer) Public() crypto.PublicKey { return s.public }
func (s *pkcs11Signer) SlotID() uint64           { return s.slot }
func (s *pkcs11Signer) KeyReference() string     { return s.cfg.KeyReference() }

func (s *pkcs11Signer) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, errors.New("HSM signer is closed")
	}
	mechanism, parameter, input, post, err := signPlan(s.public, digest, opts)
	if err != nil {
		audit(s.sink, s.cfg, s.slot, "SIGN", "FAILED")
		return nil, err
	}
	raw, err := s.session.Sign(s.key, mechanism, parameter, input)
	if err != nil {
		audit(s.sink, s.cfg, s.slot, "SIGN", "FAILED")
		return nil, errors.New("HSM signing failed")
	}
	sig, err := post(raw)
	if err != nil {
		audit(s.sink, s.cfg, s.slot, "SIGN", "FAILED")
		return nil, err
	}
	audit(s.sink, s.cfg, s.slot, "SIGN", "SUCCESS")
	return sig, nil
}

func (s *pkcs11Signer) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	err := s.session.Close()
	rel := s.release
	s.release = nil
	s.mu.Unlock()
	if rel != nil {
		rel()
	}
	audit(s.sink, s.cfg, s.slot, "SESSION_CLOSE", "SUCCESS")
	return err
}

func (s *pkcs11Signer) HealthCheck(ctx context.Context) HealthStatus {
	h := HealthStatus{Provider: s.cfg.Provider, Slot: fmt.Sprintf("%d", s.slot), KeyReference: s.cfg.KeyReference(), Status: "READY", Checks: []HealthCheck{{"provider", "READY"}, {"slot", "READY"}, {"token", "READY"}, {"key", "READY"}}}
	if err := ctx.Err(); err != nil {
		h.Status = "UNAVAILABLE"
		return h
	}
	if err := VerifySignerAssociation(s, s.public); err != nil {
		h.Status = "UNAVAILABLE"
		h.Checks[len(h.Checks)-1].Status = "UNAVAILABLE"
	}
	return h
}

const (
	ckmRSAPKCS = uint64(0x00000001)
	ckmRSAPSS  = uint64(0x0000000D)
	ckmECDSA   = uint64(0x00001041)
)

type pssParam struct{ HashAlg, MGF, SaltLen uint64 }

func encodePSS(p pssParam) []byte { return encodeNativeULongs(p.HashAlg, p.MGF, p.SaltLen) }

func signPlan(public crypto.PublicKey, digest []byte, opts crypto.SignerOpts) (uint64, []byte, []byte, func([]byte) ([]byte, error), error) {
	identity := func(b []byte) ([]byte, error) { return b, nil }
	hash := opts.HashFunc()
	if hash == 0 || len(digest) != hash.Size() {
		return 0, nil, nil, nil, errors.New("unsupported or invalid HSM digest")
	}
	switch pub := public.(type) {
	case *rsa.PublicKey:
		if pss, ok := opts.(*rsa.PSSOptions); ok {
			hashAlg, mgf, err := pkcs11Hash(hash)
			if err != nil {
				return 0, nil, nil, nil, err
			}
			salt := pss.SaltLength
			if salt == rsa.PSSSaltLengthEqualsHash {
				salt = hash.Size()
			}
			if salt == rsa.PSSSaltLengthAuto {
				emLen := (pub.N.BitLen() - 1 + 7) / 8
				salt = emLen - hash.Size() - 2
			}
			if salt < 0 {
				return 0, nil, nil, nil, errors.New("invalid RSA-PSS salt length")
			}
			return ckmRSAPSS, encodePSS(pssParam{hashAlg, mgf, uint64(salt)}), digest, identity, nil
		}
		prefix, err := digestInfoPrefix(hash)
		if err != nil {
			return 0, nil, nil, nil, err
		}
		in := append(append([]byte(nil), prefix...), digest...)
		return ckmRSAPKCS, nil, in, identity, nil
	case *ecdsa.PublicKey:
		size := (pub.Curve.Params().BitSize + 7) / 8
		post := func(raw []byte) ([]byte, error) {
			if len(raw) != 2*size {
				return nil, errors.New("unexpected PKCS#11 ECDSA signature length")
			}
			r := new(big.Int).SetBytes(raw[:size])
			ss := new(big.Int).SetBytes(raw[size:])
			return asn1.Marshal(struct{ R, S *big.Int }{r, ss})
		}
		return ckmECDSA, nil, digest, post, nil
	default:
		return 0, nil, nil, nil, fmt.Errorf("unsupported HSM key type %T", public)
	}
}

func digestInfoPrefix(h crypto.Hash) ([]byte, error) {
	switch h {
	case crypto.SHA1:
		return hex.DecodeString("3021300906052b0e03021a05000414")
	case crypto.SHA256:
		return hex.DecodeString("3031300d060960864801650304020105000420")
	case crypto.SHA384:
		return hex.DecodeString("3041300d060960864801650304020205000430")
	case crypto.SHA512:
		return hex.DecodeString("3051300d060960864801650304020305000440")
	default:
		return nil, errors.New("unsupported RSA hash for PKCS#11")
	}
}

func pkcs11Hash(h crypto.Hash) (uint64, uint64, error) {
	switch h {
	case crypto.SHA1:
		return 0x00000220, 0x00000001, nil
	case crypto.SHA256:
		return 0x00000250, 0x00000002, nil
	case crypto.SHA384:
		return 0x00000260, 0x00000003, nil
	case crypto.SHA512:
		return 0x00000270, 0x00000004, nil
	default:
		return 0, 0, errors.New("unsupported RSA-PSS hash for PKCS#11")
	}
}
