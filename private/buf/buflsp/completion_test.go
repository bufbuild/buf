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

func TestCompletion(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/completion/test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name                string
		line                uint32
		character           uint32
		expectedContains    []string
		expectedNotContains []string
		expectNoCompletions bool
	}{
		{
			name:             "complete_builtin_toplevel",
			line:             5,
			character:        1, // After the "m"
			expectedContains: []string{"message"},
		},
		{
			name:             "complete_message_field_types",
			line:             9, // Empty line in User message where field would go
			character:        2, // Indented position where field type would be
			expectedContains: []string{"string", "int32", "int64", "bool", "bytes", "User", "GetUserRequest", "GetUserResponse"},
		},
		{
			name:             "complete_builtin_service",
			line:             14,
			character:        2, // Indented position where "rpc" would be
			expectedContains: []string{"rpc", "option"},
		},
		{
			name:             "complete_rpc_request_type",
			line:             13,
			character:        uint32(len("  rpc GetUser(Get") - 1),
			expectedContains: []string{"GetUserRequest", "GetUserResponse"},
		},
		{
			name:             "complete_rpc_response_type",
			line:             13,
			character:        uint32(len("  rpc GetUser(Get) returns (Get") - 1),
			expectedContains: []string{"GetUserRequest", "GetUserResponse"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var completionList *protocol.CompletionList
			_, completionErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: testURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
			}, &completionList)
			require.NoError(t, completionErr)
			if tt.expectNoCompletions {
				assert.Nil(t, completionList, "expected no completions")
				return
			}
			require.NotNil(t, completionList, "expected completion list to be non-nil")
			labels := make([]string, 0, len(completionList.Items))
			for _, item := range completionList.Items {
				labels = append(labels, item.Label)
			}
			for _, expected := range tt.expectedContains {
				assert.Contains(t, labels, expected, "expected completion list to contain %q", expected)
			}
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, labels, notExpected, "expected completion list to not contain %q", notExpected)
			}
		})
	}
}
