#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname ${0})/.." && pwd)"
cd "${DIR}"

TEST_DIR="./private/buf/cmd/buf/testdata/largeworkspace"
rm -rf "${TEST_DIR}"
mkdir -p "${TEST_DIR}"
cd "${TEST_DIR}"

cat <<EOF >buf.work.yaml
version: v1
directories:
EOF

for i in $(seq 1 20); do
  echo "  - proto${i}" >> buf.work.yaml
  mkdir -p "proto${i}/pkg${i}/v1"
  for j in $(seq 1 50); do
    cat <<EOF > "proto${i}/pkg${i}/v1/${j}.proto"
syntax = "proto3";

package pkg${i}.v1;

message FooPkg${i}File${j} {
  int64 one = 1;
}
EOF
  done
done
