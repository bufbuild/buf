# Layout

## Modules in BSR

- `buf.testing/acme/extension`

- `buf.testing/acme/date`
  - Direct Dep: `buf.testing/acme/extension`

## Modules in workspace

- `buf.testing/acme/geo` at `common/geo/proto`

- `buf.testing/acme/money` at `common/money/proto`

- `buf.testing/acme/bond` at `finance/bond/proto`
  - Direct Dep: `buf.testing/acme/date`
  - Direct Dep: `buf.testing/acme/geo`
  - Direct Dep: `buf.testing/acme/money`
  - Transitive Dep: `buf.testing/acme/extension`

- `finance/portfolio/proto` (unnamed)
  - Direct Dep: `buf.testing/acme/bond`
  - Transitive Dep: `buf.testing/acme/date`
  - Transitive Dep: `buf.testing/acme/geo`
  - Transitive Dep: `buf.testing/acme/money`
  - Transitive Dep: `buf.testing/acme/extension`

## Development

```
bash scripts/digests.bash
```
