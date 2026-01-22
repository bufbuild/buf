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

package buflsp

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf16"

	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
)

// formatRange formats a range within a proto file, expanding to declaration boundaries.
//
// The approach:
//  1. Find top-level declarations in the parsed AST that overlap with the range
//  2. Format the ENTIRE file (required for consistency, e.g., aligned field numbers across the file)
//  3. Parse the formatted file and match declarations by simple identifiers to find where they ended up
//  4. Return just the formatted text for those declarations
//
// This ensures that range formatting produces the same output as full-file formatting for the
// affected declarations, maintaining consistency with formatting conventions.
func formatRange(
	parsed *ast.FileNode,
	fileText string,
	filename string,
	startOffset, endOffset int,
) (formattedSegment string, origStart, origEnd int, err error) {
	// Find top-level declarations that overlap with the selected range.
	// We only format complete declarations, not arbitrary text ranges.
	var startDecl, endDecl ast.Node
	var startID, endID *declIdentifier
	positionCounters := make(map[string]int)

	for _, decl := range parsed.Decls {
		id := getDeclIdentifier(decl, positionCounters)
		if id == nil {
			continue // syntax, package, import, option - not formattable
		}

		nodeInfo := parsed.NodeInfo(decl)
		declStart := nodeInfo.Start().Offset
		declEnd := nodeInfo.End().Offset

		// Check if this declaration overlaps with [startOffset, endOffset)
		if declStart < endOffset && declEnd > startOffset {
			if startDecl == nil {
				startDecl = decl
				startID = id
			}
			endDecl = decl
			endID = id
		}
	}

	if startDecl == nil || endDecl == nil || startID == nil || endID == nil {
		return "", 0, 0, fmt.Errorf("no formattable declarations found in range")
	}

	// Format the entire file to maintain consistency (e.g., field alignment).
	var out strings.Builder
	if err := formatProtoFile(&out, parsed); err != nil {
		return "", 0, 0, err
	}
	formattedText := out.String()

	// Early return if no changes needed
	if formattedText == fileText {
		return "", 0, 0, nil
	}

	// Find where the same declarations ended up in the formatted file
	formattedStartOffset, formattedEndOffset, err := findMatchingDeclsInFormatted(
		formattedText,
		filename,
		startID,
		endID,
	)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to match declarations: %w", err)
	}

	// Extract the formatted text segment
	formattedSegment = formattedText[formattedStartOffset:formattedEndOffset]

	// Return the offsets in the original file
	startNodeInfo := parsed.NodeInfo(startDecl)
	endNodeInfo := parsed.NodeInfo(endDecl)
	origStart = startNodeInfo.Start().Offset
	origEnd = endNodeInfo.End().Offset

	return formattedSegment, origStart, origEnd, nil
}

// formatFullFile formats an entire proto file and returns the edit, or nil if no changes needed.
func formatFullFile(fileText, filename string) (*protocol.TextEdit, error) {
	parsed, err := parseProtoFileSimple(filename, fileText)
	if err != nil {
		return nil, err
	}

	var out strings.Builder
	if err := formatProtoFile(&out, parsed); err != nil {
		return nil, err
	}
	newText := out.String()

	// No changes needed
	if newText == fileText {
		return nil, nil
	}

	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   fileEndPosition(fileText),
		},
		NewText: newText,
	}, nil
}

// findMatchingDeclsInFormatted finds the byte offsets in the formatted file that correspond
// to the given declarations from the original file, by parsing the formatted file and matching by identifier.
func findMatchingDeclsInFormatted(
	formattedText string,
	filename string,
	startID, endID *declIdentifier,
) (startOffset, endOffset int, err error) {
	formattedParsed, err := parseProtoFileSimple(filename, formattedText)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse formatted file: %w", err)
	}

	// Build declaration map for the formatted file
	formattedDeclMap := buildDeclMap(formattedParsed)

	// Find matching declarations in the formatted AST by identifier
	startDecl, startFound := formattedDeclMap[*startID]
	endDecl, endFound := formattedDeclMap[*endID]

	if !startFound || !endFound {
		return 0, 0, fmt.Errorf("could not find matching declarations (start=%v, end=%v)", startFound, endFound)
	}

	startNodeInfo := formattedParsed.NodeInfo(startDecl)
	endNodeInfo := formattedParsed.NodeInfo(endDecl)
	startOffset = startNodeInfo.Start().Offset
	endOffset = endNodeInfo.End().Offset

	return startOffset, endOffset, nil
}

