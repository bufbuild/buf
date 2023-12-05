#!/usr/bin/env bash

set -euo pipefail

DIR="$(CDPATH= cd "$(dirname "${0}")/.." && pwd)"
cd "${DIR}"

DATE_DIGEST="$(buf-digest bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep date | cut -f 2 -d ' ')"
EXTENSION_DIGEST="$(buf-digest bsr/buf.testing/acme/date bsr/buf.testing/acme/extension | grep extension | cut -f 2 -d ' ')"

rm -f workspacev1/finance/bond/proto/buf.lock
cat <<EOF > workspacev1/finance/bond/proto/buf.lock
version: v2
deps:
  - name: buf.testing/acme/date
    digest: ${DATE_DIGEST}
  - name: buf.testing/acme/extension
    digest: ${EXTENSION_DIGEST}
EOF

rm -f workspacev1/finance/portfolio/proto/buf.lock
cat <<EOF > workspacev1/finance/portfolio/proto/buf.lock
version: v2
deps:
  - name: buf.testing/acme/date
    digest: ${DATE_DIGEST}
  - name: buf.testing/acme/extension
    digest: ${EXTENSION_DIGEST}
EOF

rm -f workspacev2/buf.lock
cat <<EOF > workspacev2/buf.lock
version: v2
deps:
  - name: buf.testing/acme/date
    digest: ${DATE_DIGEST}
  - name: buf.testing/acme/extension
    digest: ${EXTENSION_DIGEST}
EOF
