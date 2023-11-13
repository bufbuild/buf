#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

# We already have set -u, but want to fail early if a required variable is not set.
: ${WEBHOOK_URL}
# However, if you are already logged in for GitHub CLI locally, you can remove this line when running it locally.
: ${GH_TOKEN}

if [[ "${VERSION}" == v* ]]; then
  echo "error: VERSION ${VERSION} must not start with 'v'" >&2
  exit 1
fi

make updateversion
make releasechangelog

BRANCH="release/v${VERSION}"
git switch -C ${BRANCH}
git add .
git commit -m "Update version to ${VERSION}"
git push --set-upstream origin --force ${BRANCH} 
url=$(gh pr create --title "Release v${VERSION}" --body "Release prepared for ${VERSION}
Reminder: Verify the changelog")

jq --null-input "{text: \"BufCLI Release v${VERSION} has started: ${url}\" }" | curl -sSL -X POST -H "Content-Type: application/json" -d@- "${WEBHOOK_URL}"
