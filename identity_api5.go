package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	apiIdentityModeDetect  = "DETECT"
	apiIdentityModeEnforce = "ENFORCE"

	apiIdentityMaxViolations     = 2048
	apiIdentityMaxJWKSBytes      = 1 << 20
	apiIdentityMaxTokenBytes     = 32 << 10
	apiIdentityDefaultJWKSMin    = 5 * time.Minute
	apiIdentityDefaultJWKSMax    = 24 * time.Hour
	apiIdentityKidMissRefreshMin = 30 * time.Second
)

// JWTIssuerConfig is operator-authored trust configuration. It contains no
// private key or token material. JWKS keys are fetched into ephemeral memory
// and are deliberately not persisted.
type JWTIssuerConfig struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Issuer       string    `json:"issuer"`
	JWKSURL      string    `json:"jwks_url"`
	Audiences    []string  `json:"audiences"`
	AllowedAlgs  []string  `json:"allowed_algs"`
	ClockSkewSec int       `json:"clock_skew_sec,omitempty"`
	CacheTTLSec  int       `json:"cache_ttl_sec,omitempty"`
	SubjectClaim string    `json:"subject_claim,omitempty"`
	TenantClaim  string    `json:"tenant_claim,omitempty"`
	ClientClaim  string    `json:"client_id_claim,omitempty"`
	RolesClaim   string    `json:"roles_claim,omitempty"`
	ScopesClaim  string    `json:"scopes_claim,omitempty"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedBy    string    `json:"created_by,omitempty"`
	UpdatedBy    string    `json:"updated_by,omitempty"`
}

// APIIdentityPolicy binds one discovered operation to one trusted issuer and
// deterministic authorization requirements. A new policy always starts in
// DETECT and must be explicitly promoted to ENFORCE.
type APIIdentityPolicy struct {
	OperationID      string    `json:"operation_id"`
	IssuerID         string    `json:"issuer_id"`
	Mode             string    `json:"mode"`
	RequireAuth      bool      `json:"require_authenticated"`
	RequiredRoles    []string  `json:"required_roles,omitempty"`
	RequiredScopes   []string  `json:"required_scopes,omitempty"`
	RoleMatch        string    `json:"role_match,omitempty"`  // all | any
	ScopeMatch       string    `json:"scope_match,omitempty"` // all | any
	RequiredTenant   string    `json:"required_tenant,omitempty"`
	AllowedClientIDs []string  `json:"allowed_client_ids,omitempty"`
	UnauthorizedCode int       `json:"unauthorized_status,omitempty"`
	ForbiddenCode    int       `json:"forbidden_status,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedBy        string    `json:"created_by,omitempty"`
	UpdatedBy        string    `json:"updated_by,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}

// VerifiedAPIIdentity exists only in request memory after cryptographic
// verification. Raw tokens and unverified claims never enter this context.
type VerifiedAPIIdentity struct {
	Issuer    string    `json:"issuer"`
	Subject   string    `json:"subject,omitempty"`
	TenantID  string    `json:"tenant_id,omitempty"`
	ClientID  string    `json:"client_id,omitempty"`
	Roles     []string  `json:"roles,omitempty"`
	Scopes    []string  `json:"scopes,omitempty"`
	Audience  []string  `json:"audience,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	KeyID     string    `json:"key_id,omitempty"`
	Algorithm string    `json:"algorithm"`
}

type APIIdentityViolation struct {
	ID          string    `json:"id"`
	Time        time.Time `json:"time"`
	OperationID string    `json:"operation_id"`
	Mode        string    `json:"mode"`
	Action      string    `json:"action"`
	Type        string    `json:"type"`
	IssuerID    string    `json:"issuer_id,omitempty"`
	SubjectHash string    `json:"subject_hash,omitempty"`
	Expected    string    `json:"expected,omitempty"`
	Observed    string    `json:"observed,omitempty"`
}

type apiIdentityStateFile struct {
	Version    int                          `json:"version"`
	Saved      time.Time                    `json:"saved"`
	Issuers    map[string]JWTIssuerConfig   `json:"issuers"`
	Policies   map[string]APIIdentityPolicy `json:"policies"`
	Violations []APIIdentityViolation       `json:"violations,omitempty"`
}

type apiIdentityContextKey struct{}

func verifiedAPIIdentity(r *http.Request) (VerifiedAPIIdentity, bool) {
	if r == nil {
		return VerifiedAPIIdentity{}, false
	}
	v, ok := r.Context().Value(apiIdentityContextKey{}).(VerifiedAPIIdentity)
	return v, ok
}

type jwksPublicKey struct {
	Kid string
	Alg string
	Key any
}

type jwksKeySet struct {
	Keys      []jwksPublicKey
	FetchedAt time.Time
	ExpiresAt time.Time
}

type jwtIssuerRuntime struct {
	config             JWTIssuerConfig
	keys               atomic.Pointer[jwksKeySet]
	refreshMu          sync.Mutex
	lastKidMissRefresh time.Time
}

type apiIdentityRuntimePolicy struct {
	Policy APIIdentityPolicy
	Issuer *jwtIssuerRuntime
}

type apiIdentityRuntimeSnapshot struct {
	ByOperation map[string]apiIdentityRuntimePolicy
}

type apiIdentityStore struct {
	mu         sync.RWMutex
	issuers    map[string]JWTIssuerConfig
	policies   map[string]APIIdentityPolicy
	violations []APIIdentityViolation
	seq        atomic.Uint64
	runtimes   map[string]*jwtIssuerRuntime
	runtime    atomic.Pointer[apiIdentityRuntimeSnapshot]
	fetchJWKS  func(context.Context, string) ([]byte, error)
}

func newAPIIdentityStore() *apiIdentityStore {
	s := &apiIdentityStore{
		issuers:   map[string]JWTIssuerConfig{},
		policies:  map[string]APIIdentityPolicy{},
		runtimes:  map[string]*jwtIssuerRuntime{},
		fetchJWKS: fetchJWKSHTTPS,
	}
	s.runtime.Store(&apiIdentityRuntimeSnapshot{ByOperation: map[string]apiIdentityRuntimePolicy{}})
	return s
}

func normalizeIdentityMode(v string) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case apiIdentityModeDetect, apiIdentityModeEnforce:
		return v, nil
	case "":
		return apiIdentityModeDetect, nil
	default:
		return "", fmt.Errorf("identity mode must be DETECT or ENFORCE")
	}
}

