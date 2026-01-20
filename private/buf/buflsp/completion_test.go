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

	"buf.build/go/standard/xslices"
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
			name:             "complete_message_field_modifiers",
			line:             9, // Empty line in User message where field modifier would go
			character:        2, // Indented position where field modifier would be
			expectedContains: []string{"map", "repeated", "optional"},
		},
		{
			name:             "complete_message_field_types",
			line:             10, // Empty line in User message where field would go
			character:        2,  // Indented position where field type would be
			expectedContains: []string{"string", "int32", "int64", "bool", "bytes", "User", "GetUserRequest", "GetUserResponse"},
		},
		{
			name:             "complete_builtin_service",
			line:             15,
			character:        2, // Indented position where "rpc" would be
			expectedContains: []string{"rpc", "option"},
		},
		{
			name:             "complete_rpc_request_type",
			line:             14,
			character:        uint32(len("  rpc GetUser(Get") - 1),
			expectedContains: []string{"GetUserRequest", "GetUserResponse"},
		},
		{
			name:             "complete_rpc_response_type",
			line:             14,
			character:        uint32(len("  rpc GetUser(Get) returns (Get") - 1),
			expectedContains: []string{"GetUserRequest", "GetUserResponse"},
		},
		{
			name:             "complete_field_number_after_reserved",
			line:             32, // Line with "User user ="
			character:        14, // After "  User user = "
			expectedContains: []string{"6"},
		},
		{
			name:             "complete_field_number_after_reserved_with_semicolon",
			line:             41, // Line with "User user = ;"
			character:        14, // After "  User user = "
			expectedContains: []string{"6"},
		},
		{
			name:             "complete_field_number_skips_protobuf_reserved_range",
			line:             47, // Line with "User user = ;"
			character:        14, // After "  User user = "
			expectedContains: []string{"20000"},
		},
		{
			name:                "complete_absolute_type_reference",
			line:                51, // Line with ".goo field_name = 1;"
			character:           4,  // After ".goo"
			expectedContains:    []string{".google.protobuf.Timestamp", ".google.protobuf.Duration", ".google.protobuf.Any"},
			expectedNotContains: []string{".example.v1.User", ".example.v1.GetUserRequest"},
		},
		{
			name:             "complete_enum_number_basic",
			line:             57, // Line with "STATUS_NEW = " (0-indexed)
			character:        15, // After "  STATUS_NEW = "
			expectedContains: []string{"2"},
		},
		{
			name:             "complete_enum_number_with_gaps",
			line:             64, // Line with "STATUS_NEW = ;" (0-indexed)
			character:        15, // After "  STATUS_NEW = "
			expectedContains: []string{"2"},
		},
		{
			name:             "complete_enum_number_with_reserved",
			line:             74, // Line with "STATUS_NEW = ;" (0-indexed from line 74)
			character:        15, // After "  STATUS_NEW = "
			expectedContains: []string{"6"},
		},
		{
			name:             "complete_enum_number_empty",
			line:             78, // Line with "STATUS_FIRST = ;" in EmptyStatus (0-indexed from line 78)
			character:        17, // After "  STATUS_FIRST = "
			expectedContains: []string{"0"},
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

func TestCompletionAfterUpdate(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	testProtoPath, err := filepath.Abs("testdata/completion/update_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Send a didChange notification to mutate the file state.
	// This simulates a user typing "str" on a new field line within the User message.
	// We replace the entire content with an updated version that has "  str" added.
	updatedContent := `syntax = "proto3";

package example.v1;

message User {
  string id = 1;
  str
}
`
	err = clientJSONConn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Version: 2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{
				Text: updatedContent,
			},
		},
	})
	require.NoError(t, err)

	// Now request completions at the position where we just inserted "str"
	// This should return completions for "string" and other types starting with "str"
	var completionList *protocol.CompletionList
	_, completionErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Position: protocol.Position{
				Line:      6,
				Character: 5, // After "  str"
			},
		},
	}, &completionList)
	require.NoError(t, completionErr)
	require.NotNil(t, completionList, "expected completion list to be non-nil after file update")

	// Check that we get "string" as a completion option
	labels := xslices.Map(completionList.Items, func(item protocol.CompletionItem) string {
		return item.Label
	})
	assert.Contains(t, labels, "string", "expected completion list to contain 'string' after partial edit")
}