// parseProtoFileSimple parses a proto file and returns just the AST, without building descriptors.
// This is used for simple formatting operations that don't need semantic information.
func parseProtoFileSimple(filename, fileText string) (*ast.FileNode, error) {
	var errorsWithPos []reporter.ErrorWithPos
	handler := reporter.NewHandler(reporter.NewReporter(
		func(errorWithPos reporter.ErrorWithPos) error {
			errorsWithPos = append(errorsWithPos, errorWithPos)
			return nil
		},
		func(errorWithPos reporter.ErrorWithPos) {},
	))
	parsed, _ := parser.Parse(filename, strings.NewReader(fileText), handler)
	if len(errorsWithPos) > 0 {
		return nil, fmt.Errorf("cannot parse file %q, %v error(s) found", filename, len(errorsWithPos))
	}
	if parsed == nil {
		return nil, fmt.Errorf("failed to parse file %q", filename)
	}
	return parsed, nil
}

// formatProtoFile formats a parsed proto file AST and writes the result to the provided writer.
// Returns an error if formatting fails.
func formatProtoFile(w io.Writer, parsed *ast.FileNode) error {
	return bufformat.FormatFileNode(w, parsed)
}

// declIdentifier uniquely identifies a top-level declaration for matching purposes.
// This is simpler than FQN-based matching and doesn't require parser.Result or descriptors.
type declIdentifier struct {
	typ      string // "message", "enum", "service", "extend"
	name     string // simple name (or extendee for extend blocks)
	position int    // position among declarations of the same type (for disambiguation)
}

// getDeclIdentifier extracts a simple identifier from a declaration node.
// Returns nil for non-formattable declarations (syntax, package, import, option).
func getDeclIdentifier(node ast.Node, positionCounters map[string]int) *declIdentifier {
	var typ, name string

	switch n := node.(type) {
	case *ast.MessageNode:
		if n.Name == nil {
			return nil
		}
		typ, name = "message", string(n.Name.AsIdentifier())
	case *ast.EnumNode:
		if n.Name == nil {
			return nil
		}
		typ, name = "enum", string(n.Name.AsIdentifier())
	case *ast.ServiceNode:
		if n.Name == nil {
			return nil
		}
		typ, name = "service", string(n.Name.AsIdentifier())
	case *ast.ExtendNode:
		if n.Extendee == nil {
			return nil
		}
		typ, name = "extend", string(n.Extendee.AsIdentifier())
	default:
		return nil // syntax, package, import, option - not formattable
	}

	// Use position counter for disambiguation (e.g., multiple extends of the same type)
	key := typ + ":" + name
	position := positionCounters[key]
	positionCounters[key]++

	return &declIdentifier{typ: typ, name: name, position: position}
}

// buildDeclMap builds a map from declaration identifiers to AST nodes.
// This allows matching declarations between original and formatted files without needing FQNs.
func buildDeclMap(parsed *ast.FileNode) map[declIdentifier]ast.Node {
	declMap := make(map[declIdentifier]ast.Node)
	positionCounters := make(map[string]int)

	for _, decl := range parsed.Decls {
		if id := getDeclIdentifier(decl, positionCounters); id != nil {
			declMap[*id] = decl
		}
	}

	return declMap
}

// positionToOffset converts a protocol position to a byte offset in the file.
func positionToOffset(file *file, position protocol.Position) int {
	positionLocation := file.file.InverseLocation(
		int(position.Line)+1,
		int(position.Character)+1,
		positionalEncoding,
	)
	return positionLocation.Offset
}

// clampPositionToFile ensures a position doesn't exceed the file bounds.
func clampPositionToFile(file *file, pos protocol.Position) protocol.Position {
	endLine := uint32(strings.Count(file.file.Text(), "\n"))
	if pos.Line > endLine {
		return protocol.Position{Line: endLine, Character: 0}
	}
	return pos
}

// offsetsToRange converts byte offsets to a protocol Range.
func offsetsToRange(file *file, startOffset, endOffset int) protocol.Range {
	startLoc := file.file.Location(startOffset, positionalEncoding)
	endLoc := file.file.Location(endOffset, positionalEncoding)
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(startLoc.Line - 1),
			Character: uint32(startLoc.Column - 1),
		},
		End: protocol.Position{
			Line:      uint32(endLoc.Line - 1),
			Character: uint32(endLoc.Column - 1),
		},
	}
}

// fileEndPosition calculates the end position of a file using UTF-16 encoding.
func fileEndPosition(fileText string) protocol.Position {
	endLine := strings.Count(fileText, "\n")
	endCharacter := 0
	lastLineStart := strings.LastIndexByte(fileText, '\n') + 1
	for _, char := range fileText[lastLineStart:] {
		endCharacter += utf16.RuneLen(char)
	}
	return protocol.Position{
		Line:      uint32(endLine),
		Character: uint32(endCharacter),
	}
}
