#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT=Path(__file__).resolve().parents[2]
checks=[]
def req(cond,msg):
    if not cond:
        print(f"API64_SOURCE_GATE_FAIL: {msg}", file=sys.stderr); sys.exit(1)
    checks.append(msg)

base=(ROOT/'sequence_api61.go').read_text(encoding='utf-8')
learn=(ROOT/'sequence_api62.go').read_text(encoding='utf-8')
detect=(ROOT/'sequence_api63.go').read_text(encoding='utf-8')
src=(ROOT/'sequence_api64.go').read_text(encoding='utf-8')
tests=(ROOT/'sequence_api64_test.go').read_text(encoding='utf-8')
admin=(ROOT/'admin.go').read_text(encoding='utf-8')
ui=(ROOT/'static/admin.html').read_text(encoding='utf-8')
build=(ROOT/'build.sh').read_text(encoding='utf-8')
ci=(ROOT/'.github/workflows/ci.yml').read_text(encoding='utf-8')

req('sequenceStateVersion                 = 4' in base, 'durable sequence state advances additively to v4')
req('RecentSessions []SequenceRecentSession' in base and 'Controls       []SequenceSiteControl' in base, 'v4 persists recent-session and operator-control state')
req('RecentSessions                  []SequenceRecentSession' in base and 'Controls                        []SequenceSiteControl' in base, 'immutable model carries operations state')
req('type SequenceSiteControl struct' in src and 'Generation uint64' in src, 'site-scoped mode and generation are explicit')
req('validSequenceMode' in src and 'sequenceDetectionLearn' in src and 'sequenceDetectionDetect' in src, 'only LEARN and DETECT are valid sequence modes')
req('len(s.controls) >= s.settings.MaxWorkflows' in src and 'maximum sequence site controls reached' in src, 'site-control state is cardinality bounded')
req('"ENFORCE"' not in src, 'API-6.4 introduces no sequence ENFORCE mode')
req('sequenceModeLocked(ev.Site) != sequenceDetectionDetect' in detect, 'API-6.3 evidence is gated by explicit API-6.4 DETECT mode')
req('DETECT requires at least one MATURE workflow for site' in src and 'workflowMaturityLocked' in src, 'DETECT promotion requires mature workflow evidence')
req('workflow.DetectionState = sequenceDetectionLearn' in base and 'sequenceModeLocked(v.Site) == sequenceDetectionDetect' in base, 'workflow read model reports operational mode plus maturity')
req('type SequenceRecentSession struct' in src and 'SessionFingerprint' in src and 'TerminalOperationID' in src, 'recent-session evidence is keyed and normalized')
req('MaxRecentSessions' in base and 'RecentSessionTTL' in base and 'evictedRecentSessions' in base, 'recent sessions are bounded by cap and TTL with eviction telemetry')
req('recordRecentSessionLocked(session, at)' in learn, 'expired/evicted active sessions enter bounded recent-session evidence')
req('pruneOperationsLocked(now)' in base and 'v.ExpiresAt.After(now)' in src, 'recent-session TTL cleanup is wired into pruning')
req('resetAndRelearn' in src and 'result.PreservedExceptions' in src and 'c.Mode = sequenceDetectionLearn' in src, 'site reset/relearn preserves exceptions and returns to LEARN')
req('if v.Site == site' in src and 'delete(s.workflows, id)' in src and 'delete(s.transitions, id)' in src, 'reset/relearn is site-scoped across learned/runtime state')
req('createSequenceException' in src and 'exception must include workflow or normalized operation scope' in src, 'exception CRUD rejects broad site-only exemptions')
req('sequenceMaximumExceptionTTL' in src and 'exception ttl exceeds maximum' in src, 'new exceptions are TTL bounded')
req('len(s.exceptions) >= s.settings.MaxExceptions' in src, 'exception CRUD enforces existing cardinality cap')
req('deleteSequenceException' in src, 'exception delete operation exists')
req('state.Version != 3' in base, 'API-6.3 v3 durable state remains accepted for additive migration')
req('cloneSequenceRecentSessions' in base and 'cloneSequenceControls' in base, 'immutable snapshots clone API-6.4 operations state')
for route in (
    'GET /api/security/sequence/recent-sessions',
    'GET /api/security/sequence/exceptions',
    'GET /api/security/sequence/learning-state',
    'POST /api/security/sequence/mode',
    'POST /api/security/sequence/relearn',
    'POST /api/security/sequence/exceptions',
    'DELETE /api/security/sequence/exceptions/{exception_id}',
):
    req(route in admin, f'admin route exists: {route}')
req('authRole(roleReviewer, a.handleSequenceMode)' in admin and 'authRole(roleReviewer, a.handleSequenceRelearn)' in admin, 'sequence control mutations require Reviewer-or-higher RBAC')
req('authRole(roleReviewer, a.handleSequenceExceptionCreate)' in admin and 'authRole(roleReviewer, a.handleSequenceExceptionDelete)' in admin, 'exception mutations require Reviewer-or-higher RBAC')
for action in ('api_sequence.mode','api_sequence.relearn','api_sequence.exception_create','api_sequence.exception_delete'):
    req(action in src, f'control mutation is audited: {action}')
req('persistSequenceMutation' in src and 'a.srv.sequence.save(a.srv.configPath)' in src, 'sequence control mutations persist durable state immediately')
for label in ('API sequence foundation + operations','Learning state','Active / recent sessions','Workflow models','Sequence violations','Exceptions','Reset / relearn'):
    req(label in ui, f'operations console exposes {label}')
req('/api/security/sequence/learning-state' in ui and '/api/security/sequence/recent-sessions' in ui and '/api/security/sequence/exceptions' in ui, 'console reads operations endpoints')
req('/api/security/sequence/mode' in ui and '/api/security/sequence/relearn' in ui, 'console exposes audited LEARN/DETECT and relearn controls')
req('Raw URLs, object values, JWTs, cookies, and claim values are not accepted as selectors' in ui, 'console documents sensitive-value exclusion boundary')
req('Authorization' not in src and 'Cookie' not in src and 'OpenAI' not in src, 'API-6.4 control/evidence code does not parse tokens/cookies or call OpenAI')
req('test-api64-source.py' in build, 'local build retains API-6.4 source gate')
req('test-api64-source.py' in ci, 'CI retains API-6.4 source gate')
for name in (
    'TestAPI64ExplicitLearnDetectModeNeverEnforces',
    'TestAPI64DetectPromotionRequiresMatureWorkflow',
    'TestAPI64SiteControlCardinalityIsBounded',
    'TestAPI64ResetRelearnIsSiteScopedPreservesExceptionsAndReturnsLearn',
    'TestAPI64RecentSessionsAreBoundedTTLAndPrivacyPreserving',
    'TestAPI64ExceptionCRUDIsScopedBoundedAndPersistent',
    'TestAPI64V3MigrationDefaultsToLearnAndV4PersistsControls',
    'TestAPI64AdminControlsRequireReviewerPersistAndAudit',
    'TestAPI64ConcurrentModeSnapshotAndRelearnStayBounded',
):
    req(name in tests, f'targeted API-6.4 test exists: {name}')

print(f"API64_SOURCE_GATE_PASS checks={len(checks)}")
