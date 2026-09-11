#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 3 ]] || { echo 'Usage: ./verify-release-signature.sh <artifact> <signature> <public-key.pem>' >&2; exit 2; }
openssl dgst -sha256 -verify "$3" -signature "$2" "$1"
