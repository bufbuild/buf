#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

# Ensure the following environment variables are set:
: ${DATE}
#: ${GH_TOKEN} # However, if you are already logged in for GitHub CLI locally, you can remove this line when running it locally.

make upgrade

BRANCH="upgrade/${DATE}"
git switch -C ${BRANCH}
git add .
git commit -m "Upgrade dependencies ${DATE}"
git push --set-upstream origin ${BRANCH}
PR_URL=$(gh pr create --title "Upgrade dependencies ${DATE}" --body "Make sure to review the changes and merge it if everything looks good." --base main --head ${BRANCH})
echo "Pull request created: ${PR_URL}"
