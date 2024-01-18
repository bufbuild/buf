#!/usr/bin/env bash

set -eo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

if [ -z "${BASE_BRANCH}" ]; then
  BASE_BRANCH="main"
fi

TODOS="$(git diff "${BASE_BRANCH}" | awk '
  /^diff / {f="?"; next}
  f=="?" {if (/^\+\+\+ /) f=substr($0, 7)"\n"; next}
  /^@@/ {n=$3; sub(/,.*/,"",n); n=0+$3; next}
  /^\+.*TODO/ {print f n ":" substr($0,2); f=""}
  substr($0,1,1)~/[ +]/ {n++}')"

FILENAME=""
while read -r line; do
  if [[ "${line}" =~ [a-z,A-Z,0-9]+\.[a-z]{2,3}$ ]]; then
    FILENAME="${line}"
  elif [[ "${line}" == *"make/buf/scripts/newtodos.bash"* ]]; then
    FILENAME="no print"
  else
    if [[ "${FILENAME}" != "no print" ]]; then 
      LINENUMBER="${line%%:*}"
      TODO="${line#*:}"
      echo "${FILENAME}":"${LINENUMBER}":"${TODO#"${TODO%%[![:space:]]*}"}"
    fi
  fi
done <<< "${TODOS}"
