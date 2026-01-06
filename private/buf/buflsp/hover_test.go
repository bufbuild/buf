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

package buflsp_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestHover(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/hover/test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name                string
		line                uint32
		character           uint32
		expectedContains    string
		expectedNotContains string
		expectNoHover       bool
	}{
		{
			name:             "hover_on_user_message",
			line:             7, // Line with "message User {"
			character:        8, // On the word "User"
			expectedContains: "User represents a user in the system",
		},
		{
			name:             "hover_on_id_field",
			line:             9,  // Line with "string id = 1;"
			character:        10, // On the word "id"
			expectedContains: "The unique identifier for the user",
		},
		{
			name:             "hover_on_status_enum",
			line:             19, // Line with "enum Status {"
			character:        5,  // On the word "Status"
			expectedContains: "Status represents the current state of a user",
		},
		{
			name:             "hover_on_status_active",
			line:             24, // Line with "STATUS_ACTIVE = 1;"
			character:        2,  // On "STATUS_ACTIVE"
			expectedContains: "The user is active",
		},
		{
			name:             "hover_on_user_service",
			line:             31, // Line with "service UserService {"
			character:        8,  // On "UserService"
			expectedContains: "UserService provides operations for managing users",
		},
		{
			name:             "hover_on_get_user_rpc",
			line:             33, // Line with "rpc GetUser"
			character:        6,  // On "GetUser"
			expectedContains: "GetUser retrieves a user by their ID",
		},
		{
			name:      "hover_on_deprecated_option",
			line:      37, // Line with "option deprecated = true;"
			character: 11, // On "deprecated"
			// We don't want the hover info to include the floating comment that is separated by newlines
			// from the comment above the option.
			// Ref: https://buf.build/protocolbuffers/wellknowntypes/file/main:google/protobuf/descriptor.proto#L946
			expectedNotContains: "Buffers.", // From last line of the previous floating comment.
			expectedContains:    "Is this method deprecated?",
		},
		{
			name:             "hover_on_status_type_reference",
			line:             15, // Line with "Status status = 3;"
			character:        2,  // On "Status" type
			expectedContains: "Status represents the current state of a user",
		},
		{
			name:             "hover_on_user_type_reference",
			line:             50, // Line with "User user = 1;"
			character:        2,  // On "User" type
			expectedContains: "User represents a user in the system",
		},
		{
			name:             "hover_on_rpc_request_type",
			line:             33, // Line with "rpc GetUser(GetUserRequest)"
			character:        14, // On "GetUserRequest"
			expectedContains: "GetUserRequest is the request message for GetUser",
		},
		{
			name:             "hover_on_rpc_response_type",
			line:             33, // Line with "returns (GetUserResponse)"
			character:        39, // On "GetUserResponse"
			expectedContains: "GetUserResponse is the response message for GetUser",
		},
		{
			name:          "hover_on_syntax_keyword",
			line:          0, // Line with "syntax = "proto3";"
			character:     0, // On "syntax"
			expectNoHover: true,
		},
		{
			name:          "hover_on_proto3_string",
			line:          0,  // Line with "syntax = "proto3";"
			character:     10, // On "proto3"
			expectNoHover: true,
		},
		{
			name:          "hover_on_package_keyword",
			line:          2, // Line with "package example.v1;"
			character:     0, // On "package"
			expectNoHover: true,
		},
		{
			name:          "hover_on_package_name",
			line:          2, // Line with "package example.v1;"
			character:     8, // On "example"
			expectNoHover: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var hover *protocol.Hover
			_, hoverErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentHover, protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: testURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
			}, &hover)
			require.NoError(t, hoverErr)

			if tt.expectNoHover {
				assert.Nil(t, hover, "expected no hover information")
			} else {
				if tt.expectedContains != "" {
					require.NotNil(t, hover, "expected hover to be non-nil")
					assert.Equal(t, protocol.Markdown, hover.Contents.Kind)
					assert.Contains(t, hover.Contents.Value, tt.expectedContains)
				}
				if tt.expectedNotContains != "" {
					require.NotNil(t, hover, "expected hover to be non-nil")
					assert.Equal(t, protocol.Markdown, hover.Contents.Kind)
					assert.NotContains(t, hover.Contents.Value, tt.expectedNotContains)
				}
			}
		})
	}
}