func TestCompletionMaps(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/completion/map_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name                string
		line                uint32
		character           uint32
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:                "complete_map_key_type_all",
			line:                16, // Line with "map<int32, string> field0 = 10;"
			character:           6,  // After "  map<" (no prefix)
			expectedContains:    []string{"int32", "int64", "bool", "string", "uint32", "sint32"},
			expectedNotContains: []string{"float", "double", "bytes"}, // Not allowed for map keys
		},
		{
			name:                "complete_map_key_type_prefix",
			line:                19,                                   // Line with "map<int32, string> field1 = 1;"
			character:           7,                                    // After "  map<i"
			expectedContains:    []string{"int32", "int64"},           // Types starting with "i"
			expectedNotContains: []string{"float", "double", "bytes"}, // Not allowed for map keys
		},
		{
			name:             "complete_map_value_type_builtin",
			line:             22, // Line with "map<int32, string> field2 = 2;"
			character:        13, // After "  map<int32, s"
			expectedContains: []string{"string", "sint32", "sint64"},
		},
		{
			name:             "complete_map_value_type_custom",
			line:             25, // Line with "map<string, User> field3 = 3;"
			character:        16, // After "  map<string, U"
			expectedContains: []string{"User"},
		},
		{
			name:                "complete_map_key_type_no_enums_or_messages",
			line:                28,                                              // Line with "map<int32, string> field4 = 4;"
			character:           6,                                               // After "  map<" (no prefix)
			expectedContains:    []string{"int32", "bool"},                       // Valid map key types
			expectedNotContains: []string{"User", "Status", "MapCompletionTest"}, // Enums and messages not allowed
		},
		{
			name:                "complete_map_key_type_incomplete_syntax",
			line:                31,                                              // Line with "map<, int32> attributes = 30;"
			character:           6,                                               // Right after "  map<" (at comma, since no space)
			expectedContains:    []string{"int32", "bool", "string"},             // Valid map key types
			expectedNotContains: []string{"User", "Status", "MapCompletionTest"}, // Enums and messages not allowed
		},
		{
			name:                "complete_map_value_type_incomplete_syntax",
			line:                34,                                    // Line with "map<int32, > attributes2 = 31;"
			character:           13,                                    // Right after "  map<int32, " (before >)
			expectedContains:    []string{"User", "Status", "string"},  // All types allowed for value
			expectedNotContains: []string{},                            // No restrictions on value types
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
			require.NotNil(t, completionList, "expected completion list to be non-nil")
			labels := xslices.Map(completionList.Items, func(item protocol.CompletionItem) string {
				return item.Label
			})
			for _, expected := range tt.expectedContains {
				assert.Contains(t, labels, expected, "expected completion list to contain %q", expected)
			}
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, labels, notExpected, "expected completion list to not contain %q", notExpected)
			}
		})
	}
}

func TestCompletionOptions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/completion/options_test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name             string
		line             uint32
		character        uint32
		expectedContains []string
	}{
		{
			name:      "complete_file_options_builtin",
			line:      6,
			character: 8, // After "option d"
			expectedContains: []string{
				"deprecated",
			},
		},
		{
			name:      "complete_file_options_custom",
			line:      7,
			character: 9, // After "option (c"
			expectedContains: []string{
				"(example.options.custom_file_option)",
			},
		},
		{
			name:      "complete_message_options_builtin",
			line:      11,
			character: 10, // After "option d"
			expectedContains: []string{
				"deprecated",
				"deprecated_legacy_json_field_conflicts",
			},
		},
		{
			name:      "complete_message_options_custom",
			line:      12,
			character: 10, // After "option ("
			expectedContains: []string{
				"(example.options.custom_message_option)",
			},
		},
		{
			name:      "complete_field_options_builtin",
			line:      14,
			character: 26, // After "[d"
			expectedContains: []string{
				"deprecated",
			},
		},
		{
			name:      "complete_field_options_custom",
			line:      15,
			character: 27, // After "[("
			expectedContains: []string{
				"(example.options.custom_field_option)",
			},
		},
		{
			name:      "complete_field_options_newline_builtin",
			line:      17,
			character: 5, // After "d"
			expectedContains: []string{
				"deprecated",
			},
		},
		{
			name:      "complete_field_options_newline_custom",
			line:      20,
			character: 5, // After "("
			expectedContains: []string{
				"(example.options.custom_field_option)",
			},
		},
		{
			name:      "complete_enum_options_builtin",
			line:      26,
			character: 10, // After "option a"
			expectedContains: []string{
				"allow_alias",
			},
		},
		{
			name:      "complete_enum_options_custom",
			line:      27,
			character: 10, // After "option ("
			expectedContains: []string{
				"(example.options.custom_enum_option)",
			},
		},
		{
			name:      "complete_enum_value_options_builtin",
			line:      29,
			character: 19, // After "[d"
			expectedContains: []string{
				"deprecated",
			},
		},
		{
			name:      "complete_enum_value_options_custom",
			line:      30,
			character: 20, // After "[("
			expectedContains: []string{
				"(example.options.custom_enum_value_option)",
			},
		},
		{
			name:      "complete_service_options_builtin",
			line:      37,
			character: 10, // After "option d"
			expectedContains: []string{
				"deprecated",
			},
		},
		{
			name:      "complete_service_options_custom",
			line:      38,
			character: 10, // After "option ("
			expectedContains: []string{
				"(example.options.custom_service_option)",
			},
		},
		{
			name:      "complete_method_options_builtin",
			line:      41,
			character: 12, // After "option i"
			expectedContains: []string{
				"idempotency_level",
			},
		},
		{
			name:      "complete_method_options_custom",
			line:      42,
			character: 12, // After "option ("
			expectedContains: []string{
				"(example.options.custom_method_option)",
			},
		},
		{
			name:      "complete_path_field_1",
			line:      48,
			character: 28, // After "(example.options.field)."
			expectedContains: []string{
				"recurse", "strings", "required", "number", "enum",
			},
		},
		{
			name:      "complete_path_field_2",
			line:      49,
			character: 36, // After "(example.options.field).recurse."
			expectedContains: []string{
				"recurse", "strings", "required", "number", "enum",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Request completions at the specified position in the static file
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
			require.NotNil(t, completionList, "expected completion list to be non-nil")

			// Extract labels from completion items
			labels := xslices.Map(completionList.Items, func(item protocol.CompletionItem) string {
				return item.Label
			})
			// Verify expected options are present
			for _, expected := range tt.expectedContains {
				assert.Contains(t, labels, expected, "expected completion list to contain %q", expected)
			}
		})
	}
}
