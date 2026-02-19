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
	"maps"
	"strings"

	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/overloads"
	"go.lsp.dev/protocol"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

const (
	// quoteOffsetAdjustment accounts for the opening quote in CEL string literals.
	quoteOffsetAdjustment = 1
)

// getCELHover returns hover documentation for CEL expressions.
// If the cursor is over a CEL expression, it returns documentation for the token at that position.
func getCELHover(
	file *file,
	position protocol.Position,
	celEnv *cel.Env,
) *protocol.Hover {
	// Convert position to byte offset
	lineColumn := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	byteOffset := lineColumn.Offset

	// Find if we're inside a CEL expression
	for _, symbol := range file.symbols {
		celExpressions := extractCELExpressions(file, symbol)
		for _, exprInfo := range celExpressions {
			// Check if cursor is within this CEL expression span
			if byteOffset < exprInfo.span.Start || byteOffset >= exprInfo.span.End {
				continue
			}

			// Calculate offset within the CEL expression
			// The span includes quotes, so adjust for opening quote
			celOffset := byteOffset - exprInfo.span.Start - quoteOffsetAdjustment
			if celOffset < 0 || celOffset >= len(exprInfo.expression) {
				continue
			}

			// Parse the CEL expression. MacroCalls are only preserved in the
			// parsed AST (they are expanded in the checked AST), so we always
			// parse first and keep both.
			parsedAST, parseIssues := celEnv.Parse(exprInfo.expression)
			if parseIssues.Err() != nil {
				continue
			}

			// Default to the parsed AST, but prefer the checked AST for full
			// type information if type-checking succeeds.
			astForTypeInfo := parsedAST
			var typeMap map[int64]*exprpb.Type

			if checkedAST, compileIssues := celEnv.Compile(exprInfo.expression); compileIssues.Err() == nil {
				astForTypeInfo = checkedAST
				if checkedExpr, err := cel.AstToCheckedExpr(checkedAST); err == nil {
					typeMap = checkedExpr.TypeMap
				}
			}

			// Find the token at the cursor position using proper AST walking.
			// Pass both ASTs: astForTypeInfo (for type info) and parsedAST (for macro info).
			hoverInfo := findCELTokenAtOffset(astForTypeInfo, parsedAST, celOffset, exprInfo, typeMap)
			if hoverInfo == nil {
				continue
			}

			// Format documentation for the hover
			docs := formatCELHoverContent(hoverInfo, celEnv)
			if docs == "" {
				continue
			}

			// Create hover range - highlight the whole token
			tokenSpan := createCELSpan(hoverInfo.start, hoverInfo.end, exprInfo.span)
			if tokenSpan.IsZero() {
				continue
			}

			hoverRange := reportSpanToProtocolRange(tokenSpan)
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: docs,
				},
				Range: &hoverRange,
			}
		}
	}

	return nil
}

// celHoverInfo represents information about a CEL token for hover display
type celHoverInfo struct {
	kind        celHoverKind
	text        string
	start       int          // byte offset in CEL expression
	end         int          // byte offset in CEL expression
	celType     *exprpb.Type // from type checker
	protoMember ir.Member    // resolved proto field
	exprID      int64
}

type celHoverKind int

const (
	celHoverFunction celHoverKind = iota
	celHoverOperator
	celHoverKeyword
	celHoverMacro
	celHoverType
	celHoverField
	celHoverVariable
	celHoverLiteral
)

