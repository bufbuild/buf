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
	"maps"
	"strings"

	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

const (
	// buf.validate extension field number on descriptor options
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate
	bufValidateExtensionNumber = 1159

	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldRules
	celFieldNumberInFieldRules = 23
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldRules
	celExpressionFieldNumberInFieldRules = 29

	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.MessageRules
	celFieldNumberInMessageRules = 3
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.MessageRules
	celExpressionFieldNumberInMessageRules = 5

	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.Rule
	expressionFieldNumberInRule = 3
)

// celExpressionInfo holds information about a CEL expression found in protovalidate options.
type celExpressionInfo struct {
	expression string      // The CEL expression string
	span       source.Span // The span of the expression string literal in the proto file
	irMember   ir.Member   // The field/member that has the option (for type context)
}

// extractCELExpressions finds CEL expressions in protovalidate options.
// It looks for (buf.validate.field).cel[].expression and (buf.validate.field).cel_expression[] patterns.
func extractCELExpressions(file *file, symbol *symbol) []celExpressionInfo {
	if symbol.ir.IsZero() {
		return nil
	}
	var optionValue ir.MessageValue
	switch symbol.ir.Kind() {
	case ir.SymbolKindField, ir.SymbolKindExtension:
		optionValue = symbol.ir.AsMember().Options()
	case ir.SymbolKindMessage:
		optionValue = symbol.ir.AsType().Options()
	case ir.SymbolKindEnum:
		optionValue = symbol.ir.AsType().Options()
	case ir.SymbolKindEnumValue:
		optionValue = symbol.ir.AsMember().Options()
	case ir.SymbolKindService:
		optionValue = symbol.ir.AsService().Options()
	case ir.SymbolKindMethod:
		optionValue = symbol.ir.AsMethod().Options()
	case ir.SymbolKindOneof:
		optionValue = symbol.ir.AsOneof().Options()
	default:
		return nil
	}
	if optionValue.IsZero() {
		return nil
	}
	// Traverse the option message value looking for buf.validate CEL expressions
	return extractCELFromMessage(file, optionValue, symbol.ir)
}

// extractCELFromMessage recursively extracts CEL expressions from an option message.
func extractCELFromMessage(file *file, msgValue ir.MessageValue, irSym ir.Symbol) []celExpressionInfo {
	var results []celExpressionInfo

	// Helper to extract member from symbol
	getMember := func() ir.Member {
		if irSym.Kind() == ir.SymbolKindField || irSym.Kind() == ir.SymbolKindExtension {
			return irSym.AsMember()
		}
		return ir.Member{}
	}

	for field := range msgValue.Fields() {
		for element := range seq.Values(field.Elements()) {
			elementField := element.Field()
			fieldNumber := elementField.Number()

			// Check if this is a top-level buf.validate extension - recurse into it
			if elementField.IsExtension() && fieldNumber == bufValidateExtensionNumber {
				nestedMsg := element.AsMessage()
				if !nestedMsg.IsZero() {
					results = append(results, extractCELFromMessage(file, nestedMsg, irSym)...)
				}
				continue
			}

			// Handle nested message field (cel.expression)
			if fieldNumber == celFieldNumberInFieldRules || fieldNumber == celFieldNumberInMessageRules {
				// This is a Rule message, look for the expression field
				nestedMsg := element.AsMessage()
				if !nestedMsg.IsZero() {
					for nestedField := range nestedMsg.Fields() {
						for nestedElement := range seq.Values(nestedField.Elements()) {
							nestedElementField := nestedElement.Field()
							if nestedElementField.Number() == expressionFieldNumberInRule {
								if exprString, ok := nestedElement.AsString(); ok {
									results = append(results, celExpressionInfo{
										expression: exprString,
										span:       nestedElement.AST().Span(),
										irMember:   getMember(),
									})
								}
							}
						}
					}
					// Recursively check nested messages
					results = append(results, extractCELFromMessage(file, nestedMsg, irSym)...)
				}
			}

			// Handle cel_expression string field
			if fieldNumber == celExpressionFieldNumberInFieldRules || fieldNumber == celExpressionFieldNumberInMessageRules {
				if exprString, ok := element.AsString(); ok {
					results = append(results, celExpressionInfo{
						expression: exprString,
						span:       element.AST().Span(),
						irMember:   getMember(),
					})
				}
			}
		}
	}

	return results
}

