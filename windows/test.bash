#!/usr/bin/env bash

set -eo pipefail

PROTOC_VERSION="22.0"
PROTOC_GEN_GO_VERSION="v1.28.2-0.20220831092852-f930b1dc76e8"
CONNECT_VERSION="v1.5.2"

# Convert DOWNLOAD_CACHE from d:\path to /d/path
DOWNLOAD_CACHE="$(echo "/${DOWNLOAD_CACHE}" | sed 's|\\|/|g' | sed 's/://')"
mkdir -p "${DOWNLOAD_CACHE}"
PATH="${DOWNLOAD_CACHE}/protoc/bin:${PATH}"

if [ -f "${DOWNLOAD_CACHE}/protoc/bin/protoc.exe" ]; then
  CACHED_PROTOC_VERSION="$("${DOWNLOAD_CACHE}/protoc/bin/protoc.exe" --version | cut -d " " -f 2)"
fi

if [ "${CACHED_PROTOC_VERSION}" != "$PROTOC_VERSION" ]; then
  PROTOC_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-win64.zip"
  curl -sSL -o "${DOWNLOAD_CACHE}/protoc.zip" "${PROTOC_URL}"
  7z x -y -o"${DOWNLOAD_CACHE}/protoc" "${DOWNLOAD_CACHE}/protoc.zip"
  mkdir -p "${DOWNLOAD_CACHE}/protoc/lib"
  cp -a "${DOWNLOAD_CACHE}/protoc/include" "${DOWNLOAD_CACHE}/protoc/lib/include"
else
  echo "Using cached protoc"
fi

PATH="${DOWNLOAD_CACHE}/protoc/bin:${PATH}"

go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}
go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@${CONNECT_VERSION}
go install ./cmd/buf \
  ./private/buf/cmd/buf/command/alpha/protoc/internal/protoc-gen-insertion-point-writer \
  ./private/buf/cmd/buf/command/alpha/protoc/internal/protoc-gen-insertion-point-receiver
go test ./...
