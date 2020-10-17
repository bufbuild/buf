# Buf

[![License](https://img.shields.io/github/license/bufbuild/buf?color=blue)](https://github.com/bufbuild/buf/blob/master/LICENSE)
[![Release](https://img.shields.io/github/v/release/bufbuild/buf?include_prereleases)](https://github.com/bufbuild/buf/releases)
[![CI](https://github.com/bufbuild/buf/workflows/ci/badge.svg)](https://github.com/bufbuild/buf/actions?workflow=ci)
[![Coverage](https://img.shields.io/codecov/c/github/bufbuild/buf/master)](https://codecov.io/gh/bufbuild/buf)
[![Docker](https://img.shields.io/docker/pulls/bufbuild/buf)](https://hub.docker.com/r/bufbuild/buf)
[![Homebrew](https://img.shields.io/badge/homebrew-v0.27.1-blue)](https://github.com/bufbuild/homebrew-buf)
[![AUR](https://img.shields.io/aur/version/buf)](https://aur.archlinux.org/packages/buf)
[![Google Group](https://img.shields.io/badge/google%20group-bufbuild--announce-blue)](https://groups.google.com/forum/#!forum/bufbuild-announce)
[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)](https://join.slack.com/t/bufbuild/shared_invite/zt-f5k547ki-VDs_iC4TblNCu7ubhRD17w)

**All documentation is hosted at [https://buf.build](https://buf.build). Please head over there for
more details.**

## Goal

Buf's goal is for Protobuf to not only be a good choice on the technical merits,
but to be so easy to use that the decision is trivial. Your organization
should not have to reinvent the wheel to use Protobuf efficiently and effectively. Stop
worrying about your Protobuf management strategy getting out of control. We'll
handle that for you, so you can worry about what matters.


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

## Overview

Using an [IDL](https://en.wikipedia.org/wiki/Interface_description_language) such as
[Protocol Buffers](https://developers.google.com/protocol-buffers) ("Protobuf")
provides numerous benefits over JSON:

- Generated stubs for each language you use.
- Forwards and backwards compatibility for your data types.
- Payload sizes are up to 10 times smaller.
- Serialization speed is up to 100 times faster.
- Structured RPCs for your APIs instead of documented HTTP endpoints.

Protobuf is the most stable, widely-adopted IDL in the software industry today. While there are
many pros and cons to Protobuf versus other IDLs such as Thrift, FlatBuffers, Avro, and Cap'n Proto,
Protobuf provides most companies the most stable platform to build on, along with the largest
ecosystem of languages and libraries available.

If you've found us today, we'll assume you're already relatively convinced of these statements.
We'll add a reference document for those new to Protobuf in the future.

If Protobuf is so great, the question is: why do so many companies still choose JSON as their
data format in 2020? Usually, the answer comes down to difficulty in adoption:

- **API Structure**: Writing maintainable, consistent Protobuf APIs isn't as widely
  understood as writing maintainable JSON/REST-based APIs, which makes sense - Protobuf
  is not as widely adopted. With no standards enforcement, inconsistency can arise across
  an organization's Protobuf APIs, and design decisions can be made that can affect your
  API's future iterability.
- **Backwards compatibility**: While forwards and backwards compatibility is a promise
  of Protobuf, actually maintaining backwards-compatible Protobuf APIs isn't widely
  practiced, and is hard to enforce.
- **Stub distribution**: Maintaining consistent stub generation is a difficult proposition.
  There is a steep learning curve to using `protoc` and associated plugins in a maintainable manner.
  Organizations end up struggling with distribution of Protobuf files and stubs, even if they use a
  build system such as Bazel - exposing APIs to external customers remains problematic.
  It ends up being more trouble than it's worth to expose APIs via Protobuf than via JSON/REST.
- **Tooling**: Lots of tooling for JSON/REST APIs exists today and is easy to use.
  Mock server generation, fuzz testing, documentation, and other daily API concerns
  are not widely standardized and easy to use for Protobuf APIs.

Done right, adopting Protobuf to represent
your structured data and APIs can quite literally produce one of the largest efficiency gains your
engineering organization can have. Much of the software we write today can be generated, and many
daily software development tasks we perform can be automated away.

In time, Buf aims to solve all this and more. However, there is a long way between that
world and the one we have now.

## Buf CLI

Phase 1 is to solve the API Structure and Backwards Compatibility problems: let's
help you maintain consistent Protobuf APIs that maintain compatibility.

This is done via the Buf CLI tool.

Buf currently contains:

- A [linter](https://buf.build/docs/lint-usage) that enforces good API design choices and structure.
- A [breaking change detector](https://buf.build/docs/breaking-usage) that enforces compatibility at the source code or wire level.
- A [generator](https://buf.build/docs/generate-usage) that invokes your protoc plugins based on a configurable
  template.
  A [protoc replacement](https://buf.build/docs/generate-protoc) that uses Buf's newly-developed [high performance
  Protobuf compiler](https://buf.build/docs/build-compiler.md).
- A configurable file [builder](https://buf.build/docs/build-overview) that produces
  [Images](https://buf.build/docs/build-images), our extension of
  [FileDescriptorSets](https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/descriptor.proto).

Buf is designed to be extremely simple to use, while providing functionality for advanced use cases.
Features of Buf's include:

- **Automatic file discovery**. By default, Buf will build your `.proto` files by walking your file
  tree and building them per your [build configuration](https://buf.build/docs/build-configuration). This means you no longer need to
  manually specify your `--proto_paths` and files every time you run the tool. However, Buf does
  allow manual file specification through command-line flags if you want no file discovery to
  occur, for example in Bazel setups.

- **Selectable configuration** of the exact lint and breaking change configuration you want.
  While we recommend using the defaults, Buf allows you to easily understand and select the exact set
  of lint and breaking change checkers your organization needs.

  Buf provides [40 available lint checkers](https://buf.build/docs/lint-checkers) and [54 available breaking
  checkers](https://buf.build/docs/breaking-checkers) to cover most needs. We believe our breaking change detection truly
  covers every scenario for your APIs.

- **Selectable error output**. By default, Buf outputs `file:line:col:message` information
  for every lint error and every breaking change, with the file path carefully outputted to
  match the input location, including if absolute paths are used, and for breaking change detection,
  including if types move across files. JSON output that includes the end line and end column
  of the lint error is also available, and JUnit output is coming soon.

- **Editor integration**. The default error output is easily parseable by any editor, making the
  feedback loop for issues very short. Currently, we only provide [Vim integration](https://buf.build/docs/editor-integration)
  for linting but will extend this in the future to include other editors such as Emacs, VS Code,
  and Intellij IDEs.

- **Check anything from anywhere**. Buf allows you to not only check a Protobuf schema stored
  locally as `.proto` files, but allows you to check many different [Inputs](https://buf.build/docs/inputs):

  - Tar or zip archives containing `.proto` files, both local and remote.
  - Git repository branches or tags containing `.proto` files, both local and remote.
  - Pre-built [Images](https://buf.build/docs/build-images) or FileDescriptorSets from `protoc`, from both local and remote
    (http/https) locations.

- **Speed**. Buf's [internal Protobuf compiler](https://buf.build/docs/build-compiler) utilizes all available cores to compile
  your Protobuf schema, while still maintaining deterministic output. Additionally files are copied into
  memory before processing. As an unscientific example, Buf can compile all 2,311 `.proto` files in
  [googleapis](https://github.com/googleapis/googleapis) in about **0.8s** on a four-core machine,
  as opposed to about 4.3s for `protoc` on the same machine. While both are very fast, this allows for
  instantaneous feedback, which is especially useful with editor integration. Buf's speed is
  directly proportional to the input size, so checking a single file only takes a few milliseconds.

## Buf Schema Registry

The Buf CLI is just the first component of Buf's modern Protobuf ecosystem. We're currently
in private beta of the **Buf Schema Registry**, a hosted and on-prem service that managed
the new Protobuf Modules paradigm. The Buf Schema Registry allows you to push and consume
modules, enabling entirely new Protobuf workflows.

There's a lot we are planning with the Buf Schema Registry. For a quick overview, see our
[roadmap](https://buf.build/docs/roadmap).

## Where to go from here

To install Buf, proceed to [installation](https://buf.build/docs/installation). This includes links to an example
repository for Travis CI and GitHub Actions integration.

Next, we recommend completing the [tour](https://buf.build/docs/tour-1). This tour should only take about 10 minutes, and
will give you an overview of most of the existing functionality of Buf.

After completing the tour, check out the remainder of the docs for your specific areas of interest.
We've aimed to provide as much documentation as we can for the various components of Buf to give
you a full understanding of Buf's surface area.

Finally, [follow the project on GitHub](https://github.com/bufbuild/buf),
and [contact us](https://buf.build/docs/contact) if you'd like to get involved.
