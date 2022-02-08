![The Buf logo](./.github/buf-logo.svg)

# Buf

[![License](https://img.shields.io/github/license/bufbuild/buf?color=blue)][badges.license]
[![Release](https://img.shields.io/github/v/release/bufbuild/buf?include_prereleases)][badges.release]
[![CI](https://github.com/bufbuild/buf/workflows/ci/badge.svg)][badges.ci]
[![Docker](https://img.shields.io/docker/pulls/bufbuild/buf)][badges.docker]
[![Homebrew](https://img.shields.io/badge/homebrew-v1.0.0--rc12-blue)][badges.homebrew]
[![AUR](https://img.shields.io/aur/version/buf)][badges.aur]
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)][badges.slack]
[![Twitter](https://img.shields.io/twitter/follow/bufbuild?style=social)][badges.twitter]

The [`buf`][buf] CLI is a tool for working with [Protocol Buffers][protobuf] APIs, offering a range of features not found in the standard `protoc` compiler, including:

- The ability to manage Protobuf assets, including [plugins] and [templates], on the [Buf Schema Registry][bsr] (BSR).
- A [linter][lint_usage] that enforces good API design choices and structure.
- A [breaking change detector][breaking_usage] that enforces compatibility at the source code or wire level.
- A [generator][generate_usage] that invokes your protoc plugins based on a configurable [template][templates].
  A [protoc replacement][protoc] that uses Buf's [high-performance Protobuf compiler][compiler].
- A configurable file [builder][build_usage] that produces [Images], our extension of Protobuf's native [FileDescriptorSets][filedescriptorset].

## Installation

### Homebrew

You can install `buf` using [Homebrew][brew] (macOS or Linux):

```sh
brew install bufbuild/buf/buf
```

This installs:

* The `buf`, [`protoc-gen-buf-breaking`][breaking], and [`protoc-gen-buf-lint`][lint] binaries
* Shell completion scripts for [Bash], [Fish], [Powershell], and [zsh]

### Other methods

For other installation methods, see our [official documentation][install], which covers:

* Installing `buf` on [Windows]
* Using `buf` as a [Docker image][docker]
* Installing as a [binary], from a [tarball], and from [source] through [GitHub Releases][releases]
* [Verifying] releases using a [minisign] public key

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
* [`buf registry`][bsr_usage] (for using the [BSR])

## Next steps

Once you've installed `buf`, we recommend completing the [Tour of Buf][tour], which provides a broad but hands-on overview of the core functionality of both the CLI and the [BSR]. The tour takes about 10 minutes to complete.

After completing the tour, check out the remainder of the [docs] for your specific areas of interest and our [roadmap] to see what we have in store for the future.

Finally, [follow the Buf CLI on GitHub][repo] and [contact us][contact] if you'd like to get involved.

[badges.aur]: https://aur.archlinux.org/packages/buf
[badges.ci]: https://github.com/bufbuild/buf/actions?workflow=ci
[badges.docker]: https://hub.docker.com/r/bufbuild/buf
[badges.homebrew]: https://github.com/bufbuild/homebrew-buf
[badges.license]: https://github.com/bufbuild/buf/blob/main/LICENSE
[badges.release]: https://github.com/bufbuild/buf/releases
[badges.slack]: https://join.slack.com/t/bufbuild/shared_invite/zt-f5k547ki-VDs_iC4TblNCu7ubhRD17w
[badges.twitter]: https://twitter.com/intent/follow?screen_name=bufbuild
[bash]: https://www.gnu.org/software/bash
[binary]: https://docs.buf.build/installation#binary
[breaking]: https://docs.buf.build/breaking
[breaking_usage]: https://docs.buf.build/breaking/usage
[brew]: https://brew.sh
[bsr]: https://docs.buf.build/bsr
[bsr_usage]: https://docs.buf.build/bsr/usage
[buf]: https://buf.build
[build_usage]: https://docs.buf.build/build/usage
[compiler]: https://docs.buf.build/build/internal-compiler
[contact]: https://docs.buf.build/contact
[docker]: https://docs.buf.build/installation#use-the-docker-image
[docs]: https://docs.buf.build
[filedescriptorset]: https://github.com/protocolbuffers/protobuf/blob/044c766fd4777713fef2d1a9a095e4308d770c68/src/google/protobuf/descriptor.proto#L57
[features]: #cli-features
[fish]: https://fishshell.com
[generate_usage]: https://docs.buf.build/generate/usage
[images]: https://docs.buf.build/reference/images
[install]: https://docs.buf.build/installation
[lint]: https://docs.buf.build/lint
[lint_usage]: https://docs.buf.build/lint/usage
[minisign]: https://github.com/jedisct1/minisign
[plugins]: https://docs.buf.build/bsr/remote-generation/concepts#plugin
[powershell]: https://docs.microsoft.com/en-us/powershell
[protobuf]: https://developers.google.com/protocol-buffers
[protoc]: https://docs.buf.build/generate/high-performance-protoc-replacement
[releases]: https://docs.buf.build/installation#github-releases
[repo]: ./
[roadmap]: https://docs.buf.build/roadmap
[source]: https://docs.buf.build/installation#from-source
[tarball]: https://docs.buf.build/installation#tarball
[templates]: https://docs.buf.build/bsr/remote-generation/concepts#template
[tour]: https://docs.buf.build/tour/introduction
[verifying]: https://docs.buf.build/installation#verifying-a-release
[windows]: https://docs.buf.build/installation#windows-support
[zsh]: https://zsh.org