// findCELTokenAtOffset finds the CEL token at the given offset using proper AST traversal.
//
// Parameters:
//   - astForTypeInfo: The primary AST to walk (checked AST if available, otherwise parsed AST).
//     Used for all expression walking and type information lookups.
//   - parsedAst: The original parsed AST. Only used for macro lookups since MacroCalls
//     are preserved only in the parsed AST (they get expanded in checked ASTs).
//   - offset: The byte offset within the CEL expression string.
//   - exprInfo: Context about the CEL expression being analyzed.
//   - typeMap: Type information from the type checker (nil if only parsed AST available).
//
// Returns hover information for the token at the offset, or nil if not found.
func findCELTokenAtOffset(astForTypeInfo *cel.Ast, parsedAst *cel.Ast, offset int, exprInfo celExpressionInfo, typeMap map[int64]*exprpb.Type) *celHoverInfo {
	// Get parsed expression and source info from the type-checked AST
	parsedExpr, err := cel.AstToParsedExpr(astForTypeInfo)
	if err != nil {
		return nil
	}

	sourceInfo := parsedExpr.GetSourceInfo()
	expr := parsedExpr.GetExpr()

	// Get offset ranges from native AST for accurate position information
	nativeAST := astForTypeInfo.NativeRep()
	nativeSourceInfo := nativeAST.SourceInfo()
	offsetRanges := nativeSourceInfo.OffsetRanges()

	// Walk the AST to find the deepest matching expression at the cursor position
	// Track comprehension variables for proper scoping
	compVars := make(map[string]bool)
	result := walkCELExprForHover(expr, sourceInfo, offsetRanges, offset, exprInfo.expression, compVars, exprInfo, typeMap)

	// If we didn't find anything in the main AST, check macro calls in the parsed AST
	// Macros are expanded in the main AST, but parsedExpr.MacroCalls preserves the original macro syntax
	if result == nil && parsedAst != nil {
		if parsedExprForMacros, err := cel.AstToParsedExpr(parsedAst); err == nil {
			result = findMacroAtOffset(parsedExprForMacros.GetSourceInfo(), offset, exprInfo.expression)
		}
	}

	return result
}

// findMacroAtOffset checks if the offset is within a CEL macro call.
// Macros like has(), all(), exists(), map(), filter() are expanded in the main AST,
// but sourceInfo.MacroCalls preserves the original macro call information.
func findMacroAtOffset(sourceInfo *exprpb.SourceInfo, offset int, exprString string) *celHoverInfo {
	if sourceInfo == nil {
		return nil
	}

	// MacroCalls might be nil if there are no macros
	if len(sourceInfo.MacroCalls) == 0 {
		return nil
	}

	// Process each macro call
	for macroID, macroExpr := range sourceInfo.MacroCalls {
		callExpr, ok := macroExpr.ExprKind.(*exprpb.Expr_CallExpr)
		if !ok {
			continue
		}

		funcName := callExpr.CallExpr.Function

		// Only process recognized CEL macros
		if !isCELMacroFunction(funcName) {
			continue
		}

		// Get the position of the macro call (rune offset from CEL)
		celRuneOffset, ok := sourceInfo.Positions[macroID]
		if !ok {
			continue
		}

		// For method-style macros like this.all(...), find the method name after the dot
		// For standalone macros like has(...), find the function name before the paren
		var funcStart, funcEnd int
		var found bool

		if callExpr.CallExpr.Target != nil {
			// Method call - search for ".funcName" after the target
			targetID := callExpr.CallExpr.Target.Id
			if targetRuneOffset, ok := sourceInfo.Positions[targetID]; ok {
				targetByteOffset := celRuneOffsetToByteOffset(exprString, targetRuneOffset)
				funcStart, funcEnd = findMethodNameAfterDot(targetByteOffset, funcName, exprString)
				found = funcStart >= 0
			}
		} else {
			// Standalone function call - CEL position points to opening paren, look backwards
			celByteOffset := celRuneOffsetToByteOffset(exprString, celRuneOffset)
			funcStart, funcEnd, found = findStandaloneFunctionName(celByteOffset, funcName, exprString)
		}

		// Check if offset is within the macro function name
		if found && offset >= funcStart && offset < funcEnd {
			return &celHoverInfo{
				kind:   celHoverMacro,
				text:   funcName,
				start:  funcStart,
				end:    funcEnd,
				exprID: macroID,
			}
		}
	}

	return nil
}

