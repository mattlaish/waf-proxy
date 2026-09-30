#!/usr/bin/env bash
# Build the same complete-source release twice with fixed SOURCE_DATE_EPOCH and
# require byte-identical ZIP output.
set -euo pipefail
usage(){ echo "Usage: $0 --version LABEL --source-date-epoch EPOCH [--variant source]" >&2; }
version=""; epoch=""; variant=source
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) version=${2:-}; shift 2;;
    --source-date-epoch) epoch=${2:-}; shift 2;;
    --variant) variant=${2:-}; shift 2;;
    *) usage; exit 2;;
  esac
done
[[ -n "$version" && "$epoch" =~ ^[0-9]+$ ]] || { usage; exit 2; }
root=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
"$root/build-release-artifact.sh" "$tmp/a.zip" --version "$version" --variant "$variant" --source-date-epoch "$epoch" --require-reproducible >/dev/null
"$root/build-release-artifact.sh" "$tmp/b.zip" --version "$version" --variant "$variant" --source-date-epoch "$epoch" --require-reproducible >/dev/null
ha=$(sha256sum "$tmp/a.zip" | awk '{print $1}'); hb=$(sha256sum "$tmp/b.zip" | awk '{print $1}')
[[ "$ha" == "$hb" ]] || { echo "FAIL reproducible release mismatch: $ha != $hb" >&2; exit 1; }
echo "PASS reproducible release sha256=$ha"
