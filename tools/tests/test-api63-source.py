#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
checks=[]
def req(cond,msg):
    if not cond:
        print(f"API63_SOURCE_GATE_FAIL: {msg}", file=sys.stderr); sys.exit(1)
    checks.append(msg)

base=(ROOT/'sequence_api61.go').read_text(encoding='utf-8')
learn=(ROOT/'sequence_api62.go').read_text(encoding='utf-8')
src=(ROOT/'sequence_api63.go').read_text(encoding='utf-8')
tests=(ROOT/'sequence_api63_test.go').read_text(encoding='utf-8')
admin=(ROOT/'admin.go').read_text(encoding='utf-8')
ui=(ROOT/'static/admin.html').read_text(encoding='utf-8')
build=(ROOT/'build.sh').read_text(encoding='utf-8')
ci=(ROOT/'.github/workflows/ci.yml').read_text(encoding='utf-8')

req('sequenceStateVersion                 = 4' in base, 'durable sequence state remains additive beyond the API-6.3 v3 baseline')
req('Violations' in base and '[]SequenceViolation' in base and 'Exceptions' in base and '[]SequenceException' in base, 'post-v3 state persists bounded violations and exception primitives')
req('json:"violations"' in base and 'json:"exceptions"' in base and '[]SequenceViolation' in base and '[]SequenceException' in base, 'immutable model carries detection evidence')
req('sequenceDetectionLearn  = "LEARN"' in src and 'sequenceDetectionDetect = "DETECT"' in src, 'LEARN to DETECT semantics exist without ENFORCE state')
req('DetectionState' in learn and 'sequenceDetectionState(maturity)' in learn, 'workflow snapshot exposes automatic learning/detection state')
for anomaly in ('UNKNOWN_TRANSITION','PREREQUISITE_SKIPPED','UNEXPECTED_ENTRY_POINT','SEQUENCE_REVERSAL','ABNORMAL_REPETITION','WORKFLOW_DIVERGENCE'):
    req(f'"{anomaly}"' in src, f'anomaly class exists: {anomaly}')
req('workflowMaturityLocked' in src and 'sequenceMaturityMature' in src, 'detection is maturity-gated')
req('transitionEvidenceReadyLocked' in src and 'MinObservations' in src and 'MinSessions' in src and 'MinLearningDuration' in src, 'transition evidence requires sample, session and age thresholds')
req('sourceBaselineReadyLocked' in src and 'ConfidenceThreshold' in src, 'unknown/skip/repetition detection requires a confident learned source baseline')
req('ctx.DirectExists' in src and 'DivergenceObservedMaxConfidence' in src and 'DivergenceExpectedMinConfidence' in src, 'workflow divergence is based on learned non-zero edge evidence plus expected confidence')
req('consecutiveOperationRun' in src and 'RepetitionTailProbability' in src and 'RepetitionMinimumRun' in src, 'abnormal repetition combines session history and learned probability')
req('findSkippedPrerequisiteLocked' in src and 'RelatedOperationID: prerequisite' in src, 'prerequisite skip carries normalized intermediate-operation evidence')
req('reverseID := sequenceWorkflowTransitionID' in src and 'sequenceAnomalySequenceReversal' in src, 'reversal uses the learned reverse transition')
req('detectUnexpectedEntryLocked' in src and 'EntryConfidenceThreshold' in src, 'unexpected entry requires mature entry baseline confidence')
req('exceptionMatchesLocked' in src and 'exceptedLocked' in src, 'exceptions are evaluated before violation evidence is retained')
req('MaxViolations' in base and 'ViolationTTL' in base and 'evictedViolations' in base, 'violation evidence is cap/TTL bounded with eviction telemetry')
req('MaxExceptions' in base and 'len(s.exceptions) >= s.settings.MaxExceptions' in base, 'exception persistence is cardinality bounded')
req('pruneDetectionLocked(now)' in base and 'v.ExpiresAt.After(now)' in src, 'expired detection evidence and exceptions are cleaned up')
req('cloneSequenceViolations' in base and 'cloneSequenceExceptions' in base, 'atomic snapshots deep-copy detection slices')
req('validSequenceViolation' in base and 'validSequenceException' in base, 'restart restore validates detection state fail-closed')
req('state.Version != 2' in base, 'API-6.2 v2 durable state remains accepted for additive migration')
req('s.detectUnexpectedEntryLocked(ev)' in base and 's.detectTransitionAnomaliesLocked(ev, prev, from)' in base, 'detector executes before current observation mutates learned edge/session history')
req('apiOperationID(site, r.Method, normalizeAPIOperationPath(r.URL.Path))' in base, 'detection still consumes API-1 normalized operation IDs')
req('verifiedAPIIdentity(r)' in base and 'r.Header.Get("Authorization")' not in base+learn+src, 'identity context remains API-5 verified only and raw Authorization is never parsed')
req('SessionFingerprint' in src and 'raw JWT' not in src and 'Cookie' not in src, 'violation evidence stores keyed fingerprints rather than raw request identity/token values')
req('http.StatusForbidden' not in src and 'WriteHeader(' not in src and 'ServeHTTP(' not in src, 'API-6.3 detector has no HTTP enforcement path')
req('sequenceDetectionEnforce' not in src and '"ENFORCE"' not in src, 'API-6.3 introduces no sequence ENFORCE mode')
req('OpenAI' not in base+learn+src, 'OpenAI is absent from sequence request-path decision code')
req('GET /api/security/sequence/violations' in admin and 'authRole(roleReviewer, a.handleSequenceViolations)' in admin, 'violation evidence read endpoint is Reviewer-gated')
req('api_sequence.violations_view' in src, 'violation evidence reads are audited')
req('API-6.3 DETECT-only anomaly evidence' in ui and 'never blocks traffic' in ui, 'admin copy states the non-enforcement boundary')
req('test-api61-source.py' in build and 'test-api62-source.py' in build and 'test-api63-source.py' in build, 'local source gates retain API-6.1 through API-6.3')
req('test-api61-source.py' in ci and 'test-api62-source.py' in ci and 'test-api63-source.py' in ci, 'CI source gates retain API-6.1 through API-6.3')
for testname in (
    'TestAPI63UnknownTransitionRequiresMatureBaselineAndNeverEnforces',
    'TestAPI63PrerequisiteSkipAndReversalUseMaturePaths',
    'TestAPI63UnexpectedEntryAndExceptionSuppression',
    'TestAPI63AbnormalRepetitionUsesSessionHistoryAndLearnedProbability',
    'TestAPI63WorkflowDivergenceRequiresLearnedRareEdgeNotZeroProbability',
    'TestAPI63BoundedEvidenceRestartPersistenceAndPrivacy',
    'TestAPI63ConcurrentDetectionSnapshotsRemainBounded',
):
    req(testname in tests, f'targeted API-6.3 test exists: {testname}')

print(f"API63_SOURCE_GATE_PASS checks={len(checks)}")
