#!/usr/bin/env bash

# Managed by makego. DO NOT EDIT.

## checkcmdpackage exits with exit code 1 if the given package has any sub-packages that are not in internal
##
## Note that we would check to make sure that there are no exported types from main, but this is a builtin
## feature of Go (you cannot export types from main to use in other packages)

set -euo pipefail

NON_INTERNAL_SUB_PACKAGES="$(go list "${1}/..." | grep -v ^$(go list "${1}")\/internal)"
if [ "${NON_INTERNAL_SUB_PACKAGES}" != "$(go list ${1})" ]; then
  echo "${1} had non-sub-packages outside of ${1}/internal which is not allowed:" >&2
  echo "${NON_INTERNAL_SUB_PACKAGES}" >&2
  exit 1
fi
