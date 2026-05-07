// Copyright 2020-2026 Buf Technologies, Inc.
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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestStripLegacyOptions(t *testing.T) {
	t.Parallel()

	t.Run("no-legacy-options", func(t *testing.T) {
		t.Parallel()
		noLegacy := getFileWithNoLegacyOptions()
		files := []*descriptorpb.FileDescriptorProto{noLegacy}
		err := stripLegacyOptions(files)
		require.NoError(t, err)
		// Slice not touched
		require.Same(t, noLegacy, files[0])
		// File descriptor not mutated in any way
		require.Empty(t, cmp.Diff(getFileWithNoLegacyOptions(), noLegacy, protocmp.Transform()))
	})
	t.Run("legacy_options", func(t *testing.T) {
		t.Parallel()
		legacy := getFileWithLegacyOptions()
		files := []*descriptorpb.FileDescriptorProto{legacy}
		err := stripLegacyOptions(files)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(getFileWithLegacyOptionsRemoved(), files[0], protocmp.Transform()))
		// Original file replaced but not touched
		require.NotSame(t, legacy, files[0])
		require.Empty(t, cmp.Diff(getFileWithLegacyOptions(), legacy, protocmp.Transform()))
	})
	t.Run("mixed", func(t *testing.T) {
		t.Parallel()
		noLegacy := getFileWithNoLegacyOptions()
		legacy := getFileWithLegacyOptions()
		files := []*descriptorpb.FileDescriptorProto{noLegacy, legacy}
		err := stripLegacyOptions(files)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(getFileWithLegacyOptionsRemoved(), files[1], protocmp.Transform()))
		// First file not touched
		require.Same(t, noLegacy, files[0])
		require.Empty(t, cmp.Diff(getFileWithNoLegacyOptions(), noLegacy, protocmp.Transform()))
		// Second file replaced but not touched
		require.NotSame(t, legacy, files[1])
		require.Empty(t, cmp.Diff(getFileWithLegacyOptions(), legacy, protocmp.Transform()))

		// Make sure Go runtime is okay with result
		_, err = NewResolver(files...)
		require.NoError(t, err)
	})
	t.Run("protobuf-go-is-happy", func(t *testing.T) {
		t.Parallel()
		// Go runtime is unhappy because of legacy features.
		files := []*descriptorpb.FileDescriptorProto{getFileWithNoLegacyOptions(), getFileWithLegacyOptions()}
		_, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: files})
		require.ErrorContains(t, err, "legacy proto1 feature that is no longer supported")

		err = stripLegacyOptions(files)
		require.NoError(t, err)

		// Now it's happy since legacy features have been removed.
		_, err = protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: files})
		require.NoError(t, err)
	})
}

func getFileWithNoLegacyOptions() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    new("no_legacy.proto"),
		Package: new("foo.bar"),
		Syntax:  new("proto2"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: new("Foo"),
				Options: &descriptorpb.MessageOptions{
					NoStandardDescriptorAccessor: new(true),
					Deprecated:                   new(true),
				},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start: proto.Int32(100),
						End:   proto.Int32(maxTagNumber + 1),
					},
				},
			},
			{
				Name: new("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   new("name"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Options: &descriptorpb.FieldOptions{
							DebugRedact: new(true),
						},
					},
				},
			},
		},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: new(".foo.bar.Foo"),
				Name:     new("a"),
				Number:   proto.Int32(maxTagNumber),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
				Options: &descriptorpb.FieldOptions{
					Jstype: descriptorpb.FieldOptions_JS_STRING.Enum(),
				},
			},
		},
	}
}

