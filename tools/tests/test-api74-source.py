#!/usr/bin/env python3
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
src=(ROOT/'bola_policy_api74.go').read_text(encoding='utf-8')
tests=(ROOT/'bola_policy_api74_test.go').read_text(encoding='utf-8')
api73=(ROOT/'bola_detection_api73.go').read_text(encoding='utf-8')
rel=(ROOT/'object_relationship_api72.go').read_text(encoding='utf-8')
main=(ROOT/'main.go').read_text(encoding='utf-8')
admin=(ROOT/'admin.go').read_text(encoding='utf-8')
persist=(ROOT/'api_security_persist.go').read_text(encoding='utf-8')
ui=(ROOT/'static/admin.html').read_text(encoding='utf-8')
build=(ROOT/'build.sh').read_text(encoding='utf-8')
ci=(ROOT/'.github/workflows/ci.yml').read_text(encoding='utf-8')

checks=[]
def req(cond,msg):
    if not cond:
        raise SystemExit('API74_SOURCE_GATE_FAIL: '+msg)
    checks.append(msg)

# Data model: policy is evidence routing only.
for token in ('type BOLAPolicy struct','OperationID','LocatorID','CandidateType','MinConfidence','Action','Enabled','CreatedAt','UpdatedAt','ExpiresAt'):
    req(token in src,f'policy model includes {token}')
req('evidence-handling policy, not a request authorization or' in src,'policy comment declares evidence-only authority')
req('bolaPolicyActionReview   = "REVIEW"' in src,'REVIEW policy action exists')
req('bolaPolicyActionSuppress = "SUPPRESS"' in src,'SUPPRESS policy action exists')
req('action must be REVIEW or SUPPRESS' in src,'policy mutation rejects other actions')
req('BOLAPolicy' in src and 'IdentityFingerprint' not in src[src.index('type BOLAPolicy struct'):src.index('// BOLAEvidenceReview')], 'policy model has no identity fingerprint selector')
policy_block=src[src.index('type BOLAPolicy struct'):src.index('// BOLAEvidenceReview')]
for forbidden in ('ObjectFingerprint','TenantFingerprint','ClientFingerprint','RawObject','Owner','Header','Cookie','Authorization'):
    req(forbidden not in policy_block,f'policy model excludes raw/identity selector {forbidden}')

# Structured evidence workflow, no arbitrary text notes.
for token in ('type BOLAEvidenceReview struct','CandidateID','State','ReasonCode','EvidenceCountAtReview','CandidateLastSeen','UpdatedAt','ExpiresAt'):
    req(token in src,f'evidence review includes {token}')
for state in ('OPEN','ACKNOWLEDGED','DISMISSED','RESOLVED'):
    req(f'"{state}"' in src,f'workflow state exists: {state}')
for reason in ('INVESTIGATING','EXPECTED_SHARED_RESOURCE','AUTHORIZED_CROSS_TENANT','TEST_TRAFFIC','FALSE_POSITIVE','FIX_DEPLOYED','OTHER_REVIEWED'):
    req(f'"{reason}"' in src,f'enumerated workflow reason exists: {reason}')
req(('free-text' in src) or ('free-\n// text' in src),'source documents no free-text workflow notes')
req('reason_code is required for reviewed workflow states' in src,'reviewed states require structured reason')
req('unsupported reason_code' in src,'unrecognized workflow reason rejected')

# Evidence view joins detector candidate without mutating/removing detector evidence.
for token in ('type BOLAEvidenceView struct','BOLACandidate `json:"candidate"`','EffectiveAction','WorkflowState','Reopened','Suppressed'):
    req(token in src,f'evidence view includes {token}')
req('EffectiveAction: bolaPolicyActionReview' in src,'default evidence action is REVIEW')
req('view.Suppressed = p.Action == bolaPolicyActionSuppress' in src,'SUPPRESS is evidence visibility/routing metadata')
req('candidates.snapshot()' in src,'API-7.4 reads API-7.3 evidence snapshot')
req('delete(candidates' not in src and 'delete(srv.bolaCandidates' not in src,'API-7.4 never deletes detector candidates')
req('CandidateLastSeen' in src and 'c.LastSeen.After(r.CandidateLastSeen)' in src,'new detector evidence reopens dismissed/resolved workflow')
req('view.WorkflowState = bolaWorkflowOpen' in src and 'view.Reopened = true' in src,'reopened evidence returns to OPEN state')

