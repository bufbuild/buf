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

func TestFoldingRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		file              string
		expectedAssertion func(t *testing.T, ranges []protocol.FoldingRange)
	}{
		{
			name: "comprehensive",
			file: "folding.proto",
			expectedAssertion: func(t *testing.T, ranges []protocol.FoldingRange) {
				t.Helper()
				require.NotEmpty(t, ranges)

				// Find import group
				assert.True(t, findRange(ranges, 4, 6, protocol.ImportsFoldingRange), "import group")

				// Find multi-line comment
				assert.True(t, findRange(ranges, 8, 10, protocol.CommentFoldingRange), "multi-line comment")

				// Find User message
				assert.True(t, findRange(ranges, 11, 15, protocol.RegionFoldingRange), "User message")

				// Find Status enum
				assert.True(t, findRange(ranges, 17, 21, protocol.RegionFoldingRange), "Status enum")

				// Find UserService
				assert.True(t, findRange(ranges, 23, 32, protocol.RegionFoldingRange), "UserService")

				// Find individual RPCs
				assert.True(t, findRange(ranges, 24, 24, protocol.RegionFoldingRange), "GetUser RPC")
				assert.True(t, findRange(ranges, 25, 25, protocol.RegionFoldingRange), "CreateUser RPC")
				assert.True(t, findRange(ranges, 28, 31, protocol.RegionFoldingRange), "UpdateUser multi-line RPC")

				// Find Config message
				assert.True(t, findRange(ranges, 60, 65, protocol.RegionFoldingRange), "Config message")

				// Find Repository message with multi-line options
				assert.True(t, findRange(ranges, 67, 74, protocol.RegionFoldingRange), "Repository message")
				assert.True(t, findRange(ranges, 68, 71, protocol.RegionFoldingRange), "multi-line field options")

				// Find Profile message with oneof and nested Address
				assert.True(t, findRange(ranges, 76, 88, protocol.RegionFoldingRange), "Profile message")
				assert.True(t, findRange(ranges, 77, 80, protocol.RegionFoldingRange), "oneof")
				assert.True(t, findRange(ranges, 83, 87, protocol.RegionFoldingRange), "nested Address")
			},
		},
		{
			name: "proto2_extensions",
			file: "extensions.proto",
			expectedAssertion: func(t *testing.T, ranges []protocol.FoldingRange) {
				t.Helper()
				require.NotEmpty(t, ranges)

				// Find User message
				assert.True(t, findRange(ranges, 5, 9, protocol.RegionFoldingRange), "User message")

				// Find extend block
				assert.True(t, findRange(ranges, 12, 15, protocol.RegionFoldingRange), "extend block")

				// Find UserMetadata message
				assert.True(t, findRange(ranges, 18, 30, protocol.RegionFoldingRange), "UserMetadata message")

				// Find nested AuditInfo
				assert.True(t, findRange(ranges, 23, 27, protocol.RegionFoldingRange), "nested AuditInfo")

				// Find Preferences message
				assert.True(t, findRange(ranges, 33, 36, protocol.RegionFoldingRange), "Preferences message")

				// Find Settings message with multi-line options
				assert.True(t, findRange(ranges, 39, 44, protocol.RegionFoldingRange), "Settings message")
				assert.True(t, findRange(ranges, 40, 43, protocol.RegionFoldingRange), "multi-line field options in proto2")
			},
		},
		{
			name: "minimal",
			file: "minimal.proto",
			expectedAssertion: func(t *testing.T, ranges []protocol.FoldingRange) {
				t.Helper()
				// Should have exactly 1 range (the Simple message)
				assert.Len(t, ranges, 1, "expected exactly 1 folding range")
				if len(ranges) > 0 {
					assert.Equal(t, uint32(5), ranges[0].StartLine)
					assert.Equal(t, uint32(7), ranges[0].EndLine)
					assert.Equal(t, protocol.RegionFoldingRange, ranges[0].Kind)
				}
			},
		},
		{
			name: "imports",
			file: "imports.proto",
			expectedAssertion: func(t *testing.T, ranges []protocol.FoldingRange) {
				t.Helper()
				// Count import groups
				importGroupCount := 0
				for _, r := range ranges {
					if r.Kind == protocol.ImportsFoldingRange {
						importGroupCount++
					}
				}
				// Should have 3 import groups (separated by different gaps)
				assert.Equal(t, 3, importGroupCount, "expected 3 import groups")
			},
		},
		{
			name: "nested",
			file: "nested.proto",
			expectedAssertion: func(t *testing.T, ranges []protocol.FoldingRange) {
				t.Helper()
				require.NotEmpty(t, ranges)

				// Find Level4 message (most deeply nested)
				assert.True(t, findRange(ranges, 14, 17, protocol.RegionFoldingRange), "Level4 nested message")

				// Find ComplexOneof with many options
				assert.True(t, findRange(ranges, 30, 39, protocol.RegionFoldingRange), "ComplexOneof")
			},
		},
		{
			name: "comments",
			file: "comments.proto",
			expectedAssertion: func(t *testing.T, ranges []protocol.FoldingRange) {
				t.Helper()
				// Count comment folding ranges
				commentCount := 0
				for _, r := range ranges {
					if r.Kind == protocol.CommentFoldingRange {
						commentCount++
					}
				}
				// Should have multiple multi-line comment blocks but NOT single-line comments
				assert.GreaterOrEqual(t, commentCount, 4, "expected at least 4 multi-line comment blocks")

				// Verify the large comment block is found (spans at least 8 lines)
				foundLarge := false
				for _, r := range ranges {
					if r.Kind == protocol.CommentFoldingRange && r.EndLine-r.StartLine >= 8 {
						foundLarge = true
						break
					}
				}
				assert.True(t, foundLarge, "expected large multi-line comment block")
			},
		},
		{
			name: "options",
			file: "options.proto",
			expectedAssertion: func(t *testing.T, ranges []protocol.FoldingRange) {
				t.Helper()
				// Count multi-line option blocks (small regions)
				optionBlockCount := 0
				for _, r := range ranges {
					if r.Kind == protocol.RegionFoldingRange && r.EndLine-r.StartLine <= 3 {
						optionBlockCount++
					}
				}
				// Should have multiple multi-line field option blocks
				assert.GreaterOrEqual(t, optionBlockCount, 5, "expected at least 5 multi-line option blocks")

				// Verify FieldOptions message is found
				assert.True(t, findRangeMinEnd(ranges, 5, 19, protocol.RegionFoldingRange), "FieldOptions message")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			protoPath, err := filepath.Abs(filepath.Join("testdata/folding_range", tt.file))
			require.NoError(t, err)

			clientJSONConn, protoURI := setupLSPServer(t, protoPath)

			var ranges []protocol.FoldingRange
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: protoURI,
					},
				},
			}, &ranges)
			require.NoError(t, err)

			// Run custom assertions
			tt.expectedAssertion(t, ranges)

			// Verify no overlapping ranges (nested is OK, but improper overlap is not)
			assertNoOverlappingFoldingRanges(t, ranges)
		})
	}
}

