# Layout

## Modules in BSR

- `buf.build/acme/extension`

- `buf.build/acme/date`
  - Direct Dep: `buf.build/acme/extension`

## Modules in workspace

- `buf.build/acme/geo` at `common/geo/proto`

- `buf.build/acme/money` at `common/money/proto`

- `buf.build/acme/bond` at `finance/bond/proto`
  - Direct Dep: `buf.build/acme/date`
  - Direct Dep: `buf.build/acme/geo`
  - Direct Dep: `buf.build/acme/money`
  - Transitive Dep: `buf.build/acme/extension`

- `finance/portfolio/proto` (unnamed)
  - Direct Dep: `buf.build/acme/bond`
  - Transitive Dep: `buf.build/acme/date`
  - Transitive Dep: `buf.build/acme/geo`
  - Transitive Dep: `buf.build/acme/money`
  - Transitive Dep: `buf.build/acme/extension`
