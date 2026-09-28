package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type api5JWTFixture struct {
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
	kid  string
}

func newAPI5JWTFixture(kid string, fill byte) api5JWTFixture {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = fill
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return api5JWTFixture{pub: pub, priv: priv, kid: kid}
}

func (f api5JWTFixture) jwks() []byte {
	doc := map[string]any{"keys": []any{map[string]any{
		"kty": "OKP", "crv": "Ed25519", "use": "sig", "key_ops": []string{"verify"},
		"alg": "EdDSA", "kid": f.kid, "x": base64.RawURLEncoding.EncodeToString(f.pub),
	}}}
	b, _ := json.Marshal(doc)
	return b
}

func (f api5JWTFixture) token(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := map[string]any{"alg": "EdDSA", "typ": "JWT", "kid": f.kid}
	return f.tokenWithHeader(t, header, claims)
}

func (f api5JWTFixture) tokenWithHeader(t *testing.T, header map[string]any, claims map[string]any) string {
	t.Helper()
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	h := base64.RawURLEncoding.EncodeToString(hb)
	c := base64.RawURLEncoding.EncodeToString(cb)
	signing := h + "." + c
	sig := ed25519.Sign(f.priv, []byte(signing))
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func api5Issuer() JWTIssuerConfig {
	return JWTIssuerConfig{
		ID: "corp", Name: "Corporate IdP", Issuer: "https://issuer.example.test",
		JWKSURL: "https://issuer.example.test/.well-known/jwks.json", Audiences: []string{"payments-api"},
		AllowedAlgs: []string{"EdDSA"}, CacheTTLSec: 300, TenantClaim: "tenant_id",
		RolesClaim: "roles", ScopesClaim: "scope", Enabled: true,
	}
}

func api5Claims() map[string]any {
	now := time.Now().UTC()
	return map[string]any{
		"iss": "https://issuer.example.test", "aud": "payments-api", "sub": "user-1001",
		"tenant_id": "tenant-a", "client_id": "checkout", "roles": []string{"customer", "admin"},
		"scope": "payment.read payment.write", "exp": now.Add(10 * time.Minute).Unix(), "nbf": now.Add(-time.Minute).Unix(),
	}
}

func api5StoreWithPolicy(t *testing.T, fetch func(context.Context, string) ([]byte, error)) (*apiIdentityStore, string) {
	t.Helper()
	store := newAPIIdentityStore()
	store.fetchJWKS = fetch
	if _, err := store.upsertIssuer(api5Issuer(), "tester"); err != nil {
		t.Fatalf("issuer: %v", err)
	}
	opID := apiOperationID("default", http.MethodPost, "/api/payment")
	_, err := store.upsertPolicy(APIIdentityPolicy{
		OperationID: opID, IssuerID: "corp", Mode: apiIdentityModeDetect, RequireAuth: true,
		RequiredRoles: []string{"admin"}, RequiredScopes: []string{"payment.write"}, RequiredTenant: "tenant-a",
		AllowedClientIDs: []string{"checkout"},
	}, "tester", func(string) bool { return true })
	if err != nil {
		t.Fatalf("policy: %v", err)
	}
	return store, opID
}

func TestAPI5VerifiedClaimsOnlyEnterRequestContext(t *testing.T) {
	fx := newAPI5JWTFixture("k1", 1)
	store, _ := api5StoreWithPolicy(t, func(context.Context, string) ([]byte, error) { return fx.jwks(), nil })
	token := fx.token(t, api5Claims())

	var got VerifiedAPIIdentity
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		got, ok = verifiedAPIIdentity(r)
		if !ok {
			t.Fatal("verified identity missing from request context")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "https://api.test/api/payment", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	store.wrap("default", next).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.Subject != "user-1001" || got.TenantID != "tenant-a" || got.ClientID != "checkout" {
		t.Fatalf("unexpected identity: %+v", got)
	}
	if !stringSliceContains(got.Roles, "admin") || !stringSliceContains(got.Scopes, "payment.write") {
		t.Fatalf("roles/scopes missing: %+v", got)
	}
}

func TestAPI5DetectThenEnforceInvalidJWT(t *testing.T) {
	trusted := newAPI5JWTFixture("k1", 2)
	attacker := newAPI5JWTFixture("k1", 3)
	store, opID := api5StoreWithPolicy(t, func(context.Context, string) ([]byte, error) { return trusted.jwks(), nil })
	bad := attacker.token(t, api5Claims())
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodPost, "https://api.test/api/payment", nil)
	req.Header.Set("Authorization", "Bearer "+bad)
	rr := httptest.NewRecorder()
	store.wrap("default", next).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("DETECT must not block, status=%d", rr.Code)
	}
	vs := store.listViolations(opID, 10)
	if len(vs) == 0 || vs[0].Type != "JWT_SIGNATURE_INVALID" || vs[0].Action != "detect" {
		t.Fatalf("unexpected detect evidence: %+v", vs)
	}

	if _, err := store.setPolicyMode(opID, apiIdentityModeEnforce, "tester", "shadow validated"); err != nil {
		t.Fatalf("promote: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "https://api.test/api/payment", nil)
	req.Header.Set("Authorization", "Bearer "+bad)
	rr = httptest.NewRecorder()
	store.wrap("default", next).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized || rr.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("ENFORCE invalid JWT status=%d headers=%v", rr.Code, rr.Header())
	}
}

func TestAPI5AuthorizationRequirementsUseVerifiedClaims(t *testing.T) {
	fx := newAPI5JWTFixture("k1", 4)
	store, opID := api5StoreWithPolicy(t, func(context.Context, string) ([]byte, error) { return fx.jwks(), nil })
	if _, err := store.setPolicyMode(opID, apiIdentityModeEnforce, "tester", "ready"); err != nil {
		t.Fatal(err)
	}
	claims := api5Claims()
	claims["roles"] = []string{"customer"}
	claims["scope"] = "payment.read"
	token := fx.token(t, claims)
	req := httptest.NewRequest(http.MethodPost, "https://api.test/api/payment", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	store.wrap("default", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	vs := store.listViolations(opID, 10)
	if len(vs) == 0 || vs[0].Type != "ROLE_MISSING" || vs[0].SubjectHash == "" {
		t.Fatalf("expected privacy-preserving role evidence, got %+v", vs)
	}
	if strings.Contains(vs[0].Observed, "user-1001") {
		t.Fatal("raw subject leaked to persisted violation evidence")
	}
}

func TestAPI5IssuerAudienceExpiryAndNBFValidation(t *testing.T) {
	fx := newAPI5JWTFixture("k1", 5)
	store, _ := api5StoreWithPolicy(t, func(context.Context, string) ([]byte, error) { return fx.jwks(), nil })
	store.mu.RLock()
	rt := store.runtimes["corp"]
	store.mu.RUnlock()

	cases := []struct {
		name string
		mut  func(map[string]any)
		code string
	}{
		{"issuer", func(c map[string]any) { c["iss"] = "https://evil.example" }, "JWT_ISSUER_INVALID"},
		{"audience", func(c map[string]any) { c["aud"] = "other-api" }, "JWT_AUDIENCE_INVALID"},
		{"expired", func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }, "JWT_EXPIRED"},
		{"future_nbf", func(c map[string]any) { c["nbf"] = time.Now().Add(time.Hour).Unix() }, "JWT_NOT_YET_VALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := api5Claims()
			tc.mut(claims)
			_, err := rt.verify(context.Background(), fx.token(t, claims), store.fetchJWKS)
			var ve *jwtValidationError
			if err == nil || !errorsAsJWT(err, &ve) || ve.Code != tc.code {
				t.Fatalf("err=%v code=%v want=%s", err, func() string {
					if ve != nil {
						return ve.Code
					}
					return ""
				}(), tc.code)
			}
		})
	}
}

func errorsAsJWT(err error, target **jwtValidationError) bool {
	for err != nil {
		if v, ok := err.(*jwtValidationError); ok {
			*target = v
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func TestAPI5KidMissRefreshesJWKSOnceForRotation(t *testing.T) {
	oldKey := newAPI5JWTFixture("old", 6)
	newKey := newAPI5JWTFixture("new", 7)
	var calls atomic.Int32
	fetch := func(context.Context, string) ([]byte, error) {
		if calls.Add(1) == 1 {
			return oldKey.jwks(), nil
		}
		return newKey.jwks(), nil
	}
	store, _ := api5StoreWithPolicy(t, fetch)
	store.mu.RLock()
	rt := store.runtimes["corp"]
	store.mu.RUnlock()
	id, err := rt.verify(context.Background(), newKey.token(t, api5Claims()), fetch)
	if err != nil || id.KeyID != "new" {
		t.Fatalf("rotation verify id=%+v err=%v", id, err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("fetch calls=%d want=2", got)
	}
}

func TestAPI5PersistenceExcludesTokensAndJWKSCache(t *testing.T) {
	fx := newAPI5JWTFixture("k1", 8)
	store, opID := api5StoreWithPolicy(t, func(context.Context, string) ([]byte, error) { return fx.jwks(), nil })
	store.mu.RLock()
	rt := store.runtimes["corp"]
	store.mu.RUnlock()
	if _, err := rt.verify(context.Background(), fx.token(t, api5Claims()), store.fetchJWKS); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	if err := store.save(cfg); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "api-identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if strings.Contains(text, base64.RawURLEncoding.EncodeToString(fx.pub)) || strings.Contains(text, "user-1001") || strings.Contains(text, "payment.write payment.read") {
		t.Fatalf("ephemeral key/token/claims leaked to persistence: %s", text)
	}
	restored := newAPIIdentityStore()
	if err := restored.load(cfg); err != nil {
		t.Fatal(err)
	}
	if len(restored.listIssuers()) != 1 || len(restored.listPolicies()) != 1 || restored.listPolicies()[0].OperationID != opID {
		t.Fatalf("restore mismatch issuers=%+v policies=%+v", restored.listIssuers(), restored.listPolicies())
	}
}

func TestAPI5PolicyLifecycleStartsDetect(t *testing.T) {
	store := newAPIIdentityStore()
	store.fetchJWKS = func(context.Context, string) ([]byte, error) { return newAPI5JWTFixture("k", 9).jwks(), nil }
	if _, err := store.upsertIssuer(api5Issuer(), "tester"); err != nil {
		t.Fatal(err)
	}
	opID := apiOperationID("default", http.MethodGet, "/api/admin/users")
	if _, err := store.upsertPolicy(APIIdentityPolicy{OperationID: opID, IssuerID: "corp", Mode: apiIdentityModeEnforce}, "tester", func(string) bool { return true }); err == nil {
		t.Fatal("new policy must not start in ENFORCE")
	}
	p, err := store.upsertPolicy(APIIdentityPolicy{OperationID: opID, IssuerID: "corp"}, "tester", func(string) bool { return true })
	if err != nil || p.Mode != apiIdentityModeDetect {
		t.Fatalf("policy=%+v err=%v", p, err)
	}
	p, err = store.setPolicyMode(opID, apiIdentityModeEnforce, "tester", "validated")
	if err != nil || p.Mode != apiIdentityModeEnforce {
		t.Fatalf("policy=%+v err=%v", p, err)
	}
	p, err = store.setPolicyMode(opID, apiIdentityModeDetect, "tester", "rollback")
	if err != nil || p.Mode != apiIdentityModeDetect {
		t.Fatalf("policy=%+v err=%v", p, err)
	}
}

func TestAPI5RS256JWKSVerification(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "rsa1"
	e := []byte{1, 0, 1}
	jwks, _ := json.Marshal(map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(priv.PublicKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(e),
	}}})
	issuer := api5Issuer()
	issuer.AllowedAlgs = []string{"RS256"}
	store := newAPIIdentityStore()
	store.fetchJWKS = func(context.Context, string) ([]byte, error) { return jwks, nil }
	if _, err := store.upsertIssuer(issuer, "tester"); err != nil {
		t.Fatal(err)
	}
	store.mu.RLock()
	rt := store.runtimes["corp"]
	store.mu.RUnlock()
	claims := api5Claims()
	header := map[string]any{"alg": "RS256", "kid": kid}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	signing := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(cb)
	h := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		t.Fatal(err)
	}
	token := signing + "." + base64.RawURLEncoding.EncodeToString(sig)
	id, err := rt.verify(context.Background(), token, store.fetchJWKS)
	if err != nil || id.Subject != "user-1001" {
		t.Fatalf("id=%+v err=%v", id, err)
	}
}

func TestAPI5SemanticChangesReturnEnforcementToDetect(t *testing.T) {
	fx := newAPI5JWTFixture("k1", 10)
	store, opID := api5StoreWithPolicy(t, func(context.Context, string) ([]byte, error) { return fx.jwks(), nil })
	if _, err := store.setPolicyMode(opID, apiIdentityModeEnforce, "tester", "validated"); err != nil {
		t.Fatal(err)
	}
	p, err := store.upsertPolicy(APIIdentityPolicy{
		OperationID: opID, IssuerID: "corp", RequiredRoles: []string{"admin"},
		RequiredScopes: []string{"payment.write", "payment.refund"}, RequiredTenant: "tenant-a", AllowedClientIDs: []string{"checkout"},
	}, "tester", func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != apiIdentityModeDetect {
		t.Fatalf("policy semantic change must reset to DETECT, got %s", p.Mode)
	}
	if _, err := store.setPolicyMode(opID, apiIdentityModeEnforce, "tester", "validated again"); err != nil {
		t.Fatal(err)
	}
	issuer := api5Issuer()
	issuer.Audiences = []string{"payments-api-v2"}
	if _, err := store.upsertIssuer(issuer, "tester"); err != nil {
		t.Fatal(err)
	}
	policies := store.listPolicies()
	if len(policies) != 1 || policies[0].Mode != apiIdentityModeDetect {
		t.Fatalf("issuer trust change must reset dependent policy to DETECT: %+v", policies)
	}
}

func TestAPI5ReferencedIssuerCannotBeDisabledAndMissingRuntimeFailsClosed(t *testing.T) {
	fx := newAPI5JWTFixture("k1", 11)
	store, opID := api5StoreWithPolicy(t, func(context.Context, string) ([]byte, error) { return fx.jwks(), nil })
	issuer := api5Issuer()
	issuer.Enabled = false
	if _, err := store.upsertIssuer(issuer, "tester"); err == nil {
		t.Fatal("referenced issuer must not be disableable")
	}
	if _, err := store.setPolicyMode(opID, apiIdentityModeEnforce, "tester", "validated"); err != nil {
		t.Fatal(err)
	}
	// Simulate corrupt/legacy persisted state with the issuer runtime missing.
	store.mu.Lock()
	store.runtimes = map[string]*jwtIssuerRuntime{}
	store.runtime.Store(&apiIdentityRuntimeSnapshot{ByOperation: map[string]apiIdentityRuntimePolicy{
		opID: {Policy: store.policies[opID], Issuer: nil},
	}})
	store.mu.Unlock()
	req := httptest.NewRequest(http.MethodPost, "https://api.test/api/payment", nil)
	req.Header.Set("Authorization", "Bearer "+fx.token(t, api5Claims()))
	rr := httptest.NewRecorder()
	store.wrap("default", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing issuer runtime must fail closed in ENFORCE, status=%d", rr.Code)
	}
}

func TestAPI5RejectsMalformedJWTProtectedHeaders(t *testing.T) {
	fx := newAPI5JWTFixture("k1", 11)
	claims := api5Claims()
	cases := map[string]map[string]any{
		"crit-string":   {"alg": "EdDSA", "kid": fx.kid, "typ": "JWT", "crit": "b64"},
		"crit-nonempty": {"alg": "EdDSA", "kid": fx.kid, "typ": "JWT", "crit": []string{"b64"}},
		"b64-false":     {"alg": "EdDSA", "kid": fx.kid, "typ": "JWT", "b64": false},
		"b64-string":    {"alg": "EdDSA", "kid": fx.kid, "typ": "JWT", "b64": "true"},
	}
	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			token := fx.tokenWithHeader(t, header, claims)
			if _, err := parseJWT(token); err == nil {
				t.Fatalf("expected protected-header rejection for %s", name)
			}
		})
	}
}

