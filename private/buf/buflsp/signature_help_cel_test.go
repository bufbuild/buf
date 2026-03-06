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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestCELSignatureHelp(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	testProtoPath, err := filepath.Abs("testdata/hover/cel_signature_help.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// All expression lines below are 0-indexed.
	//
	// Line 11: `    expression: "size(this) > 0"`
	//           0123456789012345678901234
	//                           ^17=s   ^22=t (inside size())
	//
	// Line 18: `    expression: "this.contains('x')"`
	//           01234567890123456789012345678901
	//                           ^17=t        ^31=' (inside contains())
	//
	// Line 25: `    expression: "this != ''"`
	//           0123456789012345678901234 5
	//                           ^17=t    ^25=' (not inside any call)
	//
	// Line 32: `    expression: "size(string(42)) > 0"`
	//           01234567890123456789012345678 9
	//                           ^17=s       ^29=4 (inside inner string())
	//
	// Line 40: `    expression: "(1 + 2) > 0"`
	//           01234567890123456789
	//                           ^17=(  ^18=1 (operator in grouping parens)
	//
	// Line 7:   `  // Global function call: size(this) > 0`
	//           Cursor here is outside any CEL string literal.
	testCases := []struct {
		name               string
		line               uint32
		char               uint32
		expectNil          bool
		wantFuncInSig      string // substring expected in at least one signature label
		wantParamIndex     uint32
		wantExactSigCount  int // if > 0, assert len(Signatures) == this value
	}{
		{
			name:           "global function: size",
			line:           11,
			char:           22, // 't' of 'this', inside size(...)
			wantFuncInSig:  "size",
			wantParamIndex: 0,
		},
		{
			// Receiver-type filtering: this is a string field, so only the
			// string overload of contains should appear (not the bytes one).
			name:              "member function: contains filtered to string receiver",
			line:              18,
			char:              31, // first "'" inside contains(...)
			wantFuncInSig:     "contains",
			wantParamIndex:    0,
			wantExactSigCount: 1,
		},
		{
			name:      "no call at position",
			line:      25,
			char:      25, // first "'" of '' in "this != ''", not inside a call
			expectNil: true,
		},
		{
			name:           "nested calls: inner string()",
			line:           32,
			char:           29, // '4' inside inner string(42)
			wantFuncInSig:  "string",
			wantParamIndex: 0,
		},
		{
			name:      "operator: no signatures for + inside grouping parens",
			line:      40,
			char:      18, // '1' in "(1 + 2) > 0" — cursor is inside the grouping () that
			// the + operator AST node maps to, but operators must not show signatures.
			expectNil: true,
		},
		{
			name:      "outside CEL expression",
			line:      7,
			char:      10, // mid-comment, no CEL string literal here
			expectNil: true,
		},
		{
			// Cursor exactly on '(' — signature help fires after typing '(', not on it.
			// targetOffset == parenStart fails the strict '>' check.
			name:      "boundary: cursor at opening paren",
			line:      11,
			char:      21, // the '(' of size(
			expectNil: true,
		},
		{
			// Cursor on ')' — the user is closing the call; signature is still useful.
			name:          "boundary: cursor on closing paren",
			line:          11,
			char:          26, // the ')' of size(this)
			wantFuncInSig: "size",
		},
		{
			// Function name not in celEnv.Functions() → nil.
			name:      "unknown function",
			line:      51,
			char:      27, // 't' of 'this' inside unknown_fn(this)
			expectNil: true,
		},
		{
			// cursor at char 32 = '1', the first argument → paramIndex 0.
			name:           "substring: first param",
			line:           58,
			char:           32, // '1' in substring(1, 3)
			wantFuncInSig:  "substring",
			wantParamIndex: 0,
		},
		{
			// cursor at char 35 = '3', the second argument → paramIndex 1.
			name:           "substring: second param",
			line:           58,
			char:           35, // '3' in substring(1, 3)
			wantFuncInSig:  "substring",
			wantParamIndex: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var result *protocol.SignatureHelp
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentSignatureHelp, protocol.SignatureHelpParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: testURI},
					Position:     protocol.Position{Line: tc.line, Character: tc.char},
				},
			}, &result)
			require.NoError(t, err)

			if tc.expectNil {
				assert.Nil(t, result, "expected nil SignatureHelp at Line:%d Char:%d", tc.line, tc.char)
				return
			}

			require.NotNil(t, result, "expected non-nil SignatureHelp at Line:%d Char:%d", tc.line, tc.char)
			require.NotEmpty(t, result.Signatures, "expected at least one signature at Line:%d Char:%d", tc.line, tc.char)

			// Verify at least one signature label contains the expected function name.
			found := false
			for _, sig := range result.Signatures {
				if strings.Contains(sig.Label, tc.wantFuncInSig) {
					found = true
					break
				}
			}
			assert.True(t, found, "no signature label contains %q; got: %v", tc.wantFuncInSig, result.Signatures)
			assert.Equal(t, tc.wantParamIndex, result.ActiveParameter, "active parameter mismatch at Line:%d Char:%d", tc.line, tc.char)
			if tc.wantExactSigCount > 0 {
				assert.Len(t, result.Signatures, tc.wantExactSigCount, "signature count mismatch at Line:%d Char:%d", tc.line, tc.char)
			}
		})
	}
}
