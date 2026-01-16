I want a new command, `buf source edit`, that allows editing of source files using the same logic as `buf format` (however, I want a new command).

I want it to have the same first positional argument as `buf format`, and the following flags:

```
Flags:
      --config string          The buf.yaml file or data to use for configuration
  -d, --diff                   Display diffs instead of rewriting files
      --deprecate strings      The prefix of the types (package, message, enum, extension, service, method) to deprecate.
                               When specified, all types under the prefix will have the `deprecated` option added to them.
      --disable-symlinks       Do not follow symlinks when reading sources or configuration from the local filesystem
                               By default, symlinks are followed in this CLI, but never followed on the Buf Schema Registry
      --error-format string    The format for build errors printed to stderr. Must be one of [text,json,msvs,junit,github-actions,gitlab-code-quality] (default "text")
      --exclude-path strings   Exclude specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
  -h, --help                   help for format
      --path strings           Limit to specific files or directories, e.g. "proto/a/a.proto", "proto/a"
                               If specified multiple times, the union is taken
```

As opposed to `buf format`, there is no `--exit-code`, `-o`, or `-w` flags, and I've added the `--deprecate` flag. The behavior of `-w` is the default for `buf source edit`.

Internally, this should just use the `bufformat` package, and largely copy the `buf format` command, however additional functional options should be added to the `bufformat` package to allow for things like deprecation. You'll likely want to have something like `func WithDeprecate(fqnPrefix string) FormatOption` that will result in `deprecated` options being added to any type that has the given fully-qualified name prefix.

For example, if I had services, messages, and enums under `package foo.bar.baz` and I gave `foo.bar` as the `fqnPrefix`, this will result in `deprecated` options being set for the package, all services, all enums, all messages, all RPCs. We will be opinionated and choose *NOT* to deprecate enum values or fields within a message UNLESS they are explicitly specified - recursion does not hit enum values or fields. That is, if I specify a package `foo.bar.baz`, then all files, enums, messages, services, RPCs are deprecated, but not enum values or fields. If I specified `foo.bar.baz.SomeMessage.some_field`, then that field is deprecated.

To implement that logic, you'll likely want to have `bufformat` only deprecate an enum value or field if `fqnPrefix == fqn`, as opposed to `strings.HasPrefix`.

Also, optimally, you will break FQN into components, so that `foo.bar.b` does not match `foo.bar.baz`, but `foo.bar` does match.

There is a lot of commonality between this command and `buf format`, like 90%. You may want to abstract it to a new package `cmd/buf/internal/command/internal`, given that really it's just a matter of what options you apply. The diff is that `buf format` allows exit code differences and a different output location, and defaults to printing to stdout instead of rewriting in place. Otherwise, this is all the same.

Example before and after in the `foo.bar.baz` base described above:

Before:

```proto

syntax = "proto3";

package foo.bar.baz;

enum One {
  ONE_UNSPECIFIED = 0;
  ONE_ONE = 1;
}

message Two {
  enum Three {
    THREE_UNSPECIFIED = 0;
    THREE_ONE = 1;
  }
  message Four {}

  string id = 1;
}

service Five {
  rpc Six(Two) returns (Two);
}
```

After if `foo.bar` is specified for `--deprecate`:

```proto

syntax = "proto3";

package foo.bar.baz;

option deprecated = true;

enum One {
  option deprecated = true;
  ONE_UNSPECIFIED = 0;
  ONE_ONE = 1;
}

message Two {
  option deprecated = true;

  enum Three {
    option deprecated = true;

    THREE_UNSPECIFIED = 0;
    THREE_ONE = 1;
  }
  message Four {
    option deprecated = true;
  }

  string id = 1;
}

service Five {
  option deprecated = true;
  rpc Six(Two) returns (Two) {
    option deprecated = true;
  }
}
```

After if `foo.bar.Two.id` is specified for `--deprecate`:

```proto

syntax = "proto3";

package foo.bar.baz;

enum One {
  ONE_UNSPECIFIED = 0;
  ONE_ONE = 1;
}

message Two {
  enum Three {
    THREE_UNSPECIFIED = 0;
    THREE_ONE = 1;
  }
  message Four {}

  string id = 1 [deprecated = true];
}

service Five {
  rpc Six(Two) returns (Two);
}
```
