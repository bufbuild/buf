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

// This file implements CEL expression signature help support for the LSP.
// CEL expressions appear in protovalidate (buf.validate) options.

package buflsp

import (
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/decls"
	"github.com/google/cel-go/common/types"
	"go.lsp.dev/protocol"
)

// celSignatureHelpTriggers lists the characters that should trigger signature
// help in a CEL expression. Kept here alongside the implementation so that the
// server capability declaration and the logic stay in sync.
var celSignatureHelpTriggers = []string{"(", ","}

// getCELSignatureHelp returns signature help if the cursor is inside a CEL
// function call argument list. Returns nil if the position is not inside a
// CEL expression or call.
func getCELSignatureHelp(
	file *file,
	position protocol.Position,
	celEnv *cel.Env,
) *protocol.SignatureHelp {
	byteOffset := positionToOffset(file, position)

	for _, symbol := range file.symbols {
		for _, exprInfo := range extractCELExpressions(file, symbol) {
			if byteOffset < exprInfo.span.Start || byteOffset >= exprInfo.span.End {
				continue
			}

			celOffset := fileByteOffsetToCELOffset(byteOffset, exprInfo.span)
			if celOffset < 0 || celOffset > len(exprInfo.expression) {
				continue
			}

			parsedAST, parseIssues := celEnv.Parse(exprInfo.expression)
			if parseIssues.Err() != nil {
				// Expression is incomplete (e.g. cursor right after '(').
				// Fall back to a text scan: find the enclosing function name
				// and return all its overloads without type filtering.
				funcName, isMember, parenOffset := celFindCallAtOffset(exprInfo.expression, celOffset)
				if funcName == "" || celIsOperatorOrInternal(funcName) {
					continue
				}
				funcDecl, ok := celEnv.Functions()[funcName]
				if !ok {
					continue
				}
				sigs := generateCELSignatures(funcDecl, isMember, nil, nil)
				if len(sigs) == 0 {
					continue
				}
				return &protocol.SignatureHelp{
					Signatures:      sigs,
					ActiveSignature: 0,
					ActiveParameter: celCountParamsBeforeOffset(exprInfo.expression, parenOffset, celOffset),
				}
			}

			nativeAST := parsedAST.NativeRep()
			call, paramIndex := findCELCallAtPosition(
				nativeAST.Expr(),
				nativeAST.SourceInfo(),
				exprInfo.expression,
				celOffset,
			)
			if call == nil {
				continue
			}

			funcName := call.FunctionName()
			// Operators (e.g. _+_, _&&_) are represented as calls in the CEL
			// AST but are not user-callable functions; skip them.
			if celIsOperatorOrInternal(funcName) {
				continue
			}
			funcDecl, ok := celEnv.Functions()[funcName]
			if !ok {
				continue
			}

			// Try the type-checked AST; used to filter overloads for both
			// member and global calls.
			var typeMap map[int64]*types.Type
			if checkedAST, compileIssues := celEnv.Compile(exprInfo.expression); compileIssues.Err() == nil {
				typeMap = checkedAST.NativeRep().TypeMap()
			}

			// For member function calls, determine the receiver type so we can
			// filter overloads to only those applicable to that receiver.
			var receiverType *types.Type
			if call.IsMemberFunction() {
				target := call.Target()
				if typeMap != nil {
					if celType, hasType := typeMap[target.ID()]; hasType {
						receiverType = celType
					}
				}
				// Fall back to the proto IR type of `this` when the target is the
				// `this` identifier and compilation didn't provide a type.
				if receiverType == nil && !exprInfo.thisIRType.IsZero() {
					if target != nil && target.Kind() == ast.IdentKind && target.AsIdent() == "this" {
						receiverType = celProtoTypeToExprType(exprInfo.thisIRType)
					}
				}
			}

			// Determine the types of the call arguments to filter overloads.
			// This handles global calls like size(this) where 'this' is a string.
			argTypes := make([]*types.Type, len(call.Args()))
			for i, arg := range call.Args() {
				if typeMap != nil {
					if celType, hasType := typeMap[arg.ID()]; hasType {
						argTypes[i] = celType
						continue
					}
				}
				// Fall back: if this arg is the 'this' identifier, use the IR type.
				if !exprInfo.thisIRType.IsZero() && arg.Kind() == ast.IdentKind && arg.AsIdent() == "this" {
					argTypes[i] = celProtoTypeToExprType(exprInfo.thisIRType)
				}
			}

			sigs := generateCELSignatures(funcDecl, call.IsMemberFunction(), receiverType, argTypes)
			if len(sigs) == 0 {
				continue
			}

			return &protocol.SignatureHelp{
				Signatures:      sigs,
				ActiveSignature: 0,
				ActiveParameter: paramIndex,
			}
		}
	}

	return nil
}

