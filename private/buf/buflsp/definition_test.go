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

	"github.com/bufbuild/buf/private/buf/buflsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestDefinition(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/definition/definition.proto")
	require.NoError(t, err)

	typesProtoPath, err := filepath.Abs("testdata/definition/types.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := buflsp.FilePathToURI(typesProtoPath)

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
			name:               "definition_of_account_type_reference",
			line:               12, // Line with "AccountType type = 2;"
			character:          2,  // On "AccountType" type
			expectedDefURI:     testURI,
			expectedDefLine:    52, // enum AccountType definition (line 53 in file)
			expectedDefCharMin: 5,
			expectedDefCharMax: 16,
		},
		{
			name:               "definition_of_person_type_reference",
			line:               15, // Line with "Person owner = 3;"
			character:          2,  // On "Person" type
			expectedDefURI:     testURI,
			expectedDefLine:    28, // message Person definition (line 29 in file)
			expectedDefCharMin: 8,
			expectedDefCharMax: 14,
		},
		{
			name:               "definition_of_address_type_reference",
			line:               36, // Line with "Address address = 3;"
			character:          2,  // On "Address" type
			expectedDefURI:     testURI,
			expectedDefLine:    40, // message Address definition (line 41 in file)
			expectedDefCharMin: 8,
			expectedDefCharMax: 15,
		},
		{
			name:               "definition_of_country_code_reference",
			line:               48, // Line with "CountryCode country = 3;"
			character:          2,  // On "CountryCode" type
			expectedDefURI:     testURI,
			expectedDefLine:    64, // enum CountryCode definition (line 65 in file)
			expectedDefCharMin: 5,
			expectedDefCharMax: 16,
		},
		{
			name:               "definition_of_rpc_request_type",
			line:               78, // Line with "rpc GetAccount(GetAccountRequest)"
			character:          18, // On "GetAccountRequest"
			expectedDefURI:     testURI,
			expectedDefLine:    85, // message GetAccountRequest definition (line 86 in file)
			expectedDefCharMin: 8,
			expectedDefCharMax: 25,
		},
		{
			name:               "definition_of_rpc_response_type",
			line:               78, // Line with "returns (GetAccountResponse)"
			character:          45, // On "GetAccountResponse"
			expectedDefURI:     testURI,
			expectedDefLine:    91, // message GetAccountResponse definition (line 92 in file)
			expectedDefCharMin: 8,
			expectedDefCharMax: 26,
		},
		{
			name:               "definition_of_account_field_in_request",
			line:               99, // Line with "Account account = 1;" in CreateAccountRequest
			character:          2,  // On "Account" type
			expectedDefURI:     testURI,
			expectedDefLine:    7, // message Account definition
			expectedDefCharMin: 8,
			expectedDefCharMax: 15,
		},
		{
			name:               "definition_of_field_name",
			line:               12, // Line with "AccountType type = 2;"
			character:          14, // On "type" field name
			expectedDefURI:     testURI,
			expectedDefLine:    12,
			expectedDefCharMin: 14,
			expectedDefCharMax: 18,
		},
		{
			name:               "definition_of_service",
			line:               76, // Line with "service AccountService {"
			character:          8,  // On "AccountService"
			expectedDefURI:     testURI,
			expectedDefLine:    76, // service AccountService (line 77 in file)
			expectedDefCharMin: 8,
			expectedDefCharMax: 22,
		},
		{
			name:               "definition_of_status_imported_type",
			line:               18, // Line with "Status status = 4;"
			character:          2,  // On "Status" type
			expectedDefURI:     typesURI,
			expectedDefLine:    5, // enum Status definition in types.proto
			expectedDefCharMin: 5,
			expectedDefCharMax: 11,
		},
		{
			name:               "definition_of_timestamp_imported_type",
			line:               21, // Line with "Timestamp created_at = 5;"
			character:          2,  // On "Timestamp" type
			expectedDefURI:     typesURI,
			expectedDefLine:    17, // message Timestamp definition in types.proto
			expectedDefCharMin: 8,
			expectedDefCharMax: 17,
		},
		{
			name:               "definition_of_syntax_keyword",
			line:               0, // Line with "syntax = "proto3";"
			character:          0, // On "syntax"
			expectNoDefinition: true,
		},
		{
			name:               "definition_of_package_keyword",
			line:               2, // Line with "package definition.v1;"
			character:          0, // On "package"
			expectNoDefinition: true,
		},
		{
			name:               "definition_of_package_name",
			line:               2, // Line with "package definition.v1;"
			character:          8, // On "definition"
			expectNoDefinition: true,
		},
		{
			name:               "definition_of_map_value_type",
			line:               24, // Line with "map<string, Person> metadata = 6;"
			character:          16, // On "Person" in map value type
			expectedDefURI:     testURI,
			expectedDefLine:    28, // message Person definition
			expectedDefCharMin: 8,
			expectedDefCharMax: 14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var locations []protocol.Location
			_, defErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
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

// TestDefinitionURLEncoding verifies that file paths with special characters
// like '@' are properly URL-encoded in the URI responses.
func TestDefinitionURLEncoding(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Use a file from a directory with '@' in the path
	testProtoPath, err := filepath.Abs("testdata/uri@encode/test.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// Note: The client may send URIs with unencoded @ symbols, but the LSP
	// server normalizes them internally to ensure consistency

	// Test definition lookup for a type reference within the same file
	var locations []protocol.Location
	_, defErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentDefinition, protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: testURI,
			},
			Position: protocol.Position{
				Line:      13, // Line with "Status status = 3;" (0-indexed, line 14 in file)
				Character: 2,  // On "Status" type
			},
		},
	}, &locations)
	require.NoError(t, defErr)

	require.Len(t, locations, 1, "expected exactly one definition location")
	location := locations[0]

	expectedURI := buflsp.FilePathToURI(testProtoPath)

	assert.Equal(t, expectedURI, location.URI, "returned URI should have @ encoded as %40")

	// Verify it points to the correct location in the file
	assert.Equal(t, uint32(17), location.Range.Start.Line, "should point to Status enum definition (0-indexed, line 18 in file)")
}
