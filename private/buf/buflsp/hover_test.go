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

	tests := []struct {
		name                string
		protoFile           string
		line                uint32
		character           uint32
		expectedContains    string
		expectedNotContains string
		expectNoHover       bool
	}{
		{
			name:             "hover_on_user_message",
			protoFile:        "testdata/hover/test.proto",
			line:             7,               // Line with "message User {"
			character:        8,               // On the word "User"
			expectedContains: "system.\nThis", // Ensure newline between comment lines
		},
		{
			name:             "hover_on_id_field",
			protoFile:        "testdata/hover/test.proto",
			line:             9,  // Line with "string id = 1;"
			character:        10, // On the word "id"
			expectedContains: "The unique identifier for the user",
		},
		{
			name:             "hover_on_status_enum",
			protoFile:        "testdata/hover/test.proto",
			line:             19, // Line with "enum Status {"
			character:        5,  // On the word "Status"
			expectedContains: "Status represents the current state of a user",
		},
		{
			name:             "hover_on_status_active",
			protoFile:        "testdata/hover/test.proto",
			line:             24, // Line with "STATUS_ACTIVE = 1;"
			character:        2,  // On "STATUS_ACTIVE"
			expectedContains: "The user is active",
		},
		{
			name:             "hover_on_user_service",
			protoFile:        "testdata/hover/test.proto",
			line:             31, // Line with "service UserService {"
			character:        8,  // On "UserService"
			expectedContains: "UserService provides operations for managing users",
		},
		{
			name:             "hover_on_get_user_rpc",
			protoFile:        "testdata/hover/test.proto",
			line:             33, // Line with "rpc GetUser"
			character:        6,  // On "GetUser"
			expectedContains: "GetUser retrieves a user by their ID",
		},
		{
			name:      "hover_on_deprecated_option",
			protoFile: "testdata/hover/test.proto",
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
			protoFile:        "testdata/hover/test.proto",
			line:             15, // Line with "Status status = 3;"
			character:        2,  // On "Status" type
			expectedContains: "Status represents the current state of a user",
		},
		{
			name:             "hover_on_user_type_reference",
			protoFile:        "testdata/hover/test.proto",
			line:             50, // Line with "User user = 1;"
			character:        2,  // On "User" type
			expectedContains: "User represents a user in the system",
		},
		{
			name:             "hover_on_rpc_request_type",
			protoFile:        "testdata/hover/test.proto",
			line:             33, // Line with "rpc GetUser(GetUserRequest)"
			character:        14, // On "GetUserRequest"
			expectedContains: "GetUserRequest is the request message for GetUser",
		},
		{
			name:             "hover_on_rpc_response_type",
			protoFile:        "testdata/hover/test.proto",
			line:             33, // Line with "returns (GetUserResponse)"
			character:        39, // On "GetUserResponse"
			expectedContains: "GetUserResponse is the response message for GetUser",
		},
		{
			name:          "hover_on_syntax_keyword",
			protoFile:     "testdata/hover/test.proto",
			line:          0, // Line with "syntax = "proto3";"
			character:     0, // On "syntax"
			expectNoHover: true,
		},
		{
			name:          "hover_on_proto3_string",
			protoFile:     "testdata/hover/test.proto",
			line:          0,  // Line with "syntax = "proto3";"
			character:     10, // On "proto3"
			expectNoHover: true,
		},
		{
			name:          "hover_on_package_keyword",
			protoFile:     "testdata/hover/test.proto",
			line:          2, // Line with "package example.v1;"
			character:     0, // On "package"
			expectNoHover: true,
		},
		{
			name:          "hover_on_package_name",
			protoFile:     "testdata/hover/test.proto",
			line:          2, // Line with "package example.v1;"
			character:     8, // On "example"
			expectNoHover: true,
		},
		{
			name:             "hover_on_wkt_timestamp",
			protoFile:        "testdata/document_link/wkt.proto",
			line:             8, // Line with "google.protobuf.Timestamp created_at = 1;"
			character:        9, // On "Timestamp" type
			expectedContains: "https://buf.build/protocolbuffers/wellknowntypes/docs/main:google.protobuf#google.protobuf.Timestamp",
		},
		{
			name:             "hover_on_wkt_duration",
			protoFile:        "testdata/document_link/wkt.proto",
			line:             9, // Line with "google.protobuf.Duration timeout = 2;"
			character:        9, // On "Duration" type
			expectedContains: "https://buf.build/protocolbuffers/wellknowntypes/docs/main:google.protobuf#google.protobuf.Duration",
		},
		{
			name:             "hover_on_map_keyword",
			protoFile:        "testdata/completion/map_test.proto",
			line:             16, // Line with "map<int32, string> field0 = 10;"
			character:        3,  // On "map" keyword
			expectedContains: "language-spec#maps",
		},
		{
			name:          "hover_on_message_after_import_with_trailing_comment",
			protoFile:     "testdata/hover/unused_import.proto",
			line:          4, // Line with "message TestTopLevel {"
			character:     8, // On "TestTopLevel"
			expectNoHover: true,
		},
		{
			name:          "hover_on_message_after_import_with_trailing_comment_no_blank_line",
			protoFile:     "testdata/hover/unused_import_no_blank_line.proto",
			line:          3, // Line with "message TestTopLevel {}"
			character:     8, // On "TestTopLevel"
			expectNoHover: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.NotEmpty(t, tt.protoFile, "protoFile must be specified")

			protoPath, err := filepath.Abs(tt.protoFile)
			require.NoError(t, err)

			clientJSONConn, testURI := setupLSPServer(t, protoPath)

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
