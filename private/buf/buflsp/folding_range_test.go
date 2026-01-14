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

	ctx := t.Context()

	foldingProtoPath, err := filepath.Abs("testdata/folding_range/folding.proto")
	require.NoError(t, err)

	clientJSONConn, foldingURI := setupLSPServer(t, foldingProtoPath)

	var ranges []protocol.FoldingRange
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: foldingURI,
			},
		},
	}, &ranges)
	require.NoError(t, err)

	// Verify we have folding ranges
	require.NotEmpty(t, ranges, "expected at least one folding range")

	// Debug: print all ranges
	t.Logf("Found %d folding ranges:", len(ranges))
	for i, r := range ranges {
		t.Logf("  Range %d: lines %d-%d, kind=%q", i, r.StartLine, r.EndLine, r.Kind)
	}

	// Check for specific folding ranges
	// The test file has:
	// - Import group (lines 4-6)
	// - Multi-line comment (lines 8-10)
	// - User message (lines 11-15)
	// - Status enum (lines 17-21)
	// - UserService service (lines 23-32) with individual RPCs
	// - Config message (lines 60-65)
	// - Repository message with multi-line field options (lines 67-74)
	// - Profile message with oneof and nested Address message (lines 76-88)
	// - Several request/response messages

	// Find the import group range
	foundImports := false
	for _, r := range ranges {
		if r.Kind == protocol.ImportsFoldingRange {
			// Import group should be on lines 4-6 (0-indexed)
			if r.StartLine == 4 && r.EndLine == 6 {
				foundImports = true
				break
			}
		}
	}
	assert.True(t, foundImports, "expected to find import group folding range")

	// Find the multi-line comment range
	foundComment := false
	for _, r := range ranges {
		if r.Kind == protocol.CommentFoldingRange {
			// Multi-line comment should be on lines 8-10 (0-indexed)
			if r.StartLine == 8 && r.EndLine == 10 {
				foundComment = true
				break
			}
		}
	}
	assert.True(t, foundComment, "expected to find multi-line comment folding range")

	// Find the User message range
	foundUserMessage := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// User message should be on lines 11-15 (0-indexed)
			if r.StartLine == 11 && r.EndLine == 15 {
				foundUserMessage = true
				break
			}
		}
	}
	assert.True(t, foundUserMessage, "expected to find User message folding range")

	// Find the Status enum range
	foundStatusEnum := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Status enum should be on lines 17-21 (0-indexed)
			if r.StartLine == 17 && r.EndLine == 21 {
				foundStatusEnum = true
				break
			}
		}
	}
	assert.True(t, foundStatusEnum, "expected to find Status enum folding range")

	// Find the UserService service range
	foundUserService := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// UserService should be on lines 23-32 (0-indexed)
			if r.StartLine == 23 && r.EndLine == 32 {
				foundUserService = true
				break
			}
		}
	}
	assert.True(t, foundUserService, "expected to find UserService service folding range")

	// Find individual RPC methods (GetUser, CreateUser, and UpdateUser)
	foundGetUserRPC := false
	foundCreateUserRPC := false
	foundUpdateUserRPC := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// GetUser RPC should be on line 24 (0-indexed, single line)
			if r.StartLine == 24 && r.EndLine == 24 {
				foundGetUserRPC = true
			}
			// CreateUser RPC should be on line 25 (0-indexed, single line)
			if r.StartLine == 25 && r.EndLine == 25 {
				foundCreateUserRPC = true
			}
			// UpdateUser RPC should be on lines 28-31 (0-indexed, multi-line with options)
			if r.StartLine == 28 && r.EndLine == 31 {
				foundUpdateUserRPC = true
			}
		}
	}
	assert.True(t, foundGetUserRPC, "expected to find GetUser RPC folding range")
	assert.True(t, foundCreateUserRPC, "expected to find CreateUser RPC folding range")
	assert.True(t, foundUpdateUserRPC, "expected to find UpdateUser multi-line RPC folding range")

	// Find the Config message
	foundConfigMessage := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Config message should be on lines 60-65 (0-indexed)
			if r.StartLine == 60 && r.EndLine == 65 {
				foundConfigMessage = true
				break
			}
		}
	}
	assert.True(t, foundConfigMessage, "expected to find Config message folding range")

	// Find the Repository message with multi-line field options
	foundRepositoryMessage := false
	foundMultiLineOptions := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Repository message should be on lines 67-74 (0-indexed)
			if r.StartLine == 67 && r.EndLine == 74 {
				foundRepositoryMessage = true
			}
			// Multi-line field options for commit_id should be on lines 68-71 (0-indexed)
			if r.StartLine == 68 && r.EndLine == 71 {
				foundMultiLineOptions = true
			}
		}
	}
	assert.True(t, foundRepositoryMessage, "expected to find Repository message folding range")
	assert.True(t, foundMultiLineOptions, "expected to find multi-line field options folding range")

	// Find the Profile message with oneof and nested Address
	foundProfileMessage := false
	foundOneof := false
	foundNestedAddress := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Profile message should be on lines 76-88 (0-indexed)
			if r.StartLine == 76 && r.EndLine == 88 {
				foundProfileMessage = true
			}
			// Oneof contact should be on lines 77-80 (0-indexed)
			if r.StartLine == 77 && r.EndLine == 80 {
				foundOneof = true
			}
			// Nested Address message should be on lines 83-87 (0-indexed)
			if r.StartLine == 83 && r.EndLine == 87 {
				foundNestedAddress = true
			}
		}
	}
	assert.True(t, foundProfileMessage, "expected to find Profile message folding range")
	assert.True(t, foundOneof, "expected to find oneof contact folding range")
	assert.True(t, foundNestedAddress, "expected to find nested Address message folding range")

	// Verify no overlapping ranges
	assertNoOverlappingFoldingRanges(t, ranges)
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