// walkCELExprForHover walks the CEL AST to find the token at the given offset.
// Returns the deepest matching expression that contains the offset.
func walkCELExprForHover(
	expr *exprpb.Expr,
	sourceInfo *exprpb.SourceInfo,
	offsetRanges map[int64]ast.OffsetRange,
	offset int,
	exprString string,
	compVars map[string]bool,
	exprInfo celExpressionInfo,
	typeMap map[int64]*exprpb.Type,
) *celHoverInfo {
	if expr == nil {
		return nil
	}

	// Get the CEL rune offset for this expression and convert to byte offset.
	// CEL tracks positions as Unicode code point (rune) offsets, not byte offsets.
	celRuneOffset, ok := sourceInfo.Positions[expr.Id]
	if !ok {
		return nil
	}
	celByteOffset := celRuneOffsetToByteOffset(exprString, celRuneOffset)

	// Variable to hold the best match (deepest expression containing offset)
	var bestMatch *celHoverInfo

	switch kind := expr.ExprKind.(type) {
	case *exprpb.Expr_IdentExpr:
		// Identifier reference (variable, field, keyword)
		ident := kind.IdentExpr
		identName := ident.Name

		// Determine the hover kind
		var hoverKind celHoverKind
		if isCELKeyword(identName) {
			hoverKind = celHoverKeyword
		} else if compVars != nil && compVars[identName] {
			// Comprehension variable
			hoverKind = celHoverVariable
		} else {
			// Field access or other identifier
			hoverKind = celHoverField
		}

		if offsetRange, ok := offsetRanges[expr.Id]; ok {
			byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
			if offset >= byteStart && offset < byteStop {
				hoverInfo := &celHoverInfo{
					kind:   hoverKind,
					text:   identName,
					start:  byteStart,
					end:    byteStop,
					exprID: expr.Id,
				}
				// Add type information if available
				if typeMap != nil {
					if celType, hasType := typeMap[expr.Id]; hasType {
						hoverInfo.celType = celType
					}
				}
				bestMatch = hoverInfo
			}
		}

	case *exprpb.Expr_SelectExpr:
		// Field access (target.field)
		sel := kind.SelectExpr

		// Walk target first (might contain the offset)
		if sel.Operand != nil {
			if targetMatch := walkCELExprForHover(sel.Operand, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); targetMatch != nil {
				bestMatch = targetMatch
			}
		}

		// Check if cursor is on the field name itself
		if sel.Operand != nil {
			if targetRuneOffset, ok := sourceInfo.Positions[sel.Operand.Id]; ok {
				targetByteOffset := celRuneOffsetToByteOffset(exprString, targetRuneOffset)
				fieldStart, fieldEnd := findMethodNameAfterDot(targetByteOffset, sel.Field, exprString)
				if fieldStart >= 0 && offset >= fieldStart && offset < fieldEnd {
					// Cursor is on the field name
					hoverInfo := &celHoverInfo{
						kind:   celHoverField,
						text:   sel.Field,
						start:  fieldStart,
						end:    fieldEnd,
						exprID: expr.Id,
					}
					// Add type information
					if typeMap != nil {
						if celType, hasType := typeMap[expr.Id]; hasType {
							hoverInfo.celType = celType
						}
					}
					// Try to resolve proto field
					hoverInfo.protoMember = resolveCELFieldAccess(sel, exprInfo)
					bestMatch = hoverInfo
				}
			}
		}

	case *exprpb.Expr_CallExpr:
		// Function call or operator
		call := kind.CallExpr

		// Walk target first (for method calls)
		if call.Target != nil {
			if targetMatch := walkCELExprForHover(call.Target, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); targetMatch != nil {
				bestMatch = targetMatch
			}
		}

		// Walk arguments to see if offset is in any of them
		for _, arg := range call.Args {
			if argMatch := walkCELExprForHover(arg, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); argMatch != nil {
				bestMatch = argMatch
			}
		}

		funcName := call.Function

		// Check if this is an operator
		if symbol, isOperator := celOperatorSymbol(funcName); isOperator {
			if offsetRange, ok := offsetRanges[expr.Id]; ok {
				byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
				if offset >= byteStart && offset < byteStop {
					bestMatch = &celHoverInfo{
						kind:   celHoverOperator,
						text:   symbol, // Store the symbol, not the internal function name
						start:  byteStart,
						end:    byteStop,
						exprID: expr.Id,
					}
				}
			}
		} else {
			// Determine the function hover kind
			var hoverKind celHoverKind
			if isCELMacroFunction(funcName) {
				hoverKind = celHoverMacro
			} else if overloads.IsTypeConversionFunction(funcName) {
				hoverKind = celHoverType
			} else {
				hoverKind = celHoverFunction
			}

			// Check if cursor is on the function name
			var funcStart, funcEnd int
			var found bool

			if call.Target != nil {
				// Method call - search for the function name after the target
				if targetRuneOffset, ok := sourceInfo.Positions[call.Target.Id]; ok {
					targetByteOffset := celRuneOffsetToByteOffset(exprString, targetRuneOffset)
					funcStart, funcEnd = findMethodNameAfterDot(targetByteOffset, funcName, exprString)
					found = funcStart >= 0
				}
			} else {
				// Standalone function call - function name is before the opening paren
				funcStart, funcEnd, found = findStandaloneFunctionName(celByteOffset, funcName, exprString)
			}

			if found && offset >= funcStart && offset < funcEnd {
				bestMatch = &celHoverInfo{
					kind:   hoverKind,
					text:   funcName,
					start:  funcStart,
					end:    funcEnd,
					exprID: expr.Id,
				}
			}
		}

	case *exprpb.Expr_ConstExpr:
		// Constant literal
		constExpr := kind.ConstExpr

		if offsetRange, ok := offsetRanges[expr.Id]; ok {
			byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
			if offset >= byteStart && offset < byteStop {
				bestMatch = &celHoverInfo{
					kind:    celHoverLiteral,
					text:    exprString[byteStart:byteStop],
					start:   byteStart,
					end:     byteStop,
					exprID:  expr.Id,
					celType: getPrimitiveTypeForConstant(constExpr),
				}
			}
		}

	case *exprpb.Expr_ListExpr:
		// List literal - walk all elements
		list := kind.ListExpr
		for _, elem := range list.Elements {
			if elemMatch := walkCELExprForHover(elem, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); elemMatch != nil {
				bestMatch = elemMatch
			}
		}

	case *exprpb.Expr_StructExpr:
		// Map/struct literal - walk all entries
		structExpr := kind.StructExpr
		for _, entry := range structExpr.Entries {
			// Handle map entries (have a key)
			if mapKey := entry.GetMapKey(); mapKey != nil {
				if keyMatch := walkCELExprForHover(mapKey, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); keyMatch != nil {
					bestMatch = keyMatch
				}
			}
			// Walk the value
			if entry.Value != nil {
				if valueMatch := walkCELExprForHover(entry.Value, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); valueMatch != nil {
					bestMatch = valueMatch
				}
			}
		}

	case *exprpb.Expr_ComprehensionExpr:
		// List comprehension - walk all parts with proper variable scoping
		comp := kind.ComprehensionExpr

		// Walk the range and init with current scope (they don't see loop variables)
		if comp.IterRange != nil {
			if rangeMatch := walkCELExprForHover(comp.IterRange, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); rangeMatch != nil {
				bestMatch = rangeMatch
			}
		}
		if comp.AccuInit != nil {
			if initMatch := walkCELExprForHover(comp.AccuInit, sourceInfo, offsetRanges, offset, exprString, compVars, exprInfo, typeMap); initMatch != nil {
				bestMatch = initMatch
			}
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
			if condMatch := walkCELExprForHover(comp.LoopCondition, sourceInfo, offsetRanges, offset, exprString, extendedVars, exprInfo, typeMap); condMatch != nil {
				bestMatch = condMatch
			}
		}
		if comp.LoopStep != nil {
			if stepMatch := walkCELExprForHover(comp.LoopStep, sourceInfo, offsetRanges, offset, exprString, extendedVars, exprInfo, typeMap); stepMatch != nil {
				bestMatch = stepMatch
			}
		}
		if comp.Result != nil {
			if resultMatch := walkCELExprForHover(comp.Result, sourceInfo, offsetRanges, offset, exprString, extendedVars, exprInfo, typeMap); resultMatch != nil {
				bestMatch = resultMatch
			}
		}
	}

	return bestMatch
}

