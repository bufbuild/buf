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

func TestTypeDefinition(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/type_definition/type_definition.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	tests := []struct {
		name                string
		line                uint32
		character           uint32
		expectedTypeDefLine uint32
		expectedTypeDefChar uint32
	}{
		{
			name:                "type_definition_of_category_field",
			line:                10, // Line with "Category category = 2;"
			character:           2,  // On "Category" type name
			expectedTypeDefLine: 23, // message Category definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_vendor_field",
			line:                13, // Line with "Vendor vendor = 3;"
			character:           2,  // On "Vendor" type name
			expectedTypeDefLine: 32, // message Vendor definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_tags_field",
			line:                16, // Line with "repeated Tag tags = 4;"
			character:           13, // On "Tag" type name
			expectedTypeDefLine: 41, // message Tag definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_metadata_field",
			line:                19, // Line with "map<string, Metadata> metadata = 5;"
			character:           16, // On "Metadata" type name
			expectedTypeDefLine: 50, // message Metadata definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_parent_field",
			line:                28, // Line with "Category parent = 2;"
			character:           2,  // On "Category" type name
			expectedTypeDefLine: 23, // message Category definition (recursive reference)
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_type_field",
			line:                37, // Line with "VendorType type = 2;"
			character:           2,  // On "VendorType" type name
			expectedTypeDefLine: 56, // enum VendorType definition
			expectedTypeDefChar: 5,
		},
		{
			name:                "type_definition_of_product_field",
			line:                73, // Line with "Product product = 2;" in Order message
			character:           2,  // On "Product" type name
			expectedTypeDefLine: 5,  // message Product definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_status_field",
			line:                76,  // Line with "OrderStatus status = 3;"
			character:           2,   // On "OrderStatus" type name
			expectedTypeDefLine: 101, // enum OrderStatus definition
			expectedTypeDefChar: 5,
		},
		{
			name:                "type_definition_of_customer_field",
			line:                79, // Line with "Customer customer = 4;"
			character:           2,  // On "Customer" type name
			expectedTypeDefLine: 83, // message Customer definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_contact_field",
			line:                88, // Line with "Contact contact = 2;"
			character:           2,  // On "Contact" type name
			expectedTypeDefLine: 92, // message Contact definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_product_in_request",
			line:                136, // Line with "Product product = 1;" in CreateProductRequest
			character:           2,   // On "Product" type name
			expectedTypeDefLine: 5,   // message Product definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_category_in_request",
			line:                139, // Line with "Category category = 2;" in CreateProductRequest
			character:           2,   // On "Category" type name
			expectedTypeDefLine: 23,  // message Category definition
			expectedTypeDefChar: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var locations []protocol.Location
			_, typeDefErr := clientJSONConn.Call(ctx, protocol.MethodTextDocumentTypeDefinition, protocol.TypeDefinitionParams{
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
			require.NoError(t, typeDefErr)

			require.Len(t, locations, 1, "expected exactly one type definition location")
			location := locations[0]
			assert.Equal(t, testURI, location.URI)
			assert.Equal(t, tt.expectedTypeDefLine, location.Range.Start.Line)
			assert.Equal(t, tt.expectedTypeDefChar, location.Range.Start.Character)
		})
	}
}
