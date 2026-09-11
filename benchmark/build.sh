#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p bin

echo "==> building wafbench"
go build -trimpath -o bin/wafbench ./cmd/wafbench

echo "built: $(pwd)/bin/wafbench"
./bin/wafbench help
