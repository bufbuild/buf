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

// This file defines all of the message handlers that involve symbols.
//
// In particular, this file handles semantic information in fileManager that have been
// *opened by the editor*, and thus do not need references to Buf modules to find.
// See imports.go for that part of the LSP.

package buflsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommentToMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single-line-comment",
			input:    "// this is a single-line comment",
			expected: " this is a single-line comment",
		},
		{
			name: "multi-line-comment",
			input: `/*
 this is a
 multi-line comment
*/`,
			expected: `
 this is a
 multi-line comment
`,
		},
		{
			name: "doxygen-style-comment",
			input: `/**
 * Documentation comment
 * with asterisks
 */`,
			expected: ` Documentation comment
 with asterisks`,
		},
		{
			name: "doxygen-mixed-indentation",
			input: `/**
 * First line
 * - Second line
 *   - Third line
 */`,
			expected: ` First line
 - Second line
   - Third line`,
		},
		{
			name:     "markdown-emphasis",
			input:    "/*This is *important**/",
			expected: "This is *important*",
		},
		{
			name:     "single-line-doxygen",
			input:    "/** Single line doc comment */",
			expected: "Single line doc comment",
		},
		{
			name:     "empty-comment",
			input:    "/**/",
			expected: "",
		},
		{
			name:     "only-space",
			input:    "/* */",
			expected: " ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, commentToMarkdown(test.input))
		})
	}
}