func TestFoldingRangeProto2Extensions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	extensionsProtoPath, err := filepath.Abs("testdata/folding_range/extensions.proto")
	require.NoError(t, err)

	clientJSONConn, extensionsURI := setupLSPServer(t, extensionsProtoPath)

	var ranges []protocol.FoldingRange
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: extensionsURI,
			},
		},
	}, &ranges)
	require.NoError(t, err)

	// Verify we have folding ranges
	require.NotEmpty(t, ranges, "expected at least one folding range")

	// Debug: print all ranges
	t.Logf("Found %d folding ranges:", len(ranges))
	for i, r := range ranges {
		t.Logf("  Range %d: lines %d-%d, kind=%q", i, r.StartLine, r.EndLine, r.Kind)
	}

	// Find the User message with extensions
	foundUserMessage := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// User message should be on lines 5-9 (0-indexed)
			if r.StartLine == 5 && r.EndLine == 9 {
				foundUserMessage = true
				break
			}
		}
	}
	assert.True(t, foundUserMessage, "expected to find User message folding range")

	// Find the extend block
	foundExtendBlock := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Extend block should be on lines 12-15 (0-indexed)
			if r.StartLine == 12 && r.EndLine == 15 {
				foundExtendBlock = true
				break
			}
		}
	}
	assert.True(t, foundExtendBlock, "expected to find extend block folding range")

	// Find the UserMetadata message
	foundUserMetadata := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// UserMetadata should be on lines 18-30 (0-indexed)
			if r.StartLine == 18 && r.EndLine == 30 {
				foundUserMetadata = true
				break
			}
		}
	}
	assert.True(t, foundUserMetadata, "expected to find UserMetadata message folding range")

	// Find the nested AuditInfo message
	foundAuditInfo := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// AuditInfo nested message should be on lines 23-27 (0-indexed)
			if r.StartLine == 23 && r.EndLine == 27 {
				foundAuditInfo = true
				break
			}
		}
	}
	assert.True(t, foundAuditInfo, "expected to find nested AuditInfo message folding range")

	// Find the Preferences message
	foundPreferences := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Preferences should be on lines 33-36 (0-indexed)
			if r.StartLine == 33 && r.EndLine == 36 {
				foundPreferences = true
				break
			}
		}
	}
	assert.True(t, foundPreferences, "expected to find Preferences message folding range")

	// Find the Settings message with multi-line field options
	foundSettings := false
	foundSettingsOptions := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Settings message should be on lines 39-44 (0-indexed)
			if r.StartLine == 39 && r.EndLine == 44 {
				foundSettings = true
			}
			// Multi-line field options should be on lines 40-43 (0-indexed)
			if r.StartLine == 40 && r.EndLine == 43 {
				foundSettingsOptions = true
			}
		}
	}
	assert.True(t, foundSettings, "expected to find Settings message folding range")
	assert.True(t, foundSettingsOptions, "expected to find multi-line field options in proto2")

	// Verify no overlapping ranges
	assertNoOverlappingFoldingRanges(t, ranges)
}

