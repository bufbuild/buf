#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

files=`(git fetch origin main:main) && (git diff --name-only main)`
for file in $files; do
if [ "$file" = "CHANGELOG.md" ]; then
    exit 0
fi
done
echo ERROR: CHANGELOG has not been updated
exit 1
