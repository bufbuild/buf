#!/usr/bin/env bash

set -eo pipefail
set -x

DIR="$(cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

fail() {
  echo "error: $@" >&2
  exit 1
}

goos() {
  case "${1}" in
    Darwin) echo darwin ;;
    Linux) echo linux ;;
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
  if [ -z "${DOCKER_IMAGE}" ]; then
    fail "DOCKER_IMAGE must be set"
  fi
  docker run --volume \
    "${DIR}:/app" \
    --workdir "/app" \
    -e INSIDE_DOCKER=1 \
    "${DOCKER_IMAGE}" \
    bash -x make/buf/scripts/release.bash
  if [ "$(uname -s)" == "Linux" ]; then
    sudo chown -R "$(whoami):$(whoami)" .build
  fi
  exit 0
fi

BASE_NAME="buf"

RELEASE_DIR=".build/release/${BASE_NAME}"
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"
cd "${RELEASE_DIR}"

for os in Darwin Linux; do
  for arch in x86_64 arm64; do
    # our goal is to have the binaries be suffixed with $(uname -s)-$(uname -m)
    # on mac, this is arm64, on linux, this is aarch64, for historical reasons
    # this is a hacky way to not have to rewrite this loop (and others below)
    if [ "${os}" == "Linux" ] && [ "${arch}" == "arm64" ]; then
      arch="aarch64"
    fi
    dir="${os}/${arch}/${BASE_NAME}"
    mkdir -p "${dir}/bin"
    for binary in \
      buf \
      protoc-gen-buf-breaking \
      protoc-gen-buf-lint \
      protoc-gen-buf-check-breaking \
      protoc-gen-buf-check-lint; do
      CGO_ENABLED=0 GOOS=$(goos "${os}") GOARCH=$(goarch "${arch}") \
        go build -a -ldflags "-s -w" -trimpath -o "${dir}/bin/${binary}" "${DIR}/cmd/${binary}/main.go"
      cp "${dir}/bin/${binary}" "${binary}-${os}-${arch}"
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
    #mkdir -p "${dir}/share/man/man1"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" bash-completion > "${dir}/etc/bash_completion.d/buf"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" fish-completion > "${dir}/share/fish/vendor_completions.d/buf.fish"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" zsh-completion > "${dir}/share/zsh/site-functions/_buf"
    #"$(uname -s)/$(uname -m)/${1}/bin/buf" manpages "${dir}/share/man/man1"
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