func TestFoldingRangeMinimal(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	minimalProtoPath, err := filepath.Abs("testdata/folding_range/minimal.proto")
	require.NoError(t, err)

	clientJSONConn, minimalURI := setupLSPServer(t, minimalProtoPath)

	var ranges []protocol.FoldingRange
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: minimalURI,
			},
		},
	}, &ranges)
	require.NoError(t, err)

	// Minimal file should have only the Simple message (lines 5-7), no foldable comments or imports
	t.Logf("Found %d folding ranges:", len(ranges))
	for i, r := range ranges {
		t.Logf("  Range %d: lines %d-%d, kind=%q", i, r.StartLine, r.EndLine, r.Kind)
	}

	// Should have exactly 1 range (the Simple message)
	assert.Len(t, ranges, 1, "expected exactly 1 folding range")

	if len(ranges) > 0 {
		// Verify it's the Simple message
		assert.Equal(t, uint32(5), ranges[0].StartLine, "expected Simple message to start on line 5")
		assert.Equal(t, uint32(7), ranges[0].EndLine, "expected Simple message to end on line 7")
		assert.Equal(t, protocol.RegionFoldingRange, ranges[0].Kind)
	}
}

func TestFoldingRangeImports(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	importsProtoPath, err := filepath.Abs("testdata/folding_range/imports.proto")
	require.NoError(t, err)

	clientJSONConn, importsURI := setupLSPServer(t, importsProtoPath)

	var ranges []protocol.FoldingRange
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: importsURI,
			},
		},
	}, &ranges)
	require.NoError(t, err)

	t.Logf("Found %d folding ranges:", len(ranges))
	for i, r := range ranges {
		t.Logf("  Range %d: lines %d-%d, kind=%q", i, r.StartLine, r.EndLine, r.Kind)
	}

	// Count import groups
	importGroupCount := 0
	for _, r := range ranges {
		if r.Kind == protocol.ImportsFoldingRange {
			importGroupCount++
		}
	}

	// Should have 3 import groups:
	// - Lines 5-7 (first group)
	// - Lines 10-11 (second group, with 1 blank line gap)
	// - Lines 15-16 (third group, after 2+ blank lines)
	// Note: Single import on line 19 should NOT be foldable
	assert.Equal(t, 3, importGroupCount, "expected 3 import groups")

	// Verify no overlapping ranges
	assertNoOverlappingFoldingRanges(t, ranges)
}

func TestFoldingRangeNested(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	nestedProtoPath, err := filepath.Abs("testdata/folding_range/nested.proto")
	require.NoError(t, err)

	clientJSONConn, nestedURI := setupLSPServer(t, nestedProtoPath)

	var ranges []protocol.FoldingRange
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: nestedURI,
			},
		},
	}, &ranges)
	require.NoError(t, err)

	t.Logf("Found %d folding ranges:", len(ranges))
	for i, r := range ranges {
		t.Logf("  Range %d: lines %d-%d, kind=%q", i, r.StartLine, r.EndLine, r.Kind)
	}

	// Should have many nested ranges for Level1 -> Level2 -> Level3 -> Level4
	// and oneofs in ComplexOneof and NestedOneofs
	require.NotEmpty(t, ranges, "expected folding ranges for nested structures")

	// Find the Level4 message (most deeply nested)
	foundLevel4 := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// Level4 should be on lines 14-17 (0-indexed)
			if r.StartLine == 14 && r.EndLine == 17 {
				foundLevel4 = true
				break
			}
		}
	}
	assert.True(t, foundLevel4, "expected to find Level4 nested message")

	// Find the ComplexOneof with many options
	foundComplexOneof := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// ComplexOneof oneof should be on lines 30-39 (0-indexed)
			if r.StartLine == 30 && r.EndLine == 39 {
				foundComplexOneof = true
				break
			}
		}
	}
	assert.True(t, foundComplexOneof, "expected to find ComplexOneof with many options")

	// Verify no overlapping ranges (nested is OK, but improper overlap is not)
	assertNoOverlappingFoldingRanges(t, ranges)
}

