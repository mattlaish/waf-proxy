#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
checks = []

def req(cond, msg):
    if not cond:
        print(f"API5_SOURCE_GATE_FAIL: {msg}", file=sys.stderr)
        sys.exit(1)
    checks.append(msg)

src = (ROOT / "identity_api5.go").read_text()
main = (ROOT / "main.go").read_text()
admin = (ROOT / "admin.go").read_text()
persist = (ROOT / "api_security_persist.go").read_text()
tests = (ROOT / "identity_api5_test.go").read_text()
ui = (ROOT / "static/admin.html").read_text()
build = (ROOT / "build.sh").read_text()
ci = (ROOT / ".github/workflows/ci.yml").read_text()

req("type JWTIssuerConfig struct" in src, "trusted issuer model exists")
req("type APIIdentityPolicy struct" in src, "operation identity policy model exists")
req("type VerifiedAPIIdentity struct" in src, "verified request identity model exists")
req("type APIIdentityViolation struct" in src, "bounded identity evidence model exists")
req("atomic.Pointer[apiIdentityRuntimeSnapshot]" in src, "runtime policy lookup uses atomic snapshot")
req("atomic.Pointer[jwksKeySet]" in src, "JWKS cache is atomic and read-mostly")
req("refreshMu" in src and "sync.Mutex" in src, "JWKS refresh is single-flight per issuer runtime")
req("apiIdentityMaxViolations" in src and "apiIdentityMaxJWKSBytes" in src and "apiIdentityMaxTokenBytes" in src, "identity/JWKS/token state is bounded")
req('u.Scheme != "https"' in src and "JWKS redirect must remain HTTPS" in src, "JWKS retrieval is HTTPS-only including redirects")
for alg in ("RS256", "PS256", "ES256", "EDDSA"):
    req(alg in src, f"{alg} validation path exists")
req("rsa.VerifyPKCS1v15" in src and "rsa.VerifyPSS" in src, "RSA PKCS1v1.5 and PSS verification exists")
req("ecdsa.Verify" in src and "ed25519.Verify" in src, "ECDSA and Ed25519 verification exists")
req("PSSSaltLengthEqualsHash" in src, "RSA-PSS uses JWT-required salt length")
req("JWT alg missing or forbidden" in src and 'alg == "NONE"' in src, "alg=none is rejected")
req("JWT critical header extensions are not supported" in src and "unencoded JWS payloads are not supported" in src, "unsupported JWS critical/b64 modes fail closed")
req('critRaw, exists := header["crit"]' in src and '!ok || len(crit) > 0' in src, "malformed or non-empty crit header fails closed")
req('b64Raw, exists := header["b64"]' in src and '!ok || !b64' in src, "malformed or b64=false protected header fails closed")
for claim in ("iss", "aud", "exp", "nbf"):
    req(f'"{claim}"' in src, f"{claim} validation exists")
