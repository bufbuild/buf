![The Buf logo](https://raw.githubusercontent.com/bufbuild/buf/main/.github/buf-logo.svg)

# Buf

[![License](https://img.shields.io/github/license/bufbuild/buf?color=blue)](https://github.com/bufbuild/buf/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/bufbuild/buf?include_prereleases)](https://github.com/bufbuild/buf/releases)
[![CI](https://github.com/bufbuild/buf/workflows/ci/badge.svg)](https://github.com/bufbuild/buf/actions?workflow=ci)
[![Docker](https://img.shields.io/docker/pulls/bufbuild/buf)](https://hub.docker.com/r/bufbuild/buf)
[![Homebrew](https://img.shields.io/homebrew/v/buf)](https://github.com/bufbuild/homebrew-buf)
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)][badges_slack]

Buf is the modern toolchain for [Protobuf][protobuf]. It replaces day-to-day `protoc` use with a fast compiler, module-aware workspaces, formatting, linting, breaking-change detection, code generation, dependency management, API calls, and a client for the [Buf Schema Registry][bsr].

If you are still driving Protobuf with shell scripts around `protoc -I ...`, Buf is the upgrade you want: the same schema language, the same generated-code plugin model, fewer moving parts, and a direct path from local `.proto` files to governed, versioned APIs.

## Start

Install `buf` with [Homebrew][brew]:

```sh
brew install bufbuild/buf/buf
```

Initialize a workspace and run the checks you should expect every Protobuf repository to pass:

```sh
buf config init
buf build
buf format -w
buf lint
buf breaking --against '.git#branch=main'
```

Generate code from a checked-in `buf.gen.yaml` instead of a hand-maintained shell command:

```sh
buf generate
```

For a guided walkthrough from an empty workspace to a working Connect service, run the [Buf CLI quickstart][cli_quickstart].

## Why Buf wins

<a name="features"></a>

| Protobuf work | With `protoc` and scripts | With Buf |
| --- | --- | --- |
| Finding files | Maintain `-I` paths and hope import order does not change behavior. | Declare modules once in `buf.yaml`; Buf discovers files and rejects ambiguous imports. |
| Compiling | Manage a local `protoc` install and parse changing stderr output. | Use Buf's internal compiler, tested against `protoc` descriptor output and built for deterministic parallel compilation. |
| Style | Rely on review comments or separate tooling. | Run `buf lint` locally, in editors, in CI, and on the BSR with 40+ built-in rules plus custom plugins. |
| Compatibility | Find breakage after generated code fails, clients fail, or serialized data becomes unreadable. | Run `buf breaking` against Git, a BSR module, a tarball, a zip file, or a Buf image before merge. |
| Code generation | Keep plugin binaries installed on every machine and encode behavior in long commands. | Put plugins, outputs, options, inputs, and managed-mode settings in `buf.gen.yaml`; use local or remote plugins. |
| Dependencies | Copy `.proto` files between repositories or vendor them by hand. | Declare BSR module dependencies in `buf.yaml` and pin them in `buf.lock`. |
| API consumers | Send people your schemas and generation instructions. | Publish to the BSR and let consumers install generated SDKs with `go get`, `npm install`, Maven, Gradle, `pip install`, NuGet, Cargo, SwiftPM, CMake, or an archive. |
| Governance | Reimplement checks in every repository and hope every team keeps them enabled. | Enforce breaking-change, uniqueness, and custom policies at the BSR layer. |

Core CLI features work without a BSR account. Signing in to the registry adds distribution, remote plugins, generated SDKs, hosted docs, dependency resolution for private modules, and server-side checks when you need them.

## Core workflow

Buf treats a directory tree of `.proto` files as a module, and a project as a workspace. A small `buf.yaml` is enough to make build, lint, breaking-change detection, generation, dependency resolution, and publishing agree on the same input.

```yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

From there, the useful commands are deliberately boring:

```sh
buf build
buf format -w
buf lint
buf breaking --against '.git#branch=main'
buf generate
buf push
```

`buf build` compiles the workspace. `buf lint` catches API-shape problems while the author is still editing. `buf breaking` compares the current schema against a previous version and flags source, JSON, or wire-format incompatibilities. `buf generate` runs `protoc` plugins from a checked-in template. `buf push` publishes named modules to the BSR.

## Code generation

`buf generate` is compatible with the normal `protoc` plugin model, but it moves generation into versioned configuration. This example generates Go Protobuf types and ConnectRPC handlers from `proto/` using remote plugins hosted on the BSR:

```yaml
version: v2
clean: true
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/acme/weather/gen/go
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/connectrpc/gosimple
    out: gen/go
    opt:
      - paths=source_relative
      - simple
inputs:
  - directory: proto
```

Remote plugins remove the need to install and maintain generator binaries on every developer machine or CI runner. Managed mode lets API producers keep language-specific file options out of `.proto` files while consumers still get correct generated package names for their target language.

Local plugins work too. If a plugin speaks the standard Protobuf plugin protocol, Buf can run it.

## Breaking changes

Protobuf compatibility is not one thing. Renaming a field can break generated source code while preserving the binary wire format; changing a field from `int32` to `string` breaks every existing serialized message. `buf breaking` makes that distinction explicit with rule categories for `FILE`, `PACKAGE`, `WIRE_JSON`, and `WIRE` compatibility.

```sh
buf breaking --against '.git#branch=main'
```

`--against` accepts a Git branch, a BSR module, a tarball, a zip file, a local directory, or a prebuilt Buf image. That matters in real repositories: the same command works on a laptop, in CI, and in release automation.

## Buf Schema Registry

<a name="the-buf-schema-registry"></a>

[Buf Schema Registry][bsr] is a Protobuf-aware registry. It stores modules, verifies they compile, renders documentation, resolves dependencies, hosts remote plugins, produces generated SDKs, and can enforce schema checks before a breaking change reaches consumers.

```sh
buf push
```

Pushing a module to the BSR gives your organization a source of truth for Protobuf APIs. Consumers can depend on the schema as a BSR module, install generated SDKs from their normal package manager, or use the BSR docs to inspect services, messages, fields, enums, references, and historical commits.

## Related projects

Buf is most useful when schemas drive more than code generation. [ConnectRPC][connectrpc] uses Protobuf schemas to build simple HTTP APIs that support Connect, gRPC, and gRPC-Web without separate service definitions. [Protobuf-ES][protobuf_es] gives JavaScript and TypeScript users a modern Protobuf runtime and generator. [Protovalidate][protovalidate] puts validation rules in the schema and runs them consistently across languages.

One contract should drive the whole workflow: compile, lint, compatibility checks, generated clients and servers, validation, API calls, package publishing, and governed changes.

## Installation

Homebrew installs the `buf`, [`protoc-gen-buf-breaking`][breaking_plugin], and [`protoc-gen-buf-lint`][lint_plugin] binaries, plus shell completion scripts for [Bash], [Fish], [PowerShell], and [zsh].

```sh
brew install bufbuild/buf/buf
```

Other supported installation methods include [npm], [Windows], [Docker], [binary downloads], [tarballs], [source builds], and [minisign verification][verifying]. See the [installation docs][install] for the full list.

## CLI stability

Buf CLI releases do not make breaking changes within a major version. Since `buf` reached v1.0, you can expect no breaking changes until v2.0. We have no plans to release v2.0.

This policy does not apply to commands behind the `buf beta` gate. Expect breaking changes for beta commands until they are promoted.

## Documentation

- [Buf CLI][cli]
- [CLI quickstart][cli_quickstart]
- [Modules and workspaces][modules_workspaces]
- [Code generation][generate]
- [Linting][lint]
- [Breaking-change detection][breaking]
- [Formatting][format]
- [Calling APIs with `buf curl`][curl]
- [Buf Schema Registry][bsr]
- [Generated SDKs][generated_sdks]
- [Remote plugins][remote_plugins]
- [Schema checks][schema_checks]
- [Migrating from `protoc`][migrate_from_protoc]

## Community

For help and discussion around Protobuf, best practices, and Buf, join us on [Slack][badges_slack].

For bugs, feature requests, and technical questions, open an issue in this repository or email [dev@buf.build][email_dev]. For general inquiries, email [info@buf.build][email_info].

[badges_slack]: https://buf.build/links/slack
[bash]: https://www.gnu.org/software/bash
[binary downloads]: https://buf.build/docs/cli/installation/#github
[breaking]: https://buf.build/docs/breaking/
[breaking_plugin]: https://buf.build/docs/breaking/
[brew]: https://brew.sh
[bsr]: https://buf.build/docs/bsr/
[cli]: https://buf.build/docs/cli/
[cli_quickstart]: https://buf.build/docs/cli/quickstart/
[connectrpc]: https://connectrpc.com
[curl]: https://buf.build/docs/curl/
[docker]: https://buf.build/docs/cli/installation/#docker
[email_dev]: mailto:dev@buf.build
[email_info]: mailto:info@buf.build
[fish]: https://fishshell.com
[format]: https://buf.build/docs/format/
[generate]: https://buf.build/docs/generate/
[generated_sdks]: https://buf.build/docs/bsr/generated-sdks/
[install]: https://buf.build/docs/cli/installation/
[lint]: https://buf.build/docs/lint/
[lint_plugin]: https://buf.build/docs/lint/
[migrate_from_protoc]: https://buf.build/docs/migration-guides/migrate-from-protoc/
[modules_workspaces]: https://buf.build/docs/cli/modules-workspaces/
[npm]: https://buf.build/docs/cli/installation/#npm
[powershell]: https://learn.microsoft.com/en-us/powershell/
[protobuf]: https://protobuf.dev
[protobuf_es]: https://github.com/bufbuild/protobuf-es
[protovalidate]: https://protovalidate.com
[remote_plugins]: https://buf.build/docs/bsr/remote-plugins/
[schema_checks]: https://buf.build/docs/bsr/checks/
[source builds]: https://buf.build/docs/cli/installation/#source
[tarballs]: https://buf.build/docs/cli/installation/#github
[verifying]: https://buf.build/docs/cli/installation/#github
[windows]: https://buf.build/docs/cli/installation/#windows
[zsh]: https://zsh.org
