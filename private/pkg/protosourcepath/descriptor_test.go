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

package protosourcepath

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestDescriptorForSourcePath(t *testing.T) {
	t.Parallel()
	fileDescriptor := testBuildFileDescriptor(t, "testdata/descriptor/test.proto")
	for _, testCase := range []struct {
		sourcePath             protoreflect.SourcePath
		expectedDescriptorType string
		expectedDescriptorName protoreflect.FullName
	}{
		// The file itself.
		{protoreflect.SourcePath{}, "file", "foo"},
		// .service(0), .service(0).method(0).
		{protoreflect.SourcePath{6, 0}, "service", "foo.Service"},
		{protoreflect.SourcePath{6, 0, 2, 0}, "method", "foo.Service.Method"},
		// .enum_type(0), .enum_type(0).value(0).
		{protoreflect.SourcePath{5, 0}, "enum", "foo.Enum"},
		{protoreflect.SourcePath{5, 0, 2, 0}, "enum value", "foo.ENUM_UNSPECIFIED"},
		// .message_type(0) and its fields and oneof.
		{protoreflect.SourcePath{4, 0}, "message", "foo.Message"},
		{protoreflect.SourcePath{4, 0, 2, 0}, "field", "foo.Message.field"},
		{protoreflect.SourcePath{4, 0, 8, 0}, "oneof", "foo.Message.one_of"},
		{protoreflect.SourcePath{4, 0, 2, 1}, "field", "foo.Message.one_of_field"},
		// The group field and the synthetic message holding the group's fields.
		{protoreflect.SourcePath{4, 0, 2, 2}, "field", "foo.Message.group"},
		{protoreflect.SourcePath{4, 0, 3, 0}, "message", "foo.Message.Group"},
		{protoreflect.SourcePath{4, 0, 3, 0, 2, 0}, "field", "foo.Message.Group.id"},
		// .message_type(0).nested_type(1) and .message_type(0).enum_type(0).
		{protoreflect.SourcePath{4, 0, 3, 1}, "message", "foo.Message.NestedMessage"},
		{protoreflect.SourcePath{4, 0, 3, 1, 2, 0}, "field", "foo.Message.NestedMessage.field"},
		{protoreflect.SourcePath{4, 0, 4, 0}, "enum", "foo.Message.NestedEnum"},
		{protoreflect.SourcePath{4, 0, 4, 0, 2, 0}, "enum value", "foo.Message.NESTED_ENUM_UNSPECIFIED"},
		// Extensions declared in a message and in the file.
		{protoreflect.SourcePath{4, 0, 6, 0}, "field", "foo.Message.message_extension"},
		{protoreflect.SourcePath{7, 0}, "field", "foo.file_extension"},
	} {
		descriptor := DescriptorForSourcePath(fileDescriptor, testCase.sourcePath)
		require.NotNil(t, descriptor, testCase.sourcePath)
		require.Equal(t, testCase.expectedDescriptorType, testDescriptorTypeName(t, descriptor), testCase.sourcePath)
		require.Equal(t, testCase.expectedDescriptorName, descriptor.FullName(), testCase.sourcePath)
	}
}

func TestDescriptorForSourcePathNotADeclaration(t *testing.T) {
	t.Parallel()
	fileDescriptor := testBuildFileDescriptor(t, "testdata/descriptor/test.proto")
	for _, sourcePath := range []protoreflect.SourcePath{
		// Attributes of a declaration rather than a declaration.
		// .syntax, .package, .message_type(0).name.
		{12},
		{2},
		{4, 0, 1},
		// Declarations without a descriptor.
		// .dependency(0), .message_type(0).extension_range(0), .message_type(0).reserved_range(0).
		{3, 0},
		{4, 0, 5, 0},
		{4, 0, 9, 0},
		// Descending into declarations that do not contain declarations.
		// A field, a oneof, an enum value, and a method.
		{4, 0, 2, 0, 2, 0},
		{4, 0, 8, 0, 2, 0},
		{5, 0, 2, 0, 2, 0},
		{6, 0, 2, 0, 2, 0},
		// Out of range indexes.
		{4, 100},
		{4, 0, 2, 100},
		{4, 0, 3, 100},
		{5, 100},
		{6, 100},
		{7, 100},
		// Negative indexes.
		{4, -1},
		{4, 0, 2, -1},
	} {
		require.Nil(t, DescriptorForSourcePath(fileDescriptor, sourcePath), sourcePath)
	}
}

// testDescriptorTypeName returns a name for the type of the given descriptor, so that tests
// can assert the kind of declaration that was resolved and not just its name.
func testDescriptorTypeName(t *testing.T, descriptor protoreflect.Descriptor) string {
	t.Helper()
	switch descriptor.(type) {
	case protoreflect.FileDescriptor:
		return "file"
	case protoreflect.MessageDescriptor:
		return "message"
	case protoreflect.FieldDescriptor:
		return "field"
	case protoreflect.OneofDescriptor:
		return "oneof"
	case protoreflect.EnumDescriptor:
		return "enum"
	case protoreflect.EnumValueDescriptor:
		return "enum value"
	case protoreflect.ServiceDescriptor:
		return "service"
	case protoreflect.MethodDescriptor:
		return "method"
	}
	t.Fatalf("unknown descriptor type %T", descriptor)
	return ""
}
