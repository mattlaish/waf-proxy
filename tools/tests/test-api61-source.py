#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
checks = []


def req(cond, msg):
    if not cond:
        print(f"API61_SOURCE_GATE_FAIL: {msg}", file=sys.stderr)
        sys.exit(1)
    checks.append(msg)


src = (ROOT / "sequence_api61.go").read_text(encoding="utf-8")
tests = (ROOT / "sequence_api61_test.go").read_text(encoding="utf-8")
main = (ROOT / "main.go").read_text(encoding="utf-8")
admin = (ROOT / "admin.go").read_text(encoding="utf-8")
persist = (ROOT / "api_security_persist.go").read_text(encoding="utf-8")
ui = (ROOT / "static/admin.html").read_text(encoding="utf-8")
build = (ROOT / "build.sh").read_text(encoding="utf-8")
ci = (ROOT / ".github/workflows/ci.yml").read_text(encoding="utf-8")

for model in ("SequenceSession", "SequenceTransition", "SequenceModel"):
    req(f"type {model} struct" in src, f"{model} foundation exists")
req("apiOperationID(site, r.Method, normalizeAPIOperationPath(r.URL.Path))" in src, "nodes use API-1 normalized operation IDs")
req("verifiedAPIIdentity(r)" in src and 'kind = "verified_identity"' in src, "identity correlation uses API-5 verified request context")
req("anonymousSequenceMaterial" in src and "hmac.New(sha256.New, key)" in src, "anonymous correlation is keyed and privacy-preserving")
req('r.Header.Get("Authorization")' not in src, "sequence foundation never reads raw Authorization data")
req("sequenceDefaultMaxSessions" in src and "sequenceDefaultMaxTransitions" in src and "sequenceDefaultRecentOps" in src, "session, transition, and history cardinality are bounded")
req("sequenceDefaultSessionTTL" in src and "sequenceDefaultTransitionTTL" in src and "pruneLocked" in src, "session and transition TTL pruning exists")
req("select {\n\tcase s.queue <- ev:" in src and "s.dropped.Add(1)" in src, "data-plane observation enqueue is non-blocking and counted")
req("atomic.Pointer[SequenceModel]" in src and "s.runtime.Store(&model)" in src, "runtime publishes immutable atomic model snapshots")
req('apiSecurityStatePath(configPath, "api-sequence.json")' in src, "sequence state uses a dedicated persistence file")
req('apiSecurityStatePath(configPath, "api-sequence.key")' in src and "f.Chmod(0o600)" in src, "pseudonymization key persists separately with restrictive mode")
req("Raw" not in src[src.find("type sequenceStateFile struct"):src.find("type sequenceStore struct")], "persistence model excludes raw identity/session inputs")
req("*sequenceStore" in main and "newSequenceStore()" in main, "server owns initialized API-6.1 store")
req("s.sequence.load(*configPath)" in main and "s.sequence.start()" in main, "sequence state restores before asynchronous collection starts")
req("handler = s.sequence.wrap(siteName, handler)" in main, "sequence telemetry is wired into the data plane")
req(main.find("handler = s.sequence.wrap(siteName, handler)") < main.find("handler = s.identity.wrap(siteName, handler)"), "sequence executes after API-5 has attached verified identity")
req("s.sequence.stopAndDrain()" in main, "shutdown drains accepted sequence observations")
req("sequence.save(configPath)" in persist, "sequence state participates in autosave and final save")
for route in (
    'GET /api/security/sequence/model',
    'GET /api/security/sequence/sessions',
    'GET /api/security/sequence/transitions',
):
    req(route in admin, f"admin visibility route wired: {route}")
req('authRole(roleReviewer, a.handleSequenceSessions)' in admin and 'authRole(roleReviewer, a.handleSequenceTransitions)' in admin, "session and transition detail requires Reviewer RBAC")
req("api_sequence.sessions_view" in src and "api_sequence.transitions_view" in src, "sensitive sequence visibility is audited")
req("API sequence foundation" in ui and "/api/security/sequence/model" in ui, "embedded admin UI exposes API-6.1 visibility")
req("Sequence analytics never blocks traffic" in ui, "UI retains the API-6.1 non-enforcement authority boundary")
req("test-api61-source.py" in build and "test-api61-source.py" in ci, "API-6.1 source gate is retained by build.sh and CI")
for testname in (
    "TestAPI61UsesNormalizedOperationIDsAndVerifiedIdentityOnly",
    "TestAPI61DeterministicSessionAndTransitionFoundation",
    "TestAPI61BoundedStateAndTTL",
    "TestAPI61RestartPersistenceExcludesRawIdentityAndAnonymousInputs",
    "TestAPI61ConcurrentNonBlockingRuntimeSnapshot",
):
    req(testname in tests, f"targeted API-6.1 test exists: {testname}")

print(f"API61_SOURCE_GATE_PASS checks={len(checks)}")
