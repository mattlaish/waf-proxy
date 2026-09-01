#!/usr/bin/env bash
# Build waf-proxy into a single self-contained binary (console is go:embed-ed).
#
# Requires: Go >= 1.22 and network access for the first `go mod tidy`
# (Coraza + OWASP CRS deps). After that it builds offline.
set -euo pipefail

cd "$(dirname "$0")"

VERSION="${VERSION:-$(date -u +%Y.%m.%d)}"
COMMIT="${COMMIT:-unknown}"

echo "==> go mod tidy (fetches Coraza; needs network the first time)"
go mod tidy

echo "==> go vet"
go vet ./... || {
  echo "!! go vet reported issues — fix before shipping" >&2
  exit 1
}

echo "==> go test (sigupdate verifier — the signed-update proof)"
go test ./... || {
  echo "!! tests failed — do NOT ship a broken update verifier" >&2
  exit 1
}

echo "==> building waf-proxy ${VERSION}"
CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags "-s -w -X main.buildVersion=${VERSION} -X main.buildCommit=${COMMIT}" \
  -o waf-proxy .

echo
echo "built: $(pwd)/waf-proxy"
ls -lh waf-proxy
echo
echo "Next: sudo ./install.sh"
