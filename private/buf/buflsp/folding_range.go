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
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"go.lsp.dev/protocol"
)

// foldingRange generates folding ranges for a file.
// It finds messages, services, enums, and multi-line comment blocks.
func (s *server) foldingRange(file *file) []protocol.FoldingRange {
	if file.ir == nil {
		return nil
	}

	var ranges []protocol.FoldingRange

	// Add folding ranges for all types (messages, enums)
	for irType := range seq.Values(file.ir.AllTypes()) {
		if span := irType.AST().Span(); !span.IsZero() {
			ranges = append(ranges, createFoldingRange(span, protocol.RegionFoldingRange))
		}

		// Add folding ranges for oneofs within messages
		for oneof := range seq.Values(irType.Oneofs()) {
			if span := oneof.AST().Span(); !span.IsZero() {
				ranges = append(ranges, createFoldingRange(span, protocol.RegionFoldingRange))
			}
		}
	}

	// Add folding ranges for services
	for service := range seq.Values(file.ir.Services()) {
		if span := service.AST().Span(); !span.IsZero() {
			ranges = append(ranges, createFoldingRange(span, protocol.RegionFoldingRange))
		}

		// Add folding ranges for individual RPC methods
		for method := range seq.Values(service.Methods()) {
			if span := method.AST().Span(); !span.IsZero() {
				ranges = append(ranges, createFoldingRange(span, protocol.RegionFoldingRange))
			}
		}
	}

	// Add folding ranges for extend blocks
	for extend := range seq.Values(file.ir.Extends()) {
		if span := extend.AST().Span(); !span.IsZero() {
			ranges = append(ranges, createFoldingRange(span, protocol.RegionFoldingRange))
		}
	}

	// Add folding ranges for multi-line comments
	astFile := file.ir.AST()
	if astFile != nil {
		ranges = append(ranges, s.commentFoldingRanges(astFile)...)
		ranges = append(ranges, s.importGroupFoldingRanges(astFile)...)
		ranges = append(ranges, s.optionBlockFoldingRanges(astFile)...)
	}

	return ranges
}

// commentFoldingRanges creates folding ranges for multi-line comment blocks.
func (s *server) commentFoldingRanges(astFile *ast.File) []protocol.FoldingRange {
	var ranges []protocol.FoldingRange
	var commentBlock []token.Token

	stream := astFile.Stream()
	if stream == nil {
		return ranges
	}

	// Collect all comment tokens first
	var commentTokens []token.Token
	for tok := range stream.All() {
		if tok.Kind() == token.Comment {
			commentTokens = append(commentTokens, tok)
		}
	}

	// Group consecutive comment tokens (on consecutive or same lines)
	for _, tok := range commentTokens {
		if len(commentBlock) == 0 {
			commentBlock = append(commentBlock, tok)
			continue
		}

		lastComment := commentBlock[len(commentBlock)-1]
		lastLine := lastComment.Span().EndLoc().Line
		currentLine := tok.Span().StartLoc().Line

		// If comments are on consecutive lines (or same line), group them
		if currentLine <= lastLine+1 {
			commentBlock = append(commentBlock, tok)
		} else {
			// End of comment block - create folding range if multi-line
			if len(commentBlock) > 1 {
				firstComment := commentBlock[0]
				lastComment := commentBlock[len(commentBlock)-1]
				span := source.Join(firstComment, lastComment)
				ranges = append(ranges, createFoldingRange(span, protocol.CommentFoldingRange))
			}
			// Start new block
			commentBlock = []token.Token{tok}
		}
	}

	// Handle final comment block
	if len(commentBlock) > 1 {
		firstComment := commentBlock[0]
		lastComment := commentBlock[len(commentBlock)-1]
		span := source.Join(firstComment, lastComment)
		ranges = append(ranges, createFoldingRange(span, protocol.CommentFoldingRange))
	}

	return ranges
}

// importGroupFoldingRanges creates folding ranges for groups of consecutive imports.
func (s *server) importGroupFoldingRanges(astFile *ast.File) []protocol.FoldingRange {
	var ranges []protocol.FoldingRange
	var importGroup []ast.DeclImport

	// Collect all imports and group consecutive ones
	for imp := range astFile.Imports() {
		if len(importGroup) == 0 {
			importGroup = append(importGroup, imp)
			continue
		}

		lastImport := importGroup[len(importGroup)-1]
		lastLine := lastImport.Span().EndLoc().Line
		currentLine := imp.Span().StartLoc().Line

		// If imports are on consecutive lines (or close together), group them
		// Allow up to 1 blank line between imports to stay in the same group
		if currentLine <= lastLine+2 {
			importGroup = append(importGroup, imp)
		} else {
			// End of import group - create folding range if we have multiple imports
			if len(importGroup) > 1 {
				firstImport := importGroup[0]
				lastImport := importGroup[len(importGroup)-1]
				span := source.Join(firstImport, lastImport)
				ranges = append(ranges, createFoldingRange(span, protocol.ImportsFoldingRange))
			}
			// Start new group
			importGroup = []ast.DeclImport{imp}
		}
	}

	// Handle final import group
	if len(importGroup) > 1 {
		firstImport := importGroup[0]
		lastImport := importGroup[len(importGroup)-1]
		span := source.Join(firstImport, lastImport)
		ranges = append(ranges, createFoldingRange(span, protocol.ImportsFoldingRange))
	}

	return ranges
}

// optionBlockFoldingRanges creates folding ranges for multi-line option blocks.
// These are options in square brackets [...] that span multiple lines, like:
//
//	string field = 1 [
//	  option1 = value1,
//	  option2 = value2
//	];
func (s *server) optionBlockFoldingRanges(astFile *ast.File) []protocol.FoldingRange {
	var ranges []protocol.FoldingRange

	stream := astFile.Stream()
	if stream == nil {
		return ranges
	}

	// Track seen spans to avoid duplicates from nested token traversal
	type spanKey struct {
		startLine int
		endLine   int
	}
	seen := make(map[spanKey]bool)

	// Iterate through all tokens looking for multi-line bracket pairs
	for tok := range stream.All() {
		// Only handle fused bracket tokens, which are used for option blocks
		if tok.Keyword() == keyword.Brackets {
			span := tok.Span()
			startLine := span.StartLoc().Line
			endLine := span.EndLoc().Line
			// Only add if it spans multiple lines and we haven't seen it yet
			if endLine > startLine {
				key := spanKey{startLine: startLine, endLine: endLine}
				if !seen[key] {
					seen[key] = true
					ranges = append(ranges, createFoldingRange(span, protocol.RegionFoldingRange))
				}
			}
		}
	}

	return ranges
}

// createFoldingRange creates a protocol.FoldingRange from a source span.
func createFoldingRange(span source.Span, kind protocol.FoldingRangeKind) protocol.FoldingRange {
	startLoc := span.StartLoc()
	endLoc := span.EndLoc()

	// Convert from 1-based to 0-based line numbers
	return protocol.FoldingRange{
		StartLine: uint32(startLoc.Line - 1),
		EndLine:   uint32(endLoc.Line - 1),
		Kind:      kind,
	}
}