func normalizeMatchMode(v string) (string, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "all", nil
	}
	if v != "all" && v != "any" {
		return "", fmt.Errorf("match mode must be all or any")
	}
	return v, nil
}

func normalizeJWTAlg(v string) (string, bool) {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512", "EDDSA":
		return v, true
	default:
		return "", false
	}
}

func normalizeIssuerConfig(in JWTIssuerConfig, actor string, existing *JWTIssuerConfig) (JWTIssuerConfig, error) {
	now := time.Now().UTC()
	in.ID = strings.TrimSpace(in.ID)
	in.Name = strings.TrimSpace(in.Name)
	in.Issuer = strings.TrimSpace(in.Issuer)
	in.JWKSURL = strings.TrimSpace(in.JWKSURL)
	if in.ID == "" || in.Issuer == "" || in.JWKSURL == "" {
		return JWTIssuerConfig{}, errors.New("issuer id, issuer, and jwks_url are required")
	}
	if len(in.ID) > 128 || len(in.Issuer) > 2048 || len(in.JWKSURL) > 4096 {
		return JWTIssuerConfig{}, errors.New("issuer configuration exceeds size limits")
	}
	u, err := url.Parse(in.JWKSURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return JWTIssuerConfig{}, errors.New("jwks_url must be an absolute https URL without userinfo or fragment")
	}
	if len(in.Audiences) == 0 {
		return JWTIssuerConfig{}, errors.New("at least one accepted audience is required")
	}
	in.Audiences, err = normalizePolicyStringSet(in.Audiences, 64, 512, "audiences")
	if err != nil {
		return JWTIssuerConfig{}, err
	}
	if len(in.Audiences) == 0 {
		return JWTIssuerConfig{}, errors.New("at least one accepted audience is required")
	}
	if len(in.AllowedAlgs) == 0 {
		in.AllowedAlgs = []string{"RS256"}
	}
	algs := make([]string, 0, len(in.AllowedAlgs))
	seen := map[string]struct{}{}
	for _, raw := range in.AllowedAlgs {
		alg, ok := normalizeJWTAlg(raw)
		if !ok {
			return JWTIssuerConfig{}, fmt.Errorf("unsupported JWT algorithm %q", raw)
		}
		if _, ok := seen[alg]; ok {
			continue
		}
		seen[alg] = struct{}{}
		algs = append(algs, alg)
	}
	sort.Strings(algs)
	in.AllowedAlgs = algs
	if in.ClockSkewSec < 0 || in.ClockSkewSec > 300 {
		return JWTIssuerConfig{}, errors.New("clock_skew_sec must be between 0 and 300")
	}
	if in.CacheTTLSec == 0 {
		in.CacheTTLSec = int(apiIdentityDefaultJWKSMin / time.Second)
	}
	if in.CacheTTLSec < int(apiIdentityDefaultJWKSMin/time.Second) || in.CacheTTLSec > int(apiIdentityDefaultJWKSMax/time.Second) {
		return JWTIssuerConfig{}, fmt.Errorf("cache_ttl_sec must be between %d and %d", int(apiIdentityDefaultJWKSMin/time.Second), int(apiIdentityDefaultJWKSMax/time.Second))
	}
	if in.SubjectClaim == "" {
		in.SubjectClaim = "sub"
	}
	if in.ClientClaim == "" {
		in.ClientClaim = "client_id"
	}
	if in.RolesClaim == "" {
		in.RolesClaim = "roles"
	}
	if in.ScopesClaim == "" {
		in.ScopesClaim = "scope"
	}
	for _, p := range []string{in.SubjectClaim, in.TenantClaim, in.ClientClaim, in.RolesClaim, in.ScopesClaim} {
		if len(p) > 256 || strings.ContainsAny(p, "[]") {
			return JWTIssuerConfig{}, errors.New("claim paths must be short dotted object paths")
		}
	}
	if existing != nil {
		in.CreatedAt = existing.CreatedAt
		in.CreatedBy = existing.CreatedBy
	} else {
		in.CreatedAt = now
		in.CreatedBy = actor
	}
	in.UpdatedAt = now
	in.UpdatedBy = actor
	return in, nil
}

func normalizeStringSet(in []string, maxItems, maxLen int) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || len(v) > maxLen {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
		if len(out) >= maxItems {
			break
		}
	}
	sort.Strings(out)
	return out
}

