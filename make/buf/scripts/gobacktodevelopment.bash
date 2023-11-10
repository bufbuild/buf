#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

# We already have set -u, but want to fail early if a required variable is not set.
: ${WEBHOOK_URL}
# However, if you are already logged in for GitHub CLI locally, you can remove this line when running it locally.
: ${GH_TOKEN}

RELEASED_VERSION_LINE=$(grep -oE 'Version.*=.*\"[0-9]\.[0-9]+\.[0-9]+[^\"]*' private/buf/bufcli/bufcli.go)
RELEASED_VERSION=${RELEASED_VERSION_LINE##Version*=*\"}

NEXT_VERSION=$(awk -F. -v OFS=. '{$NF += 1 ; print}' <<< ${RELEASED_VERSION})
NEXT_VERSION="${NEXT_VERSION}-dev"

make updateversion VERSION=${NEXT_VERSION}

if [[ "${OSTYPE}" == "linux-gnu"* ]]; then
  SED_BIN=sed
elif [[ "${OSTYPE}" == "darwin"* ]]; then
  SED_BIN=gsed
else
  echo "unsupported OSTYPE: ${OSTYPE}"
  exit 1
fi

${SED_BIN} -i "/^# Changelog/ {
N;
a\
## [Unreleased]\\
\\
- No changes yet.\\

}" CHANGELOG.md

${SED_BIN} -i "/^Initial beta release.$/ {
N;
a\
[Unreleased]: https://github.com/bufbuild/buf/compare/v${RELEASED_VERSION}...HEAD
}" CHANGELOG.md

BRANCH="next/v${RELEASED_VERSION}"
git switch -C ${BRANCH}
git add .
git commit -m "Back to development"
git push --set-upstream origin --force ${BRANCH} 
url=$(gh pr create --title "Return to Development" --body "Release complete for v${RELEASED_VERSION}")

jq --null-input "{ text: \"PR back to development: ${url}\" }" | curl -sSL -X POST -H 'Content-Type: application/json' -d@- "${WEBHOOK_URL}"
