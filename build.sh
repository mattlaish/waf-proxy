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
GO_VERSION="$(GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' | sed 's/^go//' || true)"
if [ -z "$GO_VERSION" ] || ! version_ge "$GO_VERSION" "1.25.0"; then
  echo "!! Go >= 1.25.0 is required for Coraza v3.7.0 (found ${GO_VERSION:-unavailable})" >&2
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

echo "==> go mod tidy -diff (dependency files must already be canonical)"
go mod tidy -diff

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

if [ -n "$VECTOR_TAGS" ]; then
  BUILD_VARIANT="native-vectorscan"
  VECTORSCAN_VERSION="$(pkg-config --modversion libhs 2>/dev/null || echo unknown)"
else
  BUILD_VARIANT="portable-coraza"
  VECTORSCAN_VERSION="not-linked"
fi

echo "==> building waf-proxy ${VERSION} (${BUILD_VARIANT})"
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

echo "==> building wafctl ${VERSION}"
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X main.version=${VERSION}" \
  -o wafctl ./cmd/wafctl

echo
printf 'built:\n'
ls -lh waf-proxy waf-tlsfront wafctl
python3 - "$VERSION" "$COMMIT" "$BUILD_VARIANT" "$GO_VERSION" "$VECTORSCAN_VERSION" <<'PY2'
import hashlib,json,pathlib,sys
version,commit,variant,gov,vs=sys.argv[1:]
files=[]
for name in ("waf-proxy","waf-tlsfront","wafctl"):
 p=pathlib.Path(name); h=hashlib.sha256(p.read_bytes()).hexdigest(); files.append({"name":name,"sha256":h,"size":p.stat().st_size})
artifact_type="NATIVE_BINARY" if variant=="native-vectorscan" else "PORTABLE_BINARY"
linkage="DYNAMIC_LIBHS_REQUIRED_AT_RUNTIME" if variant=="native-vectorscan" else "NO_LIBHS_LINK"
obj={"schema_version":2,"version":version,"commit":commit,"artifact_type":artifact_type,"build_variant":variant,"go_version":gov,"coraza_version":"v3.7.0","vectorscan_version":vs,"linkage_expectation":linkage,"binaries":files}
pathlib.Path("BUILD_PROVENANCE.json").write_text(json.dumps(obj,indent=2,sort_keys=True)+"\n",encoding="utf-8")
pathlib.Path("BUILD_SHA256SUMS.txt").write_text("".join(f"{x['sha256']}  {x['name']}\n" for x in files),encoding="utf-8")
PY2
printf '\nbuild variant: %s\n' "$BUILD_VARIANT"
printf 'provenance: BUILD_PROVENANCE.json, BUILD_SHA256SUMS.txt\n'
printf '\nNext: sudo ./install.sh\n'