// collectCELTokens compiles a CEL expression and collects semantic tokens from its AST.
func collectCELTokens(
	baseCelEnv *cel.Env,
	exprInfo celExpressionInfo,
	collectToken func(span source.Span, semanticType, semanticModifier uint32, kw keyword.Keyword),
) {
	// Skip empty expressions
	if exprInfo.expression == "" {
		return
	}

	// Parse the CEL expression (without type-checking) to get syntax highlighting even for invalid expressions
	// This provides a better user experience as users can see highlighting while writing
	ast, issues := baseCelEnv.Parse(exprInfo.expression)
	if issues.Err() != nil {
		// Skip on parse errors (syntax errors)
		return
	}

	// Get the native AST which has offset ranges for all expressions
	nativeAST := ast.NativeRep()
	nativeSourceInfo := nativeAST.SourceInfo()
	offsetRanges := nativeSourceInfo.OffsetRanges()

	// Get the expression AST and source info (for compatibility with existing code)
	parsedExpr, err := cel.AstToParsedExpr(ast)
	if err != nil {
		return // Skip on error
	}
	expr := parsedExpr.GetExpr()
	sourceInfo := parsedExpr.GetSourceInfo()

	// Walk the CEL AST and collect tokens
	walkCELExprWithVars(expr, sourceInfo, offsetRanges, exprInfo.span, exprInfo.expression, collectToken, nil)

	// Process macro calls separately since they're expanded in the main AST
	// but we want to highlight the original macro function names (has, all, exists, map, filter, etc.)
	collectMacroTokens(sourceInfo, exprInfo.span, exprInfo.expression, collectToken)
}

// createCELSpan creates a source.Span for a CEL token given its start and end offsets within the CEL expression.
// The exprLiteralSpan is the span of the string literal containing the CEL expression (including quotes).
func createCELSpan(celStart, celEnd int32, exprLiteralSpan source.Span) source.Span {
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
	fileStart := exprLiteralSpan.Start + 1 + int(celStart)
	fileEnd := exprLiteralSpan.Start + 1 + int(celEnd)

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
func createCELSpanMultiline(celStart, celEnd int32, multilineSpan source.Span) source.Span {
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
			if celPos == int(celStart) {
				fileStart := multilineSpan.Start + i
				fileEnd := fileStart + int(celEnd-celStart)
				return source.Span{File: multilineSpan.File, Start: fileStart, End: fileEnd}
			}
			celPos++
			i++
		}
	}

	return source.Span{}
}

// collectMacroTokens processes CEL macro calls to highlight macro function names.
// Macros like has(), all(), exists(), map(), filter() are expanded in the main AST,
// but CEL preserves the original macro calls in sourceInfo.MacroCalls.
func collectMacroTokens(
	sourceInfo *exprpb.SourceInfo,
	exprLiteralSpan source.Span,
	exprString string,
	collectToken func(span source.Span, semanticType, semanticModifier uint32, kw keyword.Keyword),
) {
	// Process each macro call
	for macroID, macroExpr := range sourceInfo.MacroCalls {
		// Only process call expressions
		callExpr, ok := macroExpr.ExprKind.(*exprpb.Expr_CallExpr)
		if !ok {
			continue
		}

		funcName := callExpr.CallExpr.Function

		// Only highlight if it's a recognized CEL macro
		if !isCELMacroFunction(funcName) {
			continue
		}

		// Get the position of the macro call
		celOffset, ok := sourceInfo.Positions[macroID]
		if !ok {
			continue
		}

		// For method-style macros like this.all(...), we need to find the method name
		// For standalone macros like has(...), we need to find the function name
		if callExpr.CallExpr.Target != nil {
			// Method call - search for ".funcName" after the target
			targetID := callExpr.CallExpr.Target.Id
			if targetOffset, ok := sourceInfo.Positions[targetID]; ok {
				tokenSpan := findNameAfterDot(targetOffset, funcName, exprString, exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeMacro, 0, keyword.Unknown)
				}
			}
		} else {
			// Standalone function call - CEL position points to opening paren, look backwards
			funcStart := int(celOffset) - len(funcName)
			if funcStart >= 0 && funcStart+len(funcName) <= len(exprString) {
				if exprString[funcStart:funcStart+len(funcName)] == funcName {
					tokenSpan := createCELSpan(int32(funcStart), int32(funcStart+len(funcName)), exprLiteralSpan)
					if !tokenSpan.IsZero() {
						collectToken(tokenSpan, semanticTypeMacro, 0, keyword.Unknown)
					}
				}
			}
		}
	}
}

