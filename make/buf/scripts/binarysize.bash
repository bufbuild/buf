#!/usr/bin/env bash

set -eo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

TMP="$(mktemp -d )"
trap 'rm -rf "${TMP}"' EXIT

fail() {
  echo "error: $@" >&2
  exit 1
}

size_bytes() {
  case "$(uname -s)" in
    Darwin) stat -f%z "${1}" ;;
    Linux) stat -c%s "${1}" ;;
    *) fail "must be run on darwin or linux" ;;
  esac
}

# Build in the same manner as we do in release.bash.
CGO_ENABLED=0 \
  GOOS=darwin \
  GOARCH=arm64 \
  go build -a -ldflags "-s -w" -trimpath -buildvcs=false \
  -o "${TMP}/bin" \
  "${1}"

echo "$(awk "BEGIN { printf(\"%.2f\", $(size_bytes "${TMP}/bin") / 1048576.0) }") MB"
