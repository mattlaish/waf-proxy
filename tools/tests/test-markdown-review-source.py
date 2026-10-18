#!/usr/bin/env python3
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[2]
mds = sorted(root.rglob('*.md'))
fail=[]
# Every Markdown file carries a dated review marker with a classification.
# Later review passes (e.g. 2026-09-29) stamp their own date, so any dated
# marker is accepted rather than pinning the original 2026-09-28 pass.
marker=re.compile(r'<!-- documentation-review: \d{4}-\d{2}-\d{2}; classification:')
for p in mds:
    text=p.read_text(encoding='utf-8')
    if not marker.search(text):
        fail.append(f'missing review marker: {p.relative_to(root)}')
ledger=root/'MARKDOWN_REVIEW_2026-09-28.md'
if not ledger.is_file(): fail.append('missing MARKDOWN_REVIEW_2026-09-28.md')
else:
    # A file must be recorded in at least one dated review ledger.
    lt='\n'.join(p.read_text(encoding='utf-8') for p in sorted(root.glob('MARKDOWN_REVIEW_*.md')))
    for p in mds:
        rel=p.relative_to(root).as_posix()
        if f'`{rel}`' not in lt:
            fail.append(f'ledger missing: {rel}')
required_current=[
 'README.md','DEVELOPMENT.md','AI_HANDOFF.md','TESTING.md','TESTING_RESULTS.md',
 'HANDOVER_STATUS.md','NEXT_CHAT_HANDOVER.md','HANDOVER_PROMPT.md','API_SECURITY_ROADMAP.md',
 'API_SECURITY_SLICE_IMPLEMENTATION_ROADMAP.md','DEVELOPMENT_ROADMAP.md','DOCUMENTATION_INDEX.md',
 'MANIFEST.md','RELEASE_PROCESS.md','API_SECURITY_CLOSURE_RESULT.md','PRODUCTION_CONTROL_PLANE_HARDENING.md',
 'INSTALL.md','PACKAGING_TOOL.md','API8_GRAPHQL_SECURITY.md','API8_POST_AUDIT_HARDENING.md'
]
for rel in required_current:
    text=(root/rel).read_text(encoding='utf-8')
    if '## Current canonical baseline — 2026-09-28' not in text:
        fail.append(f'current baseline section missing: {rel}')
if 'The `web/` Vite console is a **scaffold**' in (root/'INSTALL.md').read_text():
    fail.append('INSTALL.md still advertises removed web scaffold')
if '`debug_bundle.go`, `debug_evidence.go`, `support_api.go`' in (root/'API8_POST_AUDIT_HARDENING.md').read_text():
    fail.append('API8 post-audit doc still claims removed debug_evidence.go as current')
if fail:
    print('MARKDOWN_REVIEW_SOURCE_GATE_FAIL')
    for item in fail: print(' -',item)
    sys.exit(1)
print(f'MARKDOWN_REVIEW_SOURCE_GATE_PASS files={len(mds)} current={len(required_current)}')
