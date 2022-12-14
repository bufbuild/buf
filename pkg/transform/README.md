![The Buf logo](../../.github/buf-logo.svg)

# Transform

[![Slack](https://img.shields.io/badge/slack-buf-%23e01563)][badges_slack]

_This project is currently in **alpha**. The API should be considered unstable and likely to change._

Allows the caller to convert a given message data blob from one format to
another by referring to a type schema for the blob.

A default client provides you with connection to the public Buf Schema Registry
and an in memory cache which will maintain the `protoencoding.Resolver` for
future transformations. You must provide it with a reference to a Buf Module and
the expected format of the data you wish to convert and the resulting format you
would like.

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
| WithCache()         | N         | Not _Required_ but highly recommended         |

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
		transform.WithNewSchemaService(http.DefaultClient, "buf.build"),
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

Discover more of the client configuration options [here](docs/client.md)

## Community

For help and discussion around Protobuf, best practices, and more, join us on [Slack][badges_slack].

[badges_slack]: https://join.slack.com/t/bufbuild/shared_invite/zt-f5k547ki-dW9LjSwEnl6qTzbyZtPojw
