#!/usr/bin/env bash
# Build waf-proxy and companion binaries.
#
# Requires Go >= 1.25 because Coraza v3.7.0 declares go 1.25.0.
# VectorScan build policy:
#   WAF_VECTORSCAN=auto      build native acceleration when libhs is available (default)
#   WAF_VECTORSCAN=required  require libhs/pkg-config and fail otherwise
#   WAF_VECTORSCAN=off       build portable Coraza-only fallback
set -euo pipefail

cd "$(dirname "$0")"

VERSION="${VERSION:-$(date -u +%Y.%m.%d)}"
COMMIT="${COMMIT:-unknown}"
VECTOR_MODE="${WAF_VECTORSCAN:-auto}"

version_ge() {
  [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}
GO_VERSION="$(go env GOVERSION | sed 's/^go//')"
if ! version_ge "$GO_VERSION" "1.25.0"; then
  echo "!! Go >= 1.25.0 is required for Coraza v3.7.0 (found ${GO_VERSION})" >&2
  exit 1
fi

case "$VECTOR_MODE" in
  auto|required|off) ;;
  *) echo "!! WAF_VECTORSCAN must be auto, required, or off" >&2; exit 1 ;;
esac

VECTOR_TAGS=""
VECTOR_CGO=0
if [ "$VECTOR_MODE" != "off" ]; then
  if command -v pkg-config >/dev/null 2>&1 && pkg-config --exists libhs; then
    VECTOR_TAGS="vectorscan"
    VECTOR_CGO=1
    echo "==> VectorScan native build: $(pkg-config --modversion libhs 2>/dev/null || echo libhs)"
  elif [ "$VECTOR_MODE" = "required" ]; then
    echo "!! VectorScan required but pkg-config/libhs was not found" >&2
    exit 1
  else
    echo "==> VectorScan native library unavailable; building Coraza-only fallback"
  fi
fi

echo "==> go mod tidy (Coraza v3.7.0; needs network/module cache)"
go mod tidy

echo "==> go vet"
if [ -n "$VECTOR_TAGS" ]; then
  CGO_ENABLED="$VECTOR_CGO" go vet -tags "$VECTOR_TAGS" ./...
else
  CGO_ENABLED=0 go vet ./...
fi

echo "==> go test"
if [ -n "$VECTOR_TAGS" ]; then
  CGO_ENABLED="$VECTOR_CGO" go test -tags "$VECTOR_TAGS" ./...
  echo "==> go test -race (native VectorScan + Coraza)"
  CGO_ENABLED=1 go test -race -tags "$VECTOR_TAGS" ./...
else
  CGO_ENABLED=0 go test ./...
  echo "==> go test -race (portable Coraza path)"
  CGO_ENABLED=1 go test -race ./...
fi

echo "==> real Coraza transaction truth gate"
CGO_ENABLED=1 go test -tags realcoraza -run 'TestRealCoraza' ./...

echo "==> building waf-proxy ${VERSION}"
BUILD_ARGS=(-trimpath -ldflags "-s -w -X main.buildVersion=${VERSION} -X main.buildCommit=${COMMIT}" -o waf-proxy)
if [ -n "$VECTOR_TAGS" ]; then
  CGO_ENABLED="$VECTOR_CGO" go build -tags "$VECTOR_TAGS" "${BUILD_ARGS[@]}" .
else
  CGO_ENABLED=0 go build "${BUILD_ARGS[@]}" .
fi

echo "==> building waf-tlsfront ${VERSION}"
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X main.buildVersion=${VERSION} -X main.buildCommit=${COMMIT}" \
  -o waf-tlsfront ./cmd/waf-tlsfront

echo
printf 'built:\n'
ls -lh waf-proxy waf-tlsfront
printf '\nNext: sudo ./install.sh\n'
