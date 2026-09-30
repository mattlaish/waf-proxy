#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"
python3 -B packaging/qualification/tests/test_package_lifecycle.py
python3 - packaging/qualification/package_lifecycle_qualify.py <<'PY'
import pathlib,sys
p=pathlib.Path(sys.argv[1]); s=p.read_text()
required=[
 'I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_HOST',
 'operator_config_preserved_on_upgrade',
 'admin_secret_preserved_on_upgrade',
 'persistent_state_preserved_on_upgrade',
 'native_config_conflict_semantics',
 'failure_fixture_rejected',
 'rollback_to_baseline',
 '.dpkg-dist', '.rpmnew', '.rpmsave',
 'secret_material_in_evidence',
]
for token in required:
    if token not in s: raise SystemExit(f'missing lifecycle invariant: {token}')
for forbidden in ['apt-get', 'apt ', 'dnf ', 'yum ', 'curl ', 'wget ']:
    if forbidden in s: raise SystemExit(f'network/package-manager dependency resolver forbidden in lifecycle runner: {forbidden.strip()}')
# Evidence must never emit a hash/fingerprint of the generated admin secret.
for forbidden in ['secret_sha', 'secret_hash', 'admin_token_hash', 'WAF_ADMIN_TOKEN=%']:
    if forbidden in s: raise SystemExit(f'secret derivative/output forbidden: {forbidden}')
print('PACKAGE_LIFECYCLE_SOURCE_POLICY_PASS')
PY
python3 - packaging/rpm/waf-proxy.spec packaging/rpm/build-rpm.sh <<'PY'
import pathlib,sys
spec=pathlib.Path(sys.argv[1]).read_text(); build=pathlib.Path(sys.argv[2]).read_text()
if '%if 0%{?waf_qualification_fail_post}' not in spec: raise SystemExit('RPM failure fixture macro missing')
if '--qualification-fail-post' not in build: raise SystemExit('RPM failure fixture build flag missing')
if 'restricted to qualification fixture versions' not in build: raise SystemExit('RPM failpoint safety guard missing')
print('PACKAGE_LIFECYCLE_RPM_FAILPOINT_SOURCE_PASS')
PY
for f in \
  packaging/qualification/build-lifecycle-fixtures.sh \
  packaging/qualification/run-package-lifecycle-qualification.sh \
  packaging/qualification/tests/test-qualification-source.sh; do
  bash -n "$f"
  [ -x "$f" ] || { echo "not executable: $f" >&2; exit 1; }
done
python3 - <<'PY'
for p in ['packaging/qualification/package_lifecycle_qualify.py','packaging/qualification/tests/test_package_lifecycle.py']:
    compile(open(p,encoding='utf-8').read(),p,'exec')
print('PACKAGE_LIFECYCLE_PYTHON_COMPILE_PASS')
PY
for f in qualification/package-lifecycle/deb-upgrade-rollback-NOT_RUN.json qualification/package-lifecycle/rpm-upgrade-rollback-NOT_RUN.json; do
  python3 - "$f" <<'PY'
import json,sys
x=json.load(open(sys.argv[1])); assert x['status']=='NOT_RUN'; assert x['real_package_manager_execution'] is False
PY
done
printf 'PACKAGE_LIFECYCLE_SOURCE_TEST_PASS\n'
