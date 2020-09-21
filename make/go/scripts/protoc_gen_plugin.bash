#!/usr/bin/env bash

# Managed by makego. DO NOT EDIT.

set -eo pipefail

fail() {
  echo "$@" >&2
  exit 1
}

usage() {
  echo "usage: ${0} \
    --proto_path=path/to/one \
    --proto_path=path/to/two \
    --proto_include_path=path/to/one \
    --proto_include_path=path/to/two \
    --plugin_name=go \
    --plugin_out=gen/proto/go \
    --plugin_opt=plugins=grpc"
}

check_flag_value_set() {
  if [ -z "${1}" ]; then
    usage
    exit 1
  fi
}

BUF_PATH=buf
PROTO_PATHS=()
PROTO_INCLUDE_PATHS=()
PLUGIN_NAME=
PLUGIN_OUT=
PLUGIN_OPT=
USE_BUF_PROTOC=
USE_BUF_PROTOC_BY_DIR=
USE_BUF_GENERATE=
while test $# -gt 0; do
  case "${1}" in
    -h|--help)
      usage
      exit 0
      ;;
    --buf_path*)
      BUF_PATH="$(echo ${1} | sed -e 's/^[^=]*=//g')"
      shift
      ;;
    --proto_path*)
      PROTO_PATHS+=("$(echo ${1} | sed -e 's/^[^=]*=//g')")
      shift
      ;;
    --proto_include_path*)
      PROTO_INCLUDE_PATHS+=("$(echo ${1} | sed -e 's/^[^=]*=//g')")
      shift
      ;;
    --plugin_name*)
      PLUGIN_NAME="$(echo ${1} | sed -e 's/^[^=]*=//g')"
      shift
      ;;
    --plugin_out*)
      PLUGIN_OUT="$(echo ${1} | sed -e 's/^[^=]*=//g')"
      shift
      ;;
    --plugin_opt*)
      PLUGIN_OPT="$(echo ${1} | sed -e 's/^[^=]*=//g')"
      shift
      ;;
    --use-buf-protoc)
      USE_BUF_PROTOC=1
      shift
      ;;
    --use-buf-protoc-by-dir)
      USE_BUF_PROTOC_BY_DIR=1
      shift
      ;;
    --use-buf-generate)
      USE_BUF_GENERATE=1
      shift
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

check_flag_value_set "${PLUGIN_NAME}"
check_flag_value_set "${PLUGIN_OUT}"
check_flag_value_set "${PROTO_PATHS[@]}"

mkdir -p "${PLUGIN_OUT}"

if [ -n "${USE_BUF_GENERATE}" ]; then
  BUF_GENERATE_FLAGS=("--plugin=${PLUGIN_NAME}" "--plugin-out=${PLUGIN_OUT}")
  if [ -n "${PLUGIN_OPT}" ]; then
    BUF_GENERATE_FLAGS+=("--plugin-opt=${PLUGIN_OPT}")
  fi
  for proto_path in "${PROTO_PATHS[@]}"; do
    for proto_file in $(find "${proto_path}" -name '*.proto'); do
      BUF_GENERATE_FLAGS+=("--file=${proto_file}")
    done
  done
  echo "${BUF_PATH}" beta generate "${BUF_GENERATE_FLAGS[@]}"
  "${BUF_PATH}" beta generate "${BUF_GENERATE_FLAGS[@]}"
  exit 0
fi

PROTOC_FLAGS=()
for proto_path in "${PROTO_PATHS[@]}"; do
  PROTOC_FLAGS+=("--proto_path=${proto_path}")
done
for proto_path in "${PROTO_INCLUDE_PATHS[@]}"; do
  PROTOC_FLAGS+=("--proto_path=${proto_path}")
done
PROTOC_FLAGS+=("--${PLUGIN_NAME}_out=${PLUGIN_OUT}")
if [ -n "${PLUGIN_OPT}" ]; then
  PROTOC_FLAGS+=("--${PLUGIN_NAME}_opt=${PLUGIN_OPT}")
fi
for proto_path in "${PROTO_PATHS[@]}"; do
  if [ -n "${USE_BUF_PROTOC_BY_DIR}" ]; then
      echo "${BUF_PATH}" protoc --by_dir "${PROTOC_FLAGS[@]}" $(find "${proto_path}" -name '*.proto')
      "${BUF_PATH}" protoc --by_dir "${PROTOC_FLAGS[@]}" $(find "${proto_path}" -name '*.proto')
  else
    for dir in $(find "${proto_path}" -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq); do
      if [ -n "${USE_BUF_PROTOC}" ]; then
        echo "${BUF_PATH}" protoc "${PROTOC_FLAGS[@]}" $(find "${dir}" -name '*.proto')
        "${BUF_PATH}" protoc "${PROTOC_FLAGS[@]}" $(find "${dir}" -name '*.proto')
      else
        echo protoc --experimental_allow_proto3_optional "${PROTOC_FLAGS[@]}" $(find "${dir}" -name '*.proto')
        protoc --experimental_allow_proto3_optional "${PROTOC_FLAGS[@]}" $(find "${dir}" -name '*.proto')
      fi
    done
  fi
done
