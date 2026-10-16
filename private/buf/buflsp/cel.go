// Copyright 2020-2026 Buf Technologies, Inc.
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
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	celast "cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token"
)

// celIsIdentChar reports whether c is a valid CEL identifier character:
// ASCII letter, digit, or underscore. This mirrors the CEL spec's IDENT token
// rule; there is no public API for this in cel-go.
func celIsIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// isCELKeyword returns true if the identifier is a CEL reserved keyword.
// See https://github.com/google/cel-spec/blob/master/doc/langdef.md#syntax
func isCELKeyword(name string) bool {
	switch name {
	case "true", "false", "null", // literals
		"this", "now": // protovalidate special variables
		return true
	}
	return false
}

// isCELMacroFunction returns true if the function name is a CEL macro (comprehension or special function).
// See https://github.com/google/cel-spec/blob/master/doc/langdef.md#macros
func isCELMacroFunction(funcName string) bool {
	return funcName == operators.Has ||
		funcName == operators.All ||
		funcName == operators.Exists ||
		funcName == operators.ExistsOne ||
		funcName == operators.Map ||
		funcName == operators.Filter
	// Note: "size" is intentionally excluded as it's more commonly used as a method
}

// celOperatorSymbol maps CEL operator function names to their operator symbols.
// CEL represents operators as function calls with names like _&&_, _||_, _>_, etc.
// Returns the operator symbol and true if the function name represents an operator.
// See https://github.com/google/cel-spec/blob/master/doc/langdef.md#operators
func celOperatorSymbol(funcName string) (string, bool) {
	// Use cel-go's operators.FindReverse to get the display symbol.
	// Only use the result when the display name is non-empty; the ternary operator
	// (_?_:_) is registered but has an empty display name, so FindReverse returns
	// ("", true) for it, which we must not treat as a valid symbol.
	if symbol, found := operators.FindReverse(funcName); found && symbol != "" {
		return symbol, true
	}

	// Special case: ternary operator — use '?' as its hover symbol.
	if funcName == operators.Conditional {
		return "?", true
	}

	return "", false
}

// celSourceSpans holds, for each byte of a CEL expression, the span of the proto
// file the byte is written as, in the one or more adjacent string literals that
// make up the expression.
//
// A byte is written as a byte of a literal's contents, or as an escape sequence,
// which decodes to a single raw byte or to a rune in UTF-8. Every byte an escape
// sequence decodes to is written as the whole escape sequence.
type celSourceSpans []source.Span

// newCELSourceSpans returns the spans of an expression written as the string
// literal str, or nil if str is zero.
func newCELSourceSpans(str token.StringToken) celSourceSpans {
	if str.IsZero() {
		return nil
	}
	// After a string's first escape sequence, Escapes has an entry for every rune
	// that follows it, escaped or not. The entries that are not escape sequences
	// have empty spans.
	var escapes []token.Escape
	for escape := range seq.Values(str.Escapes()) {
		if escape.Len() > 0 {
			escapes = append(escapes, escape)
		}
	}
	file := str.RawContent().File
	spans := make(celSourceSpans, 0, len(str.Text()))
	plain := func(from, to int) {
		for i := from; i < to; i++ {
			spans = append(spans, file.Span(i, i+1))
		}
	}
	next := 0 // The first escape sequence not yet added.
	for _, part := range celStringParts(str) {
		content := part.RawContent()
		pos := content.Start
		for ; next < len(escapes) && escapes[next].Start < content.End; next++ {
			escape := escapes[next]
			plain(pos, escape.Start)
			// The lexer writes an escaped rune in UTF-8, and one that is not a valid
			// rune, which it reports, as U+FFFD; converting it to a string does the
			// same. A raw byte has a zero Rune, which also takes one byte.
			for range len(string(escape.Rune)) {
				spans = append(spans, escape.Span)
			}
			pos = escape.End
		}
		plain(pos, content.End)
	}
	if len(spans) != len(str.Text()) {
		// Shouldn't happen in practice.
		return nil
	}
	return spans
}

// celStringParts returns the string literals that str is concatenated from, in
// order, or str alone if it is a single literal.
func celStringParts(str token.StringToken) []token.StringToken {
	start, end := str.Token().StartEnd()
	parts := []token.StringToken{start.AsString()}
	if start == end {
		return parts
	}
	if children := str.Token().Children(); children != nil {
		for child := range children.Rest() {
			if child == end {
				break
			}
			if part := child.AsString(); !part.IsZero() {
				parts = append(parts, part)
			}
		}
	}
	return append(parts, end.AsString())
}

