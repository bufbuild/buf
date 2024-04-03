#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

B4_DATE_DIGEST="$(buf-digest --digest-type shake256 bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep date | cut -f 2 -d ' ')"
B4_EXTENSION_DIGEST="$(buf-digest --digest-type shake256 bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep extension | cut -f 2 -d ' ')"

B5_DATE_DIGEST="$(buf-digest --digest-type b5 bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep date | cut -f 2 -d ' ')"
B5_EXTENSION_DIGEST="$(buf-digest --digest-type b5 bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep extension | cut -f 2 -d ' ')"

DATE_COMMIT_ID="ffded0b4cf6b47cab74da08d291a3c2f"
EXTENSION_COMMIT_ID="b8488077ea6d4f6d9562a337b98259c8"

rm -f workspacev1/finance/bond/proto/buf.lock
cat <<EOF > workspacev1/finance/bond/proto/buf.lock
version: v1
deps:
  - remote: buf.testing
    owner: acme
    repository: date
    commit: ${DATE_COMMIT_ID}
    digest: ${B4_DATE_DIGEST}
  - remote: buf.testing
    owner: acme
    repository: extension
    commit: ${EXTENSION_COMMIT_ID}
    digest: ${B4_EXTENSION_DIGEST}
EOF

rm -f workspacev1/finance/portfolio/proto/buf.lock
cp workspacev1/finance/bond/proto/buf.lock workspacev1/finance/portfolio/proto/buf.lock

rm -f workspacev2/buf.lock
cat <<EOF > workspacev2/buf.lock
version: v2
deps:
  - name: buf.testing/acme/date
    commit: ${DATE_COMMIT_ID}
    digest: ${B5_DATE_DIGEST}
  - name: buf.testing/acme/extension
    commit: ${EXTENSION_COMMIT_ID}
    digest: ${B5_EXTENSION_DIGEST}
EOF

rm -f workspace_undeclared_dep/buf.lock
cp workspacev2/buf.lock workspace_undeclared_dep/buf.lock

rm -f workspace_unused_dep/buf.lock
cp workspacev2/buf.lock workspace_unused_dep/buf.lock
