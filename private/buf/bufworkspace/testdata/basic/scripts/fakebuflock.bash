#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

DATE_DIGEST="$(buf-digest bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep date | cut -f 2 -d ' ')"
EXTENSION_DIGEST="$(buf-digest bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep extension | cut -f 2 -d ' ')"
DATE_COMMIT_ID="$(buf-new-commit-id)"
EXTENSION_COMMIT_ID="$(buf-new-commit-id)"

rm -f workspacev1/finance/bond/proto/buf.lock
cat <<EOF > workspacev1/finance/bond/proto/buf.lock
version: v2
deps:
  - name: buf.testing/acme/date
    commit: ${DATE_COMMIT_ID}
    digest: ${DATE_DIGEST}
  - name: buf.testing/acme/extension
    commit: ${EXTENSION_COMMIT_ID}
    digest: ${EXTENSION_DIGEST}
EOF

rm -f workspacev1/finance/portfolio/proto/buf.lock
cp workspacev1/finance/bond/proto/buf.lock workspacev1/finance/portfolio/proto/buf.lock

rm -f workspacev2/buf.lock
cp workspacev1/finance/bond/proto/buf.lock workspacev2/buf.lock

rm -f workspace_undeclared_dep/buf.lock
cp workspacev1/finance/bond/proto/buf.lock workspace_undeclared_dep/buf.lock

rm -f workspace_unused_dep/buf.lock
cp workspacev1/finance/bond/proto/buf.lock workspace_unused_dep/buf.lock
