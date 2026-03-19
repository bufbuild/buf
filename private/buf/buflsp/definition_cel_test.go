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

package buflsp_test

import (
	"path/filepath"
	"testing"

	protocol "github.com/bufbuild/buf/private/pkg/lspprotocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCELDefinition(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/definition/cel_definition.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name               string
		line               uint32
		character          uint32
		expectedDefURI     protocol.URI
		expectedDefLine    uint32
		expectedDefCharMin uint32
		expectedDefCharMax uint32
		expectNoDefinition bool
	}{
		{
			// message-level: `this` in `expression: "this.name.size() > 0"` (line 10, 0-indexed)
			// refers to TestMessage, defined on line 7 (0-indexed).
			name:               "message_level_this_navigates_to_message",
			line:               10,
			character:          17, // start of "this" in `    expression: "this..."`
			expectedDefURI:     testURI,
			expectedDefLine:    7, // message TestMessage {
			expectedDefCharMin: 8, // "TestMessage" starts at char 8
			expectedDefCharMax: 19,
		},
		{
			// field-level: `this` in `expression: "this.contains('@')"` (line 20, 0-indexed)
			// refers to the email field, defined on line 18 (0-indexed).
			name:               "field_level_this_navigates_to_field",
			line:               20,
			character:          17, // start of "this" in `    expression: "this..."`
			expectedDefURI:     testURI,
			expectedDefLine:    18, // string email = 1 [...]
			expectedDefCharMin: 9,  // "email" starts at char 9
			expectedDefCharMax: 14,
		},
		{
			// `name` in `expression: "this.name.size() > 0"` (line 10, 0-indexed)
			// refers to the name field, defined on line 13 (0-indexed).
			name:               "field_access_navigates_to_field_declaration",
			line:               10,
			character:          22, // start of "name" in `    expression: "this.name..."`
			expectedDefURI:     testURI,
			expectedDefLine:    13, // string name = 1;
			expectedDefCharMin: 9,  // "name" starts at char 9
			expectedDefCharMax: 13,
		},
		{
			// oneof field-level: `this` in `expression: "this.contains('@')"` (line 31, 0-indexed)
			// refers to the email field inside the oneof, defined on line 28 (0-indexed).
			name:               "oneof_field_level_this_navigates_to_oneof_field",
			line:               30,
			character:          19, // start of "this" in `      expression: "this..."`
			expectedDefURI:     testURI,
			expectedDefLine:    28, // string email = 1 [(buf.validate.field).cel = {
			expectedDefCharMin: 11, // "email" starts at char 11 (inside oneof, 4-space indent)
			expectedDefCharMax: 16,
		},
		{
			// oneof message-level: `this` in `expression: "this.name.size() > 0"` (line 41, 0-indexed)
			// refers to TestOneofMessage, defined on line 37 (0-indexed).
			name:               "oneof_message_level_this_navigates_to_message",
			line:               40,
			character:          17, // start of "this" in `    expression: "this..."`
			expectedDefURI:     testURI,
			expectedDefLine:    37, // message TestOneofMessage {
			expectedDefCharMin: 8,  // "TestOneofMessage" starts at char 8
			expectedDefCharMax: 24,
		},
		{
			// oneof message-level: `name` in `expression: "this.name.size() > 0"` (line 41, 0-indexed)
			// refers to the name field inside the oneof, defined on line 44 (0-indexed).
			name:               "oneof_field_access_navigates_to_oneof_field",
			line:               40,
			character:          22, // start of "name" in `    expression: "this.name..."`
			expectedDefURI:     testURI,
			expectedDefLine:    44, // string name = 1;
			expectedDefCharMin: 11, // "name" starts at char 11 (inside oneof, 4-space indent)
			expectedDefCharMax: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var locations []protocol.Location
			defErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: testURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
			}, &locations)
			require.NoError(t, defErr)

			if tt.expectNoDefinition {
				assert.Empty(t, locations, "expected no definition locations")
			} else {
				require.Len(t, locations, 1, "expected exactly one definition location")
				location := locations[0]
				assert.Equal(t, tt.expectedDefURI, location.URI)
				assert.Equal(t, tt.expectedDefLine, location.Range.Start.Line)
				assert.GreaterOrEqual(t, location.Range.Start.Character, tt.expectedDefCharMin)
				assert.LessOrEqual(t, location.Range.Start.Character, tt.expectedDefCharMax)
			}
		})
	}
}