func normalizePolicyStringSet(in []string, maxItems, maxLen int, field string) ([]string, error) {
	if len(in) > maxItems {
		return nil, fmt.Errorf("%s exceeds %d items", field, maxItems)
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		v := strings.TrimSpace(raw)
		if v == "" {
			return nil, fmt.Errorf("%s contains an empty value", field)
		}
		if len(v) > maxLen {
			return nil, fmt.Errorf("%s value exceeds %d bytes", field, maxLen)
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

func (s *apiIdentityStore) upsertIssuer(in JWTIssuerConfig, actor string) (JWTIssuerConfig, error) {
	if s == nil {
		return JWTIssuerConfig{}, errors.New("identity store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var existing *JWTIssuerConfig
	if old, ok := s.issuers[strings.TrimSpace(in.ID)]; ok {
		copyOld := old
		existing = &copyOld
	}
	normalized, err := normalizeIssuerConfig(in, actor, existing)
	if err != nil {
		return JWTIssuerConfig{}, err
	}
	old, hadOld := s.issuers[normalized.ID]
	if !normalized.Enabled {
		for _, p := range s.policies {
			if p.IssuerID == normalized.ID {
				return JWTIssuerConfig{}, errors.New("issuer cannot be disabled while identity policies reference it")
			}
		}
	}
	s.issuers[normalized.ID] = normalized
	if hadOld && !sameIssuerRuntimeConfig(old, normalized) {
		delete(s.runtimes, normalized.ID)
		for opID, p := range s.policies {
			if p.IssuerID == normalized.ID && p.Mode == apiIdentityModeEnforce {
				p.Mode = apiIdentityModeDetect
				p.UpdatedAt = time.Now().UTC()
				p.UpdatedBy = actor
				p.Reason = "issuer trust semantics changed; returned to DETECT"
				s.policies[opID] = p
			}
		}
	}
	s.rebuildRuntimeLocked()
	return normalized, nil
}

func (s *apiIdentityStore) listIssuers() []JWTIssuerConfig {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]JWTIssuerConfig, 0, len(s.issuers))
	for _, v := range s.issuers {
		v.Audiences = append([]string(nil), v.Audiences...)
		v.AllowedAlgs = append([]string(nil), v.AllowedAlgs...)
		out = append(out, v)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sameIdentityPolicySemantics(a, b APIIdentityPolicy) bool {
	return a.OperationID == b.OperationID && a.IssuerID == b.IssuerID &&
		a.RequireAuth == b.RequireAuth && a.RoleMatch == b.RoleMatch && a.ScopeMatch == b.ScopeMatch &&
		a.RequiredTenant == b.RequiredTenant &&
		strings.Join(a.RequiredRoles, "\x00") == strings.Join(b.RequiredRoles, "\x00") &&
		strings.Join(a.RequiredScopes, "\x00") == strings.Join(b.RequiredScopes, "\x00") &&
		strings.Join(a.AllowedClientIDs, "\x00") == strings.Join(b.AllowedClientIDs, "\x00")
}

func normalizeIdentityPolicy(in APIIdentityPolicy, actor string, existing *APIIdentityPolicy) (APIIdentityPolicy, error) {
	now := time.Now().UTC()
	in.OperationID = strings.TrimSpace(in.OperationID)
	in.IssuerID = strings.TrimSpace(in.IssuerID)
	if in.OperationID == "" || in.IssuerID == "" {
		return APIIdentityPolicy{}, errors.New("operation_id and issuer_id are required")
	}
	mode, err := normalizeIdentityMode(in.Mode)
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	if existing == nil {
		if mode == apiIdentityModeEnforce {
			return APIIdentityPolicy{}, errors.New("new identity policy must start in DETECT")
		}
		in.Mode = apiIdentityModeDetect
	} else {
		// Mode changes are intentionally isolated to setPolicyMode so an upsert
		// cannot bypass the DETECT -> ENFORCE review transition.
		in.Mode = existing.Mode
	}
	if !in.RequireAuth {
		in.RequireAuth = true
	}
	in.RequiredRoles, err = normalizePolicyStringSet(in.RequiredRoles, 64, 256, "required_roles")
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	in.RequiredScopes, err = normalizePolicyStringSet(in.RequiredScopes, 128, 256, "required_scopes")
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	in.AllowedClientIDs, err = normalizePolicyStringSet(in.AllowedClientIDs, 64, 512, "allowed_client_ids")
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	in.RoleMatch, err = normalizeMatchMode(in.RoleMatch)
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	in.ScopeMatch, err = normalizeMatchMode(in.ScopeMatch)
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	if in.UnauthorizedCode == 0 {
		in.UnauthorizedCode = http.StatusUnauthorized
	}
	if in.ForbiddenCode == 0 {
		in.ForbiddenCode = http.StatusForbidden
	}
	if in.UnauthorizedCode != http.StatusUnauthorized || in.ForbiddenCode != http.StatusForbidden {
		return APIIdentityPolicy{}, errors.New("identity policy status codes are fixed at 401/403")
	}
	if existing != nil {
		in.CreatedAt = existing.CreatedAt
		in.CreatedBy = existing.CreatedBy
	} else {
		in.CreatedAt = now
		in.CreatedBy = actor
	}
	in.UpdatedAt = now
	in.UpdatedBy = actor
	return in, nil
}

func (s *apiIdentityStore) upsertPolicy(in APIIdentityPolicy, actor string, opExists func(string) bool) (APIIdentityPolicy, error) {
	if s == nil {
		return APIIdentityPolicy{}, errors.New("identity store unavailable")
	}
	if opExists != nil && !opExists(strings.TrimSpace(in.OperationID)) {
		return APIIdentityPolicy{}, errors.New("matching discovered API operation is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	issuer, ok := s.issuers[strings.TrimSpace(in.IssuerID)]
	if !ok || !issuer.Enabled {
		return APIIdentityPolicy{}, errors.New("enabled JWT issuer is required")
	}
	var existing *APIIdentityPolicy
	if old, ok := s.policies[strings.TrimSpace(in.OperationID)]; ok {
		copyOld := old
		existing = &copyOld
		if strings.TrimSpace(in.Mode) == "" {
			in.Mode = old.Mode
		}
	}
	normalized, err := normalizeIdentityPolicy(in, actor, existing)
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	if existing != nil && !sameIdentityPolicySemantics(*existing, normalized) {
		normalized.Mode = apiIdentityModeDetect
		normalized.Reason = "policy semantics changed; returned to DETECT"
	}
	s.policies[normalized.OperationID] = normalized
	s.rebuildRuntimeLocked()
	return normalized, nil
}

func (s *apiIdentityStore) setPolicyMode(operationID, mode, actor, reason string) (APIIdentityPolicy, error) {
	if s == nil {
		return APIIdentityPolicy{}, errors.New("identity store unavailable")
	}
	mode, err := normalizeIdentityMode(mode)
	if err != nil {
		return APIIdentityPolicy{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.policies[strings.TrimSpace(operationID)]
	if !ok {
		return APIIdentityPolicy{}, errors.New("identity policy not found")
	}
	if p.Mode == mode {
		return p, nil
	}
	if p.Mode == apiIdentityModeEnforce && mode != apiIdentityModeDetect {
		return APIIdentityPolicy{}, errors.New("ENFORCE may only return to DETECT")
	}
	if p.Mode == apiIdentityModeDetect && mode != apiIdentityModeEnforce {
		return APIIdentityPolicy{}, errors.New("DETECT may only promote to ENFORCE")
	}
	p.Mode = mode
	p.UpdatedAt = time.Now().UTC()
	p.UpdatedBy = actor
	p.Reason = strings.TrimSpace(reason)
	s.policies[p.OperationID] = p
	s.rebuildRuntimeLocked()
	return p, nil
}

func (s *apiIdentityStore) deletePolicy(operationID string) error {
	if s == nil {
		return errors.New("identity store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[operationID]; !ok {
		return errors.New("identity policy not found")
	}
	delete(s.policies, operationID)
	s.rebuildRuntimeLocked()
	return nil
}

func (s *apiIdentityStore) listPolicies() []APIIdentityPolicy {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]APIIdentityPolicy, 0, len(s.policies))
	for _, v := range s.policies {
		v.RequiredRoles = append([]string(nil), v.RequiredRoles...)
		v.RequiredScopes = append([]string(nil), v.RequiredScopes...)
		v.AllowedClientIDs = append([]string(nil), v.AllowedClientIDs...)
		out = append(out, v)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].OperationID < out[j].OperationID })
	return out
}

func sameIssuerRuntimeConfig(a, b JWTIssuerConfig) bool {
	return a.ID == b.ID && a.Issuer == b.Issuer && a.JWKSURL == b.JWKSURL &&
		a.ClockSkewSec == b.ClockSkewSec && a.CacheTTLSec == b.CacheTTLSec &&
		a.SubjectClaim == b.SubjectClaim && a.TenantClaim == b.TenantClaim &&
		a.ClientClaim == b.ClientClaim && a.RolesClaim == b.RolesClaim &&
		a.ScopesClaim == b.ScopesClaim &&
		strings.Join(a.Audiences, "\x00") == strings.Join(b.Audiences, "\x00") &&
		strings.Join(a.AllowedAlgs, "\x00") == strings.Join(b.AllowedAlgs, "\x00")
}

func (s *apiIdentityStore) rebuildRuntimeLocked() {
	// Keep an ephemeral verifier/cache runtime for every enabled issuer so an
	// operator can validate/refresh JWKS before binding the issuer to traffic.
	active := make(map[string]*jwtIssuerRuntime, len(s.issuers))
	for id, issuer := range s.issuers {
		if !issuer.Enabled {
			continue
		}
		rt := s.runtimes[id]
		if rt == nil || !sameIssuerRuntimeConfig(rt.config, issuer) {
			rt = &jwtIssuerRuntime{config: issuer}
		}
		active[id] = rt
	}
	s.runtimes = active

	next := &apiIdentityRuntimeSnapshot{ByOperation: make(map[string]apiIdentityRuntimePolicy, len(s.policies))}
	for opID, policy := range s.policies {
		// Keep the policy visible even if the issuer state is corrupt/missing.
		// ENFORCE must fail closed rather than disappear from the runtime map.
		next.ByOperation[opID] = apiIdentityRuntimePolicy{Policy: policy, Issuer: s.runtimes[policy.IssuerID]}
	}
	s.runtime.Store(next)
}

func (s *apiIdentityStore) runtimePolicy(operationID string) (apiIdentityRuntimePolicy, bool) {
	if s == nil {
		return apiIdentityRuntimePolicy{}, false
	}
	snap := s.runtime.Load()
	if snap == nil {
		return apiIdentityRuntimePolicy{}, false
	}
	v, ok := snap.ByOperation[operationID]
	return v, ok
}

func fetchJWKSHTTPS(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, errors.New("invalid JWKS URL")
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many JWKS redirects")
			}
			if req.URL.Scheme != "https" || req.URL.User != nil || req.URL.Fragment != "" {
				return errors.New("JWKS redirect must remain HTTPS")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS HTTP status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, apiIdentityMaxJWKSBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > apiIdentityMaxJWKSBytes {
		return nil, errors.New("JWKS document exceeds size limit")
	}
	return b, nil
}

func parseJWKS(data []byte) ([]jwksPublicKey, error) {
	var doc struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if len(data) == 0 || len(data) > apiIdentityMaxJWKSBytes {
		return nil, errors.New("invalid JWKS size")
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid JWKS JSON: %w", err)
	}
	if len(doc.Keys) == 0 || len(doc.Keys) > 256 {
		return nil, errors.New("JWKS must contain between 1 and 256 keys")
	}
	out := make([]jwksPublicKey, 0, len(doc.Keys))
	for _, raw := range doc.Keys {
		var j struct {
			Kty    string   `json:"kty"`
			Kid    string   `json:"kid"`
			Use    string   `json:"use"`
			Alg    string   `json:"alg"`
			KeyOps []string `json:"key_ops"`
			N      string   `json:"n"`
			E      string   `json:"e"`
			Crv    string   `json:"crv"`
			X      string   `json:"x"`
			Y      string   `json:"y"`
		}
		if err := json.Unmarshal(raw, &j); err != nil {
			continue
		}
		if j.Use != "" && j.Use != "sig" {
			continue
		}
		if len(j.KeyOps) > 0 && !stringSliceContains(j.KeyOps, "verify") {
			continue
		}
		alg := strings.ToUpper(strings.TrimSpace(j.Alg))
		var key any
		switch strings.ToUpper(j.Kty) {
		case "RSA":
			nb, errN := base64.RawURLEncoding.DecodeString(j.N)
			eb, errE := base64.RawURLEncoding.DecodeString(j.E)
			if errN != nil || errE != nil || len(nb) < 256 || len(eb) == 0 || len(eb) > 4 {
				continue
			}
			e := 0
			for _, b := range eb {
				e = (e << 8) | int(b)
			}
			if e < 3 || e%2 == 0 {
				continue
			}
			key = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}
		case "EC":
			var curve elliptic.Curve
			switch j.Crv {
			case "P-256":
				curve = elliptic.P256()
			case "P-384":
				curve = elliptic.P384()
			case "P-521":
				curve = elliptic.P521()
			default:
				continue
			}
			xb, errX := base64.RawURLEncoding.DecodeString(j.X)
			yb, errY := base64.RawURLEncoding.DecodeString(j.Y)
			if errX != nil || errY != nil {
				continue
			}
			x, y := new(big.Int).SetBytes(xb), new(big.Int).SetBytes(yb)
			if !curve.IsOnCurve(x, y) {
				continue
			}
			key = &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		case "OKP":
			if j.Crv != "Ed25519" {
				continue
			}
			xb, err := base64.RawURLEncoding.DecodeString(j.X)
			if err != nil || len(xb) != ed25519.PublicKeySize {
				continue
			}
			key = ed25519.PublicKey(append([]byte(nil), xb...))
		default:
			continue
		}
		out = append(out, jwksPublicKey{Kid: strings.TrimSpace(j.Kid), Alg: alg, Key: key})
	}
	if len(out) == 0 {
		return nil, errors.New("JWKS contains no usable signature verification keys")
	}
	return out, nil
}

func stringSliceContains(in []string, want string) bool {
	for _, v := range in {
		if v == want {
			return true
		}
	}
	return false
}

func (s *apiIdentityStore) refreshIssuer(ctx context.Context, issuerID string) error {
	if s == nil {
		return errors.New("identity store unavailable")
	}
	s.mu.RLock()
	issuer, ok := s.issuers[issuerID]
	rt := s.runtimes[issuerID]
	fetch := s.fetchJWKS
	s.mu.RUnlock()
	if !ok || !issuer.Enabled || rt == nil {
		return errors.New("enabled issuer not found")
	}
	_, err := rt.refresh(ctx, fetch, true)
	return err
}

func (rt *jwtIssuerRuntime) refresh(ctx context.Context, fetch func(context.Context, string) ([]byte, error), force bool) (*jwksKeySet, error) {
	if rt == nil {
		return nil, errors.New("issuer runtime unavailable")
	}
	if !force {
		if current := rt.keys.Load(); current != nil && time.Now().UTC().Before(current.ExpiresAt) {
			return current, nil
		}
	}
	rt.refreshMu.Lock()
	defer rt.refreshMu.Unlock()
	if !force {
		if current := rt.keys.Load(); current != nil && time.Now().UTC().Before(current.ExpiresAt) {
			return current, nil
		}
	}
	if fetch == nil {
		return nil, errors.New("JWKS fetcher unavailable")
	}
	data, err := fetch(ctx, rt.config.JWKSURL)
	if err != nil {
		return nil, err
	}
	keys, err := parseJWKS(data)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	set := &jwksKeySet{Keys: keys, FetchedAt: now, ExpiresAt: now.Add(time.Duration(rt.config.CacheTTLSec) * time.Second)}
	rt.keys.Store(set)
	return set, nil
}

func (rt *jwtIssuerRuntime) refreshForKeyMiss(ctx context.Context, fetch func(context.Context, string) ([]byte, error), observed *jwksKeySet) (*jwksKeySet, error) {
	if rt == nil {
		return nil, errors.New("issuer runtime unavailable")
	}
	rt.refreshMu.Lock()
	defer rt.refreshMu.Unlock()
	current := rt.keys.Load()
	if current != nil && observed != nil && current != observed {
		// Another request already refreshed the key set while this request was
		// waiting. Reuse it rather than issuing a duplicate network fetch.
		return current, nil
	}
	now := time.Now().UTC()
	if !rt.lastKidMissRefresh.IsZero() && now.Sub(rt.lastKidMissRefresh) < apiIdentityKidMissRefreshMin {
		if current != nil {
			return current, nil
		}
		return nil, errors.New("JWKS key-miss refresh is rate limited")
	}
	if fetch == nil {
		return nil, errors.New("JWKS fetcher unavailable")
	}
	data, err := fetch(ctx, rt.config.JWKSURL)
	if err != nil {
		return nil, err
	}
	keys, err := parseJWKS(data)
	if err != nil {
		return nil, err
	}
	set := &jwksKeySet{Keys: keys, FetchedAt: now, ExpiresAt: now.Add(time.Duration(rt.config.CacheTTLSec) * time.Second)}
	rt.keys.Store(set)
	rt.lastKidMissRefresh = now
	return set, nil
}

type parsedJWT struct {
	Header      map[string]any
	Claims      map[string]any
	SigningData []byte
	Signature   []byte
	Alg         string
	Kid         string
}

func parseJWT(token string) (parsedJWT, error) {
	if len(token) == 0 || len(token) > apiIdentityMaxTokenBytes {
		return parsedJWT{}, errors.New("token size invalid")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return parsedJWT{}, errors.New("token must have three compact JWS segments")
	}
	decodeJSON := func(seg string) (map[string]any, error) {
		b, err := base64.RawURLEncoding.DecodeString(seg)
		if err != nil || len(b) > apiIdentityMaxTokenBytes {
			return nil, errors.New("invalid JWT base64url segment")
		}
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.UseNumber()
		var v map[string]any
		if err := dec.Decode(&v); err != nil || v == nil {
			return nil, errors.New("invalid JWT JSON segment")
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			return nil, errors.New("JWT JSON segment contains trailing data")
		}
		return v, nil
	}
	header, err := decodeJSON(parts[0])
	if err != nil {
		return parsedJWT{}, err
	}
	claims, err := decodeJSON(parts[1])
	if err != nil {
		return parsedJWT{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) == 0 || len(sig) > 2048 {
		return parsedJWT{}, errors.New("invalid JWT signature encoding")
	}
	alg, _ := header["alg"].(string)
	kid, _ := header["kid"].(string)
	alg = strings.ToUpper(strings.TrimSpace(alg))
	if alg == "" || alg == "NONE" {
		return parsedJWT{}, errors.New("JWT alg missing or forbidden")
	}
	if critRaw, exists := header["crit"]; exists {
		crit, ok := critRaw.([]any)
		if !ok || len(crit) > 0 {
			return parsedJWT{}, errors.New("JWT critical header extensions are not supported")
		}
	}
	if b64Raw, exists := header["b64"]; exists {
		b64, ok := b64Raw.(bool)
		if !ok || !b64 {
			return parsedJWT{}, errors.New("unencoded JWS payloads are not supported")
		}
	}
	return parsedJWT{Header: header, Claims: claims, SigningData: []byte(parts[0] + "." + parts[1]), Signature: sig, Alg: alg, Kid: strings.TrimSpace(kid)}, nil
}

func jwtAlgAllowed(allowed []string, alg string) bool {
	for _, v := range allowed {
		if v == alg {
			return true
		}
	}
	return false
}

func compatibleJWTKey(alg string, key any) bool {
	switch {
	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "PS"):
		_, ok := key.(*rsa.PublicKey)
		return ok
	case strings.HasPrefix(alg, "ES"):
		pub, ok := key.(*ecdsa.PublicKey)
		if !ok || pub.Curve == nil {
			return false
		}
		switch alg {
		case "ES256":
			return pub.Curve.Params().Name == elliptic.P256().Params().Name
		case "ES384":
			return pub.Curve.Params().Name == elliptic.P384().Params().Name
		case "ES512":
			return pub.Curve.Params().Name == elliptic.P521().Params().Name
		}
		return false
	case alg == "EDDSA":
		_, ok := key.(ed25519.PublicKey)
		return ok
	}
	return false
}

func selectJWTKey(set *jwksKeySet, kid, alg string) (jwksPublicKey, error) {
	if set == nil {
		return jwksPublicKey{}, errors.New("JWKS unavailable")
	}
	matches := make([]jwksPublicKey, 0, 2)
	for _, k := range set.Keys {
		if kid != "" && k.Kid != kid {
			continue
		}
		if k.Alg != "" && k.Alg != alg {
			continue
		}
		if !compatibleJWTKey(alg, k.Key) {
			continue
		}
		matches = append(matches, k)
	}
	if len(matches) == 0 {
		return jwksPublicKey{}, errors.New("no matching JWKS key")
	}
	if len(matches) != 1 {
		if kid == "" {
			return jwksPublicKey{}, errors.New("JWT kid required because verification key is ambiguous")
		}
		return jwksPublicKey{}, errors.New("JWKS kid is ambiguous")
	}
	return matches[0], nil
}

func verifyJWTSignature(jwt parsedJWT, key jwksPublicKey) error {
	var digest []byte
	var hash crypto.Hash
	switch jwt.Alg {
	case "RS256", "PS256", "ES256":
		h := sha256.Sum256(jwt.SigningData)
		digest, hash = h[:], crypto.SHA256
	case "RS384", "PS384", "ES384":
		h := sha512.Sum384(jwt.SigningData)
		digest, hash = h[:], crypto.SHA384
	case "RS512", "PS512", "ES512":
		h := sha512.Sum512(jwt.SigningData)
		digest, hash = h[:], crypto.SHA512
	case "EDDSA":
		pub, ok := key.Key.(ed25519.PublicKey)
		if !ok || !ed25519.Verify(pub, jwt.SigningData, jwt.Signature) {
			return errors.New("JWT signature verification failed")
		}
		return nil
	default:
		return errors.New("unsupported JWT algorithm")
	}
	switch pub := key.Key.(type) {
	case *rsa.PublicKey:
		if strings.HasPrefix(jwt.Alg, "PS") {
			if err := rsa.VerifyPSS(pub, hash, digest, jwt.Signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: hash}); err != nil {
				return errors.New("JWT signature verification failed")
			}
			return nil
		}
		if err := rsa.VerifyPKCS1v15(pub, hash, digest, jwt.Signature); err != nil {
			return errors.New("JWT signature verification failed")
		}
		return nil
	case *ecdsa.PublicKey:
		size := (pub.Curve.Params().BitSize + 7) / 8
		if len(jwt.Signature) != size*2 {
			return errors.New("invalid ECDSA JWT signature length")
		}
		r := new(big.Int).SetBytes(jwt.Signature[:size])
		s := new(big.Int).SetBytes(jwt.Signature[size:])
		if !ecdsa.Verify(pub, digest, r, s) {
			return errors.New("JWT signature verification failed")
		}
		return nil
	default:
		return errors.New("JWT key type does not match algorithm")
	}
}

type jwtValidationError struct {
	Code string
	Err  error
}

func (e *jwtValidationError) Error() string {
	if e == nil || e.Err == nil {
		return "JWT validation failed"
	}
	return e.Err.Error()
}

func jwtErr(code, msg string) error { return &jwtValidationError{Code: code, Err: errors.New(msg)} }

func numericDate(claims map[string]any, name string) (time.Time, bool, error) {
	v, ok := claims[name]
	if !ok {
		return time.Time{}, false, nil
	}
	var f float64
	switch n := v.(type) {
	case json.Number:
		var err error
		f, err = strconv.ParseFloat(n.String(), 64)
		if err != nil {
			return time.Time{}, true, errors.New("invalid numeric date")
		}
	case float64:
		f = n
	default:
		return time.Time{}, true, errors.New("invalid numeric date type")
	}
	if f < 0 || f > 1<<40 {
		return time.Time{}, true, errors.New("numeric date out of range")
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC(), true, nil
}

func audienceValues(claims map[string]any) []string {
	v, ok := claims["aud"]
	if !ok {
		return nil
	}
	switch a := v.(type) {
	case string:
		return normalizeStringSet([]string{a}, 32, 512)
	case []any:
		out := make([]string, 0, len(a))
		for _, item := range a {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return normalizeStringSet(out, 32, 512)
	}
	return nil
}

func claimAtPath(claims map[string]any, path string) any {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	var cur any = claims
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[part]
		if !ok {
			return nil
		}
	}
	return cur
}

func claimString(claims map[string]any, path string) string {
	v := claimAtPath(claims, path)
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func claimStrings(claims map[string]any, path string, splitSpaces bool) []string {
	v := claimAtPath(claims, path)
	var out []string
	switch x := v.(type) {
	case string:
		if splitSpaces {
			out = strings.Fields(x)
		} else if strings.TrimSpace(x) != "" {
			out = []string{x}
		}
	case []any:
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	}
	return normalizeStringSet(out, 256, 512)
}

func valuesIntersect(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

func valuesContainAll(have, required []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, v := range have {
		set[v] = struct{}{}
	}
	for _, v := range required {
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}

func (rt *jwtIssuerRuntime) verify(ctx context.Context, token string, fetch func(context.Context, string) ([]byte, error)) (VerifiedAPIIdentity, error) {
	jwt, err := parseJWT(token)
	if err != nil {
		return VerifiedAPIIdentity{}, jwtErr("JWT_MALFORMED", err.Error())
	}
	if !jwtAlgAllowed(rt.config.AllowedAlgs, jwt.Alg) {
		return VerifiedAPIIdentity{}, jwtErr("JWT_ALG_FORBIDDEN", "JWT algorithm is not allowed")
	}
	set, err := rt.refresh(ctx, fetch, false)
	if err != nil {
		return VerifiedAPIIdentity{}, jwtErr("JWKS_UNAVAILABLE", "JWKS is unavailable")
	}
	key, err := selectJWTKey(set, jwt.Kid, jwt.Alg)
	if err != nil {
		// A kid miss commonly means key rotation. Refresh exactly once before
		// rejecting instead of trusting stale cached keys indefinitely.
		set, refreshErr := rt.refreshForKeyMiss(ctx, fetch, set)
		if refreshErr != nil {
			return VerifiedAPIIdentity{}, jwtErr("JWKS_UNAVAILABLE", "JWKS refresh failed")
		}
		key, err = selectJWTKey(set, jwt.Kid, jwt.Alg)
		if err != nil {
			return VerifiedAPIIdentity{}, jwtErr("JWT_KEY_NOT_FOUND", "JWT verification key not found")
		}
	}
	if err := verifyJWTSignature(jwt, key); err != nil {
		return VerifiedAPIIdentity{}, jwtErr("JWT_SIGNATURE_INVALID", err.Error())
	}
	iss, _ := jwt.Claims["iss"].(string)
	if strings.TrimSpace(iss) != rt.config.Issuer {
		return VerifiedAPIIdentity{}, jwtErr("JWT_ISSUER_INVALID", "JWT issuer does not match trusted issuer")
	}
	aud := audienceValues(jwt.Claims)
	if len(aud) == 0 || !valuesIntersect(aud, rt.config.Audiences) {
		return VerifiedAPIIdentity{}, jwtErr("JWT_AUDIENCE_INVALID", "JWT audience is not accepted")
	}
	now := time.Now().UTC()
	skew := time.Duration(rt.config.ClockSkewSec) * time.Second
	exp, present, err := numericDate(jwt.Claims, "exp")
	if err != nil || !present {
		return VerifiedAPIIdentity{}, jwtErr("JWT_EXP_INVALID", "JWT exp is required and must be a numeric date")
	}
	if !now.Before(exp.Add(skew)) {
		return VerifiedAPIIdentity{}, jwtErr("JWT_EXPIRED", "JWT has expired")
	}
	if nbf, present, err := numericDate(jwt.Claims, "nbf"); err != nil {
		return VerifiedAPIIdentity{}, jwtErr("JWT_NBF_INVALID", "JWT nbf must be a numeric date")
	} else if present && now.Add(skew).Before(nbf) {
		return VerifiedAPIIdentity{}, jwtErr("JWT_NOT_YET_VALID", "JWT is not yet valid")
	}
	return VerifiedAPIIdentity{
		Issuer:    iss,
		Subject:   claimString(jwt.Claims, rt.config.SubjectClaim),
		TenantID:  claimString(jwt.Claims, rt.config.TenantClaim),
		ClientID:  claimString(jwt.Claims, rt.config.ClientClaim),
		Roles:     claimStrings(jwt.Claims, rt.config.RolesClaim, false),
		Scopes:    claimStrings(jwt.Claims, rt.config.ScopesClaim, true),
		Audience:  aud,
		ExpiresAt: exp,
		KeyID:     jwt.Kid,
		Algorithm: jwt.Alg,
	}, nil
}

func bearerToken(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("request unavailable")
	}
	values := r.Header.Values("Authorization")
	if len(values) == 0 {
		return "", nil
	}
	if len(values) != 1 {
		return "", errors.New("multiple Authorization headers are not accepted")
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("Bearer authorization is required")
	}
	return parts[1], nil
}

func evaluateIdentityRequirements(policy APIIdentityPolicy, id VerifiedAPIIdentity) (string, string, string) {
	if len(policy.RequiredRoles) > 0 {
		ok := valuesContainAll(id.Roles, policy.RequiredRoles)
		if policy.RoleMatch == "any" {
			ok = valuesIntersect(id.Roles, policy.RequiredRoles)
		}
		if !ok {
			return "ROLE_MISSING", "required role policy", "verified identity lacks required role"
		}
	}
	if len(policy.RequiredScopes) > 0 {
		ok := valuesContainAll(id.Scopes, policy.RequiredScopes)
		if policy.ScopeMatch == "any" {
			ok = valuesIntersect(id.Scopes, policy.RequiredScopes)
		}
		if !ok {
			return "SCOPE_MISSING", "required scope policy", "verified identity lacks required scope"
		}
	}
	if policy.RequiredTenant != "" && id.TenantID != policy.RequiredTenant {
		return "TENANT_MISMATCH", "required tenant", "verified tenant does not match"
	}
	if len(policy.AllowedClientIDs) > 0 && !stringSliceContains(policy.AllowedClientIDs, id.ClientID) {
		return "CLIENT_ID_FORBIDDEN", "allowed client_id", "verified client_id is not allowed"
	}
	return "", "", ""
}

func identitySubjectHash(id VerifiedAPIIdentity) string {
	if id.Subject == "" {
		return ""
	}
	h := sha256.Sum256([]byte(id.Issuer + "\x00" + id.Subject))
	return hex.EncodeToString(h[:12])
}

func (s *apiIdentityStore) recordViolation(v APIIdentityViolation) {
	if s == nil {
		return
	}
	v.Time = time.Now().UTC()
	seq := s.seq.Add(1)
	v.ID = fmt.Sprintf("idv-%d-%d", v.Time.UnixNano(), seq)
	s.mu.Lock()
	s.violations = append(s.violations, v)
	if len(s.violations) > apiIdentityMaxViolations {
		s.violations = append([]APIIdentityViolation(nil), s.violations[len(s.violations)-apiIdentityMaxViolations:]...)
	}
	s.mu.Unlock()
}

func (s *apiIdentityStore) listViolations(operationID string, limit int) []APIIdentityViolation {
	if s == nil {
		return nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	s.mu.RLock()
	out := make([]APIIdentityViolation, 0, limit)
	for i := len(s.violations) - 1; i >= 0 && len(out) < limit; i-- {
		v := s.violations[i]
		if operationID != "" && v.OperationID != operationID {
			continue
		}
		out = append(out, v)
	}
	s.mu.RUnlock()
	return out
}

func identityAction(mode string) string {
	if mode == apiIdentityModeEnforce {
		return "block"
	}
	return "detect"
}

func writeIdentityReject(w http.ResponseWriter, status int) {
	if status != http.StatusUnauthorized && status != http.StatusForbidden && status != http.StatusServiceUnavailable {
		status = http.StatusForbidden
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"request rejected by API identity policy"}`))
}

func (s *apiIdentityStore) wrap(site string, next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			next.ServeHTTP(w, r)
			return
		}
		opID := apiOperationID(site, r.Method, normalizeAPIOperationPath(r.URL.Path))
		rp, ok := s.runtimePolicy(opID)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		token, bearerErr := bearerToken(r)
		if token == "" || bearerErr != nil {
			vType := "JWT_MISSING"
			observed := "bearer token missing"
			if bearerErr != nil {
				vType = "JWT_MALFORMED_AUTHORIZATION"
				observed = "authorization header is not a single Bearer credential"
			}
			v := APIIdentityViolation{OperationID: opID, Mode: rp.Policy.Mode, Action: identityAction(rp.Policy.Mode), Type: vType, IssuerID: rp.Policy.IssuerID, Expected: "cryptographically verified Bearer JWT", Observed: observed}
			s.recordViolation(v)
			if rp.Policy.Mode == apiIdentityModeEnforce {
				writeIdentityReject(w, rp.Policy.UnauthorizedCode)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		if rp.Issuer == nil {
			s.recordViolation(APIIdentityViolation{OperationID: opID, Mode: rp.Policy.Mode, Action: identityAction(rp.Policy.Mode), Type: "JWT_ISSUER_UNAVAILABLE", IssuerID: rp.Policy.IssuerID, Expected: "enabled trusted issuer runtime", Observed: "issuer configuration unavailable"})
			if rp.Policy.Mode == apiIdentityModeEnforce {
				writeIdentityReject(w, http.StatusServiceUnavailable)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		id, err := rp.Issuer.verify(r.Context(), token, s.fetchJWKS)
		if err != nil {
			code := "JWT_INVALID"
			if ve := new(jwtValidationError); errors.As(err, &ve) {
				code = ve.Code
			}
			s.recordViolation(APIIdentityViolation{OperationID: opID, Mode: rp.Policy.Mode, Action: identityAction(rp.Policy.Mode), Type: code, IssuerID: rp.Policy.IssuerID, Expected: "valid trusted JWT", Observed: "JWT verification failed"})
			if rp.Policy.Mode == apiIdentityModeEnforce {
				status := rp.Policy.UnauthorizedCode
				if code == "JWKS_UNAVAILABLE" {
					status = http.StatusServiceUnavailable
				}
				writeIdentityReject(w, status)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), apiIdentityContextKey{}, id)
		r = r.WithContext(ctx)
		if typ, expected, observed := evaluateIdentityRequirements(rp.Policy, id); typ != "" {
			s.recordViolation(APIIdentityViolation{OperationID: opID, Mode: rp.Policy.Mode, Action: identityAction(rp.Policy.Mode), Type: typ, IssuerID: rp.Policy.IssuerID, SubjectHash: identitySubjectHash(id), Expected: expected, Observed: observed})
			if rp.Policy.Mode == apiIdentityModeEnforce {
				writeIdentityReject(w, rp.Policy.ForbiddenCode)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *apiIdentityStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	state := apiIdentityStateFile{
		Version:    apiSecurityStateVersion,
		Saved:      time.Now().UTC(),
		Issuers:    cloneMap(s.issuers),
		Policies:   cloneMap(s.policies),
		Violations: append([]APIIdentityViolation(nil), s.violations...),
	}
	s.mu.RUnlock()
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-identity.json"), state)
}

func (s *apiIdentityStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state apiIdentityStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-identity.json"), &state); err != nil {
		return err
	}
	if err := validateAPISecurityStateVersion("api-identity", state.Version); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if state.Issuers != nil {
		s.issuers = state.Issuers
	}
	if state.Policies != nil {
		s.policies = state.Policies
	}
	if len(state.Violations) > apiIdentityMaxViolations {
		state.Violations = state.Violations[len(state.Violations)-apiIdentityMaxViolations:]
	}
	s.violations = append([]APIIdentityViolation(nil), state.Violations...)
	if s.issuers == nil {
		s.issuers = map[string]JWTIssuerConfig{}
	}
	if s.policies == nil {
		s.policies = map[string]APIIdentityPolicy{}
	}
	if s.runtimes == nil {
		s.runtimes = map[string]*jwtIssuerRuntime{}
	}
	s.rebuildRuntimeLocked()
	return nil
}

func (a *adminServer) handleIdentityIssuers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.srv.identity.listIssuers())
}

func (a *adminServer) handleIdentityIssuerUpsert(w http.ResponseWriter, r *http.Request) {
	var req JWTIssuerConfig
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	issuer, err := a.srv.identity.upsertIssuer(req, who(r).user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.srv.identity.save(a.srv.configPath); err != nil {
		http.Error(w, "could not persist identity state", http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_identity.issuer_upsert", issuer.ID)
	writeJSON(w, issuer)
}

func (a *adminServer) handleIdentityIssuerRefresh(w http.ResponseWriter, r *http.Request) {
	issuerID := r.PathValue("issuer_id")
	ctx, cancel := context.WithTimeout(r.Context(), 7*time.Second)
	defer cancel()
	if err := a.srv.identity.refreshIssuer(ctx, issuerID); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	a.audit.add(who(r).user, "api_identity.issuer_refresh", issuerID)
	writeJSON(w, map[string]any{"ok": true, "issuer_id": issuerID})
}

func (a *adminServer) handleIdentityPolicies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.srv.identity.listPolicies())
}

func (a *adminServer) handleIdentityPolicyUpsert(w http.ResponseWriter, r *http.Request) {
	var req APIIdentityPolicy
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.OperationID = r.PathValue("operation_id")
	policy, err := a.srv.identity.upsertPolicy(req, who(r).user, func(id string) bool {
		_, ok := a.srv.apiOps.get(id)
		return ok
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.srv.identity.save(a.srv.configPath); err != nil {
		http.Error(w, "could not persist identity state", http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_identity.policy_upsert", policy.OperationID+" issuer="+policy.IssuerID)
	writeJSON(w, policy)
}

func (a *adminServer) handleIdentityPolicyMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode   string `json:"mode"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	policy, err := a.srv.identity.setPolicyMode(r.PathValue("operation_id"), req.Mode, who(r).user, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.srv.identity.save(a.srv.configPath); err != nil {
		http.Error(w, "could not persist identity state", http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_identity.policy_mode", policy.OperationID+" -> "+policy.Mode)
	writeJSON(w, policy)
}

func (a *adminServer) handleIdentityPolicyDelete(w http.ResponseWriter, r *http.Request) {
	opID := r.PathValue("operation_id")
	if err := a.srv.identity.deletePolicy(opID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := a.srv.identity.save(a.srv.configPath); err != nil {
		http.Error(w, "could not persist identity state", http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_identity.policy_delete", opID)
	writeJSON(w, map[string]any{"ok": true})
}

func (a *adminServer) handleIdentityViolations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, a.srv.identity.listViolations(r.URL.Query().Get("operation_id"), limit))
}
