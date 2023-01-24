#!/usr/bin/env bash

set -x

if [[ -z "$VERSION" ]]; then
    echo "Must provide VERSION in environment to publish" 1>&2
    exit 1
fi

# Allow an additional version to let us bump the version published to npm.

if [[ -z "${REVISION}" ]]; then
	NPM_VERSION="$VERSION"
else
	NPM_VERSION="$VERSION-$REVISION"
fi

function update_npm_platform_files() {
	mkdir -p "npm/@bufbuild/buf-$2/bin"

	if [[ $2 =~ "win32" ]]
	then
		# Download the binaries
		curl -Lf https://github.com/bufbuild/buf/releases/download/v$VERSION/buf-$1.exe -o npm/@bufbuild/buf-$2/bin/buf.exe
		curl -Lf https://github.com/bufbuild/buf/releases/download/v$VERSION/protoc-gen-buf-breaking-$1.exe -o npm/@bufbuild/buf-$2/bin/protoc-gen-buf-breaking.exe
		curl -Lf https://github.com/bufbuild/buf/releases/download/v$VERSION/protoc-gen-buf-breaking-$1.exe -o npm/@bufbuild/buf-$2/bin/protoc-gen-buf-lint.exe
	else
		# Download the binaries
		curl -Lf https://github.com/bufbuild/buf/releases/download/v$VERSION/buf-$1 -o npm/@bufbuild/buf-$2/bin/buf
		curl -Lf https://github.com/bufbuild/buf/releases/download/v$VERSION/protoc-gen-buf-breaking-$1 -o npm/@bufbuild/buf-$2/bin/protoc-gen-buf-breaking
		curl -Lf https://github.com/bufbuild/buf/releases/download/v$VERSION/protoc-gen-buf-breaking-$1 -o npm/@bufbuild/buf-$2/bin/protoc-gen-buf-lint
		chmod +x npm/@bufbuild/buf-$2/bin/buf
		chmod +x npm/@bufbuild/buf-$2/bin/protoc-gen-buf-breaking
		chmod +x npm/@bufbuild/buf-$2/bin/protoc-gen-buf-lint
	fi
	

	# Update the version in the package.json to newest version
	jq ".version=\"$NPM_VERSION\"" npm/@bufbuild/buf-$2/package.json > npm/@bufbuild/buf-$2/package.json.tmp
	mv npm/@bufbuild/buf-$2/package.json.tmp npm/@bufbuild/buf-$2/package.json

	# Update the version referenced in @bufbuild/buf
	jq ".optionalDependencies.\"@bufbuild/buf-$2\"=\"$NPM_VERSION\"" npm/@bufbuild/buf/package.json > npm/@bufbuild/buf/package.json.tmp
	mv npm/@bufbuild/buf/package.json.tmp npm/@bufbuild/buf/package.json
	(cd npm/@bufbuild/buf-$2 && npm publish --access restricted)
}

function update_npm_package() {
	# Update the version in the package.json to newest version
	jq ".version=\"$NPM_VERSION\"" npm/@bufbuild/buf/package.json > npm/@bufbuild/buf/package.json.tmp
	mv npm/@bufbuild/buf/package.json.tmp npm/@bufbuild/buf/package.json
	(cd npm/@bufbuild/buf && npm install --ignore-scripts && npm publish --access restricted)
}

update_npm_platform_files Darwin-arm64 darwin-arm64
update_npm_platform_files Darwin-x86_64 darwin-x64
update_npm_platform_files Linux-aarch64 linux-aarch64
update_npm_platform_files Linux-x86_64 linux-x64
update_npm_platform_files Windows-arm64 win32-arm64
update_npm_platform_files Windows-x86_64 win32-x64
update_npm_package
