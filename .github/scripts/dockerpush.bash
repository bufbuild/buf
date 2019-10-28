#!/usr/bin/env bash

set -eo pipefail

fail() {
  echo "error: $@" >&2
  exit 1
}

check_env_var() {
  if [ -z "${!1}" ]; then
    fail "${1} not set"
  fi
}

if [ -z "${GITHUB_HEAD_REF}" ]; then
  echo "skipping due to fork"
  exit 0
fi

check_env_var GITHUB_ACTOR
check_env_var GITHUB_REF
check_env_var GITHUB_TOKEN


check_env_var IMAGE
check_env_var LATEST_BRANCH
check_env_var VERSION_TAG_PREFIX

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

echo "IMAGE_VERSION: ${IMAGE_VERSION}"

echo "${GITHUB_TOKEN}" | docker login docker.pkg.github.com --username "${GITHUB_ACTOR}" --password-stdin
echo docker push "${IMAGE}:${IMAGE_VERSION}"
