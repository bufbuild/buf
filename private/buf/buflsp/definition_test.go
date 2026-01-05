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

func TestDefinition(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/definition/definition.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name               string
		line               uint32
		character          uint32
		expectedDefLine    uint32
		expectedDefCharMin uint32
		expectedDefCharMax uint32
		expectNoDefinition bool
	}{
		{
			name:               "definition_of_account_type_reference",
			line:               10, // Line with "AccountType type = 2;"
			character:          2,  // On "AccountType" type
			expectedDefLine:    41, // enum AccountType definition
			expectedDefCharMin: 5,
			expectedDefCharMax: 16,
		},
		{
			name:               "definition_of_person_type_reference",
			line:               13, // Line with "Person owner = 3;"
			character:          2,  // On "Person" type
			expectedDefLine:    17, // message Person definition
			expectedDefCharMin: 8,
			expectedDefCharMax: 14,
		},
		{
			name:               "definition_of_address_type_reference",
			line:               25, // Line with "Address address = 3;"
			character:          2,  // On "Address" type
			expectedDefLine:    29, // message Address definition
			expectedDefCharMin: 8,
			expectedDefCharMax: 15,
		},
		{
			name:               "definition_of_country_code_reference",
			line:               37, // Line with "CountryCode country = 3;"
			character:          2,  // On "CountryCode" type
			expectedDefLine:    53, // enum CountryCode definition
			expectedDefCharMin: 5,
			expectedDefCharMax: 16,
		},
		{
			name:               "definition_of_rpc_request_type",
			line:               67, // Line with "rpc GetAccount(GetAccountRequest)"
			character:          18, // On "GetAccountRequest"
			expectedDefLine:    74, // message GetAccountRequest definition
			expectedDefCharMin: 8,
			expectedDefCharMax: 25,
		},
		{
			name:               "definition_of_rpc_response_type",
			line:               67, // Line with "returns (GetAccountResponse)"
			character:          45, // On "GetAccountResponse"
			expectedDefLine:    80, // message GetAccountResponse definition
			expectedDefCharMin: 8,
			expectedDefCharMax: 26,
		},
		{
			name:               "definition_of_account_field_in_request",
			line:               88, // Line with "Account account = 1;" in CreateAccountRequest
			character:          2,  // On "Account" type
			expectedDefLine:    5,  // message Account definition
			expectedDefCharMin: 8,
			expectedDefCharMax: 15,
		},
		{
			name:               "definition_of_field_name",
			line:               10, // Line with "AccountType type = 2;"
			character:          14, // On "type" field name
			expectedDefLine:    10,
			expectedDefCharMin: 14,
			expectedDefCharMax: 18,
		},
		{
			name:               "definition_of_service",
			line:               65, // Line with "service AccountService {"
			character:          8,  // On "AccountService"
			expectedDefLine:    65,
			expectedDefCharMin: 8,
			expectedDefCharMax: 22,
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
				assert.Equal(t, testURI, location.URI)
				assert.Equal(t, tt.expectedDefLine, location.Range.Start.Line)
				assert.GreaterOrEqual(t, location.Range.Start.Character, tt.expectedDefCharMin)
				assert.LessOrEqual(t, location.Range.Start.Character, tt.expectedDefCharMax)
			}
		})
	}
}
