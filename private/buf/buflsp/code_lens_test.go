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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCodeLens(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/code_lens/code_lens.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Request code lenses for the file
	var lenses []protocol.CodeLens
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentCodeLens, protocol.CodeLensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &lenses)
	require.NoError(t, err)

	// We should have code lenses for referenceable symbols (excluding fields and oneofs)
	// Expected: Status, Priority, Unused (enums), User, Task, Project, UnusedMessage,
	// GetUserRequest, CreateUserRequest, Response, Contact (messages), UserService (service)
	// That's 12 top-level symbols (enums, messages, services) - fields and oneofs are excluded
	require.Greater(t, len(lenses), 0, "should have at least one code lens")

	t.Run("positive_cases", func(t *testing.T) {
		// Test specific code lens entries that should exist
		tests := []struct {
			name         string
			line         uint32
			expectedText string // After resolve
		}{
			{
				name:         "status_enum_multiple_refs",
				line:         5, // enum Status { (line 6 in editor, line 5 in LSP 0-based)
				expectedText: "2 references",
			},
			{
				name:         "priority_enum_one_ref",
				line:         12, // enum Priority { (line 13 in editor, line 12 in LSP)
				expectedText: "1 reference",
			},
			{
				name:         "unused_enum_zero_refs",
				line:         19, // enum Unused { (line 20 in editor, line 19 in LSP)
				expectedText: "0 references",
			},
			{
				name:         "user_message_multiple_refs",
				line:         25,             // message User { (line 26 in editor, line 25 in LSP)
				expectedText: "3 references", // Task.assigned_to, GetUser return, CreateUser return
			},
			{
				name:         "task_message_multiple_refs",
				line:         31,            // message Task { (line 32 in editor, line 31 in LSP)
				expectedText: "1 reference", // Project.tasks field
			},
			{
				name:         "project_message_one_ref",
				line:         38, // message Project { (line 39 in editor, line 38 in LSP)
				expectedText: "1 reference",
			},
			{
				name:         "unused_message_zero_refs",
				line:         44, // message UnusedMessage { (line 45 in editor, line 44 in LSP)
				expectedText: "0 references",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// Find the code lens at the expected line
				idx := slices.IndexFunc(lenses, func(lens protocol.CodeLens) bool {
					return lens.Range.Start.Line == tt.line
				})
				assert.NotEqual(t, -1, idx, "expected code lens at line %d not found", tt.line)

				if idx == -1 {
					return // Skip resolve test if lens not found
				}

				// Resolve the code lens to get the command
				var resolvedLens protocol.CodeLens
				_, err := clientJSONConn.Call(ctx, protocol.MethodCodeLensResolve, lenses[idx], &resolvedLens)
				require.NoError(t, err)

				// Verify the command title matches expected text
				require.NotNil(t, resolvedLens.Command, "resolved code lens should have a command")
				assert.Equal(t, tt.expectedText, resolvedLens.Command.Title)

				// Code lens is informational-only (not clickable)
				assert.Empty(t, resolvedLens.Command.Command, "informational code lens should have no command")
				assert.Empty(t, resolvedLens.Command.Arguments, "informational code lens should have no arguments")
			})
		}
	})

	t.Run("negative_cases", func(t *testing.T) {
		// Test that fields and oneofs do NOT have code lenses
		tests := []struct {
			name string
			line uint32
			desc string
		}{
			{
				name: "no_code_lens_on_field",
				line: 26, // User.name field (line 27 in editor, line 26 in LSP)
				desc: "field should not have code lens",
			},
			{
				name: "no_code_lens_on_oneof",
				line: 73, // oneof contact_method (line 74 in editor, line 73 in LSP)
				desc: "oneof should not have code lens",
			},
			{
				name: "no_code_lens_on_oneof_field",
				line: 74, // email field inside oneof (line 75 in editor, line 74 in LSP)
				desc: "field inside oneof should not have code lens",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// Check that no code lens exists at this line
				idx := slices.IndexFunc(lenses, func(lens protocol.CodeLens) bool {
					return lens.Range.Start.Line == tt.line
				})
				assert.Equal(t, -1, idx, "%s: found code lens at line %d but expected none", tt.desc, tt.line)
			})
		}
	})
}
