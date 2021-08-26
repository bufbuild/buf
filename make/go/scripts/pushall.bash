#!/usr/bin/env bash

# Managed by makego. DO NOT EDIT.

set -eo pipefail

fail() {
  echo "error: $@" >&2
  exit 1
}

SYSTEM_GIT="$(command -v git)"

git() {
  if ! "${SYSTEM_GIT}" rev-parse --git-dir 2>/dev/null >/dev/null; then
    echo "fatal: not a git repository (or any of the parent directories)" >&2
    return 1
  fi
  if [ -n "${GIT_USER_NAME}" ] && [ -n "${GIT_USER_EMAIL}" ]; then
    "${SYSTEM_GIT}" -c "user.name=${GIT_USER_NAME}" -c "user.email=${GIT_USER_EMAIL}" "$@"
  else
    "${SYSTEM_GIT}" "$@"
  fi
}

if [ -z "${1}" ]; then
  fail "Usage: ${0} path/to/git/clone"
fi

cd "${1}"

git add --all .
git status
git diff main
if [ -z "$(git status -s)" ]; then
  echo "Nothing to copy, exiting." >&2
  exit 0
fi

while true; do
  read -p "Do you want to commit and push all files [y/n]: " push_all
  case "${push_all}" in
    [Yy] )
      while true; do
        read -p "Enter commit message: " commit_message
        read -p "Is \"${commit_message}\" correct [y/n]: " correct
        case "${correct}" in
          [Yy] )
            git commit -am "${commit_message}"
            git push origin main
            break
            ;;
          [Nn] )
            echo "Aborting."
            exit 1
            ;;
          * )
            echo "Please answer yes or no."
            ;;
        esac
      done
      break
      ;;
    [Nn] )
      echo "Aborting."
      exit 1
      ;;
    * )
      echo "Please answer yes or no."
      ;;
  esac
done
