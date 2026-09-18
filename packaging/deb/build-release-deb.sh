#!/usr/bin/env bash
# Qualified release-host convenience wrapper: build Go binaries, then package
# those exact provenance-bound bytes into the deterministic Debian package.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
case "${SOURCE_DATE_EPOCH:-}" in
  ''|*[!0-9]*) echo "SOURCE_DATE_EPOCH must be set for a release .deb" >&2; exit 1 ;;
esac
./build.sh
exec ./packaging/deb/build-deb.sh --binaries-dir . "$@"
