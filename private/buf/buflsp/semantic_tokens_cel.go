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

	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/overloads"
	"github.com/google/cel-go/common/types"
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

			// Handle nested Rule message (cel field containing Rule messages with expressions)
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
				continue
			}

			// Handle cel_expression string field
			if fieldNumber == celExpressionFieldNumberInFieldRules || fieldNumber == celExpressionFieldNumberInMessageRules {
				if exprString, ok := element.AsString(); ok {
					results = append(results, celExpressionInfo{
						expression: exprString,
						span:       element.AST().Span(),
						irMember:   getMember(),
					})
					continue
				}
				// Not a string value (e.g. a nested FieldRules message at the same field number
				// as a MessageRules cel_expression field) - fall through to general recursion.
			}

			// General case: recursively search any nested message for CEL expressions.
			// This handles nested structures like repeated.items.cel, map.keys.cel,
			// map.values.cel, and any other type-specific rule nesting in protovalidate.
			if nestedMsg := element.AsMessage(); !nestedMsg.IsZero() {
				results = append(results, extractCELFromMessage(file, nestedMsg, irSym)...)
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
	parsedCELAST, issues := baseCelEnv.Parse(exprInfo.expression)
	if issues.Err() != nil {
		// Skip on parse errors (syntax errors)
		return
	}

	// Get the native AST which has offset ranges for all expressions
	nativeAST := parsedCELAST.NativeRep()
	nativeSourceInfo := nativeAST.SourceInfo()

	// Walk the CEL AST and collect tokens
	walkCELExprWithVars(nativeAST.Expr(), nativeSourceInfo, exprInfo.span, exprInfo.expression, collectToken, nil)

	// Process macro calls separately since they're expanded in the main AST
	// but we want to highlight the original macro function names (has, all, exists, map, filter, etc.)
	collectMacroTokens(nativeSourceInfo, exprInfo.span, exprInfo.expression, collectToken)
}