req("JWT_ISSUER_INVALID" in src and "JWT_AUDIENCE_INVALID" in src and "JWT_EXPIRED" in src and "JWT_NOT_YET_VALID" in src, "standard claim failures are typed")
req("claimString(jwt.Claims, rt.config.SubjectClaim)" in src, "subject enters context only after signature and claim validation")
req("context.WithValue(r.Context(), apiIdentityContextKey{}, id)" in src, "verified identity is attached to request context")
req("ROLE_MISSING" in src and "SCOPE_MISSING" in src and "TENANT_MISMATCH" in src and "CLIENT_ID_FORBIDDEN" in src, "identity-aware role/scope/tenant/client policy exists")
req("identitySubjectHash" in src and "SubjectHash" in src, "persisted evidence pseudonymizes subject identity")
req('apiSecurityStatePath(configPath, "api-identity.json")' in src, "API-5 state has dedicated durable file")
req("Keys" not in src[src.find("type apiIdentityStateFile struct"):src.find("type apiIdentityContextKey")], "JWKS keys are not part of persistence model")
req("new identity policy must start in DETECT" in src, "new policy cannot start in ENFORCE")
req("policy semantics changed; returned to DETECT" in src, "policy semantic changes invalidate ENFORCE")
req("issuer trust semantics changed; returned to DETECT" in src, "issuer semantic changes invalidate dependent ENFORCE")
req("issuer cannot be disabled while identity policies reference it" in src, "referenced issuer cannot be disabled")
req("JWT_ISSUER_UNAVAILABLE" in src and "StatusServiceUnavailable" in src, "missing issuer/JWKS runtime fails closed under ENFORCE")
req("A kid miss commonly means key rotation" in src and "refreshForKeyMiss" in src and "apiIdentityKidMissRefreshMin" in src, "kid miss triggers bounded single-flight JWKS refresh")
req("*apiIdentityStore" in main and "newAPIIdentityStore()" in main, "server owns initialized API-5 store")
req("s.identity.load(*configPath)" in main, "API-5 state restores at startup")
req("handler = s.identity.wrap(siteName, handler)" in main, "API-5 middleware is wired into data plane")
req(main.find("handler = s.identity.wrap(siteName, handler)") < main.find("handler = l7Abuse.wrap(siteName, handler)"), "identity policy executes inside abuse gate and before schema/Coraza/backend")
req("identity.save(configPath)" in persist, "API-5 participates in autosave")
for route in (
    'GET /api/security/identity/issuers',
    'POST /api/security/identity/issuers',
    'POST /api/security/identity/issuers/{issuer_id}/refresh',
    'GET /api/security/identity/policies',
    'POST /api/security/identity/policies/{operation_id}',
    'POST /api/security/identity/policies/{operation_id}/mode',
    'DELETE /api/security/identity/policies/{operation_id}',
    'GET /api/security/identity/violations',
):
    req(route in admin, f"route wired: {route}")
req("JWT + identity-aware API security" in ui and "api-op-identity" in ui and "api-idpol-mode" in ui, "admin UI exposes issuer and operation identity workflow")
req("Only cryptographically verified claims enter policy context" in ui, "UI states verified-claims-only boundary")
req("test-api5-source.py" in build and "test-api5-source.py" in ci, "API-5 source gate is wired into local build and CI")
req("test-api12-source.py" in build and "test-api3-source.py" in build, "local build retains earlier API source gates")
req("test-api12-source.py" in ci and "test-api3-source.py" in ci, "CI retains earlier API source gates")
for testname in (
    "TestAPI5VerifiedClaimsOnlyEnterRequestContext",
    "TestAPI5DetectThenEnforceInvalidJWT",
    "TestAPI5AuthorizationRequirementsUseVerifiedClaims",
    "TestAPI5IssuerAudienceExpiryAndNBFValidation",
    "TestAPI5KidMissRefreshesJWKSOnceForRotation",
    "TestAPI5PersistenceExcludesTokensAndJWKSCache",
    "TestAPI5PolicyLifecycleStartsDetect",
    "TestAPI5RS256JWKSVerification",
    "TestAPI5SemanticChangesReturnEnforcementToDetect",
    "TestAPI5ReferencedIssuerCannotBeDisabledAndMissingRuntimeFailsClosed",
    "TestAPI5RejectsMalformedJWTProtectedHeaders",
    "TestAPI5KidMissRefreshIsBoundedAgainstUnknownKidStorm",
    "TestAPI5RejectsAmbiguousJWKSKeyID",
    "TestAPI5OperatorPolicySetsFailClosedInsteadOfTruncating",
    "TestAPI5RejectsTrailingJWTJSONData",
    "TestAPI5PSSAndECDSAJWKSVerification",
    "TestAPI5ConcurrentKidMissRefreshIsSingleFlight",
):
    req(testname in tests, f"deterministic test exists: {testname}")
print(f"API5_SOURCE_GATE_PASS checks={len(checks)}")
