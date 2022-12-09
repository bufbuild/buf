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
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"testing"
)

type fakeSchemaService struct {
	registryv1alpha1connect.UnimplementedSchemaServiceHandler
	getSchemaFunc func(ctx context.Context, req *connect.Request[registryv1alpha1.GetSchemaRequest]) (*connect.Response[registryv1alpha1.GetSchemaResponse], error)
}

func (s fakeSchemaService) GetSchema(ctx context.Context, req *connect.Request[registryv1alpha1.GetSchemaRequest]) (*connect.Response[registryv1alpha1.GetSchemaResponse], error) {
	return s.getSchemaFunc(ctx, req)
}

type clientBuilder func(*Client)

func newClient(_ *testing.T, opts ...clientBuilder) *Client {
	out := &Client{
		cache: cache.NewCache(),
	}
	for _, apply := range opts {
		apply(out)
	}
	return out
}

func withGetSchemaResponse(version string, descriptor *descriptorpb.FileDescriptorSet) clientBuilder {
	return func(c *Client) {
		c.client = &fakeSchemaService{
			getSchemaFunc: func(ctx context.Context, req *connect.Request[registryv1alpha1.GetSchemaRequest]) (*connect.Response[registryv1alpha1.GetSchemaResponse], error) {
				return connect.NewResponse(&registryv1alpha1.GetSchemaResponse{
					Commit:      version,
					SchemaFiles: descriptor,
				}), nil
			},
		}
	}
}

func withGetSchemaError(err error) clientBuilder {
	return func(c *Client) {
		c.client = &fakeSchemaService{
			getSchemaFunc: func(ctx context.Context, req *connect.Request[registryv1alpha1.GetSchemaRequest]) (*connect.Response[registryv1alpha1.GetSchemaResponse], error) {
				return nil, err
			},
		}
	}
}

func TestClient_Transform(t *testing.T) {
	sourceFile := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Syntax:  proto.String("proto2"),
		Package: proto.String("foo.bar"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Message"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("name"),
						Number:   proto.Int32(1),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						JsonName: proto.String("name"),
					},
					{
						Name:     proto.String("id"),
						Number:   proto.Int32(2),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						JsonName: proto.String("id"),
					},
					{
						Name:     proto.String("child"),
						Number:   proto.Int32(3),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".foo.bar.Message"),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						JsonName: proto.String("children"),
					},
					{
						Name:     proto.String("kind"),
						Number:   proto.Int32(4),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".foo.bar.Kind"),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						JsonName: proto.String("kind"),
					},
				},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start: proto.Int32(100),
						End:   proto.Int32(10000),
					},
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Kind"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:   proto.String("UNKNOWN"),
						Number: proto.Int32(0),
					},
					{
						Name:   proto.String("GOOD"),
						Number: proto.Int32(1),
					},
					{
						Name:   proto.String("BAD"),
						Number: proto.Int32(2),
					},
					{
						Name:   proto.String("UGLY"),
						Number: proto.Int32(3),
					},
				},
			},
		},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Name:     proto.String("xt"),
				Extendee: proto.String(".foo.bar.Message"),
				Number:   proto.Int32(123),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			},
		},
	}
	// create test message
	fd, err := protodesc.NewFile(sourceFile, nil)
	require.NoError(t, err)
	md := fd.Messages().Get(0)
	msg := dynamicpb.NewMessage(md)
	msg.Set(md.Fields().ByNumber(1), protoreflect.ValueOfString("abcdef"))
	msg.Set(md.Fields().ByNumber(2), protoreflect.ValueOfInt64(12345678))
	list := msg.Mutable(md.Fields().ByNumber(3)).List()
	list.Append(protoreflect.ValueOfMessage(msg.New()))
	list.Append(protoreflect.ValueOfMessage(msg.New()))
	list.Append(protoreflect.ValueOfMessage(msg.New()))
	msg.Set(md.Fields().ByNumber(4), protoreflect.ValueOfEnum(3))
	inputData, err := proto.Marshal(msg)
	require.NoError(t, err)
	sourceFiles := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{sourceFile}}
	c := newClient(t, withGetSchemaResponse("123456789", sourceFiles))
	got, err := c.Transform(
		context.Background(),
		"foo",
		"bar",
		"baz",
		"123456789",
		false,
		false,
		"foo.bar.Message",
		registryv1alpha1.Format_FORMAT_BINARY,
		inputData,
	)
	require.NoError(t, err)
	fmt.Println(string(got))
}
