#!/usr/bin/env bash

# Publishes the packages in dist/ to a local Verdaccio registry and verifies
# that @bufbuild/buf installs and runs on the current platform.

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

: "${VERDACCIO_PORT:=4873}"

VERDACCIO_VERSION="6.10.4"
REGISTRY="http://localhost:${VERDACCIO_PORT}"
# jq on Windows writes CRLF line endings, so strip the carriage returns.
VERSION="$(jq -r .version buf/package.json | tr -d '\r')"
WORK_DIR="$(mktemp -d)"
VERDACCIO_PID=""

cleanup() {
  if [ -n "${VERDACCIO_PID}" ]; then
    kill "${VERDACCIO_PID}" || true
  fi
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

start_verdaccio() {
  cat >"${WORK_DIR}/verdaccio.yaml" <<EOF
storage: ./storage
auth:
  htpasswd:
    file: ./htpasswd
uplinks: {}
packages:
  "@bufbuild/*":
    access: \$all
    publish: \$all
max_body_size: 500mb
log: { type: stdout, format: pretty, level: warn }
EOF
  npx --yes "verdaccio@${VERDACCIO_VERSION}" \
    --config "${WORK_DIR}/verdaccio.yaml" \
    --listen "${VERDACCIO_PORT}" &
  VERDACCIO_PID="${!}"
  local attempt
  for attempt in $(seq 60); do
    if curl -sf "${REGISTRY}/-/ping" >/dev/null; then
      return
    fi
    sleep 1
  done
  echo "verdaccio did not start" >&2
  exit 1
}

publish_packages() {
  # Verdaccio allows anonymous publishing but npm requires some token.
  echo "//localhost:${VERDACCIO_PORT}/:_authToken=anonymous" >"${WORK_DIR}/npmrc"
  NPM_CONFIG_REGISTRY="${REGISTRY}" bash scripts/publish.bash
  # Publish a second time to verify the publish script skips existing versions
  # without failing.
  echo "Re-running publish to check already published packages are skipped"
  NPM_CONFIG_REGISTRY="${REGISTRY}" bash scripts/publish.bash
}

# Checks for a platform package that is referenced by @bufbuild/buf but was not
# published.
check_optional_dependencies() {
  local dependency
  for dependency in $(jq -r '.optionalDependencies | keys[]' buf/package.json | tr -d '\r'); do
    echo "Checking ${dependency}@${VERSION} is published"
    npm view --registry "${REGISTRY}" "${dependency}@${VERSION}" version >/dev/null
  done
}

# Installs @bufbuild/buf into a new project and runs its binaries.
# ${1} is the scenario name, ${2} is the project package.json, and any
# remaining arguments are passed to npm install.
check_install() {
  local name="${1}"
  local package_json="${2}"
  shift 2
  local project_dir="${WORK_DIR}/${name}"

  echo "Checking install: ${name}"
  mkdir -p "${project_dir}"
  echo "${package_json}" >"${project_dir}/package.json"
  (
    cd "${project_dir}"
    # macOS ships bash 3.2, where an empty "${@}" is unbound under set -u.
    npm install --registry "${REGISTRY}" --no-audit --no-fund ${@+"${@}"} "@bufbuild/buf@${VERSION}"

    local actual_version
    actual_version="$(npm exec --no -- buf --version)"
    if [ "${actual_version}" != "${VERSION}" ]; then
      echo "expected buf version ${VERSION}, got ${actual_version}" >&2
      exit 1
    fi

    # The plugins reject an empty request, which is enough to show that the
    # binary was found and executed.
    local plugin
    for plugin in protoc-gen-buf-breaking protoc-gen-buf-lint; do
      local output
      output="$(npm exec --no -- "${plugin}" </dev/null 2>&1 || true)"
      if [[ "${output}" != *"CodeGeneratorRequest"* ]]; then
        echo "unexpected output from ${plugin}: ${output}" >&2
        exit 1
      fi
    done
  )
}

start_verdaccio
export NPM_CONFIG_USERCONFIG="${WORK_DIR}/npmrc"
publish_packages
check_optional_dependencies

# Recent npm versions skip dependency install scripts unless allowed, so only
# bin.ts is exercised.
check_install default '{"private":true}'
# install.ts verifies the binary from the platform package.
check_install allow-scripts '{"private":true,"allowScripts":{"@bufbuild/buf":true}}'
# install.ts falls back to installing the platform package itself.
check_install omit-optional '{"private":true,"allowScripts":{"@bufbuild/buf":true}}' --omit=optional

echo "Smoke test passed"
