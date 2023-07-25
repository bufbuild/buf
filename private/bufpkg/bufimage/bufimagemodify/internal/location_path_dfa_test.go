// Copyright 2020-2023 Buf Technologies, Inc.
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

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPathForFieldOptions(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description string
		path        []int32
		expected    pathType
	}{
		{
			description: "field options in single message",
			path:        []int32{4, 100, 2, 101, 8},
			expected:    pathTypeFieldOptions,
		},
		{
			description: "field options in a nested message",
			path:        []int32{4, 100, 3, 101, 2, 102, 8},
			expected:    pathTypeFieldOptions,
		},
		{
			description: "field options in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 2, 104, 8},
			expected:    pathTypeFieldOptions,
		},
		{
			description: "field options in extension field on top level",
			path:        []int32{7, 100, 8},
			expected:    pathTypeFieldOptions,
		},
		{
			description: "field options in extension field in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 6, 104, 8},
			expected:    pathTypeFieldOptions,
		},
		{
			description: "empty path",
			path:        []int32{},
			expected:    pathTypeEmpty,
		},
		{
			description: "invalid first index in path of length one",
			path:        []int32{1},
			expected:    pathTypeInvalid,
		},
		{
			description: "invalid first index",
			path:        []int32{1, 0, 2, 0, 8},
			expected:    pathTypeInvalid,
		},
		{
			description: "messages",
			path:        []int32{4},
			expected:    pathTypeMessages,
		},
		{
			description: "top level extensions",
			path:        []int32{7},
			expected:    pathTypeFields,
		},
		{
			description: "field in single message",
			path:        []int32{4, 100, 2, 101},
			expected:    pathTypeField,
		},
		{
			description: "field option jstype in non-nested message",
			path:        []int32{4, 100, 2, 101, 8, 6},
			expected:    pathTypeFieldOption,
		},
		{
			description: "field label",
			path:        []int32{4, 100, 2, 101, 4},
			expected:    pathTypeInvalid,
		},
		{
			description: "enum in message",
			path:        []int32{4, 100, 4, 100},
			expected:    pathTypeInvalid,
		},
		{
			description: "one field in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 2, 104},
			expected:    pathTypeField,
		},
		{
			description: "one field option in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 2, 104, 8, 1},
			expected:    pathTypeFieldOption,
		},
		{
			description: "one field option of extension field on top level",
			path:        []int32{7, 100, 8, 1},
			expected:    pathTypeFieldOption,
		},
		{
			description: "complex field option",
			path:        []int32{4, 0, 2, 1, 8, 50000, 1},
			expected:    pathTypeFieldOption,
		},
		{
			description: "extension field",
			path:        []int32{7, 100},
			expected:    pathTypeField,
		},
		{
			description: "extension field in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 6, 104},
			expected:    pathTypeField,
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testcase.expected,
				getPathType(testcase.path),
			)
		})
	}
}
