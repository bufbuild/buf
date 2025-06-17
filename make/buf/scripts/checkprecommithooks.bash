#!/usr/bin/env bash

set -eo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

GO_VERSION=$(grep '^go ' go.mod | awk '{print $2}')

function lesser_version() {
  echo -e "$1\n$2" | sort -V | head -n 1
}

for version in $(yq ".[].language_version" .pre-commit-hooks.yaml)
do
  LESSER_VERSION=$(lesser_version "${GO_VERSION}" "${version}")
  if [ ${version} == "${LESSER_VERSION}" ]; then
    echo "found lower pre-commit hook version ${version} compared to go.mod version ${GO_VERSION}"
    exit 1
  fi
done