// walkCELExprWithVars recursively walks a CEL expression AST and collects semantic tokens.
// The compVars parameter tracks comprehension variables in scope.
func walkCELExprWithVars(
	expr *exprpb.Expr,
	sourceInfo *exprpb.SourceInfo,
	offsetRanges map[int64]ast.OffsetRange,
	exprLiteralSpan source.Span,
	exprString string,
	collectToken func(span source.Span, semanticType, semanticModifier uint32, kw keyword.Keyword),
	compVars map[string]bool,
) {
	if expr == nil {
		return
	}

	// Get the byte offset for this expression within the CEL string
	celOffset, ok := sourceInfo.Positions[expr.Id]
	if !ok {
		return
	}

	var tokenSpan source.Span

	switch kind := expr.ExprKind.(type) {
	case *exprpb.Expr_IdentExpr:
		// Identifier reference - use offset ranges from CEL's native AST
		ident := kind.IdentExpr
		identName := ident.Name

		// Determine the token type
		var tokenType uint32
		var tokenModifier uint32

		if isCELKeyword(identName) {
			// CEL keywords (true, false, null, this, etc.)
			tokenType = semanticTypeKeyword
		} else if compVars != nil && compVars[identName] {
			// Comprehension variables
			tokenType = semanticTypeVariable
		} else {
			// Field access or other identifier
			tokenType = semanticTypeProperty
		}

		if offsetRange, ok := offsetRanges[expr.Id]; ok {
			tokenSpan = createCELSpan(int32(offsetRange.Start), int32(offsetRange.Stop), exprLiteralSpan)
			if !tokenSpan.IsZero() {
				collectToken(tokenSpan, tokenType, tokenModifier, keyword.Unknown)
			}
		}

	case *exprpb.Expr_SelectExpr:
		// Field access (target.field)
		sel := kind.SelectExpr

		// Walk target first
		if sel.Operand != nil {
			walkCELExprWithVars(sel.Operand, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
		}

		// Highlight the field name
		if sel.Operand != nil {
			if targetOffset, ok := sourceInfo.Positions[sel.Operand.Id]; ok {
				tokenSpan = findNameAfterDot(targetOffset, sel.Field, exprString, exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeProperty, 0, keyword.Unknown)
				}
			}
		}

	case *exprpb.Expr_CallExpr:
		// Function call (or operator)
		call := kind.CallExpr

		// Walk target first (for method calls)
		if call.Target != nil {
			walkCELExprWithVars(call.Target, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
		}

		funcName := call.Function

		// Check if this is an operator (CEL represents operators as functions with special names)
		// Operators in CEL have names like _&&_, _||_, _>_, _==_, etc.
		if _, isOperator := celOperatorSymbol(funcName); isOperator {
			// This is an operator - use offset ranges from CEL's native AST
			if offsetRange, ok := offsetRanges[expr.Id]; ok {
				tokenSpan = createCELSpan(int32(offsetRange.Start), int32(offsetRange.Stop), exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeOperator, 0, keyword.Unknown)
				}
			}
		} else {
			// Determine the token type based on the function
			var tokenType uint32
			var tokenModifier uint32

			// Check for special function types (macros, type functions)
			if isCELMacroFunction(funcName) {
				// Macro functions (has, all, exists, map, filter)
				tokenType = semanticTypeMacro
			} else if isCELBuiltinTypeFunction(funcName) {
				// Built-in type conversion functions (int, uint, string, etc.)
				tokenType = semanticTypeType
				tokenModifier = semanticModifierDefaultLibrary
			} else if call.Target != nil {
				// Method call (e.g., this.size())
				tokenType = semanticTypeMethod
			} else {
				// Standalone function call (e.g., size(this))
				tokenType = semanticTypeFunction
			}

			if call.Target != nil {
				// Method call - search for the function name after the target
				if targetOffset, ok := sourceInfo.Positions[call.Target.Id]; ok {
					tokenSpan = findNameAfterDot(targetOffset, funcName, exprString, exprLiteralSpan)
					if !tokenSpan.IsZero() {
						collectToken(tokenSpan, tokenType, tokenModifier, keyword.Unknown)
					}
				}
			} else {
				// Standalone function call (no target)
				// CEL's position typically points to the opening paren, so look backwards for the function name
				funcStart := int(celOffset) - len(funcName)
				if funcStart >= 0 && funcStart+len(funcName) <= len(exprString) {
					if exprString[funcStart:funcStart+len(funcName)] == funcName {
						tokenSpan = createCELSpan(int32(funcStart), int32(funcStart+len(funcName)), exprLiteralSpan)
						if !tokenSpan.IsZero() {
							collectToken(tokenSpan, tokenType, tokenModifier, keyword.Unknown)
						}
					}
				}
			}
		}

		// Walk arguments
		for _, arg := range call.Args {
			walkCELExprWithVars(arg, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
		}

	case *exprpb.Expr_ConstExpr:
		// Constant literal
		constExpr := kind.ConstExpr

		switch constExpr.ConstantKind.(type) {
		case *exprpb.Constant_StringValue:
			// String literal - use offset ranges from CEL's native AST
			if offsetRange, ok := offsetRanges[expr.Id]; ok {
				tokenSpan = createCELSpan(int32(offsetRange.Start), int32(offsetRange.Stop), exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeString, 0, keyword.Unknown)
				}
			}

		case *exprpb.Constant_Int64Value, *exprpb.Constant_Uint64Value, *exprpb.Constant_DoubleValue:
			// Number literal - use offset ranges from CEL's native AST
			if offsetRange, ok := offsetRanges[expr.Id]; ok {
				tokenSpan = createCELSpan(int32(offsetRange.Start), int32(offsetRange.Stop), exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeNumber, 0, keyword.Unknown)
				}
			}

		case *exprpb.Constant_BoolValue:
			// Boolean literal - use offset ranges from CEL's native AST
			if offsetRange, ok := offsetRanges[expr.Id]; ok {
				tokenSpan = createCELSpan(int32(offsetRange.Start), int32(offsetRange.Stop), exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeKeyword, 0, keyword.Unknown)
				}
			}
		}

	case *exprpb.Expr_ListExpr:
		// List literal - walk all elements
		list := kind.ListExpr
		for _, elem := range list.Elements {
			walkCELExprWithVars(elem, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
		}

	case *exprpb.Expr_StructExpr:
		// Map/struct literal - walk all entries
		structExpr := kind.StructExpr
		for _, entry := range structExpr.Entries {
			// Handle map entries (have a key)
			if mapKey := entry.GetMapKey(); mapKey != nil {
				walkCELExprWithVars(mapKey, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
			}
			// Walk the value
			if entry.Value != nil {
				walkCELExprWithVars(entry.Value, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
			}
		}

	case *exprpb.Expr_ComprehensionExpr:
		// List comprehension - walk all parts
		comp := kind.ComprehensionExpr

		// Walk the range and init with current scope (they don't see loop variables)
		if comp.IterRange != nil {
			walkCELExprWithVars(comp.IterRange, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
		}
		if comp.AccuInit != nil {
			walkCELExprWithVars(comp.AccuInit, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, compVars)
		}

		// Create extended scope with comprehension variables for the loop body
		extendedVars := compVars
		if comp.IterVar != "" || comp.AccuVar != "" {
			// Copy the current compVars map and add the new variables
			if compVars != nil {
				extendedVars = make(map[string]bool, len(compVars)+2)
				maps.Copy(extendedVars, compVars)
			} else {
				extendedVars = make(map[string]bool, 2)
			}
			if comp.IterVar != "" {
				extendedVars[comp.IterVar] = true
			}
			if comp.AccuVar != "" {
				extendedVars[comp.AccuVar] = true
			}
		}

		// Walk the loop body with extended scope (they can see loop variables)
		if comp.LoopCondition != nil {
			walkCELExprWithVars(comp.LoopCondition, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, extendedVars)
		}
		if comp.LoopStep != nil {
			walkCELExprWithVars(comp.LoopStep, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, extendedVars)
		}
		if comp.Result != nil {
			walkCELExprWithVars(comp.Result, sourceInfo, offsetRanges, exprLiteralSpan, exprString, collectToken, extendedVars)
		}
	}
}

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

// isCELBuiltinTypeFunction returns true if the function name is a CEL built-in type conversion function.
// See https://github.com/google/cel-spec/blob/master/doc/langdef.md#gradual-type-checking
func isCELBuiltinTypeFunction(funcName string) bool {
	builtins := map[string]bool{
		"int":       true,
		"uint":      true,
		"double":    true,
		"bool":      true,
		"string":    true,
		"bytes":     true,
		"duration":  true,
		"timestamp": true,
		"dyn":       true,
		"type":      true,
	}
	return builtins[funcName]
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
	// Use cel-go's operators.FindReverse to get the display symbol
	if symbol, found := operators.FindReverse(funcName); found {
		return symbol, true
	}

	// Special case: ternary operator doesn't have a text representation in FindReverse
	// We highlight the '?' part of the ternary operator
	if funcName == operators.Conditional {
		return "?", true
	}

	return "", false
}

// findNameAfterDot searches for ".name" after targetOffset and returns the span of just the name (without the dot).
// Returns zero span if not found.
func findNameAfterDot(
	targetOffset int32,
	name string,
	exprString string,
	exprLiteralSpan source.Span,
) source.Span {
	searchStart := int(targetOffset)
	searchRegion := exprString[searchStart:]

	if idx := strings.Index(searchRegion, "."+name); idx >= 0 {
		nameStart := searchStart + idx + 1 // +1 to skip the dot
		nameEnd := nameStart + len(name)
		return createCELSpan(int32(nameStart), int32(nameEnd), exprLiteralSpan)
	}
	return source.Span{}
}
