#!/usr/bin/env bash

# Managed by makego. DO NOT EDIT.

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

# Ensure the following environment variables are set:
: "${GH_TOKEN}" # However, if you are already logged in for GitHub CLI locally, you can remove this line when running it locally.

make upgrade

if ! [[ $(git status --porcelain) ]]; then
  echo "No changes detected. Exiting."
  exit 0
fi

DATE=$(date +"%Y-%m-%d")
BRANCH="make-upgrade-${DATE}"
git switch -C "${BRANCH}"
git add .
git commit -m "Make upgrade"
git push --set-upstream origin "${BRANCH}"
PR_URL=$(gh pr create --title "Make upgrade" --body "Created on ${DATE}." --base main --head "${BRANCH}")
echo "Pull request created: ${PR_URL}"
