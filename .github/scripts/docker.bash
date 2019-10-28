#!/usr/bin/env bash

set -eo pipefail

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
cd "${DIR}"

fail() {
  echo "error: $@" >&2
  exit 1
}

check_env_var() {
  if [ -z "${!1}" ]; then
    fail "${1} not set"
  fi
}

# only set on forks
# https://help.github.com/en/github/automating-your-workflow-with-github-actions/virtual-environments-for-github-actions#environment-variables
if [ -n "${GITHUB_HEAD_REF}" ]; then
  echo "skipping due to fork"
  exit 0
fi

check_env_var GITHUB_ACTOR
check_env_var GITHUB_REF
check_env_var GITHUB_TOKEN

MAKE_BUILD_TARGET=dockerbuildbuf
IMAGE=docker.pkg.github.com/bufbuild/buf/buf
LATEST_BRANCH=master
VERSION_TAG_PREFIX=v
IMAGE_VERSION=

if echo "${GITHUB_REF}" | grep ^refs/heads/${LATEST_BRANCH}$ >/dev/null; then
  IMAGE_VERSION=latest
elif echo ${GITHUB_REF} | grep ^refs/tags/${VERSION_TAG_PREFIX} >/dev/null; then
  IMAGE_VERSION="$(echo "${GITHUB_REF}" | sed "s/refs\/tags\/${VERSION_TAG_PREFIX}//")"
fi

if [ -z "${IMAGE_VERSION}" ]; then
  echo "skipping due to GITHUB_REF: ${GITHUB_REF}"
  exit 0
fi

echo make "${MAKE_BUILD_TARGET}"
make "${MAKE_BUILD_TARGET}"
echo "${GITHUB_TOKEN}" | docker login docker.pkg.github.com --username "${GITHUB_ACTOR}" --password-stdin
if [ "${IMAGE_VERSION}" != "latest" ]; then
  echo docker tag "${IMAGE}:latest" "${IMAGE}:${IMAGE_VERSION}"
  docker tag "${IMAGE}:latest" "${IMAGE}:${IMAGE_VERSION}"
fi
echo docker push "${IMAGE}:${IMAGE_VERSION}"
docker push "${IMAGE}:${IMAGE_VERSION}"
