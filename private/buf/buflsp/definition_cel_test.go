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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestCELDefinition(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/definition/cel_definition.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Predefined rules extend buf.validate.*Rules, which proto3 only allows for
	// options, so they live in a separate editions file.
	predefinedProtoPath, err := filepath.Abs("testdata/definition/cel_definition_predefined.proto")
	require.NoError(t, err)
	predefinedURI := uri.File(predefinedProtoPath)
	openFileFromDisk(ctx, t, clientJSONConn, predefinedURI)
	validateProtoPath, err := filepath.Abs("testdata/definition/vendor/protovalidate/buf/validate/validate.proto")
	require.NoError(t, err)
	validateURI := uri.File(validateProtoPath)

	tests := []struct {
		name               string
		predefined         bool // use cel_definition_predefined.proto
		line               uint32
		character          uint32
		expectedDefURI     uri.URI
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
		{
			// predefined: `rule` in `expression: "rules.min_len <= uint(rule)"` (line 11)
			// refers to the max_words extension, defined on line 9.
			name:               "predefined_rule_navigates_to_extension",
			predefined:         true,
			line:               11,
			character:          39, // start of "rule" in `uint(rule)`
			expectedDefURI:     predefinedURI,
			expectedDefLine:    9, // int32 max_words = 80048954 [...]
			expectedDefCharMin: 8, // "max_words" starts at char 8
			expectedDefCharMax: 17,
		},
		{
			// predefined: `rules` refers to the extended buf.validate.StringRules.
			name:               "predefined_rules_navigates_to_extended_message",
			predefined:         true,
			line:               11,
			character:          17, // start of "rules"
			expectedDefURI:     validateURI,
			expectedDefLine:    3111, // message StringRules {
			expectedDefCharMin: 8,
			expectedDefCharMax: 19,
		},
		{
			// predefined: `min_len` in `rules.min_len` refers to StringRules.min_len.
			name:               "predefined_rules_field_access_navigates_to_field",
			predefined:         true,
			line:               11,
			character:          23, // start of "min_len"
			expectedDefURI:     validateURI,
			expectedDefLine:    3153, // optional uint64 min_len = 2 [...]
			expectedDefCharMin: 18,
			expectedDefCharMax: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			docURI := testURI
			if tt.predefined {
				docURI = predefinedURI
			}
			var locations []protocol.Location
			_, defErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: docURI,
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