func TestAPI5KidMissRefreshIsBoundedAgainstUnknownKidStorm(t *testing.T) {
	trusted := newAPI5JWTFixture("trusted", 12)
	attacker := newAPI5JWTFixture("unknown", 13)
	var calls atomic.Int32
	fetch := func(context.Context, string) ([]byte, error) {
		calls.Add(1)
		return trusted.jwks(), nil
	}
	store, _ := api5StoreWithPolicy(t, fetch)
	store.mu.RLock()
	rt := store.runtimes["corp"]
	store.mu.RUnlock()
	for i := 0; i < 20; i++ {
		claims := api5Claims()
		_, err := rt.verify(context.Background(), attacker.token(t, claims), fetch)
		if err == nil {
			t.Fatal("unknown kid unexpectedly verified")
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("unknown-kid storm caused %d JWKS fetches; want initial fetch + one bounded key-miss refresh", got)
	}
}

func TestAPI5RejectsAmbiguousJWKSKeyID(t *testing.T) {
	a := newAPI5JWTFixture("dup", 14)
	b := newAPI5JWTFixture("dup", 15)
	var da, db struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(a.jwks(), &da); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b.jwks(), &db); err != nil {
		t.Fatal(err)
	}
	doc, _ := json.Marshal(map[string]any{"keys": []json.RawMessage{da.Keys[0], db.Keys[0]}})
	keys, err := parseJWKS(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selectJWTKey(&jwksKeySet{Keys: keys}, "dup", "EDDSA"); err == nil {
		t.Fatal("duplicate kid must fail closed as ambiguous")
	}
}

func TestAPI5OperatorPolicySetsFailClosedInsteadOfTruncating(t *testing.T) {
	issuer := api5Issuer()
	issuer.Audiences = make([]string, 65)
	for i := range issuer.Audiences {
		issuer.Audiences[i] = fmt.Sprintf("aud-%d", i)
	}
	if _, err := normalizeIssuerConfig(issuer, "tester", nil); err == nil {
		t.Fatal("oversized audience policy must be rejected, not truncated")
	}
	policy := APIIdentityPolicy{OperationID: "op", IssuerID: "corp", RequiredRoles: []string{strings.Repeat("r", 257)}}
	if _, err := normalizeIdentityPolicy(policy, "tester", nil); err == nil {
		t.Fatal("oversized required role must be rejected, not silently dropped")
	}
}

func TestAPI5RejectsTrailingJWTJSONData(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"EdDSA"}{"extra":true}`))
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"https://issuer.example.test"}`))
	token := header + "." + claims + "." + base64.RawURLEncoding.EncodeToString([]byte{1})
	if _, err := parseJWT(token); err == nil {
		t.Fatal("JWT JSON segment with trailing data must be rejected")
	}
}

