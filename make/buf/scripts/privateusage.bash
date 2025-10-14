#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

FILE_NAME="usage.gen.go"

find ./private -name "${FILE_NAME}" -delete
for dir_and_name in $(go list -f '{{.Dir}},{{.Name}}' ./private/... | grep -v \.\/private\/usage); do
  dir_path="$(echo "${dir_and_name}" | cut -f 1 -d ,)"
  name="$(echo "${dir_and_name}" | cut -f 2 -d ,)"
  file_path="${dir_path}/${FILE_NAME}"
  cat <<EOF > "${file_path}"
// Generated. DO NOT EDIT.

package ${name}

import _ "github.com/bufbuild/buf/private/usage"
EOF
done
