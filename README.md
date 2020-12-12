# Buf

[![License](https://img.shields.io/github/license/bufbuild/buf?color=blue)](https://github.com/bufbuild/buf/blob/master/LICENSE)
[![Release](https://img.shields.io/github/v/release/bufbuild/buf?include_prereleases)](https://github.com/bufbuild/buf/releases)
[![CI](https://github.com/bufbuild/buf/workflows/ci/badge.svg)](https://github.com/bufbuild/buf/actions?workflow=ci)
[![Coverage](https://img.shields.io/codecov/c/github/bufbuild/buf/master)](https://codecov.io/gh/bufbuild/buf)
[![Docker](https://img.shields.io/docker/pulls/bufbuild/buf)](https://hub.docker.com/r/bufbuild/buf)
[![Homebrew](https://img.shields.io/badge/homebrew-v0.33.0-blue)](https://github.com/bufbuild/homebrew-buf)
[![AUR](https://img.shields.io/aur/version/buf)](https://aur.archlinux.org/packages/buf)
[![Google Group](https://img.shields.io/badge/google%20group-bufbuild--announce-blue)](https://groups.google.com/forum/#!forum/bufbuild-announce)
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)](https://join.slack.com/t/bufbuild/shared_invite/zt-f5k547ki-VDs_iC4TblNCu7ubhRD17w)

**All documentation is hosted at [https://buf.build](https://buf.build). Please head over there for
more details.**

## Goal
---

Buf’s long-term goal is to enable schema-driven development: a future where APIs are defined consistently, in a way that service owners and clients can depend on.

Defining APIs using an [IDL](https://en.wikipedia.org/wiki/Interface_description_language) provides a number of benefits over simply exposing JSON/REST services, and today, [Protobuf](https://developers.google.com/protocol-buffers) is the most stable, widely-adopted IDL in the industry.

However, as it stands, using Protobuf is much more difficult than using JSON as your data transfer format.

Enter Buf: We’re building tooling to make Protobuf reliable and easy to use for service owners and clients, while keeping it the obvious choice on the technical merits.

Your organization should not have to reinvent the wheel to create, maintain, and consume Protobuf APIs efficiently and effectively. We'll handle your Protobuf management strategy for you, so you can focus on what matters.

We’re working quickly to build a modern Protobuf ecosystem. Our first tool is the **Buf CLI**, built to help you create consistent Protobuf APIs that preserve compatibility and comply with design best-practices. The tool is currently available on an open-source basis.

Our second tool, the **Buf Schema Registry (“BSR”)**, will be the hub of our ecosystem. The BSR is a platform that serves as the source of truth for your organization's Protobuf files, enabling you to centrally maintain compatibility and manage dependencies, while enabling your clients to consume APIs reliably and efficiently. The BSR will be available for a limited, free private beta shortly.

## Quick Links

However, we recommend you read the below introduction first!

- [Tour of existing functionality- takes about 10 minutes to complete](https://buf.build/docs/tour-1)
- [Overview of our 40 lint checkers](https://buf.build/docs/lint-checkers)
- [Overview of our 54 breaking change checkers](https://buf.build/docs/breaking-checkers)
- [Simple code generation](https://buf.build/docs/generate-usage)
- [High-performance protoc replacement](https://buf.build/docs/generate-protoc)
- [Protobuf Style Guide](https://buf.build/docs/style-guide)
- [Migration from Protolock](https://buf.build/docs/migration-protolock)
- [Migration from Prototool](https://buf.build/docs/migration-prototool)

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

Our tools will address many of the problems above, ultimately allowing you to redirect much of your time and energy from managing Protobuf files to implementing your core features and infrastructure.

### The Buf CLI

The Buf CLI incorporates the following components to help you create consistent Protobuf APIs:

- A [linter](https://buf.build/docs/lint-usage) that enforces good API design choices and structure.
- A [breaking change detector](https://buf.build/docs/breaking-usage) that enforces compatibility at the source code or wire level.
- A [generator](https://buf.build/docs/generate-usage) that invokes your protoc plugins based on a configurable
  template.
  A [protoc replacement](https://buf.build/docs/generate-protoc) that uses Buf's newly-developed [high performance
  Protobuf compiler](https://buf.build/docs/build-compiler.md).
- A configurable file [builder](https://buf.build/docs/build-overview) that produces
  [Images](https://buf.build/docs/build-images), our extension of
  [FileDescriptorSets](https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/descriptor.proto).

The Buf CLI is designed to be extremely simple to use, while providing functionality for advanced use cases. Features of the CLI include:

- **Automatic file discovery**: By default, Buf will build your `.proto` files by walking your file
  tree and building them per your [build configuration](https://buf.build/docs/build-configuration). This means you no longer need to
  manually specify your `--proto_paths` and files every time you run the tool. However, Buf does
  allow manual file specification through command-line flags if you want no file discovery to
  occur, for example in Bazel setups.

- **Selectable configuration**: of the exact lint and breaking change configuration you want.
  While we recommend using the defaults, Buf allows you to easily understand and select the exact set
  of lint and breaking change checkers your organization needs.

  Buf provides [40 available lint checkers](https://buf.build/docs/lint-checkers) and [54 available breaking
  checkers](https://buf.build/docs/breaking-checkers) to cover most needs. We believe our breaking change detection truly
  covers every scenario for your APIs.

- **Selectable error output**: By default, Buf outputs `file:line:col:message` information
  for every lint error and every breaking change, with the file path carefully outputted to
  match the input location, including if absolute paths are used, and for breaking change detection,
  including if types move across files. JSON output that includes the end line and end column
  of the lint error is also available, and JUnit output is coming soon.

- **Editor integration**: The default error output is easily parseable by any editor, making the
  feedback loop for issues very short. Currently, we only provide [Vim integration](https://buf.build/docs/editor-integration)
  for linting but will extend this in the future to include other editors such as Emacs, VS Code,
  and Intellij IDEs.

- **Check anything from anywhere**: Buf allows you to not only check a Protobuf schema stored
  locally as `.proto` files, but allows you to check many different [Inputs](https://buf.build/docs/inputs):

  - Tar or zip archives containing `.proto` files, both local and remote.
  - Git repository branches or tags containing `.proto` files, both local and remote.
  - Pre-built [Images](https://buf.build/docs/build-images) or FileDescriptorSets from `protoc`, from both local and remote
    (http/https) locations.

- **Speed**: Buf's [internal Protobuf compiler](https://buf.build/docs/build-compiler) utilizes all available cores to compile
  your Protobuf schema, while still maintaining deterministic output. Additionally files are copied into
  memory before processing. As an unscientific example, Buf can compile all 2,311 `.proto` files in
  [googleapis](https://github.com/googleapis/googleapis) in about **0.8s** on a four-core machine,
  as opposed to about 4.3s for `protoc` on the same machine. While both are very fast, this allows for
  instantaneous feedback, which is especially useful with editor integration. Buf's speed is
  directly proportional to the input size, so checking a single file only takes a few milliseconds.

### The Buf Schema Registry

The Buf Schema Registry will be a powerful hosted SaaS platform to serve as your organization’s source of truth for your Protobuf APIs, built around the primitive of Protobuf Modules. We’re introducing the concept of Protobuf Modules to enable the BSR to manage a group of Protobuf files together, similar to a Go Module.

Initially, the BSR will offer the following key features:

- **Centrally managed dependencies**: Resolve diamond dependency issues caused by haphazard versioning, even with external repository dependants.

- **Automatically enforce forwards and backwards compatibility**: Ensure API clients never break, without wasteful team-to-team communication or custom SLAs.

- **Generated libraries produced by a managed compiler**: Language-specific stub generation using Buf’s high-performance, drop-in protoc replacement.

Over time, our goal is to make the BSR the only tool you need to manage your Protobuf workflow from end to end. To that end, there's a lot we are planning with the Buf Schema Registry. For a quick overview, see our [roadmap](https://buf.build/docs/roadmap).

## Where to go from here

To install Buf, proceed to [installation](https://buf.build/docs/installation.mdx). This includes links to an example
repository for Travis CI and GitHub Actions integration.

Next, we recommend completing the [tour](https://buf.build/docs/tour-1). This tour should only take about 10 minutes, and
will give you an overview of most of the existing functionality of Buf.

After completing the tour, check out the remainder of the docs for your specific areas of interest.
We've aimed to provide as much documentation as we can for the various components of Buf to give
you a full understanding of Buf's surface area.

Finally, [follow the project on GitHub](https://github.com/bufbuild/buf),
and [contact us](https://buf.build/docs/contact) if you'd like to get involved.
