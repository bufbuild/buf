![The Buf logo](../../.github/buf-logo.svg)

### 🚧 Documentation Under Construction

# Transform

Allows the caller to convert a given message data blob from one format to
another by referring to a type schema for the blob.

A default client provides you with connection to the public Buf Schema Registry
and an in memory cache which expires unused items after 5 minutes of activity.
You must provide it with a reference to a Buf Module and the expected format of
the data you wish to convert and the resulting format you would like.

```go
package foo

import (
	"context"
	"net/http"

	"github.com/bufbuild/buf/pkg/transform"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

const (
	useEnumNumbers  = false
	includeDefaults = false
)

// ConvertBinaryToJSON receives a typical protobuf encoded bag of 
// bytes and returns a JSON representation. 
func ConvertBinaryToJSON(ctx context.Context, inputData []byte) error {
	client, err := transform.DefaultClient(
		"bufbuild",
		"registry",
		"01b54e71e6b84514a9141323afdb95a1",
		registryv1alpha1.Format_FORMAT_BINARY,
		transform.ToDefaultJSONOutput(),
	)
	if err != nil {
		return err
	}
	result, err := client.ConvertMessage(ctx, "foo.bar.Message", inputData)
	if err != nil {
		return err
	}
	println(result)
	return nil
}
```

### Minimum requirements

Some basic requirements must be met when creating a transformation client, these
ensure that the package understands what it needs to get, what its being given
and what actions it needs to perform.

| Configuration       | Mandatory | Options                                       |
|---------------------|-----------|-----------------------------------------------|
| WithSchemaService() | Y         | -                                             |
| WithBufModule()     | Y         | `version` optional                            |
| FromFormat()        | Y         | `FORMAT_BINARY`, `FORMAT_JSON`, `FORMAT_TEXT` |
| ToFormat()          | Y         | `Binary`, `Json`, `Text`                      |
| WithCache()         | N         | Not Required but highly recommended           |

`NewClient` builds a transform client, processing various configurable options,
let us discover some of those options. This will allow you to further control
transform behaviour to suit your needs.

```go
package foo

import (
	"context"
	"net/http"

	"github.com/bufbuild/buf/pkg/transform"
	"github.com/bufbuild/buf/pkg/transform/internal/cache"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

const (
	useEnumNumbers  = false
	includeDefaults = false
	discardUnknown  = false
)

// ConvertBinaryToJSON receives a typical protobuf encoded bag of 
// bytes and returns a JSON representation. 
func ConvertBinaryToJSON(ctx context.Context, inputData []byte) error {
	client, err := transform.NewClient(
		// Buf Schema Registry location where the protobuf schema is stored 
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		// Add a cache to save on network call overhead
		transform.WithCache(),
		// Supplying the Buf module which describes the inputData
		transform.WithBufModule(
			"bufbuild",
			"registry",
			"01b54e71e6b84514a9141323afdb95a1",
		),
		// In our case, we are converting from binary, you could also 
		// use text or json as your source format
		transform.FromFormat(registryv1alpha1.Format_FORMAT_BINARY, discardUnknown),
		// Our resulting format
		transform.ToJSONOutput(useEnumNumbers, includeDefaults),
	)
	if err != nil {
		return err
	}
	result, err := client.ConvertMessage(ctx, "foo.bar.Message", inputData)
	if err != nil {
		return err
	}
	println(result)
	return nil
}

```

`WithSchemaService` configures the remote Buf Schema Registry. Accepting a
`HTTPClient`, `baseURL` and connect client options.

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
+		transform.WithSchemaService(http.DefaultClient, "buf.build"),
	)
}
```

[//]: # (TODO: expose delete expired to make this statement true)
`WithCache` To keep the file descriptor for the requested Buf Module in memory, 
avoid unnecessary network calls we recommend you initialize the cache,
without it the package will fetch the schema before every message conversion.

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
+		transform.WithCache(),
	)
}
```

`WithBufModule` accepts the owner of the repo that contains the schema to
retrieve (a username or organization name).
The name of the repo that contains the schema to retrieve.
(Optional) version of the repo. This can be a tag or branch name or a commit.
If version is unspecified, defaults to the latest version on the repo's
"main" branch.

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
+		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
	)
}
```

`IncludeTypes` accepts Zero or more types names. The names may refer to
messages, enums, services, methods, or extensions. All names must be
fully-qualified. If any name is unknown, the request will fail and no schema
will be returned.

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
+		transform.IncludeTypes("foo.bar.Baz", "fizz.buzz.FizzBuzz"),
	)
}
```

