#!/usr/bin/env bash

# Managed by makego. DO NOT EDIT.

set -euo pipefail

STATUS_SHORT_PRE_FILE="$(mktemp)"
STATUS_SHORT_POST_FILE="$(mktemp)"
STATUS_SHORT_DIFF_FILE="$(mktemp)"
trap 'rm -rf "${STATUS_SHORT_PRE_FILE}" "${STATUS_SHORT_POST_FILE}" "${STATUS_SHORT_DIFF_FILE}"' EXIT

git status --short > "${STATUS_SHORT_PRE_FILE}"
"$@"
git status --short > "${STATUS_SHORT_POST_FILE}"
set +e
diff "${STATUS_SHORT_PRE_FILE}" "${STATUS_SHORT_POST_FILE}" > "${STATUS_SHORT_DIFF_FILE}"
set -e

if [ -s "${STATUS_SHORT_DIFF_FILE}" ]; then
  echo "error: $@ produced a diff,  make sure to check these in:" >&2
  grep '<\|>' "${STATUS_SHORT_DIFF_FILE}" >&2
  exit 1
fi
