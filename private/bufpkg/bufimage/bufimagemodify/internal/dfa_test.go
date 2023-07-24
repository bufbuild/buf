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
		expected    bool
	}{
		{
			description: "field in single message",
			path:        []int32{4, 100, 2, 101, 8},
			expected:    true,
		},
		{
			description: "field in a nested message",
			path:        []int32{4, 100, 3, 101, 2, 102, 8},
			expected:    true,
		},
		{
			description: "field in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 2, 104, 8},
			expected:    true,
		},
		{
			description: "extension field on top level",
			path:        []int32{7, 100, 8},
			expected:    true,
		},
		{
			description: "extension field in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 6, 104, 8},
			expected:    true,
		},
		{
			description: "empty path",
			path:        []int32{},
			expected:    false,
		},
		{
			description: "invalid first index in path of length one",
			path:        []int32{1},
			expected:    false,
		},
		{
			description: "invalid first index",
			path:        []int32{1, 0, 2, 0, 8},
			expected:    false,
		},
		{
			description: "messages",
			path:        []int32{4},
			expected:    false,
		},
		{
			description: "top level extensions",
			path:        []int32{7},
			expected:    false,
		},
		{
			description: "field in single message, without options tag",
			path:        []int32{4, 100, 2, 101},
			expected:    false,
		},
		{
			description: "one field option",
			path:        []int32{4, 100, 2, 101, 8, 6},
			expected:    false,
		},
		{
			description: "field label",
			path:        []int32{4, 100, 2, 101, 4},
			expected:    false,
		},
		{
			description: "enum in message",
			path:        []int32{4, 100, 4, 100},
			expected:    false,
		},
		{
			description: "one field option in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 2, 104, 8, 1},
			expected:    false,
		},
		{
			description: "one field option of extension field on top level",
			path:        []int32{7, 100, 8, 1},
			expected:    false,
		},
		{
			description: "extension field itself",
			path:        []int32{7, 100},
			expected:    false,
		},
		{
			description: "extension field itself in a deeply nested message",
			path:        []int32{4, 100, 3, 101, 3, 102, 3, 103, 3, 0, 6, 104},
			expected:    false,
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				testcase.expected,
				isPathForFieldOptions(testcase.path),
			)
		})
	}
}
