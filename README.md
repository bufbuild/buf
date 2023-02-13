![The Buf logo](./.github/buf-logo.svg)

# Buf

[![License](https://img.shields.io/github/license/bufbuild/buf?color=blue)][badges_license]
[![Release](https://img.shields.io/github/v/release/bufbuild/buf?include_prereleases)][badges_release]
[![CI](https://github.com/bufbuild/buf/workflows/ci/badge.svg)][badges_ci]
[![Docker](https://img.shields.io/docker/pulls/bufbuild/buf)][badges_docker]
[![Homebrew](https://img.shields.io/badge/homebrew-v1.14.0-blue)][badges_homebrew]
[![AUR](https://img.shields.io/aur/version/buf)][badges_aur]
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)][badges_slack]

The [`buf`][buf] CLI is a tool for working with [Protocol Buffers][protobuf].

<a id="features"></a>

- The ability to manage Protobuf assets on the [Buf Schema Registry][bsr] (BSR).
- A [linter][lint_usage] that enforces good API design choices and structure.
- A [breaking change detector][breaking_usage] that enforces compatibility at the source code or wire level.
- A [generator][generate_usage] that invokes your plugins based on configurable [templates][templates].
- A [formatter][format_usage] that formats your Protobuf files in accordance with industry standards.
- Integration with the [Buf Schema Registry][bsr], including full dependency management.

## Installation

### Homebrew

You can install `buf` using [Homebrew][brew] (macOS or Linux):

```sh
brew install bufbuild/buf/buf
```

This installs:

- The `buf`, [`protoc-gen-buf-breaking`][breaking], and [`protoc-gen-buf-lint`][lint] binaries
- Shell completion scripts for [Bash], [Fish], [Powershell], and [zsh]

### Other methods

For other installation methods, see our [official documentation][install], which covers:

- Installing `buf` via [npm]
- Installing `buf` on [Windows]
- Using `buf` as a [Docker image][docker]
- Installing as a [binary], from a [tarball], and from [source] through [GitHub Releases][releases]
- [Verifying] releases using a [minisign] public key


## Usage

Buf's help interface provides summaries for commands and flags:

```sh
buf --help
```

For more comprehensive usage information, consult Buf's [documentation][docs], especially these guides:

* [`buf breaking`][breaking_usage]
* [`buf build`][build_usage]
* [`buf generate`][generate_usage]
* [`buf lint`][lint_usage]
* [`buf format`][format_usage]
* [`buf registry`][bsr_usage] (for using the [BSR])

## CLI breaking change policy

We will never make breaking changes within a given major version of the CLI. Once `buf` reaches v1.0, you can expect no breaking changes until v2.0. But as we have no plans to ever release a v2.0, we will likely never break the `buf` CLI.

> This breaking change policy does _not_ apply to commands behind the `buf beta` gate, and you should expect breaking changes to commands like `buf beta registry`. The policy does go into effect, however, when those commands or flags are elevated out of beta.

## Our goals for Protobuf

[Buf]'s goal is to replace the current paradigm of API development, centered around REST/JSON, with a **schema-driven** paradigm. Defining APIs using an [IDL] provides numerous benefits over REST/JSON, and [Protobuf] is by far the most stable and widely adopted IDL in the industry. We've chosen to build on this widely trusted foundation rather than creating a new IDL from scratch.

But despite its technical merits, actually _using_ Protobuf has long been more challenging than it needs to be. The Buf CLI and the [BSR](#the-buf-schema-registry) are the cornerstones of our effort to change that for good and to make Protobuf reliable and easy to use for service owners and clients alike—in other words, to create a **modern Protobuf ecosystem**.

While we intend to incrementally improve on the `buf` CLI and the [BSR](#the-buf-schema-registry), we're confident that the basic groundwork for such an ecosystem is _already_ in place.

## The Buf Schema Registry

The [Buf Schema Registry][bsr] (BSR) is a SaaS platform for managing your Protobuf APIs. It provides a centralized registry and a single source of truth for all of your Protobuf assets, including not just your `.proto` files but also [remote plugins][bsr_plugins]. Although the BSR provides an intuitive browser UI, `buf` enables you to perform most BSR-related tasks from the command line, such as [pushing] Protobuf sources to the registry and managing [users] and [repositories]. The BSR is currently in [**beta**][bsr_post].

> The BSR is not required to use `buf`. We've made the core [features] of the `buf` CLI available to _all_ Protobuf users.

## More advanced CLI features

While `buf`'s [core features][features] should cover most use cases, we've included some more advanced features to cover edge cases:

* **Automatic file discovery**. Buf walks your file tree and builds your `.proto` files in accordance with your supplied [build configuration][build_config], which means that you no longer need to manually specify `--proto_paths`. You can still, however, specify `.proto` files manually through CLI flags in cases where file discovery needs to be disabled.
* **Fine-grained rule configuration** for [linting][lint_rules] and [breaking changes][breaking_rules]. While we do have recommended defaults, you can always select the exact set of rules that your use case requires, with [40 lint rules][lint_rules] and [53 breaking change rules][breaking_rules] available.
* **Configurable error formats** for CLI output. `buf` outputs information in `file:line:column:message` form by default for each lint error and breaking change it encounters, but you can also select JSON and, in the near future, JUnit output.
* **Editor integration** driven by `buf`'s granular error output. We currently provide linting integrations for both [Vim and Visual Studio Code][ide] but we plan to support other editors, such as Emacs and [JetBrains IDEs][jetbrains] like IntelliJ and GoLand, in the future.
* **Universal Input targeting**. Buf enables you to perform actions like linting and breaking change detection not just against local `.proto` files but also against a broad range of other [Inputs], such as tarballs and ZIP files, remote Git repositories, and pre-built [image][images] files.
* **Speed**. Buf's internal Protobuf compiler compiles your Protobuf sources using all available cores without compromising deterministic output, which is considerably faster than `protoc`. This allows for near-instantaneous feedback, which is of special importance for features like [editor integration][ide].

## Next steps

Once you've installed `buf`, we recommend completing the [Tour of Buf][tour], which provides a broad but hands-on overview of the core functionality of both the CLI and the [BSR]. The tour takes about 10 minutes to complete.

After completing the tour, check out the remainder of the [docs] for your specific areas of interest.

## Community

For help and discussion around Protobuf, best practices, and more, join us on [Slack][badges_slack].

For updates on the Buf CLI, [follow this repo on GitHub][repo].

For feature requests, bugs, or technical questions, email us at [dev@buf.build][email_dev]. For general inquiries or inclusion in our upcoming feature betas, email us at [info@buf.build][email_info].

[badges_aur]: https://aur.archlinux.org/packages/buf
[badges_ci]: https://github.com/bufbuild/buf/actions?workflow=ci
[badges_docker]: https://hub.docker.com/r/bufbuild/buf
[badges_homebrew]: https://github.com/bufbuild/homebrew-buf
[badges_license]: https://github.com/bufbuild/buf/blob/main/LICENSE
[badges_release]: https://github.com/bufbuild/buf/releases
[badges_slack]: https://buf.build/links/slack
[bash]: https://www.gnu.org/software/bash
[binary]: https://docs.buf.build/installation#binary
[breaking]: https://docs.buf.build/breaking
[breaking_rules]: https://docs.buf.build/breaking/rules
[breaking_usage]: https://docs.buf.build/breaking/usage
[brew]: https://brew.sh
[bsr]: https://docs.buf.build/bsr
[bsr_plugins]: https://buf.build/plugins
[bsr_post]: https://buf.build/blog/announcing-bsr
[bsr_usage]: https://docs.buf.build/bsr/usage
[buf]: https://buf.build
[build_config]: https://docs.buf.build/build/usage/#configuration
[build_usage]: https://docs.buf.build/build/usage
[compiler]: https://docs.buf.build/build/internal-compiler
[contact]: https://docs.buf.build/contact
[docker]: https://docs.buf.build/installation#use-the-docker-image
[docs]: https://docs.buf.build
[email_dev]: mailto:dev@buf.build
[email_info]: mailto:info@buf.build
[filedescriptorset]: https://github.com/protocolbuffers/protobuf/blob/044c766fd4777713fef2d1a9a095e4308d770c68/src/google/protobuf/descriptor.proto#L57
[features]: #features
[fish]: https://fishshell.com
[format_usage]: https://docs.buf.build/format/usage
[generate_usage]: https://docs.buf.build/generate/usage
[ide]: https://docs.buf.build/editor-integration
[idl]: https://en.wikipedia.org/wiki/Interface_description_language
[images]: https://docs.buf.build/reference/images
[inputs]: https://docs.buf.build/reference/inputs
[install]: https://docs.buf.build/installation
[jetbrains]: https://docs.buf.build/editor-integration#jetbrains-ides
[lint]: https://docs.buf.build/lint
[lint_rules]: https://docs.buf.build/lint/rules
[lint_usage]: https://docs.buf.build/lint/usage
[npm]: https://docs.buf.build/installation#npm
[minisign]: https://github.com/jedisct1/minisign
[powershell]: https://docs.microsoft.com/en-us/powershell
[protobuf]: https://developers.google.com/protocol-buffers
[pushing]: https://docs.buf.build/bsr/usage#push-a-module
[releases]: https://docs.buf.build/installation#github-releases
[repo]: https://github.com/bufbuild/buf/
[repositories]: https://docs.buf.build/bsr/overview#module
[roadmap]: https://docs.buf.build/roadmap
[source]: https://docs.buf.build/installation#from-source
[tarball]: https://docs.buf.build/installation#tarball
[templates]: https://docs.buf.build/configuration/v1/buf-gen-yaml
[tour]: https://docs.buf.build/tour/introduction
[users]: https://docs.buf.build/bsr/user-management#organization-roles
[verifying]: https://docs.buf.build/installation#verifying-a-release
[windows]: https://docs.buf.build/installation#windows-support
[zsh]: https://zsh.org
