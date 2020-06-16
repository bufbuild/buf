#!/bin/sh
# Copyright 2020 Buf Technologies, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


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