// findRange returns true if a folding range with the exact start/end lines and kind exists.
func findRange(ranges []protocol.FoldingRange, startLine, endLine uint32, kind protocol.FoldingRangeKind) bool {
	for _, r := range ranges {
		if r.StartLine == startLine && r.EndLine == endLine && r.Kind == kind {
			return true
		}
	}
	return false
}

// findRangeMinEnd returns true if a folding range with the exact start line, at least the end line, and kind exists.
func findRangeMinEnd(ranges []protocol.FoldingRange, startLine, minEndLine uint32, kind protocol.FoldingRangeKind) bool {
	for _, r := range ranges {
		if r.StartLine == startLine && r.EndLine >= minEndLine && r.Kind == kind {
			return true
		}
	}
	return false
}

// assertNoOverlappingFoldingRanges verifies that no two folding ranges improperly overlap.
func assertNoOverlappingFoldingRanges(t *testing.T, ranges []protocol.FoldingRange) {
	t.Helper()

	for i := 0; i < len(ranges); i++ {
		for j := i + 1; j < len(ranges); j++ {
			r1 := ranges[i]
			r2 := ranges[j]

			// Check for improper overlap (one starts inside another but doesn't nest properly)
			// Proper nesting: r2 is completely inside r1, or r1 is completely inside r2, or no overlap
			r1ContainsR2 := r1.StartLine <= r2.StartLine && r1.EndLine >= r2.EndLine
			r2ContainsR1 := r2.StartLine <= r1.StartLine && r2.EndLine >= r1.EndLine
			noOverlap := r1.EndLine < r2.StartLine || r2.EndLine < r1.StartLine

			assert.True(
				t,
				r1ContainsR2 || r2ContainsR1 || noOverlap,
				"Folding ranges improperly overlap:\nRange %d: lines %d-%d\nRange %d: lines %d-%d",
				i, r1.StartLine, r1.EndLine,
				j, r2.StartLine, r2.EndLine,
			)
		}
	}
}
