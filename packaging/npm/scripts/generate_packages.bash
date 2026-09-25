#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

GITHUB_RELEASES_BASE="https://github.com/bufbuild/buf/releases/download"
BINARIES=(buf protoc-gen-buf-breaking protoc-gen-buf-lint)

VERSION="$(jq -r .version buf/package.json)"
CHECKSUMS="$(curl -sSLf "${GITHUB_RELEASES_BASE}/v${VERSION}/sha256.txt")"

verify_checksum() {
  local path="${1}"
  local filename="${2}"
  local expected
  expected="$(awk -v name="${filename}" '$2 == name { print $1 }' <<<"${CHECKSUMS}")"
  local actual
  actual="$(shasum -a 256 "${path}" | awk '{ print $1 }')"
  if [ -z "${expected}" ] || [ "${expected}" != "${actual}" ]; then
    echo "checksum mismatch for ${filename}: expected ${expected}, got ${actual}" >&2
    exit 1
  fi
}

# Downloads the binaries for a single platform and packs its npm package.
# ${1} is the buf release platform suffix, ${2} is the npm package suffix.
generate_platform_package() {
  local buf_platform="${1}"
  local npm_platform="${2}"
  local package_dir="buf-${npm_platform}"
  local ext=""
  if [[ "${npm_platform}" == win32-* ]]; then
    ext=".exe"
  fi

  echo "Generating package for ${buf_platform} (${npm_platform})"
  rm -rf "${package_dir}/bin"
  mkdir -p "${package_dir}/bin"
  local binary
  for binary in "${BINARIES[@]}"; do
    local filename="${binary}-${buf_platform}${ext}"
    local dest="${package_dir}/bin/${binary}${ext}"
    echo "  Downloading ${GITHUB_RELEASES_BASE}/v${VERSION}/${filename}"
    curl -sSLf "${GITHUB_RELEASES_BASE}/v${VERSION}/${filename}" -o "${dest}"
    verify_checksum "${dest}" "${filename}"
    chmod +x "${dest}"
  done

  npm pack "./${package_dir}" --pack-destination dist/platforms
}

echo "Generating npm packages for buf v${VERSION}"
rm -rf dist
mkdir -p dist/platforms dist/main

generate_platform_package Darwin-arm64 darwin-arm64
generate_platform_package Darwin-x86_64 darwin-x64
generate_platform_package Linux-aarch64 linux-aarch64
generate_platform_package Linux-armv7 linux-armv7
generate_platform_package Linux-x86_64 linux-x64
generate_platform_package Windows-arm64 win32-arm64
generate_platform_package Windows-x86_64 win32-x64

echo "Generating package for @bufbuild/buf"
cp ../../README.md buf/README.md
# The platform packages for this version may not be published yet; npm skips
# optional dependencies that fail to install.
(cd buf && npm install --ignore-scripts --no-audit --no-fund)
npm pack ./buf --pack-destination dist/main

echo "Done. Packages written to dist/"