// formatCELHoverContent formats hover information into markdown documentation
func formatCELHoverContent(info *celHoverInfo, celEnv *cel.Env) string {
	if info == nil {
		return ""
	}

	switch info.kind {
	case celHoverKeyword:
		return getCELKeywordDocs(info.text)
	case celHoverFunction:
		result := getCELFunctionDocs(info.text, celEnv)
		// Add inferred return type if available
		if info.celType != nil {
			result += fmt.Sprintf("\n\n**Return type**: `%s`", getCELTypeString(info.celType))
		}
		return result
	case celHoverOperator:
		// Convert internal operator name to symbol if needed
		if symbol, ok := celOperatorSymbol(info.text); ok {
			return getCELOperatorDocs(symbol, celEnv)
		}
		return getCELOperatorDocs(info.text, celEnv)
	case celHoverMacro:
		return getCELMacroDocs(info.text, celEnv)
	case celHoverType:
		return getCELTypeDocs(info.text, celEnv)
	case celHoverField:
		// For fields, show comprehensive proto field info
		result := fmt.Sprintf("**Field**: `%s`", info.text)

		// Add CEL type information
		if info.celType != nil {
			result += fmt.Sprintf("\n\n**Type**: `%s`", getCELTypeString(info.celType))
		}

		// Add proto field information if resolved
		if !info.protoMember.IsZero() {
			result += "\n\n" + getProtoFieldDocumentation(info.protoMember)
		}

		return result
	case celHoverVariable:
		// Comprehension variable
		result := fmt.Sprintf("**Variable**: `%s`\n\nLoop variable from comprehension.", info.text)
		if info.celType != nil {
			result += fmt.Sprintf("\n\n**Type**: `%s`", getCELTypeString(info.celType))
		}
		return result
	case celHoverLiteral:
		// Literal value
		typeName := "value"
		if info.celType != nil {
			typeName = getCELTypeString(info.celType)
		}
		return fmt.Sprintf("**Literal**: `%s`\n\n**Type**: %s", info.text, typeName)
	default:
		return ""
	}
}