If no names are provided, the full schema for the module is returned.
Otherwise, the resulting schema contains only the named elements and all of
their dependencies. This is enough information for the caller to construct
a dynamic message for any requested message types or to dynamically invoke
an RPC for any requested methods or services.

`FromFormat` requires the format of the input data and an option to discard
unknown values. If true, any unresolvable fields in the input are discarded.
For formats other than FORMAT_BINARY, this means that the operation will
fail if the input contains unrecognized field names. For FORMAT_BINARY,
unrecognized fields can be retained and possibly included in the reformatted
output (depending on the requested output format).

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
		transform.IncludeTypes("foo.bar.Baz", "fizz.buzz.FizzBuzz"),
+		transform.FromFormat(registryv1alpha1.Format_FORMAT_BINARY, discardUnknown),
	)
}
```

[//]: # (TODO: supply logic in constructor or support the user through this flow)
`Exclude` configures the schema that is fetched from the schema service,
providing 2 configurable options:

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
		transform.IncludeTypes("foo.bar.Baz", "fizz.buzz.FizzBuzz"),
		transform.FromFormat(registryv1alpha1.Format_FORMAT_BINARY, discardUnknown),
+		transform.Exclude(excludeCustomOptions, excludeKnownExtensions),
	)
}
```

`excludeCustomOptions` - If true, the returned schema will not include
extension definitions for custom options that appear on schema elements.
When filtering the schema based on the given element names, options on all
encountered elements are usually examined as well. But that is not the case
if excluding custom options.

This flag is ignored if `IncludeTypes()` is empty as the entire schema is always
returned in that case.

`excludeKnownExtensions` - If true, the returned schema will not include known
extensions for extendable messages for schema elements.

`IfNotCommit` is a commit that the client already has cached. So if the
given module version resolves to this same commit, the server should not
send back any descriptors since the client already has them.
This allows a client to efficiently poll for updates: after the initial RPC
to get a schema, the client can cache the descriptors and the resolved
commit. It then includes that commit in subsequent requests in this field,
and the server will only reply with a schema (and new commit) if/when the
resolved commit changes.

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
		transform.IncludeTypes("foo.bar.Baz", "fizz.buzz.FizzBuzz"),
		transform.FromFormat(registryv1alpha1.Format_FORMAT_BINARY, discardUnknown),
		transform.Exclude(excludeCustomOptions, excludeKnownExtensions),
+		transform.IfNotCommit("foo"),
	)
}
```

`ToBinaryOutput` specifies the output format as Binary

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
		transform.IncludeTypes("foo.bar.Baz", "fizz.buzz.FizzBuzz"),
		transform.FromFormat(registryv1alpha1.Format_FORMAT_BINARY, discardUnknown),
		transform.Exclude(excludeCustomOptions, excludeKnownExtensions),
		transform.IfNotCommit("foo"),
+		transform.ToBinaryOutput(),
	)
}
```

`ToJSONOutput` specifies the output format as JSON. Accepts `UseEnumNumbers`
for Enum fields will be emitted as numeric values. If false (the default),
enum fields are emitted as strings that are the enum values' names.
`includeDefaults` Includes fields that have their default values. This applies
only to fields defined in proto3 syntax that have no explicit "optional"
keyword. Other optional fields will be included if present in the input data.

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
		transform.IncludeTypes("foo.bar.Baz", "fizz.buzz.FizzBuzz"),
		transform.FromFormat(registryv1alpha1.Format_FORMAT_BINARY, discardUnknown),
		transform.Exclude(excludeCustomOptions, excludeKnownExtensions),
		transform.IfNotCommit("foo"),
+		transform.ToJSONOutput(UseEnumNumbers, includeDefaults),
	)
}
```

`ToTextOutput` specifies the output format as Text

```diff
func Foo(ctx context.Context) (*transform.Client, error) {
	return transform.NewClient(
		transform.WithSchemaService(http.DefaultClient, "buf.build"),
		transform.WithCache(),
		transform.WithBufModule("bufbuild", "registry", "01b54e71e6b84514a9141323afdb95a1"),
		transform.IncludeTypes("foo.bar.Baz", "fizz.buzz.FizzBuzz"),
		transform.FromFormat(registryv1alpha1.Format_FORMAT_BINARY, discardUnknown),
		transform.Exclude(excludeCustomOptions, excludeKnownExtensions),
		transform.IfNotCommit("foo"),
+		transform.ToTextOutput(),
	)
}
```