// findCELCallAtPosition walks the CEL AST to find the deepest call expression
// whose argument list contains celOffset. Returns the call and the 0-based index
// of the active parameter.
func findCELCallAtPosition(
	expr ast.Expr,
	sourceInfo *ast.SourceInfo,
	exprString string,
	targetOffset int,
) (ast.CallExpr, uint32) {
	var result ast.CallExpr
	var paramIndex uint32
	var bestRange [2]int

	var walk func(ast.Expr)
	walk = func(e ast.Expr) {
		if e == nil {
			return
		}

		switch e.Kind() {
		case ast.CallKind:
			call := e.AsCall()
			offsetRange, hasOffset := sourceInfo.GetOffsetRange(e.ID())
			if hasOffset {
				byteStart, _ := celOffsetRangeToByteRange(exprString, offsetRange)
				// Find the opening paren of the argument list.
				parenIdx := strings.Index(exprString[byteStart:], "(")
				if parenIdx >= 0 {
					parenStart := byteStart + parenIdx
					parenEnd := parenStart + 1
					depth := 1
					for parenEnd < len(exprString) && depth > 0 {
						switch exprString[parenEnd] {
						case '(':
							depth++
						case ')':
							depth--
						}
						parenEnd++
					}
					// Cursor must be inside the parens: after '(' and before (or at) ')'.
					if targetOffset > parenStart && targetOffset <= parenEnd {
						callRange := parenEnd - parenStart
						if result == nil || callRange < (bestRange[1]-bestRange[0]) {
							result = call
							paramIndex = countCELParametersBeforeCursor(exprString, parenStart, targetOffset, call)
							bestRange = [2]int{parenStart, parenEnd}
						}
					}
				}
			}
			for _, arg := range call.Args() {
				walk(arg)
			}
			if call.IsMemberFunction() {
				walk(call.Target())
			}
		case ast.ListKind:
			for _, elem := range e.AsList().Elements() {
				walk(elem)
			}
		case ast.MapKind:
			for _, entry := range e.AsMap().Entries() {
				mapEntry := entry.AsMapEntry()
				walk(mapEntry.Key())
				walk(mapEntry.Value())
			}
		case ast.StructKind:
			for _, field := range e.AsStruct().Fields() {
				walk(field.AsStructField().Value())
			}
		case ast.SelectKind:
			sel := e.AsSelect()
			if sel.Operand() != nil {
				walk(sel.Operand())
			}
		case ast.ComprehensionKind:
			comp := e.AsComprehension()
			walk(comp.IterRange())
			walk(comp.AccuInit())
			walk(comp.LoopCondition())
			walk(comp.LoopStep())
			walk(comp.Result())
		}
	}
	walk(expr)
	return result, paramIndex
}

// countCELParametersBeforeCursor returns the 0-based index of the active
// parameter at cursorOffset, counting depth-0 commas after the opening paren.
func countCELParametersBeforeCursor(exprString string, parenStart, cursorOffset int, call ast.CallExpr) uint32 {
	var paramIndex uint32
	depth := 0
	for i := parenStart + 1; i < len(exprString) && i < cursorOffset; i++ {
		switch exprString[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				paramIndex++
			}
		}
	}
	if len(call.Args()) == 0 {
		return 0
	}
	if int(paramIndex) >= len(call.Args()) {
		return uint32(len(call.Args()) - 1)
	}
	return paramIndex
}

