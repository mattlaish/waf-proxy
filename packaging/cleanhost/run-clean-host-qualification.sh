#!/usr/bin/env bash
# Operator entrypoint for Enterprise Linux Distribution Packaging Slice D.
# Preflight is non-mutating. Real execution is restricted to dedicated clean
# hosts and requires the explicit acknowledgement documented in README.md.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
if [ "$#" -eq 0 ]; then
  exec python3 "$ROOT/packaging/cleanhost/clean_host_qualify.py" --help
fi
exec python3 "$ROOT/packaging/cleanhost/clean_host_qualify.py" "$@"
