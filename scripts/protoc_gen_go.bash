#!/usr/bin/env bash

set -eo pipefail

fail() {
  echo "$@" >&2
  exit 1
}

if [ -z "${1}" ] || [ -z "${2}" ]; then
  fail "usage: ${0} proto_path protoc_gen_go_out"
fi

PROTO_PATH="${1}"
PROTOC_GEN_GO_OUT="${2}"

rm -rf "${PROTOC_GEN_GO_OUT}"
mkdir -p "${PROTOC_GEN_GO_OUT}"
for dir in $(find "${PROTO_PATH}" -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq); do
  protoc "--proto_path=${PROTO_PATH}" "--go_out=${PROTOC_GEN_GO_OUT}" $(find "${dir}" -name '*.proto')
done