// span returns the span of the file that the expression's bytes [start, end) are
// written as, or the zero span if they are not all in the expression.
func (s celSourceSpans) span(start, end int) source.Span {
	if start < 0 || start > end || end > len(s) || start == len(s) {
		return source.Span{}
	}
	first := s[start]
	if start == end {
		return source.Span{File: first.File, Start: first.Start, End: first.Start}
	}
	return source.Span{File: first.File, Start: first.Start, End: s[end-1].End}
}

// offset returns the offset in the expression of the byte written at fileOffset,
// or -1 if no byte of the expression is written there, as for a quote or the
// space between adjacent literals. An offset inside an escape sequence gives the
// first byte the escape sequence decodes to.
func (s celSourceSpans) offset(fileOffset int) int {
	// The first byte whose span ends after fileOffset.
	i, _ := slices.BinarySearchFunc(s, fileOffset+1, func(span source.Span, end int) int {
		return cmp.Compare(span.End, end)
	})
	if i == len(s) || s[i].Start > fileOffset {
		return -1
	}
	return i
}

// celLocByteOffset converts a CEL source location (line, col) to a byte offset
// within exprString. line and col come from common.Location, which returns int,
// but ComputeOffset requires int32; the conversion is safe for any realistic
// CEL expression length.
func celLocByteOffset(line, col int, sourceInfo *celast.SourceInfo, exprString string) int {
	runeOffset := int32(col) + sourceInfo.ComputeOffset(int32(line), 0)
	return celRuneOffsetToByteOffset(exprString, runeOffset)
}

// celRuneOffsetToByteOffset converts a CEL source position (Unicode code point offset)
// to a UTF-8 byte offset within the expression string.
//
// CEL-go tracks source positions as Unicode code point (rune) offsets, not byte offsets.
// We need byte offsets to correctly slice Go strings and compute file spans.
func celRuneOffsetToByteOffset(s string, runeOffset int32) int {
	byteIdx := 0
	for runeIdx := int32(0); runeIdx < runeOffset && byteIdx < len(s); runeIdx++ {
		_, size := utf8.DecodeRuneInString(s[byteIdx:])
		byteIdx += size
	}
	return byteIdx
}

// celOffsetRangeToByteRange converts a CEL ast.OffsetRange to byte start and stop offsets.
//
// CEL stores OffsetRange.Start as a Unicode code point (rune) offset, but
// OffsetRange.Stop = Start + len(tokenText) where len uses Go's byte count (UTF-8 bytes).
// Therefore Stop-Start is the byte length of the token as the parser decoded it, which
// matches the source except where the source is not valid UTF-8: each invalid byte is
// decoded as U+FFFD, three bytes long. So the stop is found by walking the source from
// the start, counting each rune at its decoded length, until the token's length is used up.
func celOffsetRangeToByteRange(exprString string, r celast.OffsetRange) (byteStart, byteStop int) {
	byteStart = celRuneOffsetToByteOffset(exprString, r.Start)
	byteStop = byteStart
	for decoded := int32(0); decoded < r.Stop-r.Start && byteStop < len(exprString); {
		c, size := utf8.DecodeRuneInString(exprString[byteStop:])
		decoded += int32(utf8.RuneLen(c))
		byteStop += size
	}
	return
}

// findMethodNameAfterDot finds the position of a method name after a dot in a CEL expression.
// targetByteOffset is a byte offset (not rune offset) within exprString.
// Returns the start and end byte positions, or -1 if not found.
func findMethodNameAfterDot(targetByteOffset int, methodName string, exprString string) (start, end int) {
	searchStart := targetByteOffset
	searchRegion := exprString[searchStart:]

	// Search for ".methodName" pattern in the remaining string
	if idx := strings.Index(searchRegion, "."+methodName); idx >= 0 {
		// Found the pattern, return the position of just the method name (skip the dot)
		nameStart := searchStart + idx + 1 // +1 to skip the dot
		nameEnd := nameStart + len(methodName)
		return nameStart, nameEnd
	}
	return -1, -1
}

// findNameAfterDot searches for ".name" after targetByteOffset and returns the span of just the name (without the dot).
// targetByteOffset is a byte offset (not rune offset) within exprString.
// Returns zero span if not found.
func findNameAfterDot(
	targetByteOffset int,
	name string,
	exprString string,
	sourceSpans celSourceSpans,
) source.Span {
	start, end := findMethodNameAfterDot(targetByteOffset, name, exprString)
	if start < 0 {
		return source.Span{}
	}
	return sourceSpans.span(start, end)
}

// findStandaloneFunctionName calculates the position of a function name before an opening paren.
// celByteOffset is the byte offset (not rune offset) of the opening paren within exprString.
func findStandaloneFunctionName(celByteOffset int, funcName string, exprString string) (start, end int, found bool) {
	start = celByteOffset - len(funcName)
	end = start + len(funcName)
	if start >= 0 && end <= len(exprString) && exprString[start:end] == funcName {
		return start, end, true
	}
	return -1, -1, false
}
