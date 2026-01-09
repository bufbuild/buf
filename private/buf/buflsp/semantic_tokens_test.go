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
				{5, 11, 6, semanticTypeProperty, "'string' as property"},
				{5, 18, 4, semanticTypeProperty, "'name' as property"},
				{5, 25, 1, semanticTypeNumber, "'1' field tag"},
				// optional modifier
				{6, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{6, 11, 5, semanticTypeProperty, "'int32' as property"},
				{6, 17, 3, semanticTypeProperty, "'age' as property"},
				{6, 23, 1, semanticTypeNumber, "'2' field tag"},
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
				// option keyword
				{10, 2, 6, semanticTypeKeyword, "'option' keyword"},
				// decorator (option)
				{10, 9, 10, semanticTypeDecorator, "'deprecated' as decorator"},
				// built-in type
				{13, 2, 6, semanticTypeProperty, "'string' as property"},
				// field name
				{13, 9, 4, semanticTypeProperty, "'name' as property"},
				{13, 16, 1, semanticTypeNumber, "'1' field tag"},
				// optional field
				{16, 2, 8, semanticTypeModifier, "'optional' modifier"},
				{16, 11, 6, semanticTypeProperty, "'string' as property"},
				{16, 18, 8, semanticTypeProperty, "'nickname' as property"},
				{16, 29, 1, semanticTypeNumber, "'5' field tag"},
				// custom type
				{19, 2, 4, semanticTypeEnum, "'Role' as enum"},
				{19, 7, 4, semanticTypeProperty, "'role' as property"},
				// repeated modifier
				{22, 2, 8, semanticTypeModifier, "'repeated' modifier"},
				// map field (line 26 in file = line 25 in 0-indexed)
				{25, 2, 18, semanticTypeProperty, "'map<string, int32>' map type"},
				{25, 6, 6, semanticTypeProperty, "'string' scalar"},
				{25, 14, 5, semanticTypeProperty, "'int32' scalar"},
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
	semanticTypeDecorator  = 7
	semanticTypeNamespace  = 8
	semanticTypeKeyword    = 9
	semanticTypeModifier   = 10
	semanticTypeComment    = 11
	semanticTypeString     = 12
	semanticTypeNumber     = 13
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
