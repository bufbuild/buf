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

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ConvertMessage allows the caller to convert a given message data blob from
// one format to another by referring to a type schema for the blob.
func (c *Client) ConvertMessage(ctx context.Context, messageName string, inputData []byte) ([]byte, error) {
	if len(c.types) > 0 && !contains(c.types, messageName) {
		return nil, fmt.Errorf("message_name '%s' is not found in filtered types %v", messageName, c.types)
	}
	// First step: get descriptors for the requested message
	resolver, err := c.getProtoEncodingResolver(ctx)
	if err != nil {
		return nil, err
	}
	// Now we can do the conversion.
	md, err := resolver.FindMessageByName(protoreflect.FullName(messageName))
	if err != nil {
		return nil, err
	}
	msg := dynamicpb.NewMessage(md.Descriptor())

	unmarshaler := getUnmarshaler(c.inputFormat, resolver, c.discardUnknown)
	if err := unmarshaler.Unmarshal(inputData, msg); err != nil {
		return nil, fmt.Errorf("input_data cannot be unmarshaled to %s in %s: %w", messageName, c.inputFormat, err)
	}
	marshaller, err := getMarshaller(c.outputFormat, resolver)
	if err != nil {
		return nil, err
	}
	data, err := marshaller.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("input message cannot be unmarshaled to %s: %w", c.inputFormat, err)
	}
	// Done!
	return data, nil
}

// getProtoEncodingResolver allows the caller to download a schema for one or more requested
// types, RPC services, or RPC methods.
func (c *Client) getProtoEncodingResolver(ctx context.Context) (protoencoding.Resolver, error) {
	if c.cache != nil {
		if out, ok := c.cache.Load(c.moduleName()); ok {
			return out, nil
		}
	}
	request := &registryv1alpha1.GetSchemaRequest{
		Owner:                  c.owner,
		Repository:             c.repository,
		Version:                c.version,
		Types:                  c.types,
		IfNotCommit:            c.ifNotCommit,
		ExcludeCustomOptions:   c.excludeCustomOptions,
		ExcludeKnownExtensions: c.excludeKnownExtensions,
	}
	resp, err := c.client.GetSchema(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, err
	}
	// TODO: unsure if this is a good idea
	c.version = resp.Msg.Commit
	resolver, err := protoencoding.NewResolver(
		protodescriptor.FileDescriptorsForFileDescriptorSet(resp.Msg.SchemaFiles)...,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create resolver from image: %w", err)
	}
	if c.cache != nil {
		go c.cache.Save(c.moduleName(), resolver)
	}
	return resolver, nil
}

func (c *Client) moduleName() string {
	return fmt.Sprintf("%s/%s/%s", c.owner, c.repository, c.version)
}

func getMarshaller(outputFormat interface{}, res protoencoding.Resolver) (protoencoding.Marshaler, error) {
	// TODO: protojson and prototext are notorious in how they handle whitespace
	//	so we should programmatically *reformat* their output so this API has more
	//	stable output data in the responses.
	//	https://github.com/golang/protobuf/issues/1121
	var out protoencoding.Marshaler
	switch format := outputFormat.(type) {
	case *registryv1alpha1.ConvertMessageRequest_OutputBinary:
		out = proto.MarshalOptions{}
	case *registryv1alpha1.ConvertMessageRequest_OutputJson:
		opts := format.OutputJson
		out = protojson.MarshalOptions{
			UseEnumNumbers:  opts.UseEnumNumbers,
			EmitUnpopulated: opts.IncludeDefaults,
			Resolver:        res,
		}
	case *registryv1alpha1.ConvertMessageRequest_OutputText:
		opts := format.OutputText
		out = prototext.MarshalOptions{
			EmitUnknown: opts.IncludeUnrecognized,
			Resolver:    res,
		}
	default:
		return nil, fmt.Errorf("output_format has unrecognized type: %T", outputFormat)
	}
	return out, nil
}

func getUnmarshaler(
	inputFormat registryv1alpha1.Format,
	resolver protoencoding.Resolver,
	discardUnknown bool,
) protoencoding.Unmarshaler {
	var unmarshaler protoencoding.Unmarshaler
	switch inputFormat {
	case registryv1alpha1.Format_FORMAT_BINARY:
		unmarshaler = proto.UnmarshalOptions{
			Resolver:       resolver,
			DiscardUnknown: discardUnknown,
		}
	case registryv1alpha1.Format_FORMAT_JSON:
		unmarshaler = protojson.UnmarshalOptions{
			Resolver:       resolver,
			DiscardUnknown: discardUnknown,
		}
	case registryv1alpha1.Format_FORMAT_TEXT:
		unmarshaler = prototext.UnmarshalOptions{
			Resolver:       resolver,
			DiscardUnknown: discardUnknown,
		}
	}
	return unmarshaler
}

func contains[T comparable](elems []T, want T) bool {
	for _, elem := range elems {
		if elem == want {
			return true
		}
	}
	return false
}
