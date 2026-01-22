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
	"strings"

	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/google/cel-go/common/operators"
)

// isCELKeyword returns true if the identifier is a CEL reserved keyword.
// See https://github.com/google/cel-spec/blob/master/doc/langdef.md#syntax
func isCELKeyword(name string) bool {
	keywords := map[string]bool{
		// Literals
		"true":  true,
		"false": true,
		"null":  true,
		// Special identifiers
		"this": true,
	}
	return keywords[name]
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

// createCELSpan creates a source.Span for a CEL token given its start and end offsets within the CEL expression.
// The exprLiteralSpan is the span of the string literal containing the CEL expression (including quotes).
func createCELSpan(celStart, celEnd int, exprLiteralSpan source.Span) source.Span {
	// Check if this is a multi-line span (covers multiple string literals)
	startLoc := exprLiteralSpan.StartLoc()
	endLoc := exprLiteralSpan.EndLoc()
	if startLoc.Line != endLoc.Line {
		// Multi-line span - use special handling for concatenated literals
		return createCELSpanMultiline(celStart, celEnd, exprLiteralSpan)
	}

	// For single-line literals, use simple offset calculation
	literalText := exprLiteralSpan.Text()
	if len(literalText) < 2 {
		return source.Span{}
	}

	// Calculate offset from start of file
	// exprLiteralSpan.Start is the byte offset of the opening quote
	// Add 1 for the quote, then add the CEL offset
	fileStart := exprLiteralSpan.Start + 1 + celStart
	fileEnd := exprLiteralSpan.Start + 1 + celEnd

	// Validate bounds
	if fileEnd > exprLiteralSpan.End {
		return source.Span{}
	}

	return source.Span{File: exprLiteralSpan.File, Start: fileStart, End: fileEnd}
}

// createCELSpanMultiline handles CEL token spans for multi-line expressions.
// Multi-line spans contain multiple quoted strings like: "first" "second"
// CEL concatenates them into: "firstsecond"
// This function maps CEL offsets back to file positions.
func createCELSpanMultiline(celStart, celEnd int, multilineSpan source.Span) source.Span {
	spanText := multilineSpan.Text()
	celPos := 0 // Current position in concatenated CEL string

	// Walk through the span text, tracking both file position and CEL position
	for i := 0; i < len(spanText); i++ {
		if spanText[i] != '"' {
			continue
		}

		// Found opening quote - scan the string content
		i++
		for i < len(spanText) && spanText[i] != '"' {
			// Check if we've found the token start
			if celPos == celStart {
				fileStart := multilineSpan.Start + i
				fileEnd := fileStart + (celEnd - celStart)
				return source.Span{File: multilineSpan.File, Start: fileStart, End: fileEnd}
			}
			celPos++
			i++
		}
	}

	return source.Span{}
}

// findMethodNameAfterDot finds the position of a method name after a dot in a CEL expression.
// Returns the start and end positions, or -1 if not found.
func findMethodNameAfterDot(targetOffset int32, methodName string, exprString string) (start, end int) {
	searchStart := int(targetOffset)
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

// findNameAfterDot searches for ".name" after targetOffset and returns the span of just the name (without the dot).
// Returns zero span if not found.
func findNameAfterDot(
	targetOffset int32,
	name string,
	exprString string,
	exprLiteralSpan source.Span,
) source.Span {
	start, end := findMethodNameAfterDot(targetOffset, name, exprString)
	if start < 0 {
		return source.Span{}
	}
	return createCELSpan(start, end, exprLiteralSpan)
}

// findStandaloneFunctionName calculates the position of a function name before an opening paren.
// celOffset is the position of the opening paren.
func findStandaloneFunctionName(celOffset int32, funcName string, exprString string) (start, end int, found bool) {
	start = int(celOffset) - len(funcName)
	end = start + len(funcName)
	if start >= 0 && end <= len(exprString) && exprString[start:end] == funcName {
		return start, end, true
	}
	return -1, -1, false
}
