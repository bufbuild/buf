# `protosourcepath`

`protosourcepath` is a simple package that takes a [Protobuf source path](source-path) and
returns a list of associated source paths. A [Protobuf source path](source-path) is a
[`SourceCodeInfo.Location`](location) path, which is an array of integers of a variable length
that identifies a Protobuf definition. Each element of the source path represents a field
number from a descriptor proto or an index, and they form a *path* from the [`FileDescriptorProto`](file-descriptor)
to the definition itself. An index is needed for any `repeated` types on the descriptor proto,
such as messages, enums, services, and extensions on [`FileDescriptorProto`](file-descriptor).

For example, let's say we have the following source path:

```
[4, 0, 2, 0, 1]
```

This path represents `.message_type(0).field(0).name`, which is the name of field at index
0 for the message at index 0. We can break down the source path by following the numbers,
starting from `FileDescriptorProto`:

- `4` is the field number of [`message_type` on `FileDescriptorProto`](message-types), which
  is a repeated field representing message declarations in the file. `0` is the index of the
  message for this path.
- `2` is the field number of [`field` on `DescriptorProto`](field), which is a repeated field
  representing field declarations of a message. `0` is the index of the field for this path.
- `1` is the field number of [`name` on `FieldDescriptorProto`](field-name), which is a field
  representing the name of field declarations.

All source paths start from `FileDescriptorProto` and end at the Protobuf definition they
are pointing to. More details on source paths can be found in [descriptor.proto](source-path).

Source paths are useful because they can be used to retrieve the [`SourceCodeInfo`](source-code-info)
of Protobuf definitions from their `FileDescriptorProto`'s. [`SourceCodeInfo`](source-code-info) provides
location-based metadata of Protobuf definitions, including comments attached to the Protobuf
definition, which we use to look for comment ignores for lint checks. And in the case of comment
ignores, a list of associated paths would allow us to check "associated locations" as potential
sites for defining a comment ignore (e.g. for a lint rule that checks fields, we want users to
be able to define a comment ignore on a specific field, but also the message a field belongs
to, since they could use that to ignore this rule for all fields for that message instead of
defining individual comment ignores for each field).

## Associated paths

Associated paths are source paths that we consider "associated" with the given source path,
which we define as parent paths or child paths.

**Parent paths** are valid source paths to "complete" Protobuf declarations that are equal or "closer"
to the `FileDescriptorProto` than the given source path. A "complete" Protobuf declaration starts
from either the keyword (e.g. `message` or `enum`) or label/name (e.g. fields may or may not
have a label, and enum values would start at the name) and terminates at the opening brace
or semicolon respectively. For example, the path we looked at earlier, `[4, 0, 2, 0, 1]`, one of
the parent paths would be `[4, 0, 2, 0]`, which is the complete field declaration of `message_type(0).field(0)`
(which starts at the label of the field and terminates at the semicolon). Parent paths are
always "complete" Protobuf declarations. The following is a breakdown of what we consider as parent paths:

- For each top-level declaration (e.g. messages, enums, services, extensions, options), we consider the
  complete declaration as a parent path. This means that a given path can be one of its parent paths,
  (e.g. if the given path is `[4, 0, 2, 0]`, then `[4, 0, 2, 0]` would be considered one of
  the parent paths).
- For each nested declaration (e.g. field; enum values; options; nested messages, enums, and
  extensions), we consider the complete declaration of the Protobuf definition as a parent path,
  and the paths of the complete declarations of all parent types as parent paths (e.g. if the
  given path is `[4, 0, 3, 2]`, which is a path to the complete declaration of a nested message,
  then the path itself, `[4, 0, 3, 2]` is considered a parent path and `[4, 0]`, the path of the complete
  declaration of the parent message is considered a parent path).
- For each specific attribute (e.g. name, label, field number, enum value number, etc.), we consider the complete
  declaration of the Protobuf definition as a parent path (as illustrated in the initial example,
  `[4, 0, 2, 0, 1]`, given the path to a field name, the complete declaration of the field
  and the complete declaration of the message would be considered parent paths).

**Child paths** are valid source paths that are *not* complete Protobuf declarations that are
equal or "closer" to the `FileDescriptorProto` than the given source path. Going back to our
example path, `[4, 0, 2, 0, 1]`, a path to a field name, it would be considered its own child
path, since it is not a complete Protobuf declaration, and other associated child paths would
include the field number, label, type, and type name. In addition, the associated child paths
of its parent type would also be considered associated child paths, in this case, the path
to the message name.

Details examples for associated paths can be found through the tests.

## API

There is a single function, `GetAssociatedSourcePaths`, that takes a `protoreflect.SourcePath`
and returns a list of associated paths.

```go
func GetAssociatedSourcePaths(
	sourcePath protoreflect.SourcePath,
) ([]protoreflect.SourcePath, error)
```

We expect there always to be at least one associated path, the path itself.

## Future

We are currently returning all associated source paths, but we have the option to exclude
child paths/Protobuf declarations that are not complete, since our use-case is primarily to
get comments, which should only be attached to complete Protobuf declarations. However, it
is currently inexpensive to check all associated source paths, so we have not exposed that
functionality on the exported function.

[location]: https://github.com/protocolbuffers/protobuf/blob/44e9777103aa864859c04159a7abc376c5a98210/src/google/protobuf/descriptor.proto#L1174
[source-path]: https://github.com/protocolbuffers/protobuf/blob/44e9777103aa864859c04159a7abc376c5a98210/src/google/protobuf/descriptor.proto#L1175-L1197
[file-descriptor]: https://github.com/protocolbuffers/protobuf/blob/44e9777103aa864859c04159a7abc376c5a98210/src/google/protobuf/descriptor.proto#L97
[message-types]: https://github.com/protocolbuffers/protobuf/blob/44e9777103aa864859c04159a7abc376c5a98210/src/google/protobuf/descriptor.proto#L110
[field]: https://github.com/protocolbuffers/protobuf/blob/44e9777103aa864859c04159a7abc376c5a98210/src/google/protobuf/descriptor.proto#L137
[field-name]: https://github.com/protocolbuffers/protobuf/blob/44e9777103aa864859c04159a7abc376c5a98210/src/google/protobuf/descriptor.proto#L268
[source-code-info]: https://github.com/protocolbuffers/protobuf/blob/44e9777103aa864859c04159a7abc376c5a98210/src/google/protobuf/descriptor.proto#L1129
[dfa]: https://en.wikipedia.org/wiki/Deterministic_finite_automaton