// resolveCELFieldAccess resolves a CEL SelectExpr to a proto field/member.
// Handles expressions like `this.fieldName` or `this.address.city`.
func resolveCELFieldAccess(sel *exprpb.Expr_Select, exprInfo celExpressionInfo) ir.Member {
	if sel == nil || exprInfo.irMember.IsZero() {
		return ir.Member{}
	}

	// Start from the context field (the field being validated)
	// For field-level validation, exprInfo.irMember is the field
	currentType := exprInfo.irMember.Element()

	// Build the field path by walking up the select chain
	var fieldPath []string

	// Walk the select chain to build the field path
	currentSel := sel
	for currentSel != nil {
		fieldPath = append([]string{currentSel.Field}, fieldPath...)

		// Check if the operand is another select expression
		if currentSel.Operand != nil {
			if nestedSel, ok := currentSel.Operand.ExprKind.(*exprpb.Expr_SelectExpr); ok {
				currentSel = nestedSel.SelectExpr
			} else if ident, ok := currentSel.Operand.ExprKind.(*exprpb.Expr_IdentExpr); ok {
				// Hit an identifier (should be 'this')
				if ident.IdentExpr.Name == "this" {
					// Start resolution from the current field's type
					break
				}
				// Unknown identifier, can't resolve
				return ir.Member{}
			} else {
				// Other expression type, can't resolve
				return ir.Member{}
			}
		} else {
			break
		}
	}

	// Now resolve the field path through the type hierarchy
	for _, fieldName := range fieldPath {
		if !currentType.IsMessage() || currentType.IsZero() {
			// Can't navigate further
			return ir.Member{}
		}

		// Find the field with this name in the current message
		var found bool
		var foundMember ir.Member
		for member := range seq.Values(currentType.Members()) {
			if member.Name() == fieldName {
				foundMember = member
				currentType = member.Element()
				found = true
				break
			}
		}

		if !found {
			// Field not found
			return ir.Member{}
		}

		// If this is the last segment, return the member
		if fieldName == fieldPath[len(fieldPath)-1] {
			return foundMember
		}
	}

	return ir.Member{}
}

// getProtoFieldDocumentation returns documentation for a proto field/member.
func getProtoFieldDocumentation(member ir.Member) string {
	if member.IsZero() {
		return ""
	}

	var parts []string

	// Add proto field details
	parts = append(parts, fmt.Sprintf("**Proto Field**: `%s`", member.FullName()))

	// Add field number if it's a field
	if fieldNum := member.Number(); fieldNum > 0 {
		parts = append(parts, fmt.Sprintf("**Field Number**: %d", fieldNum))
	}

	// Add field type
	elemType := member.Element()
	if !elemType.IsZero() {
		parts = append(parts, fmt.Sprintf("**Proto Type**: `%s`", elemType.FullName()))
	}

	// Add field documentation from proto comments
	if doc := irMemberDoc(member); doc != "" {
		parts = append(parts, doc)
	}

	return strings.Join(parts, "\n\n")
}

