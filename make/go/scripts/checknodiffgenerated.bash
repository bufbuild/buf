#!/usr/bin/env bash

# Managed by makego. DO NOT EDIT.

set -euo pipefail

fail() {
  echo "error: $@" >&2
  exit 1
}

STATUS_SHORT_PRE_FILE="$(mktemp)"
STATUS_SHORT_POST_FILE="$(mktemp)"
STATUS_SHORT_DIFF_FILE="$(mktemp)"
trap 'rm -rf "${STATUS_SHORT_PRE_FILE}"' EXIT
trap 'rm -rf "${STATUS_SHORT_PRE_FILE}"' EXIT
trap 'rm -rf "${STATUS_SHORT_PRE_FILE}"' EXIT

git status --short > "${STATUS_SHORT_PRE_FILE}"
"$@"
git status --short > "${STATUS_SHORT_POST_FILE}"
set +e
diff "${STATUS_SHORT_PRE_FILE}" "${STATUS_SHORT_POST_FILE}" > "${STATUS_SHORT_DIFF_FILE}"
set -e

if [ -s "${STATUS_SHORT_DIFF_FILE}" ]; then
  fail "$@ produced a diff,  make sure to check these in:
$(grep '<\|>' "${STATUS_SHORT_DIFF_FILE}")"
fi
