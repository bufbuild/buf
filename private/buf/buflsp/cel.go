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
	"strconv"
	"strings"
	"unicode/utf8"

	celast "cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/operators"
	"github.com/bufbuild/protocompile/experimental/source"
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

// protoEscapeWidths returns the number of source bytes consumed by the proto
// string escape sequence beginning at s[i] (where s[i] == '\\') and the number
// of bytes that escape decodes to in the string value.
// Handles simple escapes (\n, \\, etc.), hex (\xNN), Unicode (\uNNNN, \UNNNNNNNN),
// and octal (\NNN) as defined by the proto language specification.
//
// Simple, hex, and octal escapes each decode to one byte. Unicode escapes
// decode to the UTF-8 encoding of their code point, which protocompile writes
// with strings.Builder.WriteRune: utf8.RuneLen bytes for a valid code point and
// the three-byte U+FFFD for an invalid one.
func protoEscapeWidths(s string, i int) (sourceWidth, decodedWidth int) {
	if i+1 >= len(s) {
		return 1, 1
	}
	switch s[i+1] {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '\'', '"', '?':
		return 2, 1
	case 'x', 'X':
		// Hex escape: \xNN — consume up to 2 hex digits.
		end := i + 2
		for end < len(s) && end < i+4 && isProtoHexDigit(s[end]) {
			end++
		}
		return end - i, 1
	case 'u':
		// Unicode escape: \uNNNN — consume up to 4 hex digits.
		return protoUnicodeEscapeWidths(s, i, 4)
	case 'U':
		// Unicode escape: \UNNNNNNNN — consume up to 8 hex digits.
		return protoUnicodeEscapeWidths(s, i, 8)
	default:
		// Octal: \NNN — 1 to 3 octal digits.
		if s[i+1] >= '0' && s[i+1] <= '7' {
			end := i + 2
			for end < len(s) && end < i+4 && s[end] >= '0' && s[end] <= '7' {
				end++
			}
			return end - i, 1
		}
		return 2, 1
	}
}

// protoUnicodeEscapeWidths returns the source and decoded widths of the \u or
// \U escape beginning at s[i], which takes up to maxDigits hex digits.
func protoUnicodeEscapeWidths(s string, i int, maxDigits int) (sourceWidth, decodedWidth int) {
	end := i + 2
	for end < len(s) && end < i+2+maxDigits && isProtoHexDigit(s[end]) {
		end++
	}
	// A missing or short digit sequence is an invalid escape that protocompile
	// reports, but it still writes the (possibly zero) code point it read, so
	// the decoded width follows the same rule.
	value, _ := strconv.ParseUint(s[i+2:end], 16, 32)
	codePoint := rune(value)
	if !utf8.ValidRune(codePoint) {
		codePoint = utf8.RuneError
	}
	return end - i, utf8.RuneLen(codePoint)
}

// isProtoHexDigit reports whether c is a valid hex digit (0-9, a-f, A-F).
//
// Equivalent to the unexported isHex in github.com/bufbuild/protocompile/linker.
func isProtoHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// createCELSpan creates a source.Span for a CEL token given its start and end byte offsets within the
// CEL expression string.
// exprLiteralSpan is the span of the proto string literal(s) containing the CEL expression (including quotes).
// It handles both single-line and multi-line (adjacent proto string literals) spans uniformly by scanning for
// opening quotes, then walking the string content while accounting for proto escape sequences. Each escape
// advances the CEL offset by the number of bytes it decodes to, so the span is correct both after an escape
// and for a token, such as a string literal, that contains one.
func createCELSpan(celStart, celEnd int, exprLiteralSpan source.Span) source.Span {
	spanText := exprLiteralSpan.Text()
	celPos := 0
	fileStart := -1

	for i := 0; i < len(spanText); i++ {
		if spanText[i] != '"' {
			continue
		}
		// Scan the content of this string literal up to its unescaped closing quote.
		i++
		for i < len(spanText) && spanText[i] != '"' {
			sourceWidth, decodedWidth := 1, 1
			if spanText[i] == '\\' && i+1 < len(spanText) {
				sourceWidth, decodedWidth = protoEscapeWidths(spanText, i)
			}
			if fileStart < 0 && celStart < celPos+decodedWidth {
				fileStart = exprLiteralSpan.Start + i
			}
			if fileStart >= 0 && celPos >= celEnd {
				return source.Span{File: exprLiteralSpan.File, Start: fileStart, End: exprLiteralSpan.Start + i}
			}
			celPos += decodedWidth
			i += sourceWidth
		}
		// The token may end at the closing quote of this literal.
		if fileStart >= 0 && celPos >= celEnd {
			return source.Span{File: exprLiteralSpan.File, Start: fileStart, End: exprLiteralSpan.Start + i}
		}
	}

	return source.Span{}
}

// fileByteOffsetToCELOffset is the inverse of createCELSpan: given a file byte
// offset that falls within exprLiteralSpan, it returns the corresponding byte
// offset within the concatenated CEL expression string. A file offset anywhere
// inside a proto escape sequence maps to the first byte the escape decodes to.
// Returns -1 if fileByteOffset is not inside any string content of the span
// (e.g. it lands on a quote character, whitespace between literals, or is out
// of range).
func fileByteOffsetToCELOffset(fileByteOffset int, exprLiteralSpan source.Span) int {
	spanText := exprLiteralSpan.Text()
	celPos := 0

	for i := 0; i < len(spanText); i++ {
		if spanText[i] != '"' {
			continue
		}
		i++
		for i < len(spanText) && spanText[i] != '"' {
			sourceWidth, decodedWidth := 1, 1
			if spanText[i] == '\\' && i+1 < len(spanText) {
				sourceWidth, decodedWidth = protoEscapeWidths(spanText, i)
			}
			fileStart := exprLiteralSpan.Start + i
			if fileByteOffset >= fileStart && fileByteOffset < fileStart+sourceWidth {
				return celPos
			}
			celPos += decodedWidth
			i += sourceWidth
		}
	}
	return -1
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
// Therefore Stop-Start equals the byte length of the token, not its rune length.
// This gives correct byte bounds for both ASCII and non-ASCII tokens.
func celOffsetRangeToByteRange(exprString string, r celast.OffsetRange) (byteStart, byteStop int) {
	byteStart = celRuneOffsetToByteOffset(exprString, r.Start)
	byteStop = byteStart + int(r.Stop-r.Start) // Stop-Start is byte length of the token
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
	exprLiteralSpan source.Span,
) source.Span {
	start, end := findMethodNameAfterDot(targetByteOffset, name, exprString)
	if start < 0 {
		return source.Span{}
	}
	return createCELSpan(start, end, exprLiteralSpan)
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
