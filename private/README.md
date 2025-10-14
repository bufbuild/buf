# Package structure

- `private/pkg` contains utility packages that are not Buf-company-specific. In theory, any of these packages could be split into a separate repository in the future, and not depend on Buf types or concepts.
- `private/bufpkg` contains utility packages that *are* Buf-company-specific, but not specific to github.com/bufbuild/buf. These packages could be split into separate repositories in the future.
- `private/buf` contains packages specific to github.com/bufbuild/buf. In other repositories, this should be named after that specific repository, for example `github.com/bufbuild/foo` would contain `private/foo`.

There's a strict dependency graph imposed at this level:

- `cmd/{binary}` packages must be `main` packages, and any sub-packages must be `internal`. `cmd/{binary}` packages can import from any `private` package.
- `private/buf` packages can import from `private/bufpkg` and `private/pkg` packages, but not `cmd` packages.
- `private/bufpkg` packages can import from `private/pkg` packages, but not `cmd` or `private/bufpkg` packages.
- `private/pkg` packages can only import from other `private/pkg` packages.

That is, the ordering is strictly `cmd -> private-buf -> private/bufpkg -> private/pkg`. This is enforced by linting tools.
