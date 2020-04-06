#!/bin/sh

set -e

DIR="$(cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

if [ -z "${1}" ]; then
  echo "usage: ${0} out_dir" >&2
  exit 1
fi

OUT_DIR="${1}"
rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}/bin"
mkdir -p "${OUT_DIR}/etc/bash_completion.d"
mkdir -p "${OUT_DIR}/share/zsh/site-functions"

set -x

go build -ldflags "-s -w" -trimpath -o "${OUT_DIR}/bin/buf" "cmd/buf/main.go"
go build -ldflags "-s -w" -trimpath -o "${OUT_DIR}/bin/protoc-gen-buf-check-breaking" "cmd/protoc-gen-buf-check-breaking/main.go"
go build -ldflags "-s -w" -trimpath -o "${OUT_DIR}/bin/protoc-gen-buf-check-lint" "cmd/protoc-gen-buf-check-lint/main.go"
"${OUT_DIR}/bin/buf" bash-completion > "${OUT_DIR}/etc/bash_completion.d/buf"
"${OUT_DIR}/bin/buf" zsh-completion > "${OUT_DIR}/share/zsh/site-functions/_buf"
cp "LICENSE" "${OUT_DIR}/LICENSE"
