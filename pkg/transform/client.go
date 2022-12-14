// Copyright 2020-2022 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transform

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bufbuild/buf/pkg/transform/internal/cache"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/connect-go"
)

// Client ..
type Client struct {
	client                 registryv1alpha1connect.SchemaServiceClient
	cache                  *cache.Cache
	owner                  string
	repository             string
	version                string
	types                  []string
	discardUnknown         bool
	inputFormat            registryv1alpha1.Format
	ifNotCommit            string
	excludeCustomOptions   bool
	excludeKnownExtensions bool
	outputFormat           interface{}
}

type ClientBuilder func(*Client)

// NewClient builds a transform client, processing various configurable options
func NewClient(ctx context.Context, builders ...ClientBuilder) (*Client, error) {
	client := &Client{}
	for _, apply := range builders {
		apply(client)
	}
	if err := client.validate(); err != nil {
		return nil, err
	}
	if client.cache != nil {
		// pre-load proto encoding resolver into cache
		if _, err := client.getProtoEncodingResolver(ctx); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func DefaultClient(
	ctx context.Context,
	owner, repository, version string,
	inputFormat registryv1alpha1.Format,
	outputFormat ClientBuilder,
) (*Client, error) {
	return NewClient(
		ctx,
		WithDefaultSchemaService(),
		WithDefaultCache(),
		WithBufModule(owner, repository, version),
		FromFormat(inputFormat, false),
		outputFormat,
	)
}

// WithNewSchemaService configures the remote Buf Schema Registry. Accepting a
// HTTPClient, baseURL and connect client options.
func WithNewSchemaService(
	httpClient connect.HTTPClient,
	baseURL string,
	opts ...connect.ClientOption,
) ClientBuilder {
	return func(c *Client) {
		c.client = registryv1alpha1connect.NewSchemaServiceClient(
			httpClient,
			baseURL,
			opts...,
		)
	}
}

func WithDefaultSchemaService() ClientBuilder {
	return WithNewSchemaService(http.DefaultClient, "buf.build")
}

func WithSchemaService(
	client registryv1alpha1connect.SchemaServiceHandler,
) ClientBuilder {
	return func(c *Client) {
		c.client = client
	}
}

// WithCache provided a default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
func WithCache() ClientBuilder {
	return func(c *Client) {
		c.cache = cache.NewCache()
	}
}

func WithDefaultCache() ClientBuilder {
	return WithCache()
}

// WithBufModule accepts the owner of the repo that contains the schema to
// retrieve (a username or organization name).
// The name of the repo that contains the schema to retrieve.
// (Optional) version of the repo. This can be a tag or branch name or a commit.
// If version is unspecified, defaults to the latest version on the repo's
// "main" branch.
func WithBufModule(
	owner string,
	repository string,
	version string,
) ClientBuilder {
	return func(c *Client) {
		c.owner = owner
		c.repository = repository
		c.version = version
	}
}

// IncludeTypes accepts Zero or more types names. The names may refer to
// messages, enums, services, methods, or extensions. All names must be
// fully-qualified. If any name is unknown, the request will fail and no schema
// will be returned.
//
// If no names are provided, the full schema for the module is returned.
// Otherwise, the resulting schema contains only the named elements and all of
// their dependencies. This is enough information for the caller to construct
// a dynamic message for any requested message types or to dynamically invoke
// an RPC for any requested methods or services.
func IncludeTypes(types ...string) ClientBuilder {
	return func(c *Client) {
		c.types = types
	}
}

// FromFormat requires the format of the input data and an option to discard
// unknown values. If true, any unresolvable fields in the input are discarded.
// For formats other than FORMAT_BINARY, this means that the operation will
// fail if the input contains unrecognized field names. For FORMAT_BINARY,
// unrecognized fields can be retained and possibly included in the reformatted
// output (depending on the requested output format).
// TODO: supply logic in constructor or support the user through this flow
func FromFormat(
	format registryv1alpha1.Format,
	discardUnknown bool,
) ClientBuilder {
	return func(c *Client) {
		c.inputFormat = format
		c.discardUnknown = discardUnknown
	}
}

// Exclude configures the schema that is fetched from the schema service,
// providing 2 configurable options:
// `excludeCustomOptions` - If true, the returned schema will not include
// extension definitions for custom options that appear on schema elements.
// When filtering the schema based on the given element names, options on all
// encountered elements are usually examined as well. But that is not the case
// if excluding custom options.
// `excludeKnownExtensions` - If true, the returned schema will not include
// known extensions for extendable messages for schema elements. If
// exclude_custom_options is true, such extensions may still be returned if the
// applicable descriptor options type is part of the requested schema.
//
// These flags are ignored if `IncludeTypes()` is empty as the entire schema is
// always returned in that case.
func Exclude(excludeCustomOptions, excludeKnownExtensions bool) ClientBuilder {
	return func(c *Client) {
		c.excludeCustomOptions = excludeCustomOptions
		c.excludeKnownExtensions = excludeKnownExtensions
	}
}

// IfNotCommit is a commit that the client already has cached. So if the
// given module version resolves to this same commit, the server should not
// send back any descriptors since the client already has them.
// This allows a client to efficiently poll for updates: after the initial RPC
// to get a schema, the client can cache the descriptors and the resolved
// commit. It then includes that commit in subsequent requests in this field,
// and the server will only reply with a schema (and new commit) if/when the
// resolved commit changes.
func IfNotCommit(in string) ClientBuilder {
	return func(c *Client) {
		c.ifNotCommit = in
	}
}

// ToBinaryOutput specifies the output format as Binary
func ToBinaryOutput() ClientBuilder {
	return func(c *Client) {
		c.outputFormat = &registryv1alpha1.ConvertMessageRequest_OutputBinary{
			OutputBinary: &registryv1alpha1.BinaryOutputOptions{},
		}
	}
}

// ToJSONOutput specifies the output format as JSON. Accepts `UseEnumNumbers`
// for Enum fields will be emitted as numeric values. If false (the default),
// enum fields are emitted as strings that are the enum values' names.
// includeDefaults Includes fields that have their default values. This applies
// only to fields defined in proto3 syntax that have no explicit "optional"
// keyword. Other optional fields will be included if present in the input data.
func ToJSONOutput(useEnumNumbers, includeDefaults bool) ClientBuilder {
	return func(c *Client) {
		c.outputFormat = &registryv1alpha1.ConvertMessageRequest_OutputJson{
			OutputJson: &registryv1alpha1.JSONOutputOptions{
				UseEnumNumbers:  useEnumNumbers,
				IncludeDefaults: includeDefaults,
			},
		}
	}
}

func ToDefaultJSONOutput() ClientBuilder {
	return ToJSONOutput(false, false)
}

// ToTextOutput specifies the output format as Text
func ToTextOutput() ClientBuilder {
	return func(c *Client) {
		c.outputFormat = &registryv1alpha1.ConvertMessageRequest_OutputText{
			OutputText: &registryv1alpha1.TextOutputOptions{},
		}
	}
}

func (c *Client) validate() error {
	if c.client == nil {
		return fmt.Errorf("SchemaServiceClient not provided")
	}
	if c.owner == "" {
		return fmt.Errorf("buf module owner not provided")
	}
	if c.repository == "" {
		return fmt.Errorf("buf module repository not provided")
	}
	if c.version == "" {
		return fmt.Errorf("buf module version not provided")
	}
	if _, ok := registryv1alpha1.Format_name[int32(c.inputFormat)]; !ok ||
		c.inputFormat == registryv1alpha1.Format_FORMAT_UNSPECIFIED {
		return fmt.Errorf("input_format value %v is not valid", c.inputFormat)
	}
	if err := validateOutputFormat(c.outputFormat); err != nil {
		return err
	}
	return nil
}

func validateOutputFormat(outputFormat interface{}) error {
	if outputFormat == nil {
		return errors.New("output_format not specified")
	}
	switch outputFormat.(type) {
	case *registryv1alpha1.ConvertMessageRequest_OutputBinary,
		*registryv1alpha1.ConvertMessageRequest_OutputJson,
		*registryv1alpha1.ConvertMessageRequest_OutputText:
		return nil
	case nil:
		return errors.New("output_format not specified")
	default:
		return fmt.Errorf("output_format has unrecognized type: %T", outputFormat)
	}
}

func GetOutputFormat(request *registryv1alpha1.ConvertMessageRequest) (ClientBuilder, error) {
	var output ClientBuilder
	switch outputFormat := request.OutputFormat.(type) {
	case *registryv1alpha1.ConvertMessageRequest_OutputBinary:
		output = ToBinaryOutput()
	case *registryv1alpha1.ConvertMessageRequest_OutputJson:
		opts := outputFormat.OutputJson
		output = ToJSONOutput(opts.UseEnumNumbers, opts.IncludeDefaults)
	case *registryv1alpha1.ConvertMessageRequest_OutputText:
		output = ToTextOutput()
	default:
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown output_format provided %s", outputFormat))
	}
	return output, nil
}

func (c *Client) Commit() string {
	return c.version
}