// getCELTypeString converts a CEL type to a human-readable string.
func getCELTypeString(t *exprpb.Type) string {
	if t == nil {
		return "unknown"
	}

	switch kind := t.TypeKind.(type) {
	case *exprpb.Type_Primitive:
		switch kind.Primitive {
		case exprpb.Type_BOOL:
			return "bool"
		case exprpb.Type_INT64:
			return "int64"
		case exprpb.Type_UINT64:
			return "uint64"
		case exprpb.Type_DOUBLE:
			return "double"
		case exprpb.Type_STRING:
			return "string"
		case exprpb.Type_BYTES:
			return "bytes"
		default:
			return "unknown"
		}
	case *exprpb.Type_ListType_:
		elemType := getCELTypeString(kind.ListType.ElemType)
		return fmt.Sprintf("list(%s)", elemType)
	case *exprpb.Type_MapType_:
		keyType := getCELTypeString(kind.MapType.KeyType)
		valueType := getCELTypeString(kind.MapType.ValueType)
		return fmt.Sprintf("map(%s, %s)", keyType, valueType)
	case *exprpb.Type_MessageType:
		return kind.MessageType
	case *exprpb.Type_TypeParam:
		return kind.TypeParam
	case *exprpb.Type_WellKnown:
		switch kind.WellKnown {
		case exprpb.Type_ANY:
			return "any"
		case exprpb.Type_DURATION:
			return "duration"
		case exprpb.Type_TIMESTAMP:
			return "timestamp"
		default:
			return "unknown"
		}
	default:
		return "unknown"
	}
}

// getCELKeywordDocs returns documentation for CEL keywords.
// These keywords are defined in the CEL specification and are not available from upstream documentation.
func getCELKeywordDocs(keyword string) string {
	// Note: true, false, and null are ConstExpr nodes in the CEL AST (not IdentExpr),
	// so only "this" can reach this function via celHoverKeyword.
	if keyword == "this" {
		return "**Special variable**\n\nRefers to the current message or field being validated.\n\n" +
			"In field-level rules, `this` refers to the field value.\n" +
			"In message-level rules, `this` refers to the entire message."
	}
	return ""
}

// getCELFunctionDocs returns documentation for CEL functions
func getCELFunctionDocs(function string, celEnv *cel.Env) string {
	// Try to get documentation from cel-go
	if celEnv != nil {
		funcs := celEnv.Functions()
		if funcDecl, ok := funcs[function]; ok {
			// Try structured documentation first
			if doc := funcDecl.Documentation(); doc != nil {
				return formatCELDoc(doc)
			}
			// Fallback to simple description
			if desc := funcDecl.Description(); desc != "" {
				return fmt.Sprintf("**Function**: `%s`\n\n%s", function, desc)
			}
		}
	}

	return ""
}

// formatCELDoc formats a CEL common.Doc into markdown
// If headerPrefix is provided (e.g., "**Operator**: " or "**Macro**: "), it will be used instead of just the name
func formatCELDoc(doc *common.Doc, headerPrefix ...string) string {
	if doc == nil {
		return ""
	}

	var result strings.Builder

	// Add signature/name with optional prefix
	var nameToDisplay string
	if doc.Signature != "" {
		nameToDisplay = doc.Signature
	} else if doc.Name != "" {
		nameToDisplay = doc.Name
	}

	if nameToDisplay != "" {
		if len(headerPrefix) > 0 && headerPrefix[0] != "" {
			// Custom header with prefix (e.g., "**Operator**: `+`")
			fmt.Fprintf(&result, "%s`%s`", headerPrefix[0], nameToDisplay)
		} else {
			// Default header (just the name)
			fmt.Fprintf(&result, "`%s`", nameToDisplay)
		}
	}

	// Add description
	if doc.Description != "" {
		if result.Len() > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(doc.Description)
	}

	// Add children information (either overloads or examples)
	if len(doc.Children) > 0 {
		// Check if children have signatures (overloads) or only descriptions (examples)
		hasSignatures := false
		for _, child := range doc.Children {
			if child.Signature != "" {
				hasSignatures = true
				break
			}
		}

		if hasSignatures {
			// Children are overloads
			result.WriteString("\n\n**Overloads**:")
			for _, child := range doc.Children {
				if child.Signature != "" {
					fmt.Fprintf(&result, "\n- `%s`", child.Signature)
					if child.Description != "" {
						fmt.Fprintf(&result, ": %s", child.Description)
					}
				}
			}
		} else {
			// Children are examples (e.g., for macros)
			result.WriteString("\n\n**Examples**:")
			for _, child := range doc.Children {
				if child.Description != "" {
					fmt.Fprintf(&result, "\n```cel\n%s\n```", child.Description)
				}
			}
		}
	}

	return result.String()
}

