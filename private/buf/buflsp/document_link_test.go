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
	"go.lsp.dev/uri"
)

func TestDocumentLink(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	mainProtoPath, err := filepath.Abs("testdata/document_link/main.proto")
	require.NoError(t, err)

	typesProtoPath, err := filepath.Abs("testdata/document_link/types.proto")
	require.NoError(t, err)

	clientJSONConn, mainURI := setupLSPServer(t, mainProtoPath)
	typesURI := uri.New(typesProtoPath)

	var links []protocol.DocumentLink
	_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentLink, protocol.DocumentLinkParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: mainURI,
		},
	}, &links)
	require.NoError(t, err)

	// main.proto has one import: "types.proto" at line 5
	// and three URLs in comments at lines 8, 9, and 12
	require.Len(t, links, 4, "expected exactly four document links")

	// First link should be the import
	importLink := links[0]
	// The import statement is on line 5 (0-indexed is line 4)
	assert.Equal(t, uint32(4), importLink.Range.Start.Line)
	// Should link to types.proto local file (since the test module has no FullName).
	// When importing from BSR modules with FullName set, the link would be to
	// https://buf.build/owner/module/docs/main:package.name (using bufconnect.DefaultRemote)
	assert.Equal(t, typesURI, importLink.Target)

	// Second link should be the first URL in the comment
	urlLink1 := links[1]
	// The URL is on line 8 (0-indexed is line 7)
	assert.Equal(t, uint32(7), urlLink1.Range.Start.Line)
	assert.Equal(t, protocol.DocumentURI("https://example.com/docs"), urlLink1.Target)

	// Third link should be the second URL in the comment
	urlLink2 := links[2]
	// The URL is on line 9 (0-indexed is line 8)
	assert.Equal(t, uint32(8), urlLink2.Range.Start.Line)
	assert.Equal(t, protocol.DocumentURI("https://github.com/example/repo"), urlLink2.Target)

	// Fourth link should be the third URL in the inline comment
	urlLink3 := links[3]
	// The URL is on line 12 (0-indexed is line 11)
	assert.Equal(t, uint32(11), urlLink3.Range.Start.Line)
	assert.Equal(t, protocol.DocumentURI("https://example.com/status"), urlLink3.Target)

	// Verify no overlapping ranges
	assertNoOverlappingRanges(t, links)
}

// assertNoOverlappingRanges verifies that no two document link ranges overlap.
func assertNoOverlappingRanges(t *testing.T, links []protocol.DocumentLink) {
	t.Helper()

	for i := 0; i < len(links); i++ {
		for j := i + 1; j < len(links); j++ {
			range1 := links[i].Range
			range2 := links[j].Range

			// Check if ranges overlap
			assert.False(
				t,
				rangesOverlap(range1, range2),
				"Document link ranges overlap:\nLink %d (target=%s): Line %d:%d to %d:%d\nLink %d (target=%s): Line %d:%d to %d:%d",
				i, links[i].Target,
				range1.Start.Line, range1.Start.Character,
				range1.End.Line, range1.End.Character,
				j, links[j].Target,
				range2.Start.Line, range2.Start.Character,
				range2.End.Line, range2.End.Character,
			)
		}
	}
}

// rangesOverlap returns true if two ranges overlap.
func rangesOverlap(r1, r2 protocol.Range) bool {
	// A range ends before another starts if r1.End <= r2.Start
	// Ranges don't overlap if one ends before the other starts
	if positionLessOrEqual(r1.End, r2.Start) || positionLessOrEqual(r2.End, r1.Start) {
		return false
	}
	return true
}

// positionLessOrEqual returns true if pos1 <= pos2.
func positionLessOrEqual(pos1, pos2 protocol.Position) bool {
	if pos1.Line < pos2.Line {
		return true
	}
	if pos1.Line == pos2.Line && pos1.Character <= pos2.Character {
		return true
	}
	return false
}
