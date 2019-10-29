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

check_env_var GITHUB_REF

check_env_var DOCKER_BUILD_MAKE_TARGET
check_env_var DOCKER_IMAGE
check_env_var DOCKER_USERNAME
check_env_var DOCKER_TOKEN
check_env_var DOCKER_LATEST_BRANCH
check_env_var DOCKER_VERSION_TAG_PREFIX

DOCKER_IMAGE_VERSION=
if echo "${GITHUB_REF}" | grep ^refs/heads/${DOCKER_LATEST_BRANCH}$ >/dev/null; then
  DOCKER_IMAGE_VERSION=latest
elif echo ${GITHUB_REF} | grep ^refs/tags/${DOCKER_VERSION_TAG_PREFIX} >/dev/null; then
  DOCKER_IMAGE_VERSION="$(echo "${GITHUB_REF}" | sed "s/refs\/tags\/${DOCKER_VERSION_TAG_PREFIX}//")"
fi
if [ -z "${DOCKER_IMAGE_VERSION}" ]; then
  echo "skipping due to GITHUB_REF: ${GITHUB_REF}"
  exit 0
fi

echo make "${DOCKER_BUILD_MAKE_TARGET}"
make "${DOCKER_BUILD_MAKE_TARGET}"
echo "${DOCKER_TOKEN}" | docker login --username "${DOCKER_USERNAME}" --password-stdin
if [ "${DOCKER_IMAGE_VERSION}" != "latest" ]; then
  echo docker tag "${DOCKER_IMAGE}:latest" "${DOCKER_IMAGE}:${DOCKER_IMAGE_VERSION}"
  docker tag "${DOCKER_IMAGE}:latest" "${DOCKER_IMAGE}:${DOCKER_IMAGE_VERSION}"
fi
echo docker push "${DOCKER_IMAGE}:${DOCKER_IMAGE_VERSION}"
docker push "${DOCKER_IMAGE}:${DOCKER_IMAGE_VERSION}"
