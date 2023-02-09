#!/usr/bin/env bash

set -eo pipefail
set -x

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

fail() {
  echo "error: $@" >&2
  exit 1
}

goos() {
  case "${1}" in
    Darwin) echo darwin ;;
    Linux) echo linux ;;
    Windows) echo windows ;;
    *) return 1 ;;
  esac
}

goarch() {
  case "${1}" in
    x86_64) echo amd64 ;;
    arm64) echo arm64 ;;
    aarch64) echo arm64 ;;
    *) return 1 ;;
  esac
}

sha256() {
  if ! type sha256sum >/dev/null 2>/dev/null; then
    if ! type shasum >/dev/null 2>/dev/null; then
      echo "sha256sum and shasum are not installed" >&2
      return 1
    else
      shasum -a 256 "$@"
    fi
  else
    sha256sum "$@"
  fi
}

if [ -z "${INSIDE_DOCKER}" ]; then
  if [ -z "${RELEASE_MINISIGN_PRIVATE_KEY}" -o -z "${RELEASE_MINISIGN_PRIVATE_KEY_PASSWORD}" ]; then
    fail "RELEASE_MINISIGN_PRIVATE_KEY and RELEASE_MINISIGN_PRIVATE_KEY_PASSWORD must be set."
  fi
  if [ -z "${DOCKER_IMAGE}" ]; then
    fail "DOCKER_IMAGE must be set"
  fi
  docker run --volume \
    "${DIR}:/app" \
    --workdir "/app" \
    --rm \
    -e INSIDE_DOCKER=1 \
    "${DOCKER_IMAGE}" \
    bash -x make/buf/scripts/release.bash
  if [ "$(uname -s)" == "Linux" ]; then
    sudo chown -R "$(id -u):$(id -g)" .build
  fi
  # Produce the signature outside the docker image where we have
  # minisign installed.
  secret_key_file="$(mktemp)"
  trap "rm ${secret_key_file}" EXIT
  # Prevent printing of private key and password
  set +x
  echo "${RELEASE_MINISIGN_PRIVATE_KEY}" > "${secret_key_file}"
  echo "${RELEASE_MINISIGN_PRIVATE_KEY_PASSWORD}" | minisign -S -s "${secret_key_file}" -m .build/release/buf/assets/sha256.txt
  set -x
  exit 0
fi

BASE_NAME="buf"

RELEASE_DIR=".build/release/${BASE_NAME}"
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"
cd "${RELEASE_DIR}"

for os in Darwin Linux Windows; do
  for arch in x86_64 arm64; do
    # our goal is to have the binaries be suffixed with $(uname -s)-$(uname -m)
    # on mac, this is arm64, on linux, this is aarch64, for historical reasons
    # this is a hacky way to not have to rewrite this loop (and others below)
    if [ "${os}" == "Linux" ] && [ "${arch}" == "arm64" ]; then
      arch="aarch64"
    fi
    extension=""
    if [ "${os}" == "Windows" ]; then
      extension=".exe"
    fi
    dir="${os}/${arch}/${BASE_NAME}"
    mkdir -p "${dir}/bin"
    for binary in \
      buf \
      protoc-gen-buf-breaking \
      protoc-gen-buf-lint; do
      CGO_ENABLED=0 GOOS=$(goos "${os}") GOARCH=$(goarch "${arch}") \
        go build -a -ldflags "-s -w" -trimpath -buildvcs=false -o "${dir}/bin/${binary}${extension}" "${DIR}/cmd/${binary}"
      cp "${dir}/bin/${binary}${extension}" "${binary}-${os}-${arch}${extension}"
    done
  done
done

for os in Darwin Linux; do
  for arch in x86_64 arm64; do
    if [ "${os}" == "Linux" ] && [ "${arch}" == "arm64" ]; then
      arch="aarch64"
    fi
    dir="${os}/${arch}/${BASE_NAME}"
    mkdir -p "${dir}/etc/bash_completion.d"
    mkdir -p "${dir}/share/fish/vendor_completions.d"
    mkdir -p "${dir}/share/zsh/site-functions"
    mkdir -p "${dir}/share/man/man1"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" completion bash > "${dir}/etc/bash_completion.d/buf"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" completion fish > "${dir}/share/fish/vendor_completions.d/buf.fish"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" completion zsh > "${dir}/share/zsh/site-functions/_buf"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" manpages "${dir}/share/man/man1"
    cp -R "${DIR}/LICENSE" "${dir}/LICENSE"
  done
done

for os in Darwin Linux; do
  for arch in x86_64 arm64; do
    if [ "${os}" == "Linux" ] && [ "${arch}" == "arm64" ]; then
      arch="aarch64"
    fi
    dir="${os}/${arch}/${BASE_NAME}"
    tar_context_dir="$(dirname "${dir}")"
    tar_dir="${BASE_NAME}"
    tarball="${BASE_NAME}-${os}-${arch}.tar.gz"
    tar -C "${tar_context_dir}" -cvzf "${tarball}" "${tar_dir}"
  done
done

for file in $(find . -maxdepth 1 -type f | sed 's/^\.\///' | sort | uniq); do
  sha256 "${file}" >> sha256.txt
done
sha256 -c sha256.txt

mkdir -p assets
for file in $(find . -maxdepth 1 -type f | sed 's/^\.\///' | sort | uniq); do
  mv "${file}" "assets/${file}"
done

echo Upload all the files in this directory to GitHub: open "${RELEASE_DIR}/assets"
