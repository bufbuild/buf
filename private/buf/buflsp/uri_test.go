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

package buflsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestNormalizeURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    protocol.URI
		expected protocol.URI
	}{
		{
			name:     "unix-path-unchanged",
			input:    "file:///home/user/project/foo.proto",
			expected: "file:///home/user/project/foo.proto",
		},
		{
			name:     "at-sign-encoded",
			input:    "file:///home/user@host/project/foo.proto",
			expected: "file:///home/user%40host/project/foo.proto",
		},
		{
			name:     "windows-drive-letter-uppercased-and-colon-encoded",
			input:    "file:///C:/Users/project/foo.proto",
			expected: "file:///c%3A/Users/project/foo.proto",
		},
		{
			name:     "windows-lowercase-drive-letter-colon-encoded",
			input:    "file:///d:/Users/project/foo.proto",
			expected: "file:///d%3A/Users/project/foo.proto",
		},
		{
			name:     "non-file-uri-colon-not-encoded",
			input:    "untitled:Untitled-1",
			expected: "untitled:Untitled-1",
		},
		{
			name:     "non-file-uri-at-sign-encoded",
			input:    "untitled:user@host",
			expected: "untitled:user%40host",
		},
		{
			name:     "at-sign-and-windows-drive-letter-both-encoded",
			input:    "file:///C:/Users/user@host/foo.proto",
			expected: "file:///c%3A/Users/user%40host/foo.proto",
		},
		{
			name:     "space-in-path-encoded-as-percent-20",
			input:    "file:///Users/bufbot/My Documents/foo.proto",
			expected: "file:///Users/bufbot/My%20Documents/foo.proto",
		},
		{
			name:     "already-encoded-space-is-idempotent",
			input:    "file:///Users/bufbot/My%20Documents/foo.proto",
			expected: "file:///Users/bufbot/My%20Documents/foo.proto",
		},
		{
			name:     "already-encoded-at-sign-is-idempotent",
			input:    "file:///home/user%40host/project/foo.proto",
			expected: "file:///home/user%40host/project/foo.proto",
		},
		{
			name:     "already-encoded-windows-drive-letter-lowercased",
			input:    "file:///C%3A/Users/project/foo.proto",
			expected: "file:///c%3A/Users/project/foo.proto",
		},
		{
			name:     "percent-literal-in-filename-not-double-encoded",
			input:    "file:///path/100%25done/foo.proto",
			expected: "file:///path/100%25done/foo.proto",
		},
		{
			name:     "space-at-sign-and-windows-drive-letter",
			input:    "file:///C:/Users/user@host/My Documents/foo.proto",
			expected: "file:///c%3A/Users/user%40host/My%20Documents/foo.proto",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.expected, normalizeURI(test.input))
		})
	}
}

func TestFilePathToURIRoundTrip(t *testing.T) {
	t.Parallel()

	// Verify that FilePathToURI produces a URI whose Filename() decodes back to
	// the original path. These are Unix absolute paths; Windows drive letters are
	// covered by TestNormalizeURI.
	paths := []string{
		"/home/user/project/foo.proto",
		"/home/user@host/project/foo.proto",
		"/Users/bufbot/My Documents/foo.proto",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, path, FilePathToURI(path).Filename())
		})
	}
}
