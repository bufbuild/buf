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
	"github.com/google/cel-go/common/overloads"
	"github.com/google/cel-go/common/types"
	"go.lsp.dev/protocol"
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

			// Calculate offset within the CEL expression.
			// For single-line literals this is simple arithmetic; for multi-line
			// (adjacent proto string literals) we must walk the span to find the
			// matching CEL byte position.
			celOffset := fileByteOffsetToCELOffset(byteOffset, exprInfo.span)
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
			var typeMap map[int64]*types.Type

			if checkedAST, compileIssues := celEnv.Compile(exprInfo.expression); compileIssues.Err() == nil {
				astForTypeInfo = checkedAST
				typeMap = checkedAST.NativeRep().TypeMap()
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
	start       int         // byte offset in CEL expression
	end         int         // byte offset in CEL expression
	celType     *types.Type // from type checker
	protoMember ir.Member   // resolved proto field
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
func findCELTokenAtOffset(astForTypeInfo *cel.Ast, parsedAst *cel.Ast, offset int, exprInfo celExpressionInfo, typeMap map[int64]*types.Type) *celHoverInfo {
	// Get expression and source info from the native AST directly
	nativeAST := astForTypeInfo.NativeRep()
	sourceInfo := nativeAST.SourceInfo()
	expr := nativeAST.Expr()

	// Walk the AST to find the deepest matching expression at the cursor position
	// Track comprehension variables for proper scoping
	compVars := make(map[string]bool)
	result := walkCELExprForHover(expr, sourceInfo, offset, exprInfo.expression, compVars, exprInfo, typeMap)

	// If we didn't find anything in the main AST, check macro calls in the parsed AST
	// Macros are expanded in the main AST, but parsedAst.MacroCalls preserves the original macro syntax
	if result == nil && parsedAst != nil {
		result = findMacroAtOffset(parsedAst.NativeRep().SourceInfo(), offset, exprInfo.expression)
	}

	return result
}

// findMacroAtOffset checks if the offset is within a CEL macro call.
// Macros like has(), all(), exists(), map(), filter() are expanded in the main AST,
// but sourceInfo.MacroCalls preserves the original macro call information.
func findMacroAtOffset(sourceInfo *ast.SourceInfo, offset int, exprString string) *celHoverInfo {
	if sourceInfo == nil {
		return nil
	}

	// Process each macro call
	for macroID, macroExpr := range sourceInfo.MacroCalls() {
		if macroExpr.Kind() != ast.CallKind {
			continue
		}
		call := macroExpr.AsCall()
		funcName := call.FunctionName()

		// Only process recognized CEL macros
		if !isCELMacroFunction(funcName) {
			continue
		}

		// Get the position of the macro call (rune offset from CEL)
		startLoc := sourceInfo.GetStartLocation(macroID)
		if startLoc.Line() <= 0 {
			continue
		}

		// For method-style macros like this.all(...), find the method name after the dot
		// For standalone macros like has(...), find the function name before the paren
		var funcStart, funcEnd int
		var found bool

		if call.IsMemberFunction() {
			// Method call - search for ".funcName" after the target
			targetID := call.Target().ID()
			targetLoc := sourceInfo.GetStartLocation(targetID)
			if targetLoc.Line() > 0 {
				targetByteOffset := celLocByteOffset(targetLoc.Line(), targetLoc.Column(), sourceInfo, exprString)
				funcStart, funcEnd = findMethodNameAfterDot(targetByteOffset, funcName, exprString)
				found = funcStart >= 0
			}
		} else {
			// Standalone function call - CEL position points to opening paren, look backwards
			celByteOffset := celLocByteOffset(startLoc.Line(), startLoc.Column(), sourceInfo, exprString)
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
	expr ast.Expr,
	sourceInfo *ast.SourceInfo,
	offset int,
	exprString string,
	compVars map[string]bool,
	exprInfo celExpressionInfo,
	typeMap map[int64]*types.Type,
) *celHoverInfo {
	if expr == nil || expr.Kind() == ast.UnspecifiedExprKind {
		return nil
	}

	// Get the CEL rune offset for this expression and convert to byte offset.
	// CEL tracks positions as Unicode code point (rune) offsets, not byte offsets.
	startLoc := sourceInfo.GetStartLocation(expr.ID())
	if startLoc.Line() <= 0 {
		return nil
	}
	celByteOffset := celLocByteOffset(startLoc.Line(), startLoc.Column(), sourceInfo, exprString)

	// Variable to hold the best match (deepest expression containing offset)
	var bestMatch *celHoverInfo

	switch expr.Kind() {
	case ast.IdentKind:
		// Identifier reference (variable, field, keyword)
		identName := expr.AsIdent()

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

		if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
			byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
			if offset >= byteStart && offset < byteStop {
				hoverInfo := &celHoverInfo{
					kind:   hoverKind,
					text:   identName,
					start:  byteStart,
					end:    byteStop,
					exprID: expr.ID(),
				}
				// Add type information if available
				if typeMap != nil {
					if celType, hasType := typeMap[expr.ID()]; hasType {
						hoverInfo.celType = celType
					}
				}
				bestMatch = hoverInfo
			}
		}

	case ast.SelectKind:
		// Field access (target.field)
		sel := expr.AsSelect()

		// Walk target first (might contain the offset)
		if sel.Operand() != nil {
			if targetMatch := walkCELExprForHover(sel.Operand(), sourceInfo, offset, exprString, compVars, exprInfo, typeMap); targetMatch != nil {
				bestMatch = targetMatch
			}
		}

		// Check if cursor is on the field name itself
		if sel.Operand() != nil {
			targetLoc := sourceInfo.GetStartLocation(sel.Operand().ID())
			if targetLoc.Line() > 0 {
				targetByteOffset := celLocByteOffset(targetLoc.Line(), targetLoc.Column(), sourceInfo, exprString)
				fieldStart, fieldEnd := findMethodNameAfterDot(targetByteOffset, sel.FieldName(), exprString)
				if fieldStart >= 0 && offset >= fieldStart && offset < fieldEnd {
					// Cursor is on the field name
					hoverInfo := &celHoverInfo{
						kind:   celHoverField,
						text:   sel.FieldName(),
						start:  fieldStart,
						end:    fieldEnd,
						exprID: expr.ID(),
					}
					// Add type information
					if typeMap != nil {
						if celType, hasType := typeMap[expr.ID()]; hasType {
							hoverInfo.celType = celType
						}
					}
					// Try to resolve proto field
					hoverInfo.protoMember = resolveCELFieldAccess(sel, exprInfo)
					bestMatch = hoverInfo
				}
			}
		}

	case ast.CallKind:
		// Function call or operator
		call := expr.AsCall()

		// Walk target first (for method calls)
		if call.IsMemberFunction() {
			if targetMatch := walkCELExprForHover(call.Target(), sourceInfo, offset, exprString, compVars, exprInfo, typeMap); targetMatch != nil {
				bestMatch = targetMatch
			}
		}

		// Walk arguments to see if offset is in any of them
		for _, arg := range call.Args() {
			if argMatch := walkCELExprForHover(arg, sourceInfo, offset, exprString, compVars, exprInfo, typeMap); argMatch != nil {
				bestMatch = argMatch
			}
		}

		funcName := call.FunctionName()

		// Check if this is an operator
		if _, isOperator := celOperatorSymbol(funcName); isOperator {
			if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
				byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
				if offset >= byteStart && offset < byteStop {
					bestMatch = &celHoverInfo{
						kind:   celHoverOperator,
						text:   funcName,
						start:  byteStart,
						end:    byteStop,
						exprID: expr.ID(),
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

			if call.IsMemberFunction() {
				// Method call - search for the function name after the target
				targetLoc := sourceInfo.GetStartLocation(call.Target().ID())
				if targetLoc.Line() > 0 {
					targetByteOffset := celLocByteOffset(targetLoc.Line(), targetLoc.Column(), sourceInfo, exprString)
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
					exprID: expr.ID(),
				}
			}
		}

	case ast.LiteralKind:
		// Constant literal
		if offsetRange, ok := sourceInfo.GetOffsetRange(expr.ID()); ok {
			byteStart, byteStop := celOffsetRangeToByteRange(exprString, offsetRange)
			if offset >= byteStart && offset < byteStop {
				bestMatch = &celHoverInfo{
					kind:    celHoverLiteral,
					text:    exprString[byteStart:byteStop],
					start:   byteStart,
					end:     byteStop,
					exprID:  expr.ID(),
					celType: getPrimitiveTypeForLiteral(expr),
				}
			}
		}

	case ast.ListKind:
		// List literal - walk all elements
		for _, elem := range expr.AsList().Elements() {
			if elemMatch := walkCELExprForHover(elem, sourceInfo, offset, exprString, compVars, exprInfo, typeMap); elemMatch != nil {
				bestMatch = elemMatch
			}
		}

	case ast.MapKind:
		// Map literal - walk all entries
		for _, entry := range expr.AsMap().Entries() {
			mapEntry := entry.AsMapEntry()
			if keyMatch := walkCELExprForHover(mapEntry.Key(), sourceInfo, offset, exprString, compVars, exprInfo, typeMap); keyMatch != nil {
				bestMatch = keyMatch
			}
			if valueMatch := walkCELExprForHover(mapEntry.Value(), sourceInfo, offset, exprString, compVars, exprInfo, typeMap); valueMatch != nil {
				bestMatch = valueMatch
			}
		}

	case ast.StructKind:
		// Struct literal - walk all field values
		for _, field := range expr.AsStruct().Fields() {
			if fieldMatch := walkCELExprForHover(field.AsStructField().Value(), sourceInfo, offset, exprString, compVars, exprInfo, typeMap); fieldMatch != nil {
				bestMatch = fieldMatch
			}
		}

	case ast.ComprehensionKind:
		// List comprehension - walk all parts with proper variable scoping
		comp := expr.AsComprehension()

		// Walk the range and init with current scope (they don't see loop variables)
		if rangeMatch := walkCELExprForHover(comp.IterRange(), sourceInfo, offset, exprString, compVars, exprInfo, typeMap); rangeMatch != nil {
			bestMatch = rangeMatch
		}
		if initMatch := walkCELExprForHover(comp.AccuInit(), sourceInfo, offset, exprString, compVars, exprInfo, typeMap); initMatch != nil {
			bestMatch = initMatch
		}

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
		if condMatch := walkCELExprForHover(comp.LoopCondition(), sourceInfo, offset, exprString, extendedVars, exprInfo, typeMap); condMatch != nil {
			bestMatch = condMatch
		}
		if stepMatch := walkCELExprForHover(comp.LoopStep(), sourceInfo, offset, exprString, extendedVars, exprInfo, typeMap); stepMatch != nil {
			bestMatch = stepMatch
		}
		if resultMatch := walkCELExprForHover(comp.Result(), sourceInfo, offset, exprString, extendedVars, exprInfo, typeMap); resultMatch != nil {
			bestMatch = resultMatch
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
func resolveCELFieldAccess(sel ast.SelectExpr, exprInfo celExpressionInfo) ir.Member {
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
	for {
		fieldPath = append([]string{currentSel.FieldName()}, fieldPath...)

		// Check if the operand is another select expression
		operand := currentSel.Operand()
		if operand == nil || operand.Kind() == ast.UnspecifiedExprKind {
			break
		}
		if operand.Kind() == ast.SelectKind {
			currentSel = operand.AsSelect()
			continue
		}
		// Non-select operand: must be 'this' to resolve
		if operand.Kind() != ast.IdentKind || operand.AsIdent() != "this" {
			return ir.Member{}
		}
		break
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
func getCELTypeString(t *types.Type) string {
	if t == nil {
		return "unknown"
	}
	switch t.Kind() {
	case types.ListKind:
		params := t.Parameters()
		if len(params) > 0 {
			return fmt.Sprintf("list(%s)", getCELTypeString(params[0]))
		}
	case types.MapKind:
		params := t.Parameters()
		if len(params) >= 2 {
			return fmt.Sprintf("map(%s, %s)", getCELTypeString(params[0]), getCELTypeString(params[1]))
		}
	}
	if name := t.TypeName(); name != "" {
		return name
	}
	return "unknown"
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

// getCELOperatorDocs returns documentation for CEL operators.
// funcName is the internal operator function name (e.g., "_&&_").
func getCELOperatorDocs(funcName string, celEnv *cel.Env) string {
	if celEnv == nil {
		return ""
	}
	funcs := celEnv.Functions()
	funcDecl, ok := funcs[funcName]
	if !ok {
		return ""
	}
	symbol, _ := celOperatorSymbol(funcName)
	if doc := funcDecl.Documentation(); doc != nil {
		doc.Name = symbol
		return formatCELDoc(doc, "**Operator**: ")
	}
	if desc := funcDecl.Description(); desc != "" {
		return fmt.Sprintf("**Operator**: `%s`\n\n%s", symbol, desc)
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

// getPrimitiveTypeForLiteral returns the CEL type for a literal expression.
func getPrimitiveTypeForLiteral(expr ast.Expr) *types.Type {
	switch expr.AsLiteral().(type) {
	case types.String:
		return types.StringType
	case types.Int:
		return types.IntType
	case types.Uint:
		return types.UintType
	case types.Double:
		return types.DoubleType
	case types.Bool:
		return types.BoolType
	case types.Bytes:
		return types.BytesType
	default:
		return nil
	}
}
