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

func TestCELCompletion(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	testProtoPath, err := filepath.Abs("testdata/hover/cel_completion.proto")
	require.NoError(t, err)

	clientJSONConn, testURI := setupLSPServer(t, testProtoPath)

	// File contents (0-indexed lines):
	//  26:  `    expression: "this."`                                              (dot context; also used for expression-start)
	//  32:  `    expression: ""`                                                   (empty expression)
	//  39:  `    expression: "1 > "`                                               (operator, no narrowing)
	//  46:  `    expression: "true == "`                                           (operator, narrowed to bool)
	//  53:  `    expression: "th"`                                                 (prefix "th", completes to "this")
	//  60:  `    expression: "size(this) > 0"`                                     (mid-token cursor, no completions)
	//  65:  `  string test_cel_expr = 7 [(buf.validate.field).cel_expression = "this."];`
	//  71:  `    expression: "this."`                                              (message-typed field)
	//  77:  `    expression: "!"`                                                  (unary NOT, narrowed to bool)
	//  84:  `    expression: "this.address.ci"`                                    (nested path; prefix "ci")
	//  91:  `    expression: "has(this."`                                          (has() arg completions)
	//  98:  `    expression: "has(this.address.ci"`                                (has() nested path; prefix "ci")
	// 105:  `    expression: "now."`                                               (Timestamp methods via "now" variable)
	// 112:  `    expression: "this."`                                              (repeated field; list methods and macros)
	// 119:  `    expression: "this."`                                              (map field; map methods)
	// 127:  `    expression: "this."`                                              (message-level CEL)
	// 138:  `    expression: "this.items.filter(addr, addr."`                      (comprehension iter var)
	// 155:  `    expression: "this.items.filter(item, item.addresses.filter(addr, addr."` (nested comprehension iter var)
	// 165:  `    expression: "this."`                                              (WKTHolder.created_at — Timestamp field)
	// 175:  `    expression: "this."`                                              (OneofHolder message-level — oneof member fields)
	// 193:  `    expression: "this.items."`                                        (RepeatedPathHolder — list receiver, not element fields)
	// 206:  `    expression: "this.items[0]."`                                     (IndexAccessHolder — indexed into list, yields element fields)
	// 210:  `    expression: "this.locations[\"key\"]."`                           (IndexAccessHolder — indexed into map, yields value fields)
	// 222:  `    expression: "this.items.filter(item, item.zip_code > 0).all(addr, addr."` (ChainedComprehensionHolder)
	// 232:  `    expression: "in"`                                                    (InOperatorHolder — "in" is an operator, not a function)
	tests := []struct {
		name                string
		line                uint32
		character           uint32
		noCompletions       bool // expect nil or empty completion list
		expectedContains    []string
		expectedNotContains []string
		expectedDocs        map[string]string // maps item label → expected documentation substring
		expectedDeprecated  []string          // item labels expected to have Deprecated: true
	}{
		{
			// Cursor at closing `"` of `"this."` — isDotContext sees `this.`
			// and returns member completions filtered to string methods.
			name:      "dot_context_member_completions",
			line:      26,
			character: 22, // position of closing `"` after `this.`
			expectedContains: []string{
				"contains",
				"endsWith",
				"matches",
				"size",
				"startsWith",
			},
			// Keywords and global-only functions are not included in member completion.
			expectedNotContains: []string{"true", "false", "null"},
		},
		{
			// Cursor at closing `"` of `""` — empty expression, return all completions.
			name:      "empty_expression_all_completions",
			line:      32,
			character: 17, // position of closing `"` in empty string
			expectedContains: []string{
				// CEL keywords / protovalidate special variables
				"true",
				"false",
				"null",
				"this",
				"now", // protovalidate runtime timestamp variable
				// Standard global functions
				"size",
				// Standard macros
				"all",
				"exists",
				"filter",
				"has",
				"map",
			},
		},
		{
			// Cursor at closing `"` of `"1 > "` — inside a CEL expression after an operator.
			// The `>` operator has mixed-type overloads (int, double, uint on the right),
			// so type narrowing does not apply and all CEL completions are returned.
			name:      "operator_context_cel_completions",
			line:      39,
			character: 21, // position of closing `"` after `1 > `
			expectedContains: []string{
				"size",
				"all",
				"exists",
				"true",
				"false",
				"null",
				"this",
				"now",
			},
		},
		{
			// Cursor at closing `"` of `"true == "`.
			// `==` has a single type-param overload; with bool on the left, expectedType
			// narrows to bool. Only bool-compatible completions are returned.
			name:      "operator_context_type_narrowed_to_bool",
			line:      46,
			character: 25, // position of closing `"` after `true == `
			expectedContains: []string{
				// bool keywords match the expected type
				"true",
				"false",
			},
			// size() returns int, not bool — excluded by type narrowing.
			// "this" is excluded because expectedType is non-nil.
			// "null" has type "value", not bool — excluded.
			// "now" is timestamp, not bool — excluded.
			expectedNotContains: []string{"size", "this", "null", "now"},
		},
		{
			// Cursor at char 17 (first character of `this` in `"this."`).
			// celContent="" so all completions are returned, just like the empty expression case.
			name:      "expression_start_all_completions",
			line:      26,
			character: 17, // char of `t` — celOffset=0, celContent=""
			expectedContains: []string{
				"true",
				"false",
				"null",
				"this",
				"now",
				"size",
				"all",
				"exists",
			},
		},
		{
			// Cursor at closing `"` of `"th"` — prefix "th" filters completions
			// to items starting with "th". The only standard CEL item is "this".
			name:      "prefix_th_completes_this",
			line:      53,
			character: 19, // closing `"` of `"th"`, celContent="th"
			expectedContains: []string{
				"this",
			},
			// Items not starting with "th" are all filtered out.
			expectedNotContains: []string{"true", "false", "null", "size", "all", "now"},
		},
		{
			// Cursor at char 18 of `    expression: "size(this) > 0"`.
			// char 18 = `i`, so the next character after the cursor is an identifier
			// character — the cursor is inside a completed token. No completions offered.
			name:          "no_completions_mid_token",
			line:          60,
			character:     18, // `i` in `size`, next char is also identifier
			noCompletions: true,
		},
		{
			// `cel_expression` field — a direct repeated-string field on FieldRules,
			// not wrapped in a Rule message. Cursor at char 72 = closing `"` after `this.`.
			name:      "cel_expression_field_dot_completions",
			line:      65,
			character: 72, // closing `"` of `"this."` in the inline option
			expectedContains: []string{
				"contains",
				"endsWith",
				"matches",
				"size",
				"startsWith",
			},
			expectedNotContains: []string{"true", "false", "null"},
		},
		{
			// Message-typed field: cursor at closing `"` of `"this."`.
			// Should return proto field names (city, country, zip_code) from Address,
			// not string member functions.
			name:      "message_field_dot_completions",
			line:      71,
			character: 22, // closing `"` of `"this."` for Address-typed field
			expectedContains: []string{
				"city",
				"country",
				"zip_code",
			},
			// String member functions and keywords are excluded.
			expectedNotContains: []string{"contains", "startsWith", "true", "false", "null"},
			// Verify that field doc comments are surfaced in completion documentation.
			expectedDocs: map[string]string{
				"city":     "The city name.",
				"country":  "The country code.",
				"zip_code": "The ZIP or postal code.",
			},
			// zip_code is marked deprecated = true in the proto.
			expectedDeprecated: []string{"zip_code"},
		},
		{
			// Unary NOT context: cursor at closing `"` of `"!"`.
			// After `!`, the operand must be bool, so completions narrow to bool-typed items.
			name:      "unary_not_bool_narrowing",
			line:      77,
			character: 18, // closing `"` of `"!"`
			expectedContains: []string{
				"true",
				"false",
			},
			// Non-bool items are excluded.
			expectedNotContains: []string{"size", "null", "this", "now"},
		},
		{
			// Nested path: cursor at end of `"this.address.ci"` on a Location-typed field.
			// `this` is Location, `this.address` is Address, prefix "ci" narrows to city.
			name:      "nested_path_completions",
			line:      84,
			character: 32, // closing `"` after `this.address.ci`
			expectedContains: []string{
				"city",
			},
			// Other Address fields don't start with "ci"; keywords and methods excluded.
			expectedNotContains: []string{"country", "zip_code", "contains", "true", "false", "null"},
		},
		{
			// has() macro argument: cursor at closing `"` of `"has(this."` on an Address field.
			// The inner `this.` selects Address fields.
			name:      "has_arg_dot_completions",
			line:      91,
			character: 26, // closing `"` after `has(this.`
			expectedContains: []string{
				"city",
				"country",
				"zip_code",
			},
			expectedNotContains: []string{"contains", "true", "false", "null"},
		},
		{
			// has() with a nested path: cursor at end of `"has(this.address.ci"`.
			// has() strips to "this.address.ci"; member access resolves to Address
			// via the nested path; prefix "ci" filters to only city.
			name:      "has_arg_nested_path_completions",
			line:      98,
			character: 36, // closing `"` after `has(this.address.ci`
			expectedContains: []string{
				"city",
			},
			expectedNotContains: []string{"country", "zip_code", "contains", "true", "false", "null"},
		},
		{
			// now. completions: cursor at closing `"` of `"now."`.
			// `now` is protovalidate's runtime Timestamp variable; "now." should yield
			// Timestamp methods from the CEL environment.
			name:      "now_dot_completions",
			line:      105,
			character: 21, // closing `"` after `now.`
			expectedContains: []string{
				"getFullYear",
				"getHours",
				"getMonth",
			},
			// Proto field names and keywords are not Timestamp methods.
			expectedNotContains: []string{"city", "name", "true", "false", "null"},
		},
		{
			// Repeated field: cursor at closing `"` of `"this."` on a `repeated Address` field.
			// CEL treats repeated proto fields as lists; `size` (list member function) and
			// comprehension macros (filter, all, exists, etc.) should be offered.
			// Address field names (city, etc.) should NOT appear since this is a list.
			name:      "repeated_field_dot_completions",
			line:      112,
			character: 22, // closing `"` after `this.`
			expectedContains: []string{
				"size",
				"filter",
				"all",
				"exists",
			},
			expectedNotContains: []string{"city", "country", "zip_code"},
		},
		{
			// Map field: cursor at closing `"` of `"this."` on a `map<string, string>` field.
			// CEL treats map fields as maps; `size` should be offered.
			// String field names should NOT appear.
			name:      "map_field_dot_completions",
			line:      119,
			character: 22, // closing `"` after `this.`
			expectedContains: []string{
				"size",
			},
			expectedNotContains: []string{"city", "country", "true", "false", "null"},
		},
		{
			// Message-level CEL rule: `(buf.validate.message).cel`.
			// Cursor at char 22 = closing `"` after `this.` in the message option.
			// `this` refers to CELCompletionMsg itself, so field completions yield `name`.
			name:      "message_level_dot_completions",
			line:      127,
			character: 22,
			expectedContains: []string{
				"name",
			},
			// String member functions and keywords are not proto fields — excluded.
			expectedNotContains: []string{"contains", "startsWith", "true", "false", "null"},
		},
		{
			// Comprehension iteration variable: cursor at closing `"` of
			// `"this.items.filter(addr, addr."`. The range `this.items` is a repeated
			// Address field; `addr` is bound to Address elements. Completions should
			// yield Address field names.
			name:      "iter_var_dot_completions",
			line:      138,
			character: 46, // closing `"` after `this.items.filter(addr, addr.`
			expectedContains: []string{
				"city",
				"country",
				"zip_code",
			},
			// List methods and keywords are not Address fields.
			expectedNotContains: []string{"size", "all", "true", "false", "null"},
		},
		{
			// Nested comprehension iteration variable: cursor at closing `"` of
			// `"this.items.filter(item, item.addresses.filter(addr, addr."`.
			// The outer range `this.items` is a repeated Item field; `item` is bound to Item.
			// The inner range `item.addresses` is a repeated Address field; `addr` is bound
			// to Address elements. Completions should yield Address field names.
			name:      "iter_nested_var_completions",
			line:      155,
			character: 74, // closing `"` after `this.items.filter(item, item.addresses.filter(addr, addr.`
			expectedContains: []string{
				"city",
				"country",
				"zip_code",
			},
			// List methods, Item fields, and keywords are not Address fields.
			expectedNotContains: []string{"size", "all", "label", "true", "false", "null"},
		},
		{
			// Well-known type field: cursor at closing `"` of `"this."` on a
			// google.protobuf.Timestamp field. The proto fields (seconds, nanos)
			// and CEL Timestamp methods (getFullYear, getHours, etc.) should be offered.
			// This exercises the IR path for WKT completions, which is distinct from
			// the `now.` test that resolves Timestamp via the CEL static environment.
			name:      "wkt_timestamp_field_dot_completions",
			line:      165,
			character: 22, // closing `"` after `this.`
			expectedContains: []string{
				"seconds",
				"nanos",
				"getFullYear",
				"getHours",
				"getMonth",
			},
			expectedNotContains: []string{"city", "country", "zip_code"},
		},
		{
			// Oneof field completions: cursor at closing `"` of `"this."` in a
			// message-level CEL rule. The oneof member fields (text, code, location)
			// are non-synthetic members in the proto IR and must appear as completions.
			name:      "oneof_field_dot_completions",
			line:      175,
			character: 22, // closing `"` after `this.`
			expectedContains: []string{
				"text",
				"code",
				"location",
			},
			expectedNotContains: []string{"true", "false", "null", "contains", "startsWith"},
		},
		{
			// Repeated path: cursor at closing `"` of `"this.items."` in a message-level
			// CEL rule on RepeatedPathHolder. `this.items` is a repeated Address field,
			// so the receiver is a list — list methods and comprehension macros should be
			// offered, not Address field names.
			name:      "repeated_path_dot_completions",
			line:      193,
			character: 28, // closing `"` after `this.items.`
			expectedContains: []string{
				"size",
				"filter",
				"all",
				"exists",
			},
			expectedNotContains: []string{"city", "country", "zip_code"},
		},
		{
			// Index access (list): cursor at closing `"` of `"this.items[0]."` in a
			// message-level CEL rule on IndexAccessHolder. After indexing into the
			// repeated Address field, the receiver is an Address element — Address
			// field names should be offered.
			name:      "index_access_list_completions",
			line:      206,
			character: 31, // closing `"` after `this.items[0].`
			expectedContains: []string{
				"city",
				"country",
				"zip_code",
			},
			expectedNotContains: []string{"size", "all", "true", "false"},
		},
		{
			// Index access (map): cursor at closing `"` of `"this.locations["key"]."` in a
			// message-level CEL rule on IndexAccessHolder. After indexing into the
			// map<string, Address> field, the receiver is an Address value — Address
			// field names should be offered, not map methods.
			name:      "index_access_map_completions",
			line:      210,
			character: 41, // closing `"` after `this.locations["key"].`
			expectedContains: []string{
				"city",
				"country",
				"zip_code",
			},
			expectedNotContains: []string{"size", "all", "true", "false"},
		},
		{
			// Chained comprehension: cursor at closing `"` of
			// `"this.items.filter(item, item.zip_code > 0).all(addr, addr."`.
			// The outer filter's result is a list of Address elements; `addr` in the
			// inner all() is bound to those elements. Address field names must appear
			// even though the range expression has a trailing `)` from the outer call.
			name:      "chained_comprehension_iter_var",
			line:      222,
			character: 75, // closing `"` after `...all(addr, addr.`
			expectedContains: []string{
				"city",
				"country",
				"zip_code",
			},
			expectedNotContains: []string{"size", "all", "true", "false", "null"},
		},
		{
			// Cursor at closing `"` of `"in"` — prefix "in".
			// `in` is a CEL binary membership operator ("value in list"), not a
			// callable function. It must NOT appear as a function completion "in()".
			name:                "in_operator_not_function_completion",
			line:                232,
			character:           19, // closing `"` after `in`
			expectedNotContains: []string{"in"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var completionList *protocol.CompletionList
			_, err := clientJSONConn.Call(ctx, protocol.MethodTextDocumentCompletion, protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: testURI,
					},
					Position: protocol.Position{
						Line:      tt.line,
						Character: tt.character,
					},
				},
			}, &completionList)
			require.NoError(t, err)

			if tt.noCompletions {
				assert.True(t,
					completionList == nil || len(completionList.Items) == 0,
					"expected no completions mid-token, got %v",
					func() []string {
						if completionList == nil {
							return nil
						}
						labels := make([]string, len(completionList.Items))
						for i, item := range completionList.Items {
							labels[i] = item.Label
						}
						return labels
					}(),
				)
				return
			}

			require.NotNil(t, completionList, "expected completion list to be non-nil")

			labels := make([]string, 0, len(completionList.Items))
			for _, item := range completionList.Items {
				labels = append(labels, item.Label)
			}
			for _, expected := range tt.expectedContains {
				assert.Contains(t, labels, expected, "expected completion list to contain %q", expected)
			}
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, labels, notExpected, "expected completion list to not contain %q", notExpected)
			}
			for label, wantDoc := range tt.expectedDocs {
				var found bool
				for _, item := range completionList.Items {
					if item.Label != label {
						continue
					}
					found = true
					var docStr string
					switch doc := item.Documentation.(type) {
					case string:
						docStr = doc
					case *protocol.MarkupContent:
						docStr = doc.Value
					case map[string]any:
						// After JSON round-trip through the LSP JSON-RPC layer,
						// *protocol.MarkupContent is decoded as a generic map.
						if v, ok := doc["value"].(string); ok {
							docStr = v
						}
					}
					assert.Contains(t, docStr, wantDoc, "item %q: expected documentation to contain %q", label, wantDoc)
					break
				}
				assert.True(t, found, "item %q not found in completion list for doc check", label)
			}
			for _, label := range tt.expectedDeprecated {
				var found bool
				for _, item := range completionList.Items {
					if item.Label != label {
						continue
					}
					found = true
					assert.True(t, item.Deprecated, "item %q: expected Deprecated to be true", label)
					break
				}
				assert.True(t, found, "item %q not found in completion list for deprecated check", label)
			}
		})
	}
}
