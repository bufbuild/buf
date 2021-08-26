#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

FILE_NAME="usage.gen.go"

find ./private -name "${FILE_NAME}" -delete
for import_path_name in $(go list -f '{{.ImportPath}},{{.Name}}' ./private/... | sed "s/github.com\/bufbuild\/buf/./" | grep -v \.\/private\/usage); do
  import_path="$(echo "${import_path_name}" | cut -f 1 -d ,)"
  name="$(echo "${import_path_name}" | cut -f 2 -d ,)"
  file_path="${import_path}/${FILE_NAME}"
  cat <<EOF > "${file_path}"
// Generated. DO NOT EDIT.

package ${name}

import _ "github.com/bufbuild/buf/private/usage"
EOF
done