# Deterministic, bounded policy matching.
req('bolaPolicyID(operationID, locatorID, candidateType, minConfidence string)' in src,'policy ID derives from normalized scope')
req('policySpecificity' in src,'policy precedence is explicit')
req('p.LocatorID != ""' in src and 'p.CandidateType != ""' in src,'specificity prefers narrower locator/type scope')
req('confidenceRank(c.Confidence) < confidenceRank(p.MinConfidence)' in src,'minimum confidence policy filter enforced')
req('p.OperationID != c.OperationID' in src,'policy scoped to normalized operation')
req('p.LocatorID != "" && p.LocatorID != c.LocatorID' in src,'optional locator scope enforced')
req('p.CandidateType == "" || p.CandidateType == c.Type' in src,'optional candidate-type scope enforced')
req('locator_id must belong to operation_id' in src,'locator scope cross-validated against API-7.1 operation')

# Resource bounds and lifecycle.
req('bolaPolicyMaxRules     = 1024' in src,'policy cardinality bounded')
req('bolaReviewMaxRecords   = 4096' in src,'review cardinality bounded')
req('bolaPolicyDefaultTTL   = 30 * 24 * time.Hour' in src,'policy default TTL bounded')
req('bolaPolicyMaxTTL       = 180 * 24 * time.Hour' in src,'policy maximum TTL bounded')
req('bolaReviewDefaultTTL   = 30 * 24 * time.Hour' in src,'review TTL bounded')
req('len(s.policies) >= bolaPolicyMaxRules' in src,'policy cap enforced')
req('len(s.reviews) >= bolaReviewMaxRecords' in src,'review cap enforced')
req('ttl < time.Minute || ttl > bolaPolicyMaxTTL' in src,'policy TTL range enforced')
req('s.pruneLocked(now)' in src,'policy/review state TTL pruning invoked')
req('!p.ExpiresAt.After(now)' in src and '!r.ExpiresAt.After(now)' in src,'expired policy/review records removed')
req('sync.RWMutex' in src,'API-7.4 state is concurrency protected')

# Restart-safe persistence.
req('bolaPolicyStateVersion = 1' in src,'API-7.4 state is versioned')
req('api-bola-policy.json' in src,'API-7.4 has dedicated durable state file')
req('state.Version != 0 && state.Version != bolaPolicyStateVersion' in src,'unknown API-7.4 state versions fail closed')
req('validBOLAPolicy(p, now)' in src,'restored policies revalidated')
req('validBOLAEvidenceReview(r, now)' in src,'restored reviews revalidated')
req('bolaPolicy          *bolaPolicyStore' in main,'server owns API-7.4 store')
req('bolaPolicy:          newBOLAPolicyStore()' in main,'server initializes API-7.4 store')
req('s.bolaPolicy.load(*configPath)' in main,'API-7.4 state restores before traffic')
req('bolaPolicy.save(configPath)' in persist,'API security autosave persists API-7.4 state')
req('s.bolaPolicy' in main[main.index('startAPISecurityAutosave'):], 'main autosave call includes API-7.4 store')

# API-7.4 is not on the request-path/detection authority chain.
req('bolaPolicy' not in rel,'API-7.2 request/background relationship processor does not consult API-7.4 policy')
req('bolaPolicy' not in api73,'API-7.3 detector does not consult API-7.4 policy')
for forbidden in ('http.StatusForbidden','http.StatusUnauthorized','ServeHTTP(','StatusCode = 403','WriteHeader(403)'):
    req(forbidden not in src,f'API-7.4 has no request-blocking primitive: {forbidden}')
req('InferenceBlocking: false' in src,'status explicitly declares inferred BOLA cannot block')
req('OpenAI' not in src,'OpenAI absent from API-7.4 policy/evidence authority')
for forbidden in ('Header.Get("X-Tenant','Header.Get("X-Owner','Authorization','Cookie('):
    req(forbidden not in src,f'API-7.4 does not trust caller-supplied authority: {forbidden}')

