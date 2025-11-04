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

// This file implements debugging utilities for the LSP, including token stream
// and AST visualization.

package buflsp

import (
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/token"
)

// formatTokenStream formats the token stream for a file into a human-readable string.
// If hasRange is true, only tokens within [startOffset, endOffset] are shown.
// Format matches the TSV format from experimental/parser/lex_test.go
func formatTokenStream(file *file, hasRange bool, startOffset, endOffset int) string {
	if file.ir.IsZero() {
		return "No AST available for this file\n"
	}

	var out strings.Builder

	stream := file.ir.AST().Context().Stream()
	if stream == nil {
		return "No token stream available\n"
	}

	// Header line matching lex_test.go format
	out.WriteString("#\t\tkind\t\tkeyword\t\toffsets\t\tlinecol\t\ttext\n")

	if hasRange {
		// Find the first token at or after startOffset
		before, after := stream.Around(startOffset)

		// Start from the token at/after the start offset
		startToken := after
		if startToken.IsZero() && !before.IsZero() {
			startToken = before
		}

		if startToken.IsZero() {
			return "No tokens found in selected range\n"
		}

		// Print tokens from startToken until we're past endOffset
		cursor := token.NewCursorAt(startToken)
		for tok := startToken; !tok.IsZero(); tok = cursor.Next() {
			span := tok.Span()
			// Stop if we're completely past the end offset
			if span.Start > endOffset {
				break
			}
			// Only print if token overlaps with the range
			if span.End >= startOffset {
				printToken(&out, tok, stream)
			}
		}
	} else {
		// Show all tokens
		for tok := range stream.All() {
			printToken(&out, tok, stream)
		}
	}

	return out.String()
}

// printToken formats a single token in TSV format matching lex_test.go
func printToken(out *strings.Builder, tok token.Token, stream *token.Stream) {
	if tok.IsZero() {
		return
	}

	sp := tok.Span()
	start := stream.Location(sp.Start, report.TermWidth)

	// Format: ID \t\t Kind \t\t Keyword \t\t Offsets \t\t LineCol \t\t Text
	fmt.Fprintf(
		out, "%v\t\t%v\t\t%#v\t\t%03d:%03d\t\t%03d:%03d\t\t%q",
		int32(tok.ID())-1,
		tok.Kind(), tok.Keyword(),
		sp.Start, sp.End,
		start.Line, start.Column,
		tok.Text(),
	)

	// Add extra info for specific token types
	switch tok.Kind() {
	case token.Number:
		n := tok.AsNumber()
		v := n.Value()
		if v.IsInt() {
			fmt.Fprintf(out, "\t\tnum:%.0f", n.Value())
		} else {
			fmt.Fprintf(out, "\t\tnum:%g", n.Value())
		}

		if prefix := n.Prefix().Text(); prefix != "" {
			fmt.Fprintf(out, "\t\tpre:%q", prefix)
		}

		if suffix := n.Suffix().Text(); suffix != "" {
			fmt.Fprintf(out, "\t\tsuf:%q", suffix)
		}

	case token.String:
		s := tok.AsString()
		fmt.Fprintf(out, "\t\tstring:%q", s.Text())

		if prefix := s.Prefix().Text(); prefix != "" {
			fmt.Fprintf(out, "\t\tpre:%q", prefix)
		}
	}

	// Show open/close relationships for paired tokens
	if a, b := tok.StartEnd(); a != b {
		if tok == a {
			fmt.Fprintf(out, "\t\tclose:%v", b.ID())
		} else {
			fmt.Fprintf(out, "\t\topen:%v", a.ID())
		}
	}

	out.WriteByte('\n')
}