func getFileWithLegacyOptions() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       new("legacy.proto"),
		Package:    new("foo.bar"),
		Syntax:     new("proto2"),
		Dependency: []string{"no_legacy.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: new("Baz"),
				Options: &descriptorpb.MessageOptions{
					NoStandardDescriptorAccessor: new(true),
					Deprecated:                   new(true),
				},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start: proto.Int32(100),
						End:   proto.Int32(maxTagNumber + 1),
					},
				},
			},
			{
				Name: new("Frob"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   new("name"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Options: &descriptorpb.FieldOptions{
							DebugRedact: new(true),
							// TO BE REMOVED
							Weak: new(true),
						},
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: new("Nitz"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   new("name"),
								Number: proto.Int32(1),
								Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Options: &descriptorpb.FieldOptions{
									// TO BE REMOVED
									Weak: new(true),
								},
							},
						},
					},
				},
				Extension: []*descriptorpb.FieldDescriptorProto{
					{
						Extendee: new(".foo.bar.Baz"),
						Name:     new("a"),
						Number:   proto.Int32(1000),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
						Options: &descriptorpb.FieldOptions{
							Jstype: descriptorpb.FieldOptions_JS_STRING.Enum(),
							// TO BE REMOVED
							Weak: new(true),
						},
					},
				},
			},
			{
				Name: new("Fizz"),
				Options: &descriptorpb.MessageOptions{
					// TO BE REMOVED
					MessageSetWireFormat: new(true),
				},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start: proto.Int32(100),
						End:   proto.Int32(1000),
					},
					{
						// TO BE MODIFIED
						Start: proto.Int32(1000),
						End:   proto.Int32(maxTagNumber * 2),
					},
					{
						// TO BE REMOVED
						Start: proto.Int32(maxTagNumber * 3),
						End:   proto.Int32(maxTagNumber*3 + 1000),
					},
					{
						// TO BE REMOVED
						Start: proto.Int32(maxTagNumber * 4),
						End:   proto.Int32(maxTagNumber*4 + 3),
					},
				},
			}},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: new(".foo.bar.Foo"),
				Name:     new("b"),
				Number:   proto.Int32(100),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
				Options: &descriptorpb.FieldOptions{
					Jstype: descriptorpb.FieldOptions_JS_STRING.Enum(),
					// TO BE REMOVED
					Weak: new(true),
				},
			},
			{
				Extendee: new(".foo.bar.Baz"),
				Name:     new("c"),
				Number:   proto.Int32(100),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
				Options: &descriptorpb.FieldOptions{
					Jstype: descriptorpb.FieldOptions_JS_STRING.Enum(),
					// TO BE REMOVED
					Weak: new(true),
				},
			},
			{
				Extendee: new(".foo.bar.Fizz"),
				Name:     new("d"),
				Number:   proto.Int32(100),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: new(".foo.bar.Foo"),
			},
			{
				Extendee: new(".foo.bar.Fizz"),
				Name:     new("e"),
				Number:   proto.Int32(maxTagNumber),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: new(".foo.bar.Bar"),
			},
			{
				// TO BE REMOVED
				Extendee: new(".foo.bar.Fizz"),
				Name:     new("f"),
				Number:   proto.Int32(maxTagNumber * 2),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: new(".foo.bar.Baz"),
			},
			{
				// TO BE REMOVED
				Extendee: new(".foo.bar.Fizz"),
				Name:     new("g"),
				Number:   proto.Int32(maxTagNumber * 4),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: new(".foo.bar.Frob"),
			},
		},
	}
}

func getFileWithLegacyOptionsRemoved() *descriptorpb.FileDescriptorProto {
	// Returns the same thing as getFileWithLegacyOptions, but without
	// legacy options/values in it.
	return &descriptorpb.FileDescriptorProto{
		Name:       new("legacy.proto"),
		Package:    new("foo.bar"),
		Syntax:     new("proto2"),
		Dependency: []string{"no_legacy.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: new("Baz"),
				Options: &descriptorpb.MessageOptions{
					NoStandardDescriptorAccessor: new(true),
					Deprecated:                   new(true),
				},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start: proto.Int32(100),
						End:   proto.Int32(maxTagNumber + 1),
					},
				},
			},
			{
				Name: new("Frob"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   new("name"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Options: &descriptorpb.FieldOptions{
							DebugRedact: new(true),
						},
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: new("Nitz"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:    new("name"),
								Number:  proto.Int32(1),
								Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:    descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								Options: &descriptorpb.FieldOptions{},
							},
						},
					},
				},
				Extension: []*descriptorpb.FieldDescriptorProto{
					{
						Extendee: new(".foo.bar.Baz"),
						Name:     new("a"),
						Number:   proto.Int32(1000),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
						Options: &descriptorpb.FieldOptions{
							Jstype: descriptorpb.FieldOptions_JS_STRING.Enum(),
						},
					},
				},
			},
			{
				Name:    new("Fizz"),
				Options: &descriptorpb.MessageOptions{},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{
						Start: proto.Int32(100),
						End:   proto.Int32(1000),
					},
					{
						Start: proto.Int32(1000),
						End:   proto.Int32(maxTagNumber + 1),
					},
				},
			}},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: new(".foo.bar.Foo"),
				Name:     new("b"),
				Number:   proto.Int32(100),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
				Options: &descriptorpb.FieldOptions{
					Jstype: descriptorpb.FieldOptions_JS_STRING.Enum(),
				},
			},
			{
				Extendee: new(".foo.bar.Baz"),
				Name:     new("c"),
				Number:   proto.Int32(100),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
				Options: &descriptorpb.FieldOptions{
					Jstype: descriptorpb.FieldOptions_JS_STRING.Enum(),
				},
			},
			{
				Extendee: new(".foo.bar.Fizz"),
				Name:     new("d"),
				Number:   proto.Int32(100),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: new(".foo.bar.Foo"),
			},
			{
				Extendee: new(".foo.bar.Fizz"),
				Name:     new("e"),
				Number:   proto.Int32(maxTagNumber),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: new(".foo.bar.Bar"),
			},
		},
	}
}
