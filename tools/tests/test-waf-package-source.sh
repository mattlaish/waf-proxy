#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

bash -n ./waf-package
python3 -m py_compile tools/waf_package_builder.py tools/tests/test_waf_package_builder.py
PYTHONDONTWRITEBYTECODE=1 python3 tools/tests/test_waf_package_builder.py

./waf-package --help | grep -q 'Build verified WAF Reverse Proxy DEB/RPM packages'
./waf-package deb --help | grep -q -- '--source-date-epoch'
./waf-package deb --help | grep -q -- '--allow-module-network'
./waf-package rpm --help | grep -q -- '--rpm-release'

# The project-specific source contract must keep canonical build/packaging entry points.
grep -q 'run_stream(\["./build.sh"\]' tools/waf_package_builder.py
grep -q './packaging/deb/build-deb.sh' tools/waf_package_builder.py
grep -q './packaging/rpm/build-rpm.sh' tools/waf_package_builder.py
grep -q 'GOTOOLCHAIN.*local' tools/waf_package_builder.py
grep -q 'PACKAGE_BUILD_BLOCKED' tools/waf_package_builder.py

printf 'WAF_PACKAGE_TOOL_SOURCE_PASS\n'
