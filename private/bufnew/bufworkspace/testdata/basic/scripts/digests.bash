#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

buf-digest \
  "bsr/buf.build/acme/date" \
  "bsr/buf.build/acme/extension" \
  "workspace/common/geo/proto" \
  "workspace/common/money/proto" \
  "workspace/finance/bond/proto" \
  "workspace/finance/portfolio/proto" \
