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

package bufcheck

import (
	"context"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const groupsProtoFileContent = `syntax = "proto2";

package a;

message Foo {
  optional group Group = 1 {
    optional string id = 1;
  }
  optional string scalar = 2;
  message Nested {
    optional group NestedGroup = 1 {
      optional string id = 1;
    }
  }
  extend Bar {
    optional group MessageExtensionGroup = 101 {
      optional string id = 1;
    }
  }
}

message Bar {
  extensions 100 to 200;
}

extend Bar {
  optional group FileExtensionGroup = 100 {
    optional string id = 1;
  }
}
`

func TestGroupFieldSyntheticMessage(t *testing.T) {
	t.Parallel()
	fileDescriptor := testCompileFileDescriptor(t, groupsProtoFileContent)
	sourceLocations := fileDescriptor.SourceLocations()
	fooMessageDescriptor := fileDescriptor.Messages().ByName("Foo")
	require.NotNil(t, fooMessageDescriptor)
	nestedMessageDescriptor := fooMessageDescriptor.Messages().ByName("Nested")
	require.NotNil(t, nestedMessageDescriptor)
	for _, testCase := range []struct {
		name                            string
		fieldDescriptor                 protoreflect.FieldDescriptor
		expectedSyntheticMessageName    protoreflect.FullName
		expectedNoSyntheticMessageFound bool
	}{
		{
			name:                         "group field",
			fieldDescriptor:              fooMessageDescriptor.Fields().ByName("group"),
			expectedSyntheticMessageName: "a.Foo.Group",
		},
		{
			name:                         "group field in nested message",
			fieldDescriptor:              nestedMessageDescriptor.Fields().ByName("nestedgroup"),
			expectedSyntheticMessageName: "a.Foo.Nested.NestedGroup",
		},
		{
			name:                         "group extension field in message",
			fieldDescriptor:              fooMessageDescriptor.Extensions().ByName("messageextensiongroup"),
			expectedSyntheticMessageName: "a.Foo.MessageExtensionGroup",
		},
		{
			name:                         "group extension field in file",
			fieldDescriptor:              fileDescriptor.Extensions().ByName("fileextensiongroup"),
			expectedSyntheticMessageName: "a.FileExtensionGroup",
		},
		{
			name:                            "non-group field",
			fieldDescriptor:                 fooMessageDescriptor.Fields().ByName("scalar"),
			expectedNoSyntheticMessageFound: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.NotNil(t, testCase.fieldDescriptor)
			sourcePath := sourceLocations.ByDescriptor(testCase.fieldDescriptor).Path
			require.NotEmpty(t, sourcePath)
			syntheticMessageDescriptor := groupFieldSyntheticMessage(fileDescriptor, sourcePath)
			if testCase.expectedNoSyntheticMessageFound {
				require.Nil(t, syntheticMessageDescriptor)
				return
			}
			require.NotNil(t, syntheticMessageDescriptor)
			require.Equal(t, testCase.expectedSyntheticMessageName, syntheticMessageDescriptor.FullName())
		})
	}
}

func TestGroupFieldSyntheticMessageNonFieldSourcePaths(t *testing.T) {
	t.Parallel()
	fileDescriptor := testCompileFileDescriptor(t, groupsProtoFileContent)
	for _, sourcePath := range []protoreflect.SourcePath{
		nil,
		{},
		// .package.
		{2},
		// .message_type(0).
		{4, 0},
		// .message_type(0).field(0).name.
		{4, 0, 2, 0, 1},
		// .message_type(0).nested_type(0).
		{4, 0, 3, 0},
		// .message_type(0).enum_type(0).
		{4, 0, 4, 0},
		// Out of range indexes.
		{4, 100, 2, 0},
		{4, 0, 2, 100},
		{7, 100},
		// Negative indexes.
		{4, -1, 2, 0},
		{4, 0, 2, -1},
	} {
		require.Nil(
			t,
			groupFieldSyntheticMessage(fileDescriptor, sourcePath),
			"source path %v", sourcePath,
		)
	}
}

func testCompileFileDescriptor(t *testing.T, fileContent string) protoreflect.FileDescriptor {
	t.Helper()
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{"a.proto": fileContent}),
		},
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	files, err := compiler.Compile(context.Background(), "a.proto")
	require.NoError(t, err)
	require.Len(t, files, 1)
	return files[0]
}
