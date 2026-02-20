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

func TestTypeDefinition(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testProtoPath, err := filepath.Abs("testdata/type_definition/type_definition.proto")
	require.NoError(t, err)

	typesProtoPath, err := filepath.Abs("testdata/type_definition/types.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
	typesURI := buflsp.FilePathToURI(typesProtoPath)

	tests := []struct {
		name                string
		line                uint32
		character           uint32
		expectedTypeDefURI  protocol.URI
		expectedTypeDefLine uint32
		expectedTypeDefChar uint32
	}{
		{
			name:                "type_definition_of_category_field",
			line:                12, // Line with "Category category = 2;"
			character:           2,  // On "Category" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 31, // message Category definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_vendor_field",
			line:                15, // Line with "Vendor vendor = 3;"
			character:           2,  // On "Vendor" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 40, // message Vendor definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_tags_field",
			line:                18, // Line with "repeated Tag tags = 4;"
			character:           13, // On "Tag" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 49, // message Tag definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_metadata_field",
			line:                21, // Line with "map<string, Metadata> metadata = 5;"
			character:           16, // On "Metadata" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 58, // message Metadata definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_parent_field",
			line:                36, // Line with "Category parent = 2;"
			character:           2,  // On "Category" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 31, // message Category definition (recursive reference)
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_type_field",
			line:                45, // Line with "VendorType type = 2;"
			character:           2,  // On "VendorType" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 64, // enum VendorType definition
			expectedTypeDefChar: 5,
		},
		{
			name:                "type_definition_of_product_field",
			line:                81, // Line with "Product product = 2;" in Order message
			character:           2,  // On "Product" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 7, // message Product definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_status_field",
			line:                84, // Line with "OrderStatus status = 3;"
			character:           2,  // On "OrderStatus" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 109, // enum OrderStatus definition
			expectedTypeDefChar: 5,
		},
		{
			name:                "type_definition_of_customer_field",
			line:                87, // Line with "Customer customer = 4;"
			character:           2,  // On "Customer" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 91, // message Customer definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_contact_field",
			line:                96, // Line with "Contact contact = 2;"
			character:           2,  // On "Contact" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 100, // message Contact definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_product_in_request",
			line:                144, // Line with "Product product = 1;" in CreateProductRequest
			character:           2,   // On "Product" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 7, // message Product definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_category_in_request",
			line:                147, // Line with "Category category = 2;" in CreateProductRequest
			character:           2,   // On "Category" type name
			expectedTypeDefURI:  testURI,
			expectedTypeDefLine: 31, // message Category definition
			expectedTypeDefChar: 8,
		},
		{
			name:                "type_definition_of_priority_imported_type",
			line:                24, // Line with "Priority priority = 6;"
			character:           2,  // On "Priority" type name
			expectedTypeDefURI:  typesURI,
			expectedTypeDefLine: 5, // enum Priority definition in types.proto
			expectedTypeDefChar: 5,
		},
		{
			name:                "type_definition_of_audit_imported_type",
			line:                27, // Line with "Audit audit = 7;"
			character:           2,  // On "Audit" type name
			expectedTypeDefURI:  typesURI,
			expectedTypeDefLine: 17, // message Audit definition in types.proto
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
			assert.Equal(t, tt.expectedTypeDefURI, location.URI)
			assert.Equal(t, tt.expectedTypeDefLine, location.Range.Start.Line)
			assert.Equal(t, tt.expectedTypeDefChar, location.Range.Start.Character)
		})
	}
}