// generateCELSignatures creates signature information for the overloads of
// funcDecl, filtered to member or global overloads based on isMemberFunction.
// receiverType filters member overloads by the receiver's type. argTypes
// filters all overloads by known argument types (nil entries are ignored).
func generateCELSignatures(funcDecl *decls.FunctionDecl, isMemberFunction bool, receiverType *types.Type, argTypes []*types.Type) []protocol.SignatureInformation {
	var sigs []protocol.SignatureInformation
	for _, o := range funcDecl.OverloadDecls() {
		if o.IsMemberFunction() != isMemberFunction {
			continue
		}
		oArgTypes := o.ArgTypes()
		start := 0
		if isMemberFunction {
			if receiverType != nil && len(oArgTypes) > 0 {
				if !oArgTypes[0].IsAssignableType(receiverType) {
					continue
				}
			}
			start = 1
		}
		// Filter by known argument types. A call with more arguments than this
		// overload accepts cannot match.
		overloadArgs := oArgTypes[start:]
		if len(argTypes) > len(overloadArgs) {
			continue
		}
		compatible := true
		for i, knownType := range argTypes {
			if knownType == nil {
				continue
			}
			if !overloadArgs[i].IsAssignableType(knownType) {
				compatible = false
				break
			}
		}
		if !compatible {
			continue
		}
		label := celFormatOverloadSignature(funcDecl.Name(), o)
		sig := protocol.SignatureInformation{
			Label:      label,
			Parameters: extractCELSignatureParameters(o),
		}
		docStr := funcDecl.Description()
		if doc := funcDecl.Documentation(); doc != nil && doc.Description != "" {
			docStr = doc.Description
		}
		if docStr != "" {
			sig.Documentation = protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: docStr,
			}
		}
		sigs = append(sigs, sig)
	}
	return sigs
}

// extractCELSignatureParameters returns ParameterInformation for each argument
// of overload o. For member functions, the receiver (first arg type) is skipped
// because it is already encoded in the signature label prefix.
func extractCELSignatureParameters(o *decls.OverloadDecl) []protocol.ParameterInformation {
	argTypes := o.ArgTypes()
	start := 0
	if o.IsMemberFunction() && len(argTypes) > 0 {
		start = 1 // skip the receiver type
	}
	if start >= len(argTypes) {
		return nil
	}
	params := make([]protocol.ParameterInformation, 0, len(argTypes)-start)
	for _, arg := range argTypes[start:] {
		params = append(params, protocol.ParameterInformation{
			Label: arg.String(),
		})
	}
	return params
}

// celFindCallAtOffset scans expression backward from offset to find the
// innermost unclosed function call. Returns the function name, whether it is
// a member call (preceded by a dot), and the byte offset of the opening paren.
// Returns ("", false, -1) if no enclosing call is found.
func celFindCallAtOffset(expression string, offset int) (funcName string, isMember bool, parenOffset int) {
	depth := 0
	for i := offset - 1; i >= 0; i-- {
		switch expression[i] {
		case ')':
			depth++
		case '(':
			if depth > 0 {
				depth--
				continue
			}
			// Found the unclosed opening paren. Extract the name before it.
			j := i - 1
			for j >= 0 && celIsIdentChar(expression[j]) {
				j--
			}
			name := expression[j+1 : i]
			if name == "" {
				return "", false, -1
			}
			member := j >= 0 && expression[j] == '.'
			return name, member, i
		}
	}
	return "", false, -1
}

// celCountParamsBeforeOffset counts depth-0 commas between parenOffset+1 and
// offset, returning the 0-based active parameter index.
func celCountParamsBeforeOffset(expression string, parenOffset, offset int) uint32 {
	var paramIndex uint32
	depth := 0
	for i := parenOffset + 1; i < len(expression) && i < offset; i++ {
		switch expression[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				paramIndex++
			}
		}
	}
	return paramIndex
}
