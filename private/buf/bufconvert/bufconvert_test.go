// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufconvert

import (
	"reflect"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestImageWithoutMessageSetWireFormatResolution(t *testing.T) {
	t.Parallel()
	file := getTestFileWithMessageSets()
	imageFile, err := bufimage.NewImageFile(
		file,
		nil,
		uuid.UUID{},
		file.GetName(),
		file.GetName(),
		false,
		false,
		nil,
	)
	require.NoError(t, err)
	image, err := bufimage.NewImage([]bufimage.ImageFile{imageFile})
	require.NoError(t, err)

	noResolveImage := ImageWithoutMessageSetWireFormatResolution(image)
	// assert.Same only supported pointers and not other reference types like slices :/
	assert.Equal(t, reflect.ValueOf(image.Files()).Pointer(), reflect.ValueOf(noResolveImage.Files()).Pointer())

	checker := resultChecker{t}
	checker.succeeds(noResolveImage.Resolver().FindDescriptorByName("foo.bar.Baz"))
	checker.fails(noResolveImage.Resolver().FindDescriptorByName("foo.bar.MessageSetBaz"))
	checker.fails(noResolveImage.Resolver().FindDescriptorByName("foo.bar.ContainsMessageSetBaz"))
	checker.fails(noResolveImage.Resolver().FindDescriptorByName("foo.bar.IndirectContainsMessageSetBaz"))
	checker.succeeds(noResolveImage.Resolver().FindDescriptorByName("foo.bar.baz"))
	checker.fails(noResolveImage.Resolver().FindDescriptorByName("foo.bar.message_set_baz"))
	checker.fails(noResolveImage.Resolver().FindDescriptorByName("foo.bar.contains_message_set_baz"))
	checker.fails(noResolveImage.Resolver().FindDescriptorByName("foo.bar.indirect_contains_message_set_baz"))
	checker.succeeds(noResolveImage.Resolver().FindDescriptorByName("foo.bar.Enum"))

	checker.succeeds(noResolveImage.Resolver().FindMessageByName("foo.bar.Baz"))
	checker.fails(noResolveImage.Resolver().FindMessageByName("foo.bar.MessageSetBaz"))
	checker.fails(noResolveImage.Resolver().FindMessageByName("foo.bar.ContainsMessageSetBaz"))
	checker.fails(noResolveImage.Resolver().FindMessageByName("foo.bar.IndirectContainsMessageSetBaz"))

	checker.succeeds(noResolveImage.Resolver().FindMessageByURL("type.googleapis.com/foo.bar.Baz"))
	checker.fails(noResolveImage.Resolver().FindMessageByURL("type.googleapis.com/foo.bar.MessageSetBaz"))
	checker.fails(noResolveImage.Resolver().FindMessageByURL("type.googleapis.com/foo.bar.ContainsMessageSetBaz"))
	checker.fails(noResolveImage.Resolver().FindMessageByURL("type.googleapis.com/foo.bar.IndirectContainsMessageSetBaz"))

	checker.succeeds(noResolveImage.Resolver().FindExtensionByName("foo.bar.str"))
	checker.succeeds(noResolveImage.Resolver().FindExtensionByName("foo.bar.baz"))
	checker.fails(noResolveImage.Resolver().FindExtensionByName("foo.bar.message_set_baz"))
	checker.fails(noResolveImage.Resolver().FindExtensionByName("foo.bar.contains_message_set_baz"))
	checker.fails(noResolveImage.Resolver().FindExtensionByName("foo.bar.indirect_contains_message_set_baz"))

	checker.succeeds(noResolveImage.Resolver().FindExtensionByNumber("foo.bar.Baz", 10101))
	checker.succeeds(noResolveImage.Resolver().FindExtensionByNumber("foo.bar.Baz", 10102))
	checker.fails(noResolveImage.Resolver().FindExtensionByNumber("foo.bar.Baz", 10103))
	checker.fails(noResolveImage.Resolver().FindExtensionByNumber("foo.bar.Baz", 10104))
	checker.fails(noResolveImage.Resolver().FindExtensionByNumber("foo.bar.Baz", 10105))

	checker.succeeds(noResolveImage.Resolver().FindEnumByName("foo.bar.Enum"))
}

func TestFindMessageInFile(t *testing.T) {
	t.Parallel()
	t.Run("no-package", func(t *testing.T) {
		t.Parallel()
		file := getTestFile("" /* no package */)

		doFindMessageInFile(t, "Foo", file, true)
		doFindMessageInFile(t, "Bar", file, true)
		doFindMessageInFile(t, "Baz", file, true)
		doFindMessageInFile(t, "Foo.Frob", file, true)
		doFindMessageInFile(t, "Bar.Buzz", file, true)
		doFindMessageInFile(t, "Baz.Abc.Xyz.Deeper.AndDeeper", file, true)

		doFindMessageInFile(t, "foo.bar.Foo", file, false)
		doFindMessageInFile(t, "Foobar", file, false)
		doFindMessageInFile(t, "Foo.Nitz.Abc", file, false)
		doFindMessageInFile(t, "Baz.Abc.Xyz.Deeper.Shallower", file, false)
	})
	t.Run("with-package", func(t *testing.T) {
		t.Parallel()
		file := getTestFile("buf.build.test")

		doFindMessageInFile(t, "buf.build.test.Foo", file, true)
		doFindMessageInFile(t, "buf.build.test.Bar", file, true)
		doFindMessageInFile(t, "buf.build.test.Baz", file, true)
		doFindMessageInFile(t, "buf.build.test.Foo.Frob", file, true)
		doFindMessageInFile(t, "buf.build.test.Bar.Buzz", file, true)
		doFindMessageInFile(t, "buf.build.test.Baz.Abc.Xyz.Deeper.AndDeeper", file, true)

		doFindMessageInFile(t, "Foo", file, false)
		doFindMessageInFile(t, "buf.Foo", file, false)
		doFindMessageInFile(t, "buf.build.Foo", file, false)
		doFindMessageInFile(t, "buf.build.test.Foobar", file, false)
		doFindMessageInFile(t, "buf.build.test.Foo.Nitz.Abc", file, false)
		doFindMessageInFile(t, "buf.build.test.Baz.Abc.Xyz.Deeper.Shallower", file, false)
	})
}

type resultChecker struct {
	t *testing.T
}

func (c resultChecker) succeeds(result any, err error) {
	c.t.Helper()
	require.NoError(c.t, err)
	require.NotNil(c.t, result)
}

func (c resultChecker) fails(_ any, err error) {
	c.t.Helper()
	var msgSetErr *messageSetNotSupportedError
	require.ErrorAs(c.t, err, &msgSetErr)
}

func doFindMessageInFile(t *testing.T, name protoreflect.FullName, file *descriptorpb.FileDescriptorProto, expectToFind bool) {
	t.Helper()
	descriptor := findMessageInFile(name, file)
	if !expectToFind {
		require.Nil(t, descriptor)
		return
	}
	require.NotNil(t, descriptor)
	require.Equal(t, descriptor.GetName(), string(name.Name()))
}

func getTestFile(pkg string) *descriptorpb.FileDescriptorProto {
	var protoPkg *string
	if pkg != "" {
		protoPkg = proto.String(pkg)
	}
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: protoPkg,
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Foo"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Frob"),
					},
					{
						Name: proto.String("Nitz"),
					},
				},
			},
			{
				Name: proto.String("Bar"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Fizz"),
					},
					{
						Name: proto.String("Buzz"),
					},
				},
			},
			{
				Name: proto.String("Baz"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Abc"),
						NestedType: []*descriptorpb.DescriptorProto{
							{
								Name: proto.String("Xyz"),
								NestedType: []*descriptorpb.DescriptorProto{
									{
										Name: proto.String("Deeper"),
										NestedType: []*descriptorpb.DescriptorProto{
											{
												Name: proto.String("AndDeeper"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func getTestFileWithMessageSets() *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
		Package: proto.String("foo.bar"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Baz"),
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{Start: proto.Int32(100), End: proto.Int32(99999)},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("name"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     proto.String("fizz"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".foo.bar.Fizz"),
					},
					{
						Name:     proto.String("buzz"),
						Number:   proto.Int32(3),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".foo.bar.Buzz"),
					},
				},
			},
			{
				Name:    proto.String("Fizz"),
				Options: &descriptorpb.MessageOptions{Deprecated: proto.Bool(true)},
			},
			{
				Name: proto.String("Buzz"),
			},
			{
				Name:    proto.String("MessageSetBaz"),
				Options: &descriptorpb.MessageOptions{MessageSetWireFormat: proto.Bool(true)},
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
					{Start: proto.Int32(1), End: proto.Int32(9999999)},
				},
			},
			{
				Name: proto.String("ContainsMessageSetBaz"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("baz"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".foo.bar.MessageSetBaz"),
					},
				},
			},
			{
				Name: proto.String("IndirectContainsMessageSetBaz"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("bazes"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".foo.bar.IndirectContainsMessageSetBaz.BazesEntry"),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name:    proto.String("BazesEntry"),
						Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   proto.String("key"),
								Number: proto.Int32(1),
								Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
							{
								Name:     proto.String("value"),
								Number:   proto.Int32(2),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
								TypeName: proto.String(".foo.bar.ContainsMessageSetBaz"),
							},
						},
					},
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Enum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:   proto.String("ZERO"),
						Number: proto.Int32(0),
					},
				},
			},
		},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: proto.String(".foo.bar.Baz"),
				Name:     proto.String("str"),
				Number:   proto.Int32(10101),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			},
			{
				Extendee: proto.String(".foo.bar.Baz"),
				Name:     proto.String("baz"),
				Number:   proto.Int32(10102),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".foo.bar.Baz"),
			},
			{
				Extendee: proto.String(".foo.bar.Baz"),
				Name:     proto.String("message_set_baz"),
				Number:   proto.Int32(10103),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".foo.bar.MessageSetBaz"),
			},
			{
				Extendee: proto.String(".foo.bar.Baz"),
				Name:     proto.String("contains_message_set_baz"),
				Number:   proto.Int32(10104),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".foo.bar.ContainsMessageSetBaz"),
			},
			{
				Extendee: proto.String(".foo.bar.Baz"),
				Name:     proto.String("indirect_contains_message_set_baz"),
				Number:   proto.Int32(10105),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".foo.bar.IndirectContainsMessageSetBaz"),
			},
		},
	}
}
