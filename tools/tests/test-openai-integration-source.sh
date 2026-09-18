#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
command -v python3 >/dev/null 2>&1 || { echo 'OPENAI_SOURCE_BLOCKED python3 unavailable' >&2; exit 77; }
command -v go >/dev/null 2>&1 || { echo 'OPENAI_SOURCE_BLOCKED go unavailable' >&2; exit 77; }

python3 - "$ROOT" <<'PY'
from pathlib import Path
import json,sys
root=Path(sys.argv[1])
ai=(root/'ai.go').read_text(); admin=(root/'admin.go').read_text(); ui=(root/'static/admin.html').read_text(); cfg=json.loads((root/'config.sample.json').read_text()); op=(root/'internal/openaiapi/responses.go').read_text(); sec=(root/'internal/secretref/secretref.go').read_text()
checks = {
 'sample uses responses': cfg['ai'].get('api_style') == 'responses',
 'sample uses secret ref': cfg['ai'].get('api_key_ref') == 'env:OPENAI_API_KEY' and 'api_key' not in cfg['ai'],
 'runtime imports provider client': 'waf-proxy/internal/openaiapi' in ai and 'openaiapi.CallResponses' in ai,
 'verdict schema strict': 'waf_verdict' in ai and 'additionalProperties' in ai and 'minimum' in ai and 'maximum' in ai,
 'profile schema strict': 'waf_profile_review' in ai and 'profileReviewSchema' in ai,
 'responses store false': 'Store:' in op and 'false' in op,
 'responses path': '+"/responses"' in op,
 'bearer auth': 'Authorization' in op and 'Bearer ' in op,
 'refusal/incomplete handled': 'refusal' in op and 'incomplete' in op and 'no output_text' in op,
 'admin redacts value/ref': 'redactAISecrets' in admin and 'c.AI.APIKeyRef = ""' in ai and 'c.AI.APIKey = ""' in ai,
 'UI secret reference only': 'ai_api_key_ref' in ui and 'id="ai_api_key"' not in ui,
 'UI exposes API style': 'ai_api_style' in ui and 'chat_completions' in ui,
 'secret ref env/file only': 'env:NAME or file:/absolute/path' in sec,
 'secret file symlink hardened': 'EvalSymlinks' in sec and 'SameFile' in sec,
 'legacy inline migration retained': 'legacy migration path only' in ai and 'return "chat_completions"' in ai,
 'Anthropic also resolves secret ref': 'authHeaders(cfg, "x-api-key")' in ai,
}
for name, ok in checks.items():
    print(('PASS' if ok else 'FAIL'), name)
if not all(checks.values()):
    raise SystemExit(1)
print(f'OPENAI_SOURCE_CONTRACT_PASS {sum(checks.values())}/{len(checks)}')
PY

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/openaiapi" "$tmp/secretref"
cp "$ROOT"/internal/openaiapi/*.go "$tmp/openaiapi/"
cp "$ROOT"/internal/secretref/*.go "$tmp/secretref/"
printf 'module waf-openai-isolated\n\ngo 1.23.0\n' > "$tmp/go.mod"
(
  cd "$tmp"
  GOTOOLCHAIN=local go test -count=1 ./openaiapi ./secretref
  GOTOOLCHAIN=local go vet ./openaiapi ./secretref
)
echo OPENAI_ISOLATED_TEST_PASS
