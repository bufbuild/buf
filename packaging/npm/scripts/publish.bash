#!/usr/bin/env bash

# Publishes the packages in dist/ to the configured npm registry.
#
# Platform packages are published first since @bufbuild/buf depends on them as
# optionalDependencies. Packages already on the registry are skipped so a
# failed run can be re-run to publish the rest.

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

publish_package() {
  local package="${1}"
  local spec
  # jq on Windows writes CRLF line endings, so strip the carriage returns.
  spec="$(tar -xOzf "${package}" package/package.json | jq -r '.name + "@" + .version' | tr -d '\r')"
  local version="${spec##*@}"
  # npm view fails for a version that does not exist, so compare its output
  # instead of checking its exit code.
  if [ "$(npm view "${spec}" version 2>/dev/null || true)" = "${version}" ]; then
    echo "Skipping ${spec}, already published"
    return
  fi
  npm publish --access public "${package}"
}

for PACKAGE in dist/platforms/*.tgz dist/main/*.tgz; do
  publish_package "${PACKAGE}"
done
