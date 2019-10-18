#!/usr/bin/env bash

set -eo pipefail

DIR="$(cd "$(dirname "${0}")/../.." && pwd)"
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

BASE_NAME="buf"

RELEASE_DIR=".build/release/${BASE_NAME}"
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"
cd "${RELEASE_DIR}"

for os in Darwin Linux; do
  for arch in x86_64; do
    dir="${os}/${arch}/${BASE_NAME}"
    mkdir -p "${dir}/bin"
    for binary in buf protoc-gen-buf-check-breaking protoc-gen-buf-check-lint; do
      CGO_ENABLED=0 GOOS=$(goos "${os}") GOARCH=$(goarch "${arch}") \
        go build -a -o "${dir}/bin/${binary}" $(find "${DIR}/cmd/${binary}" -name '*.go')
      cp "${dir}/bin/${binary}" "${binary}-${os}-${arch}"
    done
  done
done

for os in Darwin Linux; do
  for arch in x86_64; do
    dir="${os}/${arch}/${BASE_NAME}"
    mkdir -p "${dir}/etc/bash_completion.d"
    mkdir -p "${dir}/etc/zsh/site-functions"
    #mkdir -p "${dir}/share/man/man1"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" bash-completion > "${dir}/etc/bash_completion.d/buf"
    "$(uname -s)/$(uname -m)/${BASE_NAME}/bin/buf" zsh-completion > "${dir}/etc/zsh/site-functions/_buf"
    #"$(uname -s)/$(uname -m)/${1}/bin/buf" manpages "${dir}/share/man/man1"
    cp -R "${DIR}/LICENSE" "${dir}/LICENSE"
  done
done

for os in Darwin Linux; do
  for arch in x86_64; do
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

VERSION="$(./${BASE_NAME}-$(uname -s)-$(uname -m) --version 2>&1)"
DARWIN_SHA256="$(grep "${BASE_NAME}-Darwin-x86_64.tar.gz" sha256.txt | cut -f 1 -d ' ')"
LINUX_SHA256="$(grep "${BASE_NAME}-Linux-x86_64.tar.gz" sha256.txt | cut -f 1 -d ' ')"

mkdir -p Formula
cat <<EOF >Formula/buf.rb
class Buf < Formula
  desc "A new way of working with Protocol Buffers."
  homepage "https://buf.build"
  version "${VERSION}"
  bottle :unneeded

  if OS.mac?
    url "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-Darwin-x86_64.tar.gz"
    sha256 "${DARWIN_SHA256}"
  elsif OS.linux?
    url "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-Linux-x86_64.tar.gz"
    sha256 "${LINUX_SHA256}"
  end

  def install
    bin.install "bin/buf"
    bin.install "bin/protoc-gen-buf-check-breaking"
    bin.install "bin/protoc-gen-buf-check-lint"
    bash_completion.install "etc/bash_completion.d/buf"
    zsh_completion.install "etc/zsh/site-functions/_buf"
    prefix.install "LICENSE"
  end

  test do
    system "#{bin}/buf --version"
  end
end
EOF

mkdir -p assets
for file in $(find . -maxdepth 1 -type f | sed 's/^\.\///' | sort | uniq); do
  mv "${file}" "assets/${file}"
done
