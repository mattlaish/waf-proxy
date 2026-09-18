#!/usr/bin/env bash
# Operator entrypoint for Enterprise Linux Distribution Packaging Slice C.
# Non-mutating preflight is the default. Real package-manager execution remains
# destructive and is permitted only when package_lifecycle_qualify.py receives
# --execute plus the exact dedicated-host acknowledgement documented in README.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

if [ "$#" -eq 0 ]; then
  exec python3 "$ROOT/packaging/qualification/package_lifecycle_qualify.py" --help
fi

# Do not resolve packages or fetch dependencies here. The Python runner uses
# only local dpkg/rpm transactions and produces secret-free JSON evidence.
exec python3 "$ROOT/packaging/qualification/package_lifecycle_qualify.py" "$@"
