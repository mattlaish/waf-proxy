#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
for f in "$ROOT/packaging/rpm/build-rpm.sh" "$ROOT/packaging/rpm/build-release-rpm.sh" "$ROOT/packaging/rpm/verify-rpm.sh" "$ROOT/packaging/rpm/tests/test-rpm-packaging.sh"; do
  bash -n "$f"
done
PYTHONDONTWRITEBYTECODE=1 python3 "$ROOT/packaging/rpm/validate-rpm-source.py"
# Explicitly prove RPM lifecycle/build sources do not contain package/network
# fetching or SELinux-disabling commands.
python3 - "$ROOT" <<'PY'
import pathlib,re,sys
root=pathlib.Path(sys.argv[1])
paths=[root/'packaging/rpm/waf-proxy.spec',root/'packaging/rpm/build-rpm.sh']
pat=re.compile(r'(^|[;&|\s])(curl|wget|dnf|yum)([\s]|$)|git\s+clone|\b(setenforce|audit2allow)\b',re.I|re.M)
for p in paths:
    if pat.search(p.read_text(encoding='utf-8')):
        raise SystemExit(f'RPM_SOURCE_TEST_FAIL forbidden lifecycle/build fetch or SELinux mutation in {p}')
print('RPM_SOURCE_NETWORK_POLICY_PASS')
PY
printf 'RPM_SOURCE_TEST_PASS\n'