func TestAPI5PSSAndECDSAJWKSVerification(t *testing.T) {
	t.Run("PS256", func(t *testing.T) {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		e := []byte{1, 0, 1}
		jwks, _ := json.Marshal(map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "ps1", "use": "sig", "alg": "PS256",
			"n": base64.RawURLEncoding.EncodeToString(priv.PublicKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(e),
		}}})
		issuer := api5Issuer()
		issuer.AllowedAlgs = []string{"PS256"}
		store := newAPIIdentityStore()
		store.fetchJWKS = func(context.Context, string) ([]byte, error) { return jwks, nil }
		if _, err := store.upsertIssuer(issuer, "tester"); err != nil {
			t.Fatal(err)
		}
		store.mu.RLock()
		rt := store.runtimes["corp"]
		store.mu.RUnlock()
		hb, _ := json.Marshal(map[string]any{"alg": "PS256", "kid": "ps1"})
		cb, _ := json.Marshal(api5Claims())
		signing := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(cb)
		h := sha256.Sum256([]byte(signing))
		sig, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, h[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := rt.verify(context.Background(), signing+"."+base64.RawURLEncoding.EncodeToString(sig), store.fetchJWKS); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ES256", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		sz := 32
		x := priv.PublicKey.X.FillBytes(make([]byte, sz))
		y := priv.PublicKey.Y.FillBytes(make([]byte, sz))
		jwks, _ := json.Marshal(map[string]any{"keys": []any{map[string]any{
			"kty": "EC", "kid": "ec1", "use": "sig", "alg": "ES256", "crv": "P-256",
			"x": base64.RawURLEncoding.EncodeToString(x), "y": base64.RawURLEncoding.EncodeToString(y),
		}}})
		issuer := api5Issuer()
		issuer.AllowedAlgs = []string{"ES256"}
		store := newAPIIdentityStore()
		store.fetchJWKS = func(context.Context, string) ([]byte, error) { return jwks, nil }
		if _, err := store.upsertIssuer(issuer, "tester"); err != nil {
			t.Fatal(err)
		}
		store.mu.RLock()
		rt := store.runtimes["corp"]
		store.mu.RUnlock()
		hb, _ := json.Marshal(map[string]any{"alg": "ES256", "kid": "ec1"})
		cb, _ := json.Marshal(api5Claims())
		signing := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(cb)
		h := sha256.Sum256([]byte(signing))
		r, ss, err := ecdsa.Sign(rand.Reader, priv, h[:])
		if err != nil {
			t.Fatal(err)
		}
		sig := append(r.FillBytes(make([]byte, sz)), ss.FillBytes(make([]byte, sz))...)
		if _, err := rt.verify(context.Background(), signing+"."+base64.RawURLEncoding.EncodeToString(sig), store.fetchJWKS); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAPI5ConcurrentKidMissRefreshIsSingleFlight(t *testing.T) {
	trusted := newAPI5JWTFixture("trusted", 16)
	attacker := newAPI5JWTFixture("unknown", 17)
	var calls atomic.Int32
	fetch := func(context.Context, string) ([]byte, error) {
		calls.Add(1)
		time.Sleep(2 * time.Millisecond)
		return trusted.jwks(), nil
	}
	store, _ := api5StoreWithPolicy(t, fetch)
	store.mu.RLock()
	rt := store.runtimes["corp"]
	store.mu.RUnlock()
	token := attacker.token(t, api5Claims())
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := rt.verify(context.Background(), token, fetch); err == nil {
				t.Errorf("unknown kid unexpectedly verified")
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := calls.Load(); got != 2 {
		t.Fatalf("concurrent unknown-kid requests caused %d JWKS fetches; want one initial + one key-miss refresh", got)
	}
}
