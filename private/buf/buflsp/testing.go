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
	"sort"
	"strings"

	"go.lsp.dev/protocol"
)

// ApplyTextEdits applies a sequence of LSP TextEdits to the given text.
// Edits are applied in reverse position order (rightmost/bottom-most first) to maintain correct positions.
// This is a test helper shared across multiple test files.
func ApplyTextEdits(text string, edits []protocol.TextEdit) string {
	// Sort edits in reverse position order (bottom-right to top-left)
	// so that applying them doesn't invalidate subsequent edit positions
	sortedEdits := make([]protocol.TextEdit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		// Sort in reverse order: later positions first
		if sortedEdits[i].Range.Start.Line != sortedEdits[j].Range.Start.Line {
			return sortedEdits[i].Range.Start.Line > sortedEdits[j].Range.Start.Line
		}
		return sortedEdits[i].Range.Start.Character > sortedEdits[j].Range.Start.Character
	})

	lines := strings.Split(text, "\n")
	// Apply edits in reverse position order to maintain line/character positions
	for i := 0; i < len(sortedEdits); i++ {
		edit := sortedEdits[i]
		startLine := int(edit.Range.Start.Line)
		startChar := int(edit.Range.Start.Character)
		endLine := int(edit.Range.End.Line)
		endChar := int(edit.Range.End.Character)

		if startLine < 0 || startLine > len(lines) {
			continue
		}
		if endLine < 0 || endLine > len(lines) {
			endLine = len(lines)
		}

		// LSP TextEdit semantics: Range.End is exclusive at character level
		// Get the prefix (part of start line before the edit)
		prefix := ""
		if startLine < len(lines) && startChar <= len(lines[startLine]) {
			prefix = lines[startLine][:startChar]
		} else if startLine == len(lines) {
			// Inserting at end of file - no prefix
			prefix = ""
		}

		// Get the suffix (part of end line after the edit)
		// If endChar is 0, we're at the start of endLine, so no suffix from that line
		suffix := ""
		if endChar > 0 && endLine < len(lines) {
			if endChar <= len(lines[endLine]) {
				suffix = lines[endLine][endChar:]
			}
		} else if startLine == endLine && endChar == 0 {
			// Pure insertion at the beginning of a line - need the whole line as suffix
			if endLine < len(lines) {
				suffix = lines[endLine]
			}
		}

		// Determine which lines to delete
		// For a pure insertion (start == end), the line gets incorporated into replacement via suffix
		// So we need to delete it from the original lines array
		// Otherwise, we delete from startLine to endLine (inclusive or exclusive depending on endChar)
		deleteEndLine := endLine
		if startLine == len(lines) {
			// Inserting at end of file - no lines to delete, just append
			deleteEndLine = startLine - 1
		} else if startLine == endLine && startChar == endChar {
			// Pure insertion - the line is preserved in suffix, so delete it from lines array
			deleteEndLine = startLine
		} else if endChar == 0 && endLine > startLine {
			// Range ends at the beginning of endLine, so don't include endLine in deletion
			deleteEndLine = endLine - 1
		}

		// Split the new text into lines
		newText := edit.NewText
		var newLines []string
		endsWithNewline := false
		if newText != "" {
			newLines = strings.Split(newText, "\n")
			// If newText ends with \n, split gives us a trailing empty string.
			// This indicates that subsequent content should be on a new line.
			endsWithNewline = strings.HasSuffix(newText, "\n")
			if endsWithNewline && len(newLines) > 0 && newLines[len(newLines)-1] == "" {
				newLines = newLines[:len(newLines)-1]
			}
		}

		// Build the replacement
		var replacement []string
		if len(newLines) == 0 {
			// No new content
			combined := prefix + suffix
			// Create a line if: there's content, we're doing a mid-line edit, or it's a pure insertion
			if combined != "" || (startChar > 0 || endChar > 0) || (startLine == endLine && startChar == endChar) {
				replacement = []string{combined}
			} else {
				// Deleting full lines with no replacement
				replacement = []string{}
			}
		} else {
			// Add prefix to first line
			replacement = make([]string, len(newLines))
			copy(replacement, newLines)
			replacement[0] = prefix + replacement[0]
			// Add suffix: if newText ended with \n, suffix goes on a new line
			// Otherwise, append it to the last line
			if suffix != "" {
				if endsWithNewline {
					replacement = append(replacement, suffix)
				} else {
					replacement[len(replacement)-1] = replacement[len(replacement)-1] + suffix
				}
			} else if startLine == endLine && startChar == endChar && endsWithNewline {
				// Pure insertion at start of line where newText ends with newline
				// Even if the original line is blank (suffix == ""), we should preserve it
				replacement = append(replacement, "")
			}
		}

		// Replace lines[startLine:deleteEndLine+1] with replacement
		lines = append(lines[:startLine], append(replacement, lines[deleteEndLine+1:]...)...)
	}
	return strings.Join(lines, "\n")
}

// RangesOverlap returns true if two protocol ranges overlap.
// Two ranges overlap if they share any character position in the document.
// This is a test helper shared across multiple test files.
func RangesOverlap(r1, r2 protocol.Range) bool {
	return PositionLess(r1.Start, r2.End) && PositionLess(r2.Start, r1.End)
}

// PositionLess returns true if p1 comes before p2 in the document.
// Positions are compared first by line, then by character within the line.
// This is a test helper shared across multiple test files.
func PositionLess(p1, p2 protocol.Position) bool {
	if p1.Line != p2.Line {
		return p1.Line < p2.Line
	}
	return p1.Character < p2.Character
}

// FormatRange formats a protocol range as a human-readable string.
// The format is [startLine:startChar-endLine:endChar].
// This is a test helper shared across multiple test files.
func FormatRange(r protocol.Range) string {
	return fmt.Sprintf("[%d:%d-%d:%d]",
		r.Start.Line, r.Start.Character,
		r.End.Line, r.End.Character)
}
