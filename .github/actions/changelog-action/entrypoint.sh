#!/usr/bin/env bash

set -euo pipefail

FILENAME="CHANGELOG.md"
PRINT="false"
# While loop to read line by line
while IFS= read -r LINE; do
  # If the line starts with ## & currently printing, disable PRINT
  if [[ "${LINE}" == "##"* ]] && [[ "${PRINT}" == "true" ]]; then
    break
  fi
  # If printing is enabled, print the line.
  if [[ "${PRINT}" == "true" ]]; then
    echo "${LINE}"
  fi
  # If the line starts with ## & not currently printing, enable PRINT
  if [[ "${LINE}" == "## [${VERSION}]"* ]] && [[ "${PRINT}" == "false" ]]; then
    PRINT="true"
  fi
done <"${FILENAME}"
