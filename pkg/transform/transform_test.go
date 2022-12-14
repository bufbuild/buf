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
	"testing"

	"buf.build/gen/go/bufbuild/buf/bufbuild/connect-go/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "buf.build/gen/go/bufbuild/buf/protocolbuffers/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/connect-go"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type fakeSchemaService struct {
	registryv1alpha1connect.UnimplementedSchemaServiceHandler
	getSchemaFunc func(ctx context.Context, req *connect.Request[registryv1alpha1.GetSchemaRequest]) (*connect.Response[registryv1alpha1.GetSchemaResponse], error)
}

func (s fakeSchemaService) GetSchema(ctx context.Context, req *connect.Request[registryv1alpha1.GetSchemaRequest]) (*connect.Response[registryv1alpha1.GetSchemaResponse], error) {
	return s.getSchemaFunc(ctx, req)
}

func withGetSchemaResponse(version string, descriptor *descriptorpb.FileDescriptorSet) ClientBuilder {
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

func TestSchemaService_ConvertMessage(t *testing.T) {
	// create schema for message to convert
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

	inputFormats := []struct {
		format    registryv1alpha1.Format
		marshaler func(proto.Message) ([]byte, error)
	}{
		{
			format:    registryv1alpha1.Format_FORMAT_BINARY,
			marshaler: proto.Marshal,
		},
		{
			format:    registryv1alpha1.Format_FORMAT_JSON,
			marshaler: protojson.Marshal,
		},
		{
			format:    registryv1alpha1.Format_FORMAT_TEXT,
			marshaler: prototext.Marshal,
		},
	}

	outputFormats := []struct {
		format      registryv1alpha1.Format
		unmarshaler func([]byte, proto.Message) error
	}{
		{
			format:      registryv1alpha1.Format_FORMAT_BINARY,
			unmarshaler: proto.Unmarshal,
		},
		{
			format:      registryv1alpha1.Format_FORMAT_JSON,
			unmarshaler: protojson.Unmarshal,
		},
		{
			format:      registryv1alpha1.Format_FORMAT_TEXT,
			unmarshaler: prototext.Unmarshal,
		},
	}

	for _, inputFormat := range inputFormats {
		for _, outputFormat := range outputFormats {
			t.Run(fmt.Sprintf("%v_to_%v", inputFormat.format, outputFormat.format), func(t *testing.T) {
				data, err := inputFormat.marshaler(msg)
				require.NoError(t, err)
				sourceFiles := &descriptorpb.FileDescriptorSet{
					File: []*descriptorpb.FileDescriptorProto{
						sourceFile,
					},
				}
				builders := []ClientBuilder{
					withGetSchemaResponse("baz", sourceFiles),
					WithBufModule("foo", "bar", "baz"),
					FromFormat(inputFormat.format, false),
					WithCache(),
				}
				switch outputFormat.format {
				case registryv1alpha1.Format_FORMAT_BINARY:
					builders = append(builders, ToBinaryOutput())
				case registryv1alpha1.Format_FORMAT_JSON:
					builders = append(builders, ToJSONOutput(false, false))
				case registryv1alpha1.Format_FORMAT_TEXT:
					builders = append(builders, ToTextOutput())
				default:
					t.Fatalf("unknown output format %v", outputFormat.format)
				}

				s, err := NewClient(context.Background(), builders...)
				require.NoError(t, err)
				resp, err := s.ConvertMessage(context.Background(), "foo.bar.Message", data)
				require.NoError(t, err)
				clone := msg.New().Interface()
				err = outputFormat.unmarshaler(resp, clone)
				require.NoError(t, err)
				diff := cmp.Diff(msg, clone, protocmp.Transform())
				if diff != "" {
					t.Errorf("round-trip failure (-want +got):\n%s", diff)
				}
			})
		}
	}
}
