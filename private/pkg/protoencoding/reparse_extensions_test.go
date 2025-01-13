// Copyright 2020-2025 Buf Technologies, Inc.
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

package protoencoding

import (
	"math"
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/gofeaturespb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReparseExtensions(t *testing.T) {
	t.Parallel()

	descriptorFile := protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto)
	durationFile := protodesc.ToFileDescriptorProto(durationpb.File_google_protobuf_duration_proto)
	timestampFile := protodesc.ToFileDescriptorProto(timestamppb.File_google_protobuf_timestamp_proto)
	validateFile := protodesc.ToFileDescriptorProto(validate.File_buf_validate_validate_proto)

	// The file will include one custom option with a known/generated type.
	fieldOpts := &descriptorpb.FieldOptions{}
	fieldConstraints := &validate.FieldConstraints{
		Required: proto.Bool(true),
		Type: &validate.FieldConstraints_Int32{
			Int32: &validate.Int32Rules{
				GreaterThan: &validate.Int32Rules_Gt{
					Gt: 0,
				},
			},
		},
	}
	proto.SetExtension(fieldOpts, validate.E_Field, fieldConstraints)
	// The file will also contain an unrecognized custom option.
	const customOptionNum = 54321
	const customOptionVal = float32(3.14159)
	var unknownOption []byte
	unknownOption = protowire.AppendTag(unknownOption, customOptionNum, protowire.Fixed32Type)
	unknownOption = protowire.AppendFixed32(unknownOption, math.Float32bits(customOptionVal))
	fieldOpts.ProtoReflect().SetUnknown(unknownOption)

	testFile := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("test.proto"),
		Syntax:     proto.String("proto3"),
		Package:    proto.String("blah.blah"),
		Dependency: []string{"buf/validate/validate.proto", "google/protobuf/descriptor.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Foo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("bar"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
						JsonName: proto.String("bar"),
						Options:  fieldOpts,
					},
				},
			},
		},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: proto.String(".google.protobuf.FieldOptions"),
				Name:     proto.String("baz"),
				Number:   proto.Int32(customOptionNum),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_FLOAT.Enum(),
			},
		},
	}

	resolver, err := NewResolver(descriptorFile, durationFile, timestampFile, validateFile, testFile)
	require.NoError(t, err)
	err = ReparseExtensions(resolver, testFile.ProtoReflect())
	require.NoError(t, err)

	require.Empty(t, fieldOpts.ProtoReflect().GetUnknown())
	var found int
	fieldOpts.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch field.Number() {
		case customOptionNum:
			found++
			assert.Equal(t, customOptionVal, value.Interface())
		case protoreflect.FieldNumber(validate.E_Field.Field):
			found++
			msg := value.Message().Interface()
			assert.NotSame(t, fieldConstraints, msg)
			_, isGenType := msg.(*validate.FieldConstraints)
			assert.False(t, isGenType)
			_, isDynamicType := msg.(*dynamicpb.Message)
			assert.True(t, isDynamicType)

			// round-trip back to gen type to check for equality with original
			data, err := proto.Marshal(msg)
			require.NoError(t, err)
			roundTrippedConstraints := &validate.FieldConstraints{}
			err = proto.Unmarshal(data, roundTrippedConstraints)
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(fieldConstraints, roundTrippedConstraints, protocmp.Transform()))
		}
		return true
	})
	assert.Equal(t, 2, found)
}

func TestReparseExtensionsGoFeatures(t *testing.T) {
	t.Parallel()

	goFeaturesMessageDesc := gofeaturespb.File_google_protobuf_go_features_proto.Messages().ByName("GoFeatures")
	dynamicGoFeatures := dynamicpb.NewMessage(goFeaturesMessageDesc)
	dynamicGoFeatures.Set(
		goFeaturesMessageDesc.Fields().ByName("api_level"),
		protoreflect.ValueOfEnum(gofeaturespb.GoFeatures_API_OPAQUE.Number()),
	)
	assert.True(t, dynamicGoFeatures.IsValid())
	dynamicExt := dynamicpb.NewExtensionType(gofeaturespb.E_Go.TypeDescriptor().Descriptor())

	featureSet := &descriptorpb.FeatureSet{}
	featureSetReflect := featureSet.ProtoReflect()
	featureSetReflect.Set(
		dynamicExt.TypeDescriptor(),
		protoreflect.ValueOfMessage(dynamicGoFeatures),
	)

	// Validates the error conditions that cause this panic.
	// See issue https://github.com/golang/protobuf/issues/1669
	assert.Panics(t, func() {
		proto.GetExtension(featureSet, gofeaturespb.E_Go)
	})
	descFileDesc, err := protoregistry.GlobalFiles.FindFileByPath("google/protobuf/descriptor.proto")
	assert.NoError(t, err)
	goFeaturesFileDesc, err := protoregistry.GlobalFiles.FindFileByPath("google/protobuf/go_features.proto")
	assert.NoError(t, err)
	fileDesc := &descriptorpb.FileDescriptorProto{
		Name: proto.String("a.proto"),
		Dependency: []string{
			"google/protobuf/go_features.proto",
		},
		Edition: descriptorpb.Edition_EDITION_2023.Enum(),
		Syntax:  proto.String("editions"),
		Options: &descriptorpb.FileOptions{
			Features: featureSet,
		},
	}
	fileSet := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(descFileDesc),
			protodesc.ToFileDescriptorProto(goFeaturesFileDesc),
			fileDesc,
		},
	}
	assert.Panics(t, func() {
		// TODO: if this no longer panics, we can remove the code handling
		// this workaround in bufcheck.imageToProtoFileDescriptors.
		_, err := protodesc.NewFiles(fileSet)
		assert.NoError(t, err)
	})

	// Run the resvoler to convert the extension.
	goFeaturesResolver, err := newGoFeaturesResolver()
	require.NoError(t, err)
	err = ReparseExtensions(goFeaturesResolver, featureSetReflect)
	require.NoError(t, err)
	goFeatures, ok := proto.GetExtension(featureSet, gofeaturespb.E_Go).(*gofeaturespb.GoFeatures)
	require.True(t, ok)
	assert.Equal(t, goFeatures.GetApiLevel(), gofeaturespb.GoFeatures_API_OPAQUE)
}
