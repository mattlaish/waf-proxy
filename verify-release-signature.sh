#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 3 ]] || { echo "Usage: $0 <artifact> <signature.minisig> <public-key-file>" >&2; exit 2; }
artifact=$1; signature=$2; pubfile=$3
[[ -f "$artifact" && -f "$signature" && -f "$pubfile" ]] || { echo "INVALID missing artifact/signature/public key" >&2; exit 1; }
command -v minisign >/dev/null 2>&1 || { echo "NOT_CONFIGURED minisign executable unavailable" >&2; exit 3; }
pub=$(awk 'NF && $1 !~ /^untrusted/ {print; exit}' "$pubfile")
[[ -n "$pub" ]] || { echo "INVALID public key file contains no key" >&2; exit 1; }
if minisign -Vm "$artifact" -x "$signature" -P "$pub" >/dev/null 2>&1; then
  echo "VALID"
  exit 0
fi
echo "INVALID" >&2
exit 1
