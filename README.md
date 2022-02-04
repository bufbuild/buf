![The Buf logo](./.github/buf-logo.svg)

# Buf

[![CI](https://github.com/bufbuild/buf/workflows/ci/badge.svg)][badges.ci]
[![Release](https://img.shields.io/github/v/release/bufbuild/buf?include_prereleases)][badges.release]
[![Homebrew](https://img.shields.io/badge/homebrew-v1.0.0--rc12-blue)][badges.homebrew]
[![AUR](https://img.shields.io/aur/version/buf)][badges.aur]
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)][badges.slack]
[![Twitter](https://img.shields.io/twitter/follow/bufbuild?style=social)][badges.twitter]
[![Docker](https://img.shields.io/docker/pulls/bufbuild/buf)][badges.docker]
[![License](https://img.shields.io/github/license/bufbuild/buf?color=blue)][badges.license]

The [`buf`][buf] CLI is a tool for working with [Protocol Buffers][protobuf] APIs, offering a range of features not found in the standard `protoc` compiler, including:

- The ability to manage Protobuf assets, including [plugins] and [templates], on the [Buf Schema Registry][bsr] (BSR).
- A [linter][lint_usage] that enforces good API design choices and structure.
- A [breaking change detector][breaking_usage] that enforces compatibility at the source code or wire level.
- A [generator][generate_usage] that invokes your protoc plugins based on a configurable [template][templates].
  A [protoc replacement][protoc] that uses Buf's [high-performance Protobuf compiler][compiler].
- A configurable file [builder][build_usage] that produces [Images], our extension of Protobuf's native [FileDescriptorSets][filedescriptorset].

See more [features] below.

## Installation

### Homebrew

You can install `buf` using [Homebrew][brew] (macOS or Linux) through the [`bufbuild/bufbrew`][tap] tap.

```sh
brew tap bufbuild/bufbrew install buf
brew install buf
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

## Commands

The best way to learn about `buf` commands and flags is through the help interface:

```sh
buf --help
```

## Goal

[Buf]'s long-term goal is to push API development toward a schema-driven approach, where APIs are defined consistently and in a way that service owners and clients can depend on—in other words, to make API development feel much closer to using standard programming languages.

Defining APIs using an [IDL] provides a number of benefits over exposing JSON/REST services, and today, [Protobuf] is the most stable, widely adopted IDL in the industry. But using Protobuf has traditionally been much more difficult than using JSON as your data transfer format. We plan to change that by providing tools that make Protobuf reliable and easy to use for service owners and clients.

Your organization shouldn't have to reinvent the wheel to create, maintain, and consume Protobuf APIs efficiently and effectively. We'll handle your Protobuf management strategy for you, so you can focus on what matters.

We're working quickly to build a modern Protobuf ecosystem. The Buf CLI (this repo) is a core pillar of that. We built it to help you create consistent Protobuf APIs that preserve compatibility and comply with design best-practices.

The other pillar, the **Buf Schema Registry** (BSR), is the hub of our ecosystem. The BSR is a platform that serves as the source of truth for your organization's Protobuf files, enabling you to centrally maintain compatibility and manage dependencies, while enabling your clients to consume APIs reliably and efficiently. The BSR is currently in **beta**.

## Quick Links

However, we recommend you read the below introduction first!

- [Tour of existing functionality - takes about 20 minutes to complete][tour]
- [Installation guide][install]
- [Overview of our 40 lint rules][lint_rules]
- [Overview of our 54 breaking change rules][breaking_rules]
- [Simple code generation][generate_usage]
- [High-performance protoc replacement][protoc]
- [Protobuf Style Guide][style]
- [Migration from Protolock][protolock]
- [Migration from Prototool][prototool]

## The problems we aim to solve

Traditionally, adopting Protobuf presents a number of challenges across the API lifecycle. These are the problems we aim to solve.

### Creating consistent Protobuf APIs

- **API designs are often inconsistent**: Writing maintainable, consistent Protobuf APIs isn't as widely understood as writing maintainable JSON/REST-based APIs. With no standards enforcement, inconsistency can arise across an organization's Protobuf APIs, and design decisions can inadvertantly affect your API's future iterability.

### Maintaining compatible, accessible Protobuf APIs

- **Dependency management is usually an afterthought**: Protobuf files are vendored manually, with an error-prone copy-and-paste process from Github repositories. There is no centralized attempt to track and manage around cross-file dependencies.

- **Forwards and backwards compatibility is not enforced**: While forwards and backwards compatibility is a promise of Protobuf, actually maintaining backwards-compatible Protobuf APIs isn't widely practiced, and is hard to enforce.

### Consuming Protobuf APIs efficiently and reliably

- **Stub distribution is a difficult, unsolved process**: Organizations have to choose to either centralize the protoc workflow and distribute generated code, or require all service clients to run protoc independently. Because there is a steep learning curve to using protoc and associated plugins in a reliable manner, organizations end up choosing to struggle with distribution of Protobuf files and stubs. This creates substantial overhead, and often requires a dedicated team to manage the process. Even when using a build system like Bazel, exposing APIs to external customers remains problematic.

- **The tooling ecosystem is limited**: Lots of easy-to-use tooling exists today for JSON/REST APIs. Mock server generation, fuzz testing, documentation, and other daily API concerns are not widely standardized and easy to use for Protobuf APIs, requiring teams to regularly reinvent the wheel and build custom tooling to replicate the JSON ecosystem.

## Buf is building a modern Protobuf ecosystem

Our tools address many of the problems above, allowing you to redirect much of your time and energy from managing Protobuf files to implementing your core features and infrastructure.

### CLI features

The Buf CLI is designed to be simple to use while also providing functionality for the most advanced use cases. Features of the CLI include:

- **Automatic file discovery**: By default, Buf builds your `.proto` files by walking your file
  tree and building them per your [build configuration][configuration]. This means you no longer need to
  manually specify your `--proto_paths` and files every time you run the tool. However, Buf does
  allow manual file specification through command-line flags if you want no file discovery to
  occur, for example in Bazel setups.

- **Selectable configuration**: of the exact lint and breaking change configuration you want.
  While we recommend using the defaults, Buf allows you to easily understand and select the exact set
  of lint and breaking change rules your organization needs.

  Buf provides [40 available lint rules][rules] and [54 available breaking rules][breaking_rules] to cover most needs. We believe our breaking change detection truly
  covers every scenario for your APIs.

- **Selectable error output**: By default, Buf outputs `file:line:col:message` information
  for every lint error and every breaking change, with the file path carefully outputted to
  match the input location, including if absolute paths are used, and for breaking change detection,
  including if types move across files. JSON output that includes the end line and end column
  of the lint error is also available, and JUnit output is coming soon.

- **Editor integration**: The default error output is easily parseable by any editor, making the
  feedback loop for issues very short. Currently, we provide
  [Vim and Visual Studio Code integration][ide] for linting but will extend this
  in the future to include other editors such as Emacs and Intellij IDEs.

- **Check anything from anywhere**: Buf allows you to not only check a Protobuf schema stored
  locally as `.proto` files, but allows you to check many different [Inputs]:

  - Tar or zip archives containing `.proto` files, both local and remote.
  - Git repository branches or tags containing `.proto` files, both local and remote.
  - Pre-built [Images] or FileDescriptorSets from `protoc`, from both local and remote
    (http/https) locations.

- **Speed**: Buf's [internal Protobuf compiler][compiler] utilizes all available cores to compile
  your Protobuf schema, while still maintaining deterministic output. Additionally files are copied into
  memory before processing. As an unscientific example, Buf can compile all 2,311 `.proto` files in
  [`googleapis`][googleapis] in about **0.8s** on a four-core machine,
  as opposed to about 4.3s for `protoc` on the same machine. While both are very fast, this allows for
  instantaneous feedback, which is especially useful with editor integration. Buf's speed is
  directly proportional to the input size, so checking a single file only takes a few milliseconds.

### The Buf Schema Registry

The [Buf Schema Registry][bsr] (BSR) is a hosted SaaS platform that serves as your organization's source of truth for Protobuf APIs, built around the primitive of Protobuf [Modules]. We're introducing the concept of Protobuf Modules to enable the BSR to manage a group of Protobuf files together, similar to a Go Module.

The BSR offers these key features:

- **Centrally managed dependencies**: Resolve diamond dependency issues caused by haphazard versioning, even with external repository dependants.

- **Automatically enforce forwards and backwards compatibility**: Ensure API clients never break, without wasteful team-to-team communication or custom SLAs.

- **Generated libraries produced by a managed compiler**: Language-specific stub generation using Buf's high-performance, drop-in protoc replacement.

Over time, our goal is to make the BSR the only tool you need to manage your Protobuf workflow from end to end. For a quick overview of our many plans for the BSR, see the [roadmap].

## Where to go from here

To install Buf, proceed to [installation][install]. This includes links to an example repository for Travis CI and GitHub Actions integration.

Next, we recommend completing the [tour]. This tour provides you with a broad overview of most of the existing functionality of Buf. It takes about 10 minutes to complete.

After completing the tour, check out the remainder of the docs for your specific areas of interest. We've aimed to provide as much documentation as we can for the various components of Buf to give you a full understanding of Buf's surface area.

Finally, [follow the project on GitHub][repo] and [contact us][contact] if you'd like to get involved.

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
[breaking_rules]: https://docs.buf.build/breaking/rules
[breaking_usage]: https://docs.buf.build/breaking/usage
[brew]: https://brew.sh
[bsr]: https://docs.buf.build/bsr
[buf]: https://buf.build
[build_usage]: https://docs.buf.build/build/usage
[compiler]: https://docs.buf.build/build/internal-compiler
[configuration]: https://docs.buf.build/build/usage/#configuration
[contact]: https://docs.buf.build/contact
[deps]: https://docs.buf.build/bsr/overview#dependencies
[docker]: https://docs.buf.build/installation#use-the-docker-image
[filedescriptorset]: https://github.com/protocolbuffers/protobuf/blob/044c766fd4777713fef2d1a9a095e4308d770c68/src/google/protobuf/descriptor.proto#L57
[features]: #cli-features
[fish]: https://fishshell.com
[generate_usage]: https://docs.buf.build/generate/usage
[googleapis]: https://github.com/googleapis/googleapis
[ide]: https://docs.buf.build/editor-integration
[idl]: https://en.wikipedia.org/wiki/Interface_description_language
[images]: https://docs.buf.build/reference/images
[inputs]: https://docs.buf.build/reference/input
[install]: https://docs.buf.build/installation
[lint]: https://docs.buf.build/lint
[lint_rules]: https://docs.buf.build/lint/rules
[lint_usage]: https://docs.buf.build/lint/usage
[minisign]: https://github.com/jedisct1/minisign
[modules]: https://docs.buf.build/bsr/overview#module
[plugins]: https://docs.buf.build/bsr/remote-generation/concepts#plugin
[powershell]: https://docs.microsoft.com/en-us/powershell
[protobuf]: https://developers.google.com/protocol-buffers
[protoc]: https://docs.buf.build/generate/high-performance-protoc-replacement
[protolock]: https://docs.buf.build/how-to/migrate-from-protolock
[prototool]: https://docs.buf.build/how-to/migrate-from-prototool
[releases]: https://docs.buf.build/installation#github-releases
[repo]: ./
[roadmap]: https://docs.buf.build/roadmap
[source]: https://docs.buf.build/installation#from-source
[style]: https://docs.buf.build/best-practices/style-guide
[tap]: https://github.com/bufbuild/homebrew-buf
[tarball]: https://docs.buf.build/installation#tarball
[templates]: https://docs.buf.build/bsr/remote-generation/concepts#template
[tour]: https://docs.buf.build/tour/introduction
[verifying]: https://docs.buf.build/installation#verifying-a-release
[windows]: https://docs.buf.build/installation#windows-support
[zsh]: https://zsh.org