#!/usr/bin/env bash

set -euo pipefail

curl -sSL "https://raw.githubusercontent.com/bufbuild/homebrew-buf/main/Formula/buf.rb" \
  | grep 'version "' \
  | sed 's/.*version "//' \
  | sed 's/"//'
