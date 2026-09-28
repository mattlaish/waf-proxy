#!/usr/bin/env bash
set -euo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
cd "$TMP"
GOTOOLCHAIN=local GO111MODULE=off go run "$ROOT/tools/tests/go-root-shape.go" "$ROOT"
