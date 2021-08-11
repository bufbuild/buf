#!/bin/sh

set -e

DIR="$(CDPATH= cd "$(dirname "${0}")/../../.." && pwd)"
cd "${DIR}"

[ -n "$FUZZ_DIR" ] || (echo "FUZZ_DIR is required" && exit 1)
[ -n "$GO_FUZZ_VERSION" ] || (echo "GO_FUZZ_VERSION is required" && exit 1)

git diff --exit-code --quiet go.mod go.sum || (echo "go.sum and go.mod must be unmodified" && exit 1)

go get github.com/dvyukov/go-fuzz/go-fuzz-dep@"$GO_FUZZ_VERSION"
trap "git checkout -- go.mod go.sum" EXIT

mkdir -p "$FUZZ_DIR"/corpus "$FUZZ_DIR"/crashers
cp internal/buf/bufimage/bufimagebuild/bufimagebuildtesting/corpus/* "$FUZZ_DIR"/corpus
cp internal/buf/bufimage/bufimagebuild/bufimagebuildtesting/crashers/* "$FUZZ_DIR"/crashers

(
  cd internal/buf/bufimage/bufimagebuild/bufimagebuildtesting
  go-fuzz-build -o "$FUZZ_DIR"/gofuzz.zip
)

go-fuzz -bin "$FUZZ_DIR"/gofuzz.zip -workdir "$FUZZ_DIR"
