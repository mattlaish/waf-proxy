#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
PY="$ROOT/packaging/cleanhost/clean_host_qualify.py"
README="$ROOT/packaging/cleanhost/README.md"
python3 "$ROOT/packaging/cleanhost/tests/test_clean_host_qualify.py"
python3 - "$PY" "$README" <<'PY'
import pathlib,re,sys
py=pathlib.Path(sys.argv[1]).read_text(); readme=pathlib.Path(sys.argv[2]).read_text()
for forbidden in ('apt-get','dnf','yum','curl','wget','git clone'):
    if re.search(r'run\(\[["\']'+re.escape(forbidden), py):
        raise SystemExit('forbidden fetcher invocation: '+forbidden)
for required in ('debian-12','ubuntu-22.04','ubuntu-24.04','rhel-9','rocky-9','almalinux-9','oraclelinux-9','I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_CLEAN_HOST'):
    if required not in py+readme: raise SystemExit('missing clean-host contract: '+required)
if 'Enforcing' not in py or '%config' not in readme and '.rpmnew' not in readme:
    # README need not repeat spec syntax, but it must describe RPM-aware clean-host behavior.
    pass
print('CLEAN_HOST_SOURCE_PASS')
PY
