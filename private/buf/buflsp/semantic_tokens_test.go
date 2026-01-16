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

// TestSemanticTokensKeywords tests that keywords, modifiers, and other token types
// are correctly identified and positioned in the semantic tokens output.
func TestSemanticTokensKeywords(t *testing.T) {
	t.Parallel()

	// expectedToken represents an expected semantic token at a specific position.
	type expectedToken struct {
		line      uint32
		startChar uint32
		length    uint32
		tokenType uint32
		desc      string // Description for test failure messages
	}

	// notExpectedToken represents a token that should NOT be present at a specific position.
	type notExpectedToken struct {
		line      uint32
		startChar uint32
		length    uint32
		tokenType uint32
		desc      string // Description for test failure messages
	}

	testCases := []struct {
		name              string
		file              string
		expectedTokens    []expectedToken
		notExpectedTokens []notExpectedToken
	}{
		{
			name: "proto2",
			file: "testdata/semantic_tokens/proto2.proto",
			expectedTokens: []expectedToken{
				// syntax declaration
				{0, 0, 6, semanticTypeKeyword, "'syntax' keyword"},
				{0, 9, 8, semanticTypeString, "'\"proto2\"' string"},
				// package declaration
				{2, 0, 7, semanticTypeKeyword, "'package' keyword"},
				{2, 8, 7, semanticTypeNamespace, "'test.v1' namespace"},
				// message declaration
				{4, 0, 7, semanticTypeKeyword, "'message' keyword"},
				{4, 8, 6, semanticTypeStruct, "'Person' struct"},
				// required modifier
				{5, 2, 8, semanticTypeModifier, "'required' modifier"},
				{5, 11, 6, semanticTypeType, "'string' as type"},
				{5, 18, 4, semanticTypeProperty, "'name' as property"},
				{5, 25, 1, semanticTypeNumber, "'1' field tag"},
				// optional modifier + scalar types
				{6, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{6, 11, 5, semanticTypeType, "'int32' as type"},
				{6, 17, 3, semanticTypeProperty, "'age' as property"},
				{6, 23, 1, semanticTypeNumber, "'2' field tag"},
				{7, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{7, 11, 6, semanticTypeType, "'double' as type"},
				{7, 18, 6, semanticTypeProperty, "'height' as property"},
				{8, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{8, 11, 5, semanticTypeType, "'float' as type"},
				{8, 17, 6, semanticTypeProperty, "'weight' as property"},
				{9, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{9, 11, 4, semanticTypeType, "'bool' as type"},
				{9, 16, 6, semanticTypeProperty, "'active' as property"},
				{10, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{10, 11, 5, semanticTypeType, "'bytes' as type"},
				{10, 17, 4, semanticTypeProperty, "'data' as property"},
				{11, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{11, 11, 5, semanticTypeType, "'int64' as type"},
				{11, 17, 10, semanticTypeProperty, "'big_number' as property"},
				{12, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{12, 11, 6, semanticTypeType, "'uint32' as type"},
				{12, 18, 7, semanticTypeProperty, "'counter' as property"},
			},
		},
		{
			name: "edition",
			file: "testdata/semantic_tokens/edition.proto",
			expectedTokens: []expectedToken{
				// edition declaration
				{0, 0, 7, semanticTypeKeyword, "'edition' keyword"},
				{0, 10, 6, semanticTypeString, "'\"2023\"' string"},
				// package declaration
				{2, 0, 7, semanticTypeKeyword, "'package' keyword"},
				{2, 8, 7, semanticTypeNamespace, "'test.v1' namespace"},
				// message declaration
				{4, 0, 7, semanticTypeKeyword, "'message' keyword"},
				{4, 8, 7, semanticTypeStruct, "'Product' struct"},
				// string fields
				{5, 2, 6, semanticTypeType, "'string' as type"},
				{5, 9, 2, semanticTypeProperty, "'id' as property"},
				{6, 2, 6, semanticTypeType, "'string' as type"},
				{6, 9, 4, semanticTypeProperty, "'name' as property"},
			},
		},
		{
			name: "comprehensive",
			file: "testdata/semantic_tokens/comprehensive.proto",
			expectedTokens: []expectedToken{
				// comment
				{2, 0, 20, semanticTypeComment, "comment on line 3"},
				// import declarations (only resolved imports are tokenized)
				{5, 0, 6, semanticTypeKeyword, "'import' keyword"},
				{5, 7, 6, semanticTypeModifier, "'public' modifier"},
				{5, 14, 34, semanticTypeString, "import path string"},
				// Note: weak import on line 6 is not tokenized because the file doesn't exist
				// option keyword
				{10, 2, 6, semanticTypeKeyword, "'option' keyword"},
				// decorator (option)
				{10, 9, 10, semanticTypeDecorator, "'deprecated' as decorator"},
				// built-in type
				{13, 2, 6, semanticTypeType, "'string' as type"},
				// field name
				{13, 9, 4, semanticTypeProperty, "'name' as property"},
				{13, 16, 1, semanticTypeNumber, "'1' field tag"},
				{14, 2, 5, semanticTypeType, "'int32' as type"},
				{14, 8, 3, semanticTypeProperty, "'age' as property"},
				// optional field
				{16, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{16, 11, 6, semanticTypeType, "'string' as type"},
				{16, 18, 8, semanticTypeProperty, "'nickname' as property"},
				{16, 29, 1, semanticTypeNumber, "'5' field tag"},
				// custom type
				{19, 2, 4, semanticTypeEnum, "'Role' as enum"},
				{19, 7, 4, semanticTypeProperty, "'role' as property"},
				// repeated modifier
				{22, 2, 8, semanticTypeModifier, "'repeated' modifier"},
				{22, 11, 6, semanticTypeType, "'string' as type"},
				{22, 18, 4, semanticTypeProperty, "'tags' as property"},
				// map field (line 26 in file = line 25 in 0-indexed)
				{25, 2, 18, semanticTypeProperty, "'map<string, int32>' map type"},
				{25, 6, 6, semanticTypeType, "'string' scalar"},
				{25, 14, 5, semanticTypeType, "'int32' scalar"},
				{25, 21, 10, semanticTypeProperty, "'attributes' property"},
				{25, 34, 2, semanticTypeNumber, "'22' field tag"},
				// reserved keyword (single field)
				{28, 2, 8, semanticTypeKeyword, "'reserved' keyword"},
				// reserved keyword with range
				{29, 2, 8, semanticTypeKeyword, "'reserved' keyword"},
				{29, 14, 2, semanticTypeKeyword, "'to' keyword"},
				// reserved keyword with string
				{30, 2, 8, semanticTypeKeyword, "'reserved' keyword"},
				// oneof keyword
				{33, 2, 5, semanticTypeKeyword, "'oneof' keyword"},
				{33, 8, 7, semanticTypeProperty, "'contact' oneof name"},
				// enum declaration
				{40, 0, 4, semanticTypeKeyword, "'enum' keyword"},
				{40, 5, 4, semanticTypeEnum, "'Role' enum"},
				// enum member
				{41, 2, 16, semanticTypeEnumMember, "'ROLE_UNSPECIFIED' as enumMember"},
				{41, 21, 1, semanticTypeNumber, "'0' enum value"},
				// message type references
				{58, 2, 4, semanticTypeStruct, "'User' message type reference"},
				{58, 7, 4, semanticTypeProperty, "'user' property"},
				{66, 2, 8, semanticTypeModifier, "'repeated' modifier"},
				{66, 11, 4, semanticTypeStruct, "'User' message type reference (repeated)"},
				{66, 16, 5, semanticTypeProperty, "'users' property"},
				// service declaration
				{46, 0, 7, semanticTypeKeyword, "'service' keyword"},
				{46, 8, 11, semanticTypeInterface, "'UserService' interface"},
				// rpc method with stream
				{48, 2, 3, semanticTypeKeyword, "'rpc' keyword"},
				{48, 6, 7, semanticTypeMethod, "'GetUser' method"},
				{48, 14, 6, semanticTypeModifier, "'stream' modifier (parameter)"},
				{48, 21, 14, semanticTypeStruct, "'GetUserRequest' type"},
				{48, 37, 7, semanticTypeKeyword, "'returns' keyword"},
				{48, 46, 6, semanticTypeModifier, "'stream' modifier (return)"},
				{48, 53, 15, semanticTypeStruct, "'GetUserResponse' type"},
				// rpc method without stream
				{50, 2, 3, semanticTypeKeyword, "'rpc' keyword"},
				{50, 6, 9, semanticTypeMethod, "'ListUsers' method"},
				{50, 16, 16, semanticTypeStruct, "'ListUsersRequest' type"},
				{50, 34, 7, semanticTypeKeyword, "'returns' keyword"},
				{50, 43, 17, semanticTypeStruct, "'ListUsersResponse' type"},
			},
			notExpectedTokens: []notExpectedToken{
				// = and ; should not be keywords
				{0, 7, 1, semanticTypeKeyword, "'=' should not be keyword"},
				{0, 17, 1, semanticTypeKeyword, "';' should not be keyword"},
				{10, 20, 1, semanticTypeKeyword, "'=' should not be keyword"},
				{10, 26, 1, semanticTypeKeyword, "';' should not be keyword"},
				{13, 14, 1, semanticTypeKeyword, "'=' should not be keyword"},
				{13, 17, 1, semanticTypeKeyword, "';' should not be keyword"},
				// Invalid import should not have string token (only resolved imports get string tokens)
				{6, 12, 13, semanticTypeString, "invalid import path should not have string token"},
			},
		},
		{
			name: "cel_expressions",
			file: "testdata/semantic_tokens/cel.proto",
			expectedTokens: []expectedToken{
				// Line 10: expression: "this.startsWith('foo') && this.size() > 5"
				// "this" keyword
				{10, 17, 4, semanticTypeKeyword, "'this' keyword in CEL"},
				// "startsWith" method
				{10, 22, 10, semanticTypeMethod, "'startsWith' method in CEL"},
				// 'foo' string literal
				{10, 33, 5, semanticTypeString, "'foo' string in CEL"},
				// "&&" operator
				{10, 40, 2, semanticTypeOperator, "'&&' operator in CEL"},
				// "size" method
				{10, 48, 4, semanticTypeMethod, "'size' method in CEL"},
				// ">" operator
				{10, 55, 1, semanticTypeOperator, "'>' operator in CEL"},
				// "5" number
				{10, 57, 1, semanticTypeNumber, "'5' number in CEL"},

				// Line 16: expression: "this > 0"
				// "this" keyword
				{16, 17, 4, semanticTypeKeyword, "'this' keyword in CEL"},
				// ">" operator
				{16, 22, 1, semanticTypeOperator, "'>' operator in CEL"},
				// "0" number
				{16, 24, 1, semanticTypeNumber, "'0' number in CEL"},

				// Line 21: cel_expression (array form): "'@' in this"
				// '@' string literal
				{20, 59, 3, semanticTypeString, "'@' string in CEL"},
				// "in" operator
				{20, 63, 2, semanticTypeOperator, "'in' operator in CEL"},
				// "this" keyword
				{20, 66, 4, semanticTypeKeyword, "'this' keyword in CEL"},

				// Line 29: message-level .cel expression: "this.value > 0"
				// "this" keyword
				{28, 17, 4, semanticTypeKeyword, "'this' keyword in message CEL"},
				// "value" property
				{28, 22, 5, semanticTypeProperty, "'value' property in message CEL"},
				// ">" operator
				{28, 28, 1, semanticTypeOperator, "'>' operator in message CEL"},
				// "0" number
				{28, 30, 1, semanticTypeNumber, "'0' number in message CEL"},

				// Line 37: message-level cel_expression: "this.count < 100"
				// "this" keyword
				{36, 50, 4, semanticTypeKeyword, "'this' keyword in message cel_expression"},
				// "count" property
				{36, 55, 5, semanticTypeProperty, "'count' property in message cel_expression"},
				// "<" operator
				{36, 61, 1, semanticTypeOperator, "'<' operator in message cel_expression"},
				// "100" number
				{36, 63, 3, semanticTypeNumber, "'100' number in message cel_expression"},
			},
		},
		{
			name: "cel_advanced",
			file: "testdata/semantic_tokens/cel_advanced.proto",
			expectedTokens: []expectedToken{
				// Line 16 (file) = line 15 (0-indexed): expression: "'@' in this"
				// Test 'in' operator
				{15, 21, 2, semanticTypeOperator, "'in' operator"},
				{15, 24, 4, semanticTypeKeyword, "'this' keyword (with in operator)"},

				// Line 22 (file) = line 21 (0-indexed): expression: "int(this) > 0 && string(int(this)) == this"
				// Test type conversion functions (built-in type functions)
				{21, 17, 3, semanticTypeType, "'int' type conversion function"},
				{21, 21, 4, semanticTypeKeyword, "'this' keyword (in int call)"},

				// Line 40 (file) = line 39 (0-indexed): expression: "has(this.street) && has(this.city)"
				// Test has() macro highlighting
				{39, 17, 3, semanticTypeMacro, "'has' macro"},
				{39, 37, 3, semanticTypeMacro, "'has' macro (second occurrence)"},
				{39, 21, 4, semanticTypeKeyword, "'this' keyword (in has call)"},
				{39, 41, 4, semanticTypeKeyword, "'this' keyword (in second has call)"},

				// Line 46 (file) = line 45 (0-indexed): expression: "this.all(tag, tag.size() > 0) && this.exists(tag, tag == 'valid')"
				// Test all() and exists() macro highlighting + comprehension variables
				{45, 17, 4, semanticTypeKeyword, "'this' keyword (before all)"},
				{45, 22, 3, semanticTypeMacro, "'all' macro"},
				{45, 31, 3, semanticTypeVariable, "'tag' comprehension variable (2nd use)"},
				{45, 50, 4, semanticTypeKeyword, "'this' keyword (before exists)"},
				{45, 55, 6, semanticTypeMacro, "'exists' macro"},
				// Note: Other 'tag' uses in exists are at different positions due to expansion

				// Line 52 (file) = line 51 (0-indexed): expression: "this > 0 ? true : false"
				{51, 17, 4, semanticTypeKeyword, "'this' keyword"},
				{51, 22, 1, semanticTypeOperator, "'>' operator"},
				{51, 24, 1, semanticTypeNumber, "'0' number"},
				{51, 26, 1, semanticTypeOperator, "'?' ternary operator"},
				{51, 28, 4, semanticTypeKeyword, "'true' keyword"},
				{51, 35, 5, semanticTypeKeyword, "'false' keyword"},

				// Line 58 (file) = line 57 (0-indexed): expression: "this.map(n, n * 2).size() > 0"
				// Test map() macro highlighting + comprehension variables
				{57, 17, 4, semanticTypeKeyword, "'this' keyword (before map)"},
				{57, 22, 3, semanticTypeMacro, "'map' macro"},
				{57, 29, 1, semanticTypeVariable, "'n' comprehension variable (2nd use)"},

				// Line 64 (file) = line 63 (0-indexed): expression: "this.filter(i, i.startsWith('test')).exists_one(i, i == 'test1')"
				// Test filter() and exists_one() macro highlighting + comprehension variables
				{63, 17, 4, semanticTypeKeyword, "'this' keyword (before filter)"},
				{63, 22, 6, semanticTypeMacro, "'filter' macro"},
				{63, 32, 1, semanticTypeVariable, "'i' comprehension variable (2nd use in filter)"},
				{63, 54, 10, semanticTypeMacro, "'exists_one' macro"},
				// Note: Other 'i' uses in exists_one are at different positions due to expansion

				// Multi-line expression test 1 (lines 72-75)
				// Line 72: "this.point_a == this.point_b ? 'point A and point B cannot be the same'"
				{72, 7, 4, semanticTypeKeyword, "'this' keyword in multi-line expression"},
				{72, 12, 7, semanticTypeProperty, "'point_a' property in multi-line expression"},
				{72, 20, 2, semanticTypeOperator, "'==' operator in multi-line expression"},
				{72, 36, 1, semanticTypeOperator, "'?' ternary operator in multi-line expression"},

				// Multi-line expression test 2 (lines 81-82)
				// Line 81: "(this.point_a.y - this.point_b.y) * (this.point_a.x - this.point_c.x)"
				{81, 8, 4, semanticTypeKeyword, "'this' keyword in multi-line arithmetic"},
				{81, 13, 7, semanticTypeProperty, "'point_a' property in multi-line arithmetic"},
				// Line 82: "!= (this.point_a.y - this.point_c.y) * (this.point_a.x - this.point_b.x)"
				{82, 7, 2, semanticTypeOperator, "'!=' operator in multi-line arithmetic"},
				{82, 11, 4, semanticTypeKeyword, "'this' keyword on line 2 of multi-line arithmetic"},
			},
		},
		{
			name: "cel_invalid",
			file: "testdata/semantic_tokens/cel_invalid.proto",
			// Invalid CEL expressions should not be highlighted at all.
			// When CEL parsing fails, we skip highlighting and treat them as plain protobuf strings.
			// This test verifies that we don't crash or highlight invalid syntax.
			expectedTokens: []expectedToken{},
		},
		{
			name: "extensions",
			file: "testdata/semantic_tokens/extensions.proto",
			expectedTokens: []expectedToken{
				// syntax declaration
				{0, 0, 6, semanticTypeKeyword, "'syntax' keyword"},
				{0, 9, 8, semanticTypeString, "'\"proto2\"' string"},
				// package declaration
				{2, 0, 7, semanticTypeKeyword, "'package' keyword"},
				{2, 8, 7, semanticTypeNamespace, "'test.v1' namespace"},
				// message with extensions
				{5, 0, 7, semanticTypeKeyword, "'message' keyword"},
				{5, 8, 3, semanticTypeStruct, "'Foo' message"},
				// extensions keyword
				{6, 2, 10, semanticTypeKeyword, "'extensions' keyword"},
				// extend keyword
				{11, 0, 6, semanticTypeKeyword, "'extend' keyword"},
				{11, 7, 3, semanticTypeStruct, "'Foo' (being extended)"},
				// extension field (my_extension)
				{12, 18, 12, semanticTypeVariable, "'my_extension' extension field"},
				{12, 33, 3, semanticTypeNumber, "'101' extension number"},
				// extension field (another_extension)
				{13, 17, 17, semanticTypeVariable, "'another_extension' extension field"},
				{13, 37, 3, semanticTypeNumber, "'102' extension number"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			testProtoPath, err := filepath.Abs(tc.file)
			require.NoError(t, err)
			clientJSONConn, testURI := setupLSPServer(t, testProtoPath)
			var semanticTokens *protocol.SemanticTokens
			_, err = clientJSONConn.Call(ctx, "textDocument/semanticTokens/full", protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
			}, &semanticTokens)
			require.NoError(t, err)
			require.NotNil(t, semanticTokens)
			require.NotEmpty(t, semanticTokens.Data)
			tokens := decodeSemanticTokens(semanticTokens.Data)

			// Check for overlapping spans (tokens with exact same position and length)
			type spanKey struct {
				line      uint32
				startChar uint32
				length    uint32
			}
			seenSpans := make(map[spanKey]int)
			for i, token := range tokens {
				key := spanKey{token.line, token.startChar, token.length}
				if prevIdx, exists := seenSpans[key]; exists {
					assert.Failf(t, "Overlapping spans detected",
						"Token %d and token %d have the same span (line=%d, col=%d, len=%d). Token %d type=%d, Token %d type=%d",
						prevIdx, i, token.line, token.startChar, token.length, prevIdx, tokens[prevIdx].tokenType, i, token.tokenType)
				}
				seenSpans[key] = i
			}

			for _, expected := range tc.expectedTokens {
				assert.True(t, findToken(tokens, expected.line, expected.startChar, expected.length, expected.tokenType),
					"Expected %s at line %d, column %d", expected.desc, expected.line+1, expected.startChar)
			}

			for _, notExpected := range tc.notExpectedTokens {
				assert.False(t, findToken(tokens, notExpected.line, notExpected.startChar, notExpected.length, notExpected.tokenType),
					"Not expected %s at line %d, column %d", notExpected.desc, notExpected.line+1, notExpected.startChar)
			}
		})
	}
}

// Semantic token types - must match semantic_tokens.go constants
const (
	semanticTypeProperty   = 0
	semanticTypeStruct     = 1
	semanticTypeVariable   = 2
	semanticTypeEnum       = 3
	semanticTypeEnumMember = 4
	semanticTypeInterface  = 5
	semanticTypeMethod     = 6
	semanticTypeFunction   = 7
	semanticTypeDecorator  = 8
	semanticTypeMacro      = 9
	semanticTypeNamespace  = 10
	semanticTypeKeyword    = 11
	semanticTypeModifier   = 12
	semanticTypeComment    = 13
	semanticTypeString     = 14
	semanticTypeNumber     = 15
	semanticTypeType       = 16
	semanticTypeOperator   = 17
)

// semanticToken represents a decoded semantic token for easier testing.
type semanticToken struct {
	line      uint32
	startChar uint32
	length    uint32
	tokenType uint32
}

// decodeSemanticTokens converts the delta-encoded token array into absolute positions.
func decodeSemanticTokens(data []uint32) []semanticToken {
	var tokens []semanticToken
	var line, startChar uint32

	for i := 0; i < len(data); i += 5 {
		deltaLine := data[i]
		deltaStartChar := data[i+1]
		length := data[i+2]
		tokenType := data[i+3]

		line += deltaLine
		if deltaLine != 0 {
			startChar = deltaStartChar
		} else {
			startChar += deltaStartChar
		}

		tokens = append(tokens, semanticToken{
			line:      line,
			startChar: startChar,
			length:    length,
			tokenType: tokenType,
		})
	}

	return tokens
}

// findToken searches for a token at the specified position with the given type.
func findToken(tokens []semanticToken, line, startChar, length, tokenType uint32) bool {
	return slices.ContainsFunc(tokens, func(token semanticToken) bool {
		return token.line == line && token.startChar == startChar &&
			token.length == length && token.tokenType == tokenType
	})
}
