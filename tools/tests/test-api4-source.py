#!/usr/bin/env python3
from pathlib import Path
import re, sys

ROOT = Path(__file__).resolve().parents[2]
checks = []

def req(cond, msg):
    if not cond:
        print(f"API4_SOURCE_GATE_FAIL: {msg}", file=sys.stderr)
        sys.exit(1)
    checks.append(msg)

src = (ROOT / "positive_schema_api4.go").read_text()
main = (ROOT / "main.go").read_text()
admin = (ROOT / "admin.go").read_text()
profiles = (ROOT / "profiles.go").read_text()
tests = (ROOT / "positive_schema_api4_test.go").read_text()
ui = (ROOT / "static/admin.html").read_text()
persist = (ROOT / "api_security_persist.go").read_text()

req("type PositiveSchemaProfileVersion struct" in src, "immutable profile-version model exists")
req("type PositiveSchemaDeployment struct" in src, "deployment model exists")
req("type PositiveSchemaException struct" in src, "exception model exists")
req("type PositiveSchemaViolation struct" in src, "violation evidence model exists")
for mode in ("LEARN", "DETECT", "ENFORCE"):
    req(mode in src, f"{mode} mode exists")
req('candidate.Status != "REVIEWED"' in src and 'candidate.Review.Source != "operator"' in src, "promotion requires operator-reviewed candidate")
req("candidate.Review.CandidateVersion != candidate.Version" in src, "stale candidate review rejected")
req("validPositiveModeTransition" in src and "ENFORCE requires an active DETECT stage first" in src, "LEARN/DETECT/ENFORCE transition gate exists")
req("atomic.Pointer[positiveSchemaRuntimeSnapshot]" in src, "data-plane policy lookup uses atomic snapshot")
req("positiveSchemaMaxViolations" in src and "positiveSchemaMaxExceptions" in src and "positiveSchemaMaxHistory" in src, "state growth is bounded")
req("BODY_TOO_LARGE_FOR_SCHEMA_VALIDATION" in src, "truncated body fails positive validation")
req("MALFORMED_BODY" in src, "malformed JSON evidence exists")
for typ in ("REQUIRED_FIELD_MISSING", "TYPE_MISMATCH", "FORMAT_MISMATCH", "ENUM_MISMATCH", "UNKNOWN_FIELD", "CONTENT_TYPE_MISMATCH"):
    req(typ in src, f"{typ} validator exists")
req("Exempted" in src and "ExceptionID" in src and "applyPositiveExceptions" in src, "exceptions are applied and evidenced")
req("request rejected by API schema policy" in src, "ENFORCE has generic block response")
req("api-positive-schema.json" in src, "API-4 state has dedicated durable file")
req("bodyTruncated bool" in profiles and "passiveBodyLimit+1" in profiles, "shared body capture records truncation without losing replay bytes")
req("positiveHandler = s.positiveSchema.wrap(siteName, positiveHandler)" in main, "API-4 middleware is wired into site runtime")
req("relationshipHandler := positiveHandler" in main and "graphqlHandler := relationshipHandler" in main and "requestBodyPrefixWrap(captureForAI, cfg.PassiveDiscoveryEnabled, true, graphqlHandler)" in main, "API-4 remains inside bounded body capture through later API-7.2/API-8 wrapper composition")
req("*positiveSchemaStore" in main and "newPositiveSchemaStore()" in main, "server owns initialized API-4 store")
req("s.positiveSchema.load(*configPath)" in main, "API-4 state restored at startup")
req("positive.save(configPath)" in persist, "API-4 participates in autosave")
for route in (
    'POST /api/security/schema/{id}/enforcement',
    'GET /api/security/schema/enforcement',
    'GET /api/security/schema/enforcement/violations',
    'POST /api/security/schema/enforcement/{operation_id}/mode',
    'POST /api/security/schema/enforcement/{operation_id}/activate',
    'POST /api/security/schema/enforcement/{operation_id}/rollback',
    'POST /api/security/schema/enforcement/{operation_id}/exceptions',
):
    req(route in admin, f"route wired: {route}")
req("Positive schema enforcement" in ui and "api-schema-promote" in ui and "api-enforce-next" in ui, "admin UI exposes promotion and mode workflow")
req("AI review never activates this table" in ui, "UI states AI non-authority boundary")
for testname in (
    "TestAPI4PromotionRequiresCurrentOperatorReview",
    "TestAPI4ModeLifecycleRequiresDetectBeforeEnforce",
    "TestAPI4DetectShadowsAndEnforceBlocks",
    "TestAPI4ExceptionBypassesSpecificViolation",
    "TestAPI4RollbackRestoresPreviousProfileAndMode",
    "TestAPI4PersistenceRestoresRuntimePolicyAndViolations",
    "TestAPI4OversizedBodyCannotPassOnPrefixOnly",
):
    req(testname in tests, f"deterministic test exists: {testname}")
req("reviewSchemaCandidate" not in src and ".ai." not in src, "API-4 runtime does not call AI")
print(f"API4_SOURCE_GATE_PASS checks={len(checks)}")