// collectMacroTokens processes CEL macro calls to highlight macro function names.
// Macros like has(), all(), exists(), map(), filter() are expanded in the main AST,
// but CEL preserves the original macro calls in sourceInfo.MacroCalls.
func collectMacroTokens(
	sourceInfo *ast.SourceInfo,
	exprLiteralSpan source.Span,
	exprString string,
	collectToken func(span source.Span, semanticType, semanticModifier uint32, kw keyword.Keyword),
) {
	// Process each macro call
	for macroID, macroExpr := range sourceInfo.MacroCalls() {
		// Only process call expressions
		if macroExpr.Kind() != ast.CallKind {
			continue
		}
		call := macroExpr.AsCall()
		funcName := call.FunctionName()

		// Only highlight if it's a recognized CEL macro
		if !isCELMacroFunction(funcName) {
			continue
		}

		// Get the position of the macro call (rune offset from CEL)
		startLoc := sourceInfo.GetStartLocation(macroID)
		if startLoc.Line() <= 0 {
			continue
		}
		// For method-style macros like this.all(...), we need to find the method name
		// For standalone macros like has(...), we need to find the function name
		if call.IsMemberFunction() {
			// Method call - search for ".funcName" after the target
			targetID := call.Target().ID()
			targetLoc := sourceInfo.GetStartLocation(targetID)
			if targetLoc.Line() > 0 {
				targetByteOffset := celLocByteOffset(targetLoc.Line(), targetLoc.Column(), sourceInfo, exprString)
				tokenSpan := findNameAfterDot(targetByteOffset, funcName, exprString, exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeMacro, 0, keyword.Unknown)
				}
			}
		} else {
			// Standalone function call - CEL position points to opening paren, look backwards
			celByteOffset := celLocByteOffset(startLoc.Line(), startLoc.Column(), sourceInfo, exprString)
			funcStart := celByteOffset - len(funcName)
			funcEnd := funcStart + len(funcName)
			if funcStart >= 0 && funcEnd <= len(exprString) {
				if exprString[funcStart:funcEnd] == funcName {
					tokenSpan := createCELSpan(funcStart, funcEnd, exprLiteralSpan)
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
	expr ast.Expr,
	sourceInfo *ast.SourceInfo,
	exprLiteralSpan source.Span,
	exprString string,
	collectToken func(span source.Span, semanticType, semanticModifier uint32, kw keyword.Keyword),
	compVars map[string]bool,
) {
	if expr == nil || expr.Kind() == ast.UnspecifiedExprKind {
		return
	}

	// Get the CEL rune offset for this expression and convert to byte offset.
	// CEL tracks positions as Unicode code point (rune) offsets, not byte offsets.
	startLoc := sourceInfo.GetStartLocation(expr.ID())
	if startLoc.Line() <= 0 {
		return
	}
	celByteOffset := celLocByteOffset(startLoc.Line(), startLoc.Column(), sourceInfo, exprString)

	var tokenSpan source.Span

	switch expr.Kind() {
	case ast.IdentKind:
		// Identifier reference - use offset ranges from CEL's native AST
		identName := expr.AsIdent()

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

		if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
			byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
			tokenSpan = createCELSpan(byteStart, byteStop, exprLiteralSpan)
			if !tokenSpan.IsZero() {
				collectToken(tokenSpan, tokenType, tokenModifier, keyword.Unknown)
			}
		}

	case ast.SelectKind:
		// Field access (target.field)
		sel := expr.AsSelect()

		// Walk target first
		if sel.Operand() != nil {
			walkCELExprWithVars(sel.Operand(), sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
		}

		// Highlight the field name
		if sel.Operand() != nil {
			targetLoc := sourceInfo.GetStartLocation(sel.Operand().ID())
			if targetLoc.Line() > 0 {
				targetByteOffset := celLocByteOffset(targetLoc.Line(), targetLoc.Column(), sourceInfo, exprString)
				tokenSpan = findNameAfterDot(targetByteOffset, sel.FieldName(), exprString, exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeProperty, 0, keyword.Unknown)
				}
			}
		}

	case ast.CallKind:
		// Function call (or operator)
		call := expr.AsCall()

		// Walk target first (for method calls)
		if call.IsMemberFunction() {
			walkCELExprWithVars(call.Target(), sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
		}

		funcName := call.FunctionName()

		// Check if this is an operator (CEL represents operators as functions with special names)
		// Operators in CEL have names like _&&_, _||_, _>_, _==_, etc.
		if _, isOperator := celOperatorSymbol(funcName); isOperator {
			// This is an operator - use offset ranges from CEL's native AST
			if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
				byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
				tokenSpan = createCELSpan(byteStart, byteStop, exprLiteralSpan)
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
			} else if overloads.IsTypeConversionFunction(funcName) {
				// Built-in type conversion functions (int, uint, string, etc.)
				tokenType = semanticTypeType
				tokenModifier = semanticModifierDefaultLibrary
			} else if call.IsMemberFunction() {
				// Method call (e.g., this.size())
				tokenType = semanticTypeMethod
			} else {
				// Standalone function call (e.g., size(this))
				tokenType = semanticTypeFunction
			}

			if call.IsMemberFunction() {
				// Method call - search for the function name after the target
				targetLoc := sourceInfo.GetStartLocation(call.Target().ID())
				if targetLoc.Line() > 0 {
					targetByteOffset := celLocByteOffset(targetLoc.Line(), targetLoc.Column(), sourceInfo, exprString)
					tokenSpan = findNameAfterDot(targetByteOffset, funcName, exprString, exprLiteralSpan)
					if !tokenSpan.IsZero() {
						collectToken(tokenSpan, tokenType, tokenModifier, keyword.Unknown)
					}
				}
			} else {
				// Standalone function call (no target)
				// CEL's position typically points to the opening paren, so look backwards for the function name
				funcStart := celByteOffset - len(funcName)
				funcEnd := funcStart + len(funcName)
				if funcStart >= 0 && funcEnd <= len(exprString) {
					if exprString[funcStart:funcEnd] == funcName {
						tokenSpan = createCELSpan(funcStart, funcEnd, exprLiteralSpan)
						if !tokenSpan.IsZero() {
							collectToken(tokenSpan, tokenType, tokenModifier, keyword.Unknown)
						}
					}
				}
			}
		}

		// Walk arguments
		for _, arg := range call.Args() {
			walkCELExprWithVars(arg, sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
		}

	case ast.LiteralKind:
		// Constant literal
		switch expr.AsLiteral().(type) {
		case types.String:
			// String literal - use offset ranges from CEL's native AST
			if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
				byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
				tokenSpan = createCELSpan(byteStart, byteStop, exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeString, 0, keyword.Unknown)
				}
			}

		case types.Int, types.Uint, types.Double:
			// Number literal - use offset ranges from CEL's native AST
			if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
				byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
				tokenSpan = createCELSpan(byteStart, byteStop, exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeNumber, 0, keyword.Unknown)
				}
			}

		case types.Bool:
			// Boolean literal - use offset ranges from CEL's native AST
			if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
				byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
				tokenSpan = createCELSpan(byteStart, byteStop, exprLiteralSpan)
				if !tokenSpan.IsZero() {
					collectToken(tokenSpan, semanticTypeKeyword, 0, keyword.Unknown)
				}
			}
		}

	case ast.ListKind:
		// List literal - walk all elements
		for _, elem := range expr.AsList().Elements() {
			walkCELExprWithVars(elem, sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
		}

	case ast.MapKind:
		// Map literal - walk all entries
		for _, entry := range expr.AsMap().Entries() {
			mapEntry := entry.AsMapEntry()
			walkCELExprWithVars(mapEntry.Key(), sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
			walkCELExprWithVars(mapEntry.Value(), sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
		}

	case ast.StructKind:
		// Struct literal - walk all field values
		for _, field := range expr.AsStruct().Fields() {
			walkCELExprWithVars(field.AsStructField().Value(), sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
		}

	case ast.ComprehensionKind:
		// List comprehension - walk all parts
		comp := expr.AsComprehension()

		// Walk the range and init with current scope (they don't see loop variables)
		walkCELExprWithVars(comp.IterRange(), sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)
		walkCELExprWithVars(comp.AccuInit(), sourceInfo, exprLiteralSpan, exprString, collectToken, compVars)

		// Create extended scope with comprehension variables for the loop body
		extendedVars := compVars
		if comp.IterVar() != "" || comp.AccuVar() != "" {
			// Copy the current compVars map and add the new variables
			if compVars != nil {
				extendedVars = make(map[string]bool, len(compVars)+2)
				maps.Copy(extendedVars, compVars)
			} else {
				extendedVars = make(map[string]bool, 2)
			}
			if comp.IterVar() != "" {
				extendedVars[comp.IterVar()] = true
			}
			if comp.AccuVar() != "" {
				extendedVars[comp.AccuVar()] = true
			}
		}

		// Walk the loop body with extended scope (they can see loop variables)
		walkCELExprWithVars(comp.LoopCondition(), sourceInfo, exprLiteralSpan, exprString, collectToken, extendedVars)
		walkCELExprWithVars(comp.LoopStep(), sourceInfo, exprLiteralSpan, exprString, collectToken, extendedVars)
		walkCELExprWithVars(comp.Result(), sourceInfo, exprLiteralSpan, exprString, collectToken, extendedVars)
	}
}
