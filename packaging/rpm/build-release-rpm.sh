#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
case "${SOURCE_DATE_EPOCH:-}" in
  ''|*[!0-9]*) echo "SOURCE_DATE_EPOCH must be set for a release .rpm" >&2; exit 1 ;;
esac
./build.sh
exec ./packaging/rpm/build-rpm.sh --binaries-dir . "$@"
