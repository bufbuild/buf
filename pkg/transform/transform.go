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
	"fmt"
	"github.com/bufbuild/buf/pkg/internal/cache"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Client ..
type Client struct {
	client registryv1alpha1connect.SchemaServiceClient
	cache  *cache.Cache
}

// NewClient ..
func NewClient(
	httpClient connect.HTTPClient,
	baseURL string,
	opts ...connect.ClientOption,
) *Client {
	return &Client{
		client: registryv1alpha1connect.NewSchemaServiceClient(
			httpClient,
			baseURL,
			opts...,
		),
		cache: cache.NewCache(),
	}
}

// GetSchema ..
func (c *Client) GetSchema(
	ctx context.Context,
	owner, repository, reference, ifNotCommit string,
	excludeCustomOptions, excludeKnownExtensions bool,
) (string, *descriptorpb.FileDescriptorSet, error) {
	if out, ok := c.checkCache(owner, repository, reference); ok {
		return out.Commit, out.SchemaFiles, nil
	}
	request := &registryv1alpha1.GetSchemaRequest{
		Owner:                  owner,
		Repository:             repository,
		Version:                reference,
		IfNotCommit:            ifNotCommit,
		ExcludeCustomOptions:   excludeCustomOptions,
		ExcludeKnownExtensions: excludeKnownExtensions,
	}
	resp, err := c.client.GetSchema(ctx, connect.NewRequest(request))
	if err != nil {
		return "", nil, err
	}
	out := resp.Msg
	go c.cache.Write(fmt.Sprintf("%s/%s/%s", owner, repository, out.Commit), out)
	return out.Commit, out.SchemaFiles, nil
}

// TODO: get this to "work" with the elementNames attribute
func (c *Client) checkCache(owner string, repository string, reference string) (*registryv1alpha1.GetSchemaResponse, bool) {
	module := fmt.Sprintf("%s/%s/%s", owner, repository, reference)
	cached, ok := c.cache.Read(module)
	if ok {
		if schema, ok := cached.(*registryv1alpha1.GetSchemaResponse); ok {
			return schema, ok
		}
	}
	return nil, ok
}

// Transform ..
func (c *Client) Transform(ctx context.Context,
	owner, repository, reference, ifNotCommit string,
	excludeCustomOptions, excludeKnownExtensions bool,
	messageName string, inputFormat registryv1alpha1.Format, inputData []byte,
) ([]byte, error) {
	_, descriptorSet, err := c.GetSchema(
		ctx,
		owner,
		repository,
		reference,
		ifNotCommit,
		excludeCustomOptions,
		excludeKnownExtensions,
	)
	if err != nil {
		return nil, err
	}
	return ConvertMessage(descriptorSet, messageName, inputFormat, inputData, false)
}

func ConvertMessage(
	schemaFiles *descriptorpb.FileDescriptorSet,
	messageName string,
	inputFormat registryv1alpha1.Format,
	inputData []byte,
	discardUnknown bool,
) ([]byte, error) {
	// First step: get descriptors for the requested message
	res, err := protoencoding.NewResolver(
		protodescriptor.FileDescriptorsForFileDescriptorSet(schemaFiles)...,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("unable to create resolver from image: %w", err))
	}
	md, err := res.FindMessageByName(protoreflect.FullName(messageName))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Now we can do the conversion.
	msg := dynamicpb.NewMessage(md.Descriptor())
	var unmarshaler protoencoding.Unmarshaler
	switch inputFormat {
	case registryv1alpha1.Format_FORMAT_BINARY:
		unmarshaler = proto.UnmarshalOptions{
			Resolver:       res,
			DiscardUnknown: discardUnknown,
		}
	case registryv1alpha1.Format_FORMAT_JSON:
		unmarshaler = protojson.UnmarshalOptions{
			Resolver:       res,
			DiscardUnknown: discardUnknown,
		}
	case registryv1alpha1.Format_FORMAT_TEXT:
		unmarshaler = prototext.UnmarshalOptions{
			Resolver:       res,
			DiscardUnknown: discardUnknown,
		}
	}
	if err := unmarshaler.Unmarshal(inputData, msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("input_data cannot be unmarshaled to %s in %s: %w", messageName, inputFormat, err))
	}

	// TODO: protojson and prototext are notorious in how they handle whitespace
	//  so we should programmatically *reformat* their output so this API has more
	//  stable output data in the responses.
	//  https://github.com/golang/protobuf/issues/1121
	var marshaler protoencoding.Marshaler
	opts := registryv1alpha1.JSONOutputOptions{}
	marshaler = protojson.MarshalOptions{
		UseEnumNumbers:  opts.UseEnumNumbers,
		EmitUnpopulated: opts.IncludeDefaults,
		Resolver:        res,
	}
	data, err := marshaler.Marshal(msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("input message cannot be unmarshaled to %s: %w", inputFormat, err))
	}

	// Done!
	return data, nil
}