# Strict JSON schema prevents smuggling raw selectors into mutation APIs.
req('DisallowUnknownFields()' in src,'API-7.4 mutation JSON rejects unknown fields')
req('http.MaxBytesReader(w, r.Body, 1<<14)' in src,'API-7.4 mutation bodies are size bounded')
req('RawObject' not in src and 'RawIdentity' not in src,'API-7.4 mutation model has no raw identity/object field')

# Reviewer-only APIs and audit trail.
routes=(
    'GET /api/security/bola/evidence',
    'POST /api/security/bola/evidence/{candidate_id}/workflow',
    'GET /api/security/bola/policies',
    'POST /api/security/bola/policies',
    'DELETE /api/security/bola/policies/{policy_id}',
    'GET /api/security/bola/policy-status',
)
for route in routes:
    req(route in admin,f'admin route exists: {route}')
for handler in ('handleBOLAEvidence','handleBOLAEvidenceWorkflow','handleBOLAPolicies','handleBOLAPolicyUpsert','handleBOLAPolicyDelete','handleBOLAPolicyStatus'):
    req(f'authRole(roleReviewer, a.{handler})' in admin,f'{handler} requires Reviewer-or-higher')
for event in ('api_bola.policies_view','api_bola.policy_upsert','api_bola.policy_delete','api_bola.evidence_view','api_bola.evidence_workflow','api_bola.policy_status_view'):
    req(event in src,f'audit event exists: {event}')
req('persistBOLAPolicyMutation' in src and 'bolaPolicy.save(a.srv.configPath)' in src,'mutations persist before success')

# Console completeness and authority messaging.
for token in ('API-7.3 / API-7.4 BOLA evidence','api_bola_evidence','api_bola_policies','api_bola_policy_save','api_bola_policy_action','api_bola_policy_type','api_bola_policy_conf','api_bola_policy_ttl'):
    req(token in ui,f'Admin console includes {token}')
for action in ('Ack','Dismiss','Resolve','Reopen'):
    req(f'>{action}<' in ui,f'Admin console workflow action exists: {action}')
req('BOLA candidates are inferred evidence only' in ui,'console labels inferred evidence as non-authoritative')
req('never authorize request-path BLOCK/DENY/403' in ui,'console states no BOLA request blocking')
req('Policies are evidence-routing controls only' in ui,'console explains evidence-policy semantics')
req('<option>REVIEW</option><option>SUPPRESS</option>' in ui,'console exposes REVIEW/SUPPRESS only')
req('ENFORCE action' in ui,'console explicitly rejects ENFORCE policy semantics')

# Tests.
for name in (
    'TestAPI74PolicyScopePrecedenceAndSuppressionNeverDeletesCandidate',
    'TestAPI74PolicyRejectsUnnormalizedOrUnknownLocatorAndInvalidTTL',
    'TestAPI74EvidenceWorkflowUsesEnumeratedStateAndReasonOnly',
    'TestAPI74DismissedOrResolvedEvidenceReopensOnNewDetection',
    'TestAPI74PersistenceIsBoundedRestartSafeAndPrivacyPreserving',
    'TestAPI74PolicyAndReviewCardinalityAreBounded',
    'TestAPI74StrictMutationJSONRejectsUnknownRawSelectors',
    'TestAPI74StatusDeclaresInferredBOLANonBlocking',
    'TestAPI74ConcurrentPolicyEvidenceSnapshotsRemainBounded',
    'TestAPI74StateFileRejectsUnknownVersion',
):
    req(name in tests,f'targeted API-7.4 test exists: {name}')

# Build/CI retains all prior gates and adds API-7.4.
for gate in ('test-api71-source.py','test-api72-source.py','test-api73-source.py','test-api74-source.py'):
    req(gate in build,f'local build includes {gate}')
    req(gate in ci,f'CI includes {gate}')

print(f'API74_SOURCE_GATE_PASS checks={len(checks)}')