func TestFoldingRangeComments(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	commentsProtoPath, err := filepath.Abs("testdata/folding_range/comments.proto")
	require.NoError(t, err)

	clientJSONConn, commentsURI := setupLSPServer(t, commentsProtoPath)

	var ranges []protocol.FoldingRange
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: commentsURI,
			},
		},
	}, &ranges)
	require.NoError(t, err)

	t.Logf("Found %d folding ranges:", len(ranges))
	for i, r := range ranges {
		t.Logf("  Range %d: lines %d-%d, kind=%q", i, r.StartLine, r.EndLine, r.Kind)
	}

	// Count comment folding ranges
	commentCount := 0
	for _, r := range ranges {
		if r.Kind == protocol.CommentFoldingRange {
			commentCount++
		}
	}

	// Should have multiple multi-line comment blocks but NOT single-line comments
	// Expected: lines 6-8, 14-16, 20-22, 33-42 (4 comment blocks)
	assert.GreaterOrEqual(t, commentCount, 4, "expected at least 4 multi-line comment blocks")

	// Verify the large comment block is found
	foundLargeComment := false
	for _, r := range ranges {
		if r.Kind == protocol.CommentFoldingRange {
			// Large comment block should span at least 8 lines
			if r.EndLine-r.StartLine >= 8 {
				foundLargeComment = true
				break
			}
		}
	}
	assert.True(t, foundLargeComment, "expected to find large multi-line comment block")

	// Verify no overlapping ranges
	assertNoOverlappingFoldingRanges(t, ranges)
}

func TestFoldingRangeOptions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	optionsProtoPath, err := filepath.Abs("testdata/folding_range/options.proto")
	require.NoError(t, err)

	clientJSONConn, optionsURI := setupLSPServer(t, optionsProtoPath)

	var ranges []protocol.FoldingRange
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentFoldingRange, protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: optionsURI,
			},
		},
	}, &ranges)
	require.NoError(t, err)

	t.Logf("Found %d folding ranges:", len(ranges))
	for i, r := range ranges {
		t.Logf("  Range %d: lines %d-%d, kind=%q", i, r.StartLine, r.EndLine, r.Kind)
	}

	// Count multi-line option blocks
	optionBlockCount := 0
	for _, r := range ranges {
		// Option blocks are RegionFoldingRange but are small (typically 2-3 lines)
		// and start/end on specific bracket positions
		if r.Kind == protocol.RegionFoldingRange && r.EndLine-r.StartLine <= 3 {
			// Could be an option block (though could also be a small message)
			optionBlockCount++
		}
	}

	// Should have multiple multi-line field option blocks
	// FieldOptions has 2, MultiOptions has 3, plus RPC method options
	assert.GreaterOrEqual(t, optionBlockCount, 5, "expected at least 5 multi-line option blocks")

	// Verify the FieldOptions message is found
	foundFieldOptions := false
	for _, r := range ranges {
		if r.Kind == protocol.RegionFoldingRange {
			// FieldOptions message should be on lines 5-20 (0-indexed)
			if r.StartLine == 5 && r.EndLine >= 19 {
				foundFieldOptions = true
				break
			}
		}
	}
	assert.True(t, foundFieldOptions, "expected to find FieldOptions message")

	// Verify no overlapping ranges
	assertNoOverlappingFoldingRanges(t, ranges)
}
