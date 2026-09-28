#!/usr/bin/env python3
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[2]
read = lambda p: (ROOT / p).read_text(encoding="utf-8")
checks = []

def req(cond, msg):
    if not cond:
        raise SystemExit("CODE_DUPLICATION_REVIEW_SOURCE_GATE_FAIL: " + msg)
    checks.append(msg)

# Removed duplicate/dead compatibility layers must not reappear.
removed = (
    "debug_evidence.go", "debug_evidence_ops_v2.go", "support_bundle_v2.go",
    "security_state.go", "security_event.go", "deployment_readiness.go",
    "reliability.go", "vectorscan_qualification.go", "vectorscan_audit_store.go",
    "vectorscan_transition_audit.go", "schema_api2_continuation.go",
    "pki_hardening.go", "internal/coverage/report.go",
)
for path in removed:
    req(not (ROOT / path).exists(), "removed duplicate/dead layer stays absent: " + path)

removed_tests = (
    "debug_evidence_test.go", "debug_evidence_ops_v2_test.go", "security_state_test.go",
    "reliability_test.go", "vectorscan_qualification_test.go", "pki_hardening_test.go",
)
for path in removed_tests:
    req(not (ROOT / path).exists(), "obsolete test for removed layer stays absent: " + path)

# Debug evidence has one runtime store and one sanitization path.
debug = read("debug_bundle.go")
sanitize = read("debug_sanitize.go")
observer = read("coraza_observer.go")
support = read("support_api.go")
for token in ("type DebugEvidenceStore struct", "func (s *DebugEvidenceStore) Put", "func (s *DebugEvidenceStore) ExportIncident"):
    req(token in debug, "authoritative debug store retained: " + token)
for token in ("isSensitiveDebugKey", "sanitizeDebugValue", "sanitizeDebugMap"):
    req(token in sanitize, "single debug sanitization path retained: " + token)
for token in ("DebugEvidenceCapture", "currentDebugEvidenceCapture", "DebugEvidenceOpsV2", "TenantExport"):
    req(token not in debug + sanitize + observer + support, "legacy debug compatibility path absent: " + token)
req("currentDebugEvidenceStore" in observer, "Coraza debug evidence uses authoritative store")

# SecLang token/action parsing is shared by offline coverage and live VectorScan.
shared = read("internal/capability/seclang.go")
coverage = read("internal/coverage/crs/parser.go")
vector = read("internal/vectoraccel/rules.go")
for token in ("func NextToken", "func SplitActions", "func SplitAction", "func TrimActionValue"):
    req(token in shared, "shared SecLang helper exists: " + token)
for token in ("shared.NextToken", "shared.SplitActions", "shared.SplitAction", "shared.TrimActionValue"):
    req(token in coverage, "coverage parser uses shared helper: " + token)
    req(token in vector, "VectorScan parser uses shared helper: " + token)
for token in ("func nextToken", "func splitActions", "func splitActionForCapability", "func trimQuotedActionValue"):
    req(token not in coverage + vector, "duplicate SecLang helper removed: " + token)

# Superseding implementations remain present.
req("func (a *adminServer) handleDoctor" in support, "Doctor remains the deployment readiness surface")
req("func validateCRLURLSyntax" in read("pki_url.go"), "wired PKI URL hardening remains")
req("func prepareCRLStore" in read("pki_url.go"), "wired PKI CRL store remains")
req("func (p *SitePlan) QualificationCandidates" in read("internal/vectoraccel/qualification.go"), "live VectorScan qualification path remains")
req("BuildCoverageReportV2" in read("internal/coverage/crs/report.go"), "current CRS coverage report remains")
req("type schemaStore struct" in read("schema_api2.go"), "live API-2 schema store remains")

# Removed placeholder models/symbols must not exist anywhere in production Go source.
prod = "\n".join(p.read_text(encoding="utf-8") for p in ROOT.rglob("*.go") if not p.name.endswith("_test.go"))
for token in (
    "DebugEvidenceCapture", "DebugEvidenceOpsV2", "SupportBundleProvenance",
    "securityStateStore", "securityEventExporter", "DeploymentReadinessReport",
    "ReliabilityReport", "VectorScanQualificationRecord", "VectorScanAuditStore",
    "VectorScanTransitionAudit", "SchemaObservationLearning", "crlFetchPolicy",
    "QualificationPassed", "TransformCapabilities", "CompareCandidates",
    "QualificationResult", "ActualListenerKey", "nonAnonymousAuthSamples",
    "sequenceTransitionID", "ClientIdentityAuditSink",
):
    req(re.search(r"\b" + re.escape(token) + r"\b", prod) is None, "obsolete symbol absent: " + token)

# Admin route authority must remain unique. Duplicate METHOD+pattern registrations
# would mean two handlers competing for the same control-plane function.
route_text = read("admin.go") + "\n" + read("update.go")
routes = re.findall(r'mux\.HandleFunc\("([A-Z]+) ([^"\\]+)"', route_text)
seen = set()
dups = []
for route in routes:
    if route in seen:
        dups.append(route)
    seen.add(route)
req(not dups, "Admin METHOD+path registrations remain unique")
req(len(routes) >= 120, "Admin route inventory remains complete after consolidation")

# Authority separation is unchanged: learned sequence/BOLA evidence is not a
# second enforcement stack; deterministic GraphQL/positive-schema/identity stay separate.
api6 = read("sequence_api63.go") + read("sequence_api64.go")
api7 = read("bola_detection_api73.go") + read("bola_policy_api74.go")
api8 = read("graphql_api8.go")
req('Mode string `json:"mode"`' in api6 and 'sequenceDetectionDetect' in api6, "sequence control remains LEARN/DETECT authority")
req("InferenceBlocking" in api7 and "false" in api7, "BOLA inferred evidence remains non-blocking")
req("graphqlModeEnforce" in api8, "GraphQL explicit deterministic ENFORCE authority remains")

# Documentation and gate wiring.
req((ROOT / "CODE_DUPLICATION_REVIEW.md").exists(), "duplication review document exists")
req("test-code-duplication-review-source.py" in read("build.sh"), "build.sh runs duplication review gate")
req("test-code-duplication-review-source.py" in read(".github/workflows/ci.yml"), "CI runs duplication review gate")

print(f"CODE_DUPLICATION_REVIEW_SOURCE_GATE_PASS checks={len(checks)} routes={len(routes)}")