// getCELOperatorDocs returns documentation for CEL operators
func getCELOperatorDocs(op string, celEnv *cel.Env) string {
	// Operators are internally represented as functions (e.g., "_&&_", "_||_")
	// Try to find the internal function name from the operator symbol
	var internalName string
	var found bool

	// operators.Find() doesn't work for some operators, so map them manually
	switch op {
	case "&&":
		internalName = operators.LogicalAnd
		found = true
	case "||":
		internalName = operators.LogicalOr
		found = true
	case "!":
		internalName = operators.LogicalNot
		found = true
	case "?", "?:":
		internalName = operators.Conditional
		found = true
	case "[]":
		internalName = operators.Index
		found = true
	default:
		internalName, found = operators.Find(op)
	}

	if found {
		if celEnv != nil {
			funcs := celEnv.Functions()
			if funcDecl, ok := funcs[internalName]; ok {
				if doc := funcDecl.Documentation(); doc != nil {
					// Use the operator symbol instead of the internal name
					doc.Name = op
					return formatCELDoc(doc, "**Operator**: ")
				}
				if desc := funcDecl.Description(); desc != "" {
					return fmt.Sprintf("**Operator**: `%s`\n\n%s", op, desc)
				}
			}
		}
	}
	return ""
}

// getCELMacroDocs returns documentation for CEL macros
func getCELMacroDocs(macro string, celEnv *cel.Env) string {
	// Macros might be available as functions in the environment
	if celEnv != nil {
		funcs := celEnv.Functions()
		if funcDecl, ok := funcs[macro]; ok {
			if doc := funcDecl.Documentation(); doc != nil {
				return formatCELDoc(doc)
			}
			if desc := funcDecl.Description(); desc != "" {
				return fmt.Sprintf("**Macro**: `%s`\n\n%s", macro, desc)
			}
		}
		// Check env.Macros() for documentation
		for _, m := range celEnv.Macros() {
			if m.Function() == macro {
				if doc, ok := m.(common.Documentor); ok {
					if documentation := doc.Documentation(); documentation != nil {
						// Use formatCELDoc with macro prefix
						documentation.Name = macro
						return formatCELDoc(documentation, "**Macro**: ")
					}
				}
				break
			}
		}
	}

	return ""
}

// getCELTypeDocs returns documentation for CEL type functions
func getCELTypeDocs(typeName string, celEnv *cel.Env) string {
	// Type conversion functions are available in the environment
	if celEnv != nil {
		funcs := celEnv.Functions()
		if funcDecl, ok := funcs[typeName]; ok {
			if doc := funcDecl.Documentation(); doc != nil {
				return formatCELDoc(doc)
			}
			if desc := funcDecl.Description(); desc != "" {
				return fmt.Sprintf("**Type**: `%s`\n\n%s", typeName, desc)
			}
		}
	}

	// All type conversion functions are documented upstream
	return fmt.Sprintf("**Type**: `%s`", typeName)
}

// getPrimitiveTypeForConstant returns the CEL type for a constant expression.
func getPrimitiveTypeForConstant(constExpr *exprpb.Constant) *exprpb.Type {
	switch constExpr.ConstantKind.(type) {
	case *exprpb.Constant_StringValue:
		return &exprpb.Type{TypeKind: &exprpb.Type_Primitive{Primitive: exprpb.Type_STRING}}
	case *exprpb.Constant_Int64Value:
		return &exprpb.Type{TypeKind: &exprpb.Type_Primitive{Primitive: exprpb.Type_INT64}}
	case *exprpb.Constant_Uint64Value:
		return &exprpb.Type{TypeKind: &exprpb.Type_Primitive{Primitive: exprpb.Type_UINT64}}
	case *exprpb.Constant_DoubleValue:
		return &exprpb.Type{TypeKind: &exprpb.Type_Primitive{Primitive: exprpb.Type_DOUBLE}}
	case *exprpb.Constant_BoolValue:
		return &exprpb.Type{TypeKind: &exprpb.Type_Primitive{Primitive: exprpb.Type_BOOL}}
	default:
		return nil
	}
}
