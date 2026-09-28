#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
checks=[]
def req(cond,msg):
    if not cond:
        print(f"API62_SOURCE_GATE_FAIL: {msg}", file=sys.stderr); sys.exit(1)
    checks.append(msg)

base=(ROOT/'sequence_api61.go').read_text(encoding='utf-8')
src=(ROOT/'sequence_api62.go').read_text(encoding='utf-8')
tests=(ROOT/'sequence_api62_test.go').read_text(encoding='utf-8')
main=(ROOT/'main.go').read_text(encoding='utf-8')
admin=(ROOT/'admin.go').read_text(encoding='utf-8')
ui=(ROOT/'static/admin.html').read_text(encoding='utf-8')
build=(ROOT/'build.sh').read_text(encoding='utf-8')
ci=(ROOT/'.github/workflows/ci.yml').read_text(encoding='utf-8')

req('sequenceStateVersion                 = 4' in base, 'durable sequence state remains additive beyond the API-6.2 v2 baseline')
for f in ('ObservationCount uint64','SessionCount     uint64','Confidence       float64','Maturity         string'):
    req(f in base, f'transition learning field exists: {f}')
for f in ('FirstSeen','LastSeen','MaxObservedDepth','EntryOperations','TerminalOperations','Maturity'):
    req(f in src, f'workflow model includes {f}')
for state in ('LEARNING','MATURE','STALE'):
    req(f'"{state}"' in src, f'{state} maturity semantic exists')
for threshold in ('sequenceDefaultMinObservations','sequenceDefaultMinSessions','sequenceDefaultMinLearningDuration','sequenceDefaultConfidenceThreshold'):
    req(threshold in base, f'cold-start threshold exists: {threshold}')
req('SessionAbsoluteTTL' in base and 'AbsoluteExpiresAt' in base, 'idle plus absolute session lifecycle exists')
req('MaxWorkflows' in base and 'MaxWorkflowOperations' in base and 'MaxSeenTransitions' in base, 'workflow/session learning cardinality is bounded')
req('ensureWorkflowCapacityLocked' in src and 'ensureSessionCapacityLocked' in base and 'ensureTransitionCapacityLocked' in base, 'bounded eviction exists for workflow/session/transition state')
req('apiOperationID(site, r.Method, normalizeAPIOperationPath(r.URL.Path))' in base, 'learning nodes remain API-1 normalized operations')
req('verifiedAPIIdentity(r)' in base and 'verifiedWorkflowMaterial(id)' in base, 'workflow identity derives only from API-5 verified context')
req('id.Subject' not in src[src.find('func verifiedWorkflowMaterial'):src.find('func containsSequenceID')], 'verified subject excluded from workflow cohort material')
req('sort.Strings(roles)' in src and 'sort.Strings(scopes)' in src, 'role/scope cohort semantics are order-stable')
req('r.Header.Get("Authorization")' not in base+src, 'sequence learning never reads raw Authorization data')
req('hmac.New(sha256.New, key)' in base and 'WorkflowID: s.digest(' in base, 'session/workflow identifiers are keyed fingerprints')
req('select {\n\tcase s.queue <- ev:' in base and 's.dropped.Add(1)' in base, 'request hot path remains non-blocking bounded enqueue')
req(base.find('func (s *sequenceStore) noteRequest') < base.find('func (s *sequenceStore) process'), 'hot-path observation is separated from background state mutation')
req('atomic.Pointer[SequenceModel]' in base and 's.runtime.Store(&model)' in base, 'readers consume atomic immutable model snapshots')
req('refreshLearningDerivedLocked' in base and 'float64(edge.ObservationCount) / float64(total)' in src, 'transition confidence is frequency-derived rather than raw count')
req('edge.SessionCount++' in base and 'SeenTransitionIDs' in base, 'transition tracks bounded unique-session evidence')
req('boundedIncrementOperation(w.EntryOperations' in src and 'boundedIncrementOperation(w.TerminalOperations' in src, 'entry and terminal operations are learned')
req('WorkflowDepth' in base and 'MaxObservedDepth' in src, 'workflow depth is learned')
req('migrateV1StateLocked' in src and 'state.Version == 1' in base, 'API-6.1 durable state has explicit v1 migration')
req('Workflows: model.Workflows' in base and 'workflowRuntimeFromModel' in src, 'workflow models persist and restore across restart')
req('GET /api/security/sequence/workflows' in admin and 'authRole(roleReviewer, a.handleSequenceWorkflows)' in admin, 'workflow detail visibility is Reviewer-gated')
req('api_sequence.workflows_view' in src, 'workflow visibility is audited')
req('API-6.2 LEARN-only workflow construction' in ui, 'console retains the API-6.2 learning boundary')
req('API-6.3 DETECT-only anomaly evidence' in ui and 'never blocks traffic' in ui, 'console distinguishes later DETECT-only evidence from API-6.2 learning')
for forbidden in ('UNKNOWN_TRANSITION','PREREQUISITE_SKIPPED','SEQUENCE_REVERSAL','ABNORMAL_REPETITION','http.StatusForbidden','BLOCK','DENY'):
    req(forbidden not in src, f'API-6.2 does not introduce detection/enforcement token {forbidden}')
req('OpenAI' not in base+src, 'OpenAI is absent from request-path sequence learning code')
req('test-api62-source.py' in build and 'test-api62-source.py' in ci, 'API-6.2 source gate retained by local build and CI')
for testname in (
    'TestAPI62LearnsTransitionFrequencyUniqueSessionsAndMaturity',
    'TestAPI62ColdStartAndStaleMaturityProtection',
    'TestAPI62AbsoluteSessionLifetimeAndTerminalLearning',
    'TestAPI62IdentityCohortsUseVerifiedClaimsOnlyAndExcludeSubject',
    'TestAPI62NormalizedCardinalityAndRandomSessionBounds',
    'TestAPI62RestartPersistenceRestoresLearningWithoutRawVerifiedClaims',
    'TestAPI62ConcurrentWorkflowLearningAndSnapshotsRemainBounded',
):
    req(testname in tests, f'targeted API-6.2 test exists: {testname}')

print(f"API62_SOURCE_GATE_PASS checks={len(checks)}")
