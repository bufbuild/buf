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

	tests := []struct {
		name          string
		protoFile     string
		expectedLinks []expectedLink
	}{
		{
			name:      "local_import_and_comment_urls",
			protoFile: "testdata/document_link/main.proto",
			expectedLinks: []expectedLink{
				{
					line:        4,  // import "types.proto" on line 5 (0-indexed line 4)
					startChar:   7,  // Start of "types.proto" (including opening quote)
					endChar:     20, // End of "types.proto" (after closing quote)
					description: "local import to types.proto",
					targetType:  linkTargetTypeLocal,
					localPath:   "testdata/document_link/types.proto",
				},
				{
					line:        7, // https://example.com/docs on line 8
					startChar:   7,
					endChar:     31,
					description: "comment URL 1",
					targetType:  linkTargetTypeURL,
					targetURL:   "https://example.com/docs",
				},
				{
					line:        8, // https://github.com/example/repo on line 9
					startChar:   14,
					endChar:     45,
					description: "comment URL 2",
					targetType:  linkTargetTypeURL,
					targetURL:   "https://github.com/example/repo",
				},
				{
					line:        11, // https://example.com/status on line 12
					startChar:   43,
					endChar:     69,
					description: "comment URL 3",
					targetType:  linkTargetTypeURL,
					targetURL:   "https://example.com/status",
				},
			},
		},
		{
			name:      "wkt_imports",
			protoFile: "testdata/document_link/wkt.proto",
			expectedLinks: []expectedLink{
				{
					line:        4,  // import "google/protobuf/timestamp.proto" on line 5
					startChar:   7,  // Start of "google/protobuf/timestamp.proto" (including opening quote)
					endChar:     40, // End of "google/protobuf/timestamp.proto" (after closing quote)
					description: "WKT Timestamp import",
					targetType:  linkTargetTypeURL,
					targetURL:   "https://buf.build/protocolbuffers/wellknowntypes/file/main:google/protobuf/timestamp.proto",
				},
				{
					line:        5,  // import "google/protobuf/duration.proto" on line 6
					startChar:   7,  // Start of "google/protobuf/duration.proto" (including opening quote)
					endChar:     39, // End of "google/protobuf/duration.proto" (after closing quote)
					description: "WKT Duration import",
					targetType:  linkTargetTypeURL,
					targetURL:   "https://buf.build/protocolbuffers/wellknowntypes/file/main:google/protobuf/duration.proto",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			protoPath, err := filepath.Abs(tt.protoFile)
			require.NoError(t, err)

			clientJSONConn, testURI := setupLSPServer(t, protoPath)

			var links []protocol.DocumentLink
			_, err = clientJSONConn.Call(ctx, protocol.MethodTextDocumentDocumentLink, protocol.DocumentLinkParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: testURI,
				},
			}, &links)
			require.NoError(t, err)

			require.Len(t, links, len(tt.expectedLinks), "unexpected number of document links")

			for i, expected := range tt.expectedLinks {
				link := links[i]
				assert.Equal(t, expected.line, link.Range.Start.Line, "link %d (%s): wrong start line", i, expected.description)
				assert.Equal(t, expected.startChar, link.Range.Start.Character, "link %d (%s): wrong start character", i, expected.description)
				assert.Equal(t, expected.line, link.Range.End.Line, "link %d (%s): wrong end line", i, expected.description)
				assert.Equal(t, expected.endChar, link.Range.End.Character, "link %d (%s): wrong end character", i, expected.description)

				switch expected.targetType {
				case linkTargetTypeLocal:
					localPath, err := filepath.Abs(expected.localPath)
					require.NoError(t, err)
					expectedURI := uri.New(localPath)
					assert.Equal(t, expectedURI, link.Target, "link %d (%s): wrong target", i, expected.description)
				case linkTargetTypeURL:
					assert.Equal(t, protocol.DocumentURI(expected.targetURL), link.Target, "link %d (%s): wrong target", i, expected.description)
				}
			}

			// Verify no overlapping ranges
			assertNoOverlappingRanges(t, links)
		})
	}
}

type linkTargetType int

const (
	linkTargetTypeLocal linkTargetType = iota
	linkTargetTypeURL
)

type expectedLink struct {
	line        uint32
	startChar   uint32 // expected starting character position
	endChar     uint32 // expected ending character position
	description string
	targetType  linkTargetType
	localPath   string // used when targetType is linkTargetTypeLocal
	targetURL   string // used when targetType is linkTargetTypeURL
}

// assertNoOverlappingRanges verifies that no two document link ranges overlap.
func assertNoOverlappingRanges(t *testing.T, links []protocol.DocumentLink) {
	t.Helper()

	for i := range links {
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
