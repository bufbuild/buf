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

// This file implements CEL expression completion support for the LSP.
// CEL expressions appear in protovalidate (buf.validate) options.

package buflsp

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bufbuild/protocompile/experimental/ast/predeclared"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/decls"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
	"go.lsp.dev/protocol"
)

// celKeywords are CEL literal keywords, the protovalidate "this" special variable,
// and the "now" runtime variable (google.protobuf.Timestamp) declared by protovalidate.
var celKeywords = []string{"true", "false", "null", "this", "now"}

// getCELCompletionItems returns completion items if the cursor is inside a CEL expression.
// Returns nil if the position is not inside a CEL expression.
func getCELCompletionItems(
	file *file,
	position protocol.Position,
	celEnv *cel.Env,
) []protocol.CompletionItem {
	byteOffset := positionToOffset(file, position)

	for _, symbol := range file.symbols {
		for _, exprInfo := range extractCELExpressions(file, symbol) {
			if byteOffset < exprInfo.span.Start || byteOffset >= exprInfo.span.End {
				continue
			}

			celOffset := fileByteOffsetToCELOffset(byteOffset, exprInfo.span)
			if celOffset < 0 {
				// Cursor is at the closing quote or inside an empty string literal;
				// treat as end-of-expression for completion purposes.
				celOffset = len(exprInfo.expression)
			}
			if celOffset > len(exprInfo.expression) {
				continue
			}

			celContent := exprInfo.expression[:celOffset]

			// If the cursor is surrounded by identifier characters on both sides,
			// it is placed inside an existing token — do not offer completions.
			// Checking both sides avoids suppressing completions at the start of
			// an expression (where celContent is empty) when the next char happens
			// to be an identifier (e.g. cursor before "this.").
			if celOffset < len(exprInfo.expression) && len(celContent) > 0 {
				next := exprInfo.expression[celOffset]
				prev := celContent[len(celContent)-1]
				if celIsIdentChar(next) && celIsIdentChar(prev) {
					return nil
				}
			}

			// Determine the content to analyze for member access by stripping outer
			// context layers: comprehension bodies first (to capture the iter var type),
			// then has() macro arguments.
			memberAccessContent := celContent
			iterVarTypes := map[string]ir.Type{}

			// Peel comprehension layers from outermost to innermost.
			// Each iteration resolves the range expression (using all known variables
			// accumulated so far) to bind the iteration variable, then advances
			// memberAccessContent into the body expression for the next pass.
			//
			// lastRangeType tracks the element type from the previous layer. In chained
			// comprehensions (e.g. "list.filter(x, pred).all(y, y."), the inner macro's
			// range candidate is the predicate expression "pred" which cannot be resolved
			// as a type. In that case we fall back to the previous element type because
			// filter/map/all preserve the element type of the input list.
			var lastRangeType ir.Type
			for {
				rangeExpr, iterVar, body, inBody := celParseIteratorBody(memberAccessContent)
				if !inBody {
					break
				}
				vars := map[string]ir.Type{}
				if !exprInfo.thisIRType.IsZero() {
					vars["this"] = exprInfo.thisIRType
				}
				maps.Copy(vars, iterVarTypes)
				rangeType, _ := celResolveExprType(rangeExpr, vars)
				if rangeType.IsZero() {
					rangeType = lastRangeType
				}
				if !rangeType.IsZero() {
					iterVarTypes[iterVar] = rangeType
					lastRangeType = rangeType
				}
				memberAccessContent = body
			}

			// has() macro argument context: strip "has(" so the inner select
			// expression is handled by the regular member access path below.
			// E.g. "has(this.f" → memberAccessContent = "this.f".
			if innerContent, inHasArg := celParseHasArg(memberAccessContent); inHasArg {
				memberAccessContent = innerContent
			}

			// Member access context: cursor immediately after a dot or typing
			// a member name. Handles both "this." and "this.ci" (prefix "ci").
			if receiverExpr, memberPrefix, ok := celParseMemberAccess(memberAccessContent); ok {
				var items []protocol.CompletionItem

				// Try CEL compile to determine receiver type (works for variables like "now").
				var receiverCELType *types.Type
				if ast, iss := celEnv.Compile(receiverExpr); iss.Err() == nil {
					receiverCELType = ast.OutputType()
				}

				// Resolve via proto IR for field names and type mapping.
				// Build a variable map from "this" and any comprehension iteration variables.
				var isListOrMapReceiver bool
				if !exprInfo.thisIRType.IsZero() || len(iterVarTypes) > 0 {
					vars := map[string]ir.Type{}
					if !exprInfo.thisIRType.IsZero() {
						vars["this"] = exprInfo.thisIRType
					}
					maps.Copy(vars, iterVarTypes)
					irType, isCollection := celResolveExprType(receiverExpr, vars)
					if !irType.IsZero() {
						// For the direct "this" receiver, use irMember cardinality:
						// thisIRType is already the element type so cardinality is not in vars.
						if receiverExpr == "this" && !exprInfo.irMember.IsZero() {
							if exprInfo.irMember.IsMap() {
								if receiverCELType == nil {
									receiverCELType = celMapEntryToCELType(irType)
								}
								irType = ir.Type{}
								isListOrMapReceiver = true
							} else if exprInfo.irMember.IsRepeated() {
								if receiverCELType == nil {
									receiverCELType = types.NewListType(celProtoTypeToExprType(irType))
								}
								irType = ir.Type{}
								isListOrMapReceiver = true
							}
						} else if isCollection {
							// Path through a repeated or map field (e.g., "this.items."):
							// offer collection methods rather than element field names.
							if receiverCELType == nil {
								if irType.IsMessage() && irType.IsMapEntry() {
									receiverCELType = celMapEntryToCELType(irType)
								} else {
									receiverCELType = types.NewListType(celProtoTypeToExprType(irType))
								}
							}
							irType = ir.Type{}
							isListOrMapReceiver = true
						}
						if !irType.IsZero() {
							if receiverCELType == nil {
								receiverCELType = celProtoTypeToExprType(irType)
							}
							if irType.IsMessage() {
								items = append(items, celProtoFieldCompletionItems(irType)...)
							}
						}
					}
				}

				items = append(items, celMemberCompletionItems(celEnv, receiverCELType)...)
				if isListOrMapReceiver {
					items = append(items, celMemberMacroCompletionItems()...)
				}
				return celFilterByPrefix(items, memberPrefix)
			}

			// Extract the identifier prefix being typed at the end of celContent
			// so that operator context detection works on the preceding content.
			prefix := celCurrentPrefix(celContent)
			contentBeforePrefix := celContent[:len(celContent)-len(prefix)]

			// Detect unary NOT or binary operator context to narrow expected type.
			var expectedType *types.Type
			trimmed := strings.TrimRight(contentBeforePrefix, " \t\r\n")
			if strings.HasSuffix(trimmed, "!") {
				// After logical NOT, the operand must be bool.
				expectedType = types.BoolType
			} else {
				expectedType = celExpectedTypeAfterOperator(contentBeforePrefix, celEnv)
			}

			var items []protocol.CompletionItem
			items = append(items, celGlobalCompletionItems(celEnv, expectedType)...)
			items = append(items, celMacroCompletionItems(celEnv)...)
			items = append(items, celKeywordCompletionItems(celEnv, expectedType)...)
			return celFilterByPrefix(items, prefix)
		}
	}

	return nil
}

// celParseMemberAccess detects if celContent is in a member access context:
// either right after a dot ("this.") or typing a member name ("this.ci").
// Returns the receiver expression, the typed prefix (possibly empty), and ok=true.
// Examples:
//   - "this."     → ("this", "", true)
//   - "this.ci"   → ("this", "ci", true)
//   - "size("     → ("", "", false)
//   - "si"        → ("", "", false)
func celParseMemberAccess(celContent string) (receiverExpr, memberPrefix string, ok bool) {
	// Peel off the identifier being typed at the end of celContent.
	wordStart := len(celContent)
	for wordStart > 0 && celIsIdentChar(celContent[wordStart-1]) {
		wordStart--
	}
	beforeWord := celContent[:wordStart]
	prefix := celContent[wordStart:]

	// Member access requires a dot immediately before the current word (or cursor).
	trimmed := strings.TrimRight(beforeWord, " \t\r\n")
	if len(trimmed) == 0 || trimmed[len(trimmed)-1] != '.' {
		return "", "", false
	}

	// Receiver expression is everything before the dot.
	receiver := strings.TrimRight(trimmed[:len(trimmed)-1], " \t\r\n")
	if receiver == "" {
		return "", "", false
	}
	return receiver, prefix, true
}

// celParseHasArg detects if the cursor is inside the argument of a has() macro call.
// The has() macro takes a single select expression: has(receiver.field).
// Returns the content after the last unqualified "has(" when the cursor is still
// within the argument (i.e. no closing paren or comma has been written yet).
// "Unqualified" means has( is not preceded by an identifier character or dot —
// which would make it a method call (e.g. "foo.has(") rather than the macro.
// Examples:
//   - "has(this."    → ("this.", true)
//   - "has(this.f"   → ("this.f", true)
//   - "!has(this.f"  → ("this.f", true)
//   - "has(this.f)"  → ("", false)  — already closed
//   - "foo.has(this." → ("", false)  — method call, not the macro
func celParseHasArg(celContent string) (innerContent string, ok bool) {
	idx := strings.LastIndex(celContent, "has(")
	if idx < 0 {
		return "", false
	}
	// Reject method calls: "expr.has(" or "ident.has(" — the char before "has(" is a
	// dot or identifier character, indicating this is a method call, not the macro.
	if idx > 0 && (celIsIdentChar(celContent[idx-1]) || celContent[idx-1] == '.') {
		return "", false
	}
	inner := celContent[idx+4:]
	// If we have already passed a closing paren or argument separator, we are
	// no longer inside the has() argument.
	if strings.ContainsAny(inner, "),") {
		return "", false
	}
	return inner, true
}

// celIteratorMacros are the CEL comprehension macros that bind an iteration variable.
var celIteratorMacros = []string{"all", "exists", "exists_one", "filter", "map"}

// celParseIteratorBody detects if the cursor is inside the body expression of a CEL
// comprehension macro call (all, exists, filter, map, etc.).
// It finds the leftmost (outermost) such macro in celContent so that callers can
// process comprehension layers from outer to inner by calling it in a loop.
// Returns the range expression, the iteration variable name, and the body content
// (the expression fragment after the separating comma), or ok=false if not matched.
// Examples:
//
//	"this.items.filter(addr, addr."  → ("this.items", "addr", "addr.", true)
//	"items.all(x, x.name"            → ("items", "x", "x.name", true)
//	"this.filter("                   → ("", "", "", false)  — no comma yet
func celParseIteratorBody(celContent string) (rangeExpr, iterVar, bodyContent string, ok bool) {
	// Find the leftmost occurrence of any comprehension macro pattern.
	bestIdx := -1
	bestMacro := ""
	for _, macro := range celIteratorMacros {
		pattern := "." + macro + "("
		if idx := strings.Index(celContent, pattern); idx >= 0 {
			if bestIdx < 0 || idx < bestIdx {
				bestIdx = idx
				bestMacro = macro
			}
		}
	}
	if bestIdx < 0 {
		return "", "", "", false
	}
	pattern := "." + bestMacro + "("
	afterParen := celContent[bestIdx+len(pattern):]
	beforeComma, afterComma, hasComma := strings.Cut(afterParen, ",")
	if !hasComma {
		return "", "", "", false
	}
	candidate := strings.TrimSpace(beforeComma)
	if candidate == "" {
		return "", "", "", false
	}
	// Iteration variable must be a plain identifier with no operators or parens.
	for i := 0; i < len(candidate); i++ {
		if !celIsIdentChar(candidate[i]) {
			return "", "", "", false
		}
	}
	body := strings.TrimLeft(afterComma, " \t\r\n")
	rangeCandidate := strings.TrimRight(celContent[:bestIdx], " \t\r\n")
	// Strip unbalanced trailing closing parens. These appear when this expression
	// is itself the body of an outer comprehension that was already peeled — e.g.
	// "this.items.filter(item, item.addresses).all(addr, addr." yields a range
	// candidate of "item.addresses)" with one unmatched ")".
	opens := strings.Count(rangeCandidate, "(")
	closes := strings.Count(rangeCandidate, ")")
	for closes > opens && strings.HasSuffix(rangeCandidate, ")") {
		rangeCandidate = strings.TrimRight(rangeCandidate[:len(rangeCandidate)-1], " \t\r\n")
		closes--
	}
	return rangeCandidate, candidate, body, true
}

// celCurrentPrefix extracts the identifier being typed at the end of celContent.
// Returns an empty string if celContent ends with a non-identifier character.
func celCurrentPrefix(celContent string) string {
	i := len(celContent)
	for i > 0 && celIsIdentChar(celContent[i-1]) {
		i--
	}
	return celContent[i:]
}

// celFilterByPrefix filters completion items to those whose label starts with prefix.
// Returns the full list unchanged when prefix is empty.
func celFilterByPrefix(items []protocol.CompletionItem, prefix string) []protocol.CompletionItem {
	if prefix == "" {
		return items
	}
	filtered := make([]protocol.CompletionItem, 0, len(items))
	for _, item := range items {
		if strings.HasPrefix(item.Label, prefix) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// celResolveExprType resolves a CEL expression to its ir.Type, supporting both
// dot access ("this.field.sub") and bracket-index notation ("this.items[0]" or
// "this.locations[\"key\"]"). The second return value is true when the final
// navigation step traverses a repeated or map field, indicating the returned type
// is the element type of a list or map (so callers can offer collection methods
// rather than individual field names).
//
// Examples (with vars = {"this": LocationType, "addr": AddressType}):
//
//	"this.address.city"          → (string, false)
//	"this.items"                 → (Address, true)   // repeated field
//	"this.items[0]"              → (Address, false)  // element after indexing
//	"this.locations[\"home\"]" → (Address, false)  // map-value after indexing
//	"addr.zip_code"              → (int32, false)
func celResolveExprType(expr string, vars map[string]ir.Type) (ir.Type, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ir.Type{}, false
	}

	// Handle bracket-index access: receiver[expr].
	// After indexing, the result is always a single element (not a collection).
	if strings.HasSuffix(expr, "]") {
		bracketOpen := strings.LastIndex(expr, "[")
		if bracketOpen < 0 {
			return ir.Type{}, false
		}
		receiverType, _ := celResolveExprType(expr[:bracketOpen], vars)
		if receiverType.IsZero() {
			return ir.Type{}, false
		}
		if receiverType.IsMessage() && receiverType.IsMapEntry() {
			// Map indexing: return the map value element type.
			_, valueMember := receiverType.EntryFields()
			if valueMember.IsZero() {
				return ir.Type{}, false
			}
			return valueMember.Element(), false
		}
		// List indexing: the receiver is already the element type.
		return receiverType, false
	}

	// Handle dot access: split on the last dot.
	dotIdx := strings.LastIndex(expr, ".")
	if dotIdx < 0 {
		// Simple root variable lookup.
		t, ok := vars[expr]
		if !ok {
			return ir.Type{}, false
		}
		return t, false
	}

	fieldName := strings.TrimSpace(expr[dotIdx+1:])
	receiverExpr := expr[:dotIdx]
	if fieldName == "" {
		return ir.Type{}, false
	}

	receiverType, _ := celResolveExprType(receiverExpr, vars)
	if receiverType.IsZero() || !receiverType.IsMessage() {
		return ir.Type{}, false
	}

	member := receiverType.MemberByName(fieldName)
	if member.IsZero() {
		return ir.Type{}, false
	}

	// Return element type and whether this step crosses a collection boundary.
	return member.Element(), member.IsRepeated() || member.IsMap()
}

// celProtoTypeToExprType maps a proto ir.Type to the corresponding CEL *types.Type.
// Returns nil if the type cannot be mapped.
func celProtoTypeToExprType(irType ir.Type) *types.Type {
	if irType.IsZero() {
		return nil
	}
	if irType.IsMessage() {
		return types.NewObjectType(string(irType.FullName()))
	}
	if irType.IsEnum() {
		return types.IntType // enums are integers in CEL
	}
	if !irType.IsPredeclared() {
		return nil
	}
	switch irType.Predeclared() {
	case predeclared.Bool:
		return types.BoolType
	case predeclared.String:
		return types.StringType
	case predeclared.Bytes:
		return types.BytesType
	case predeclared.Float, predeclared.Double:
		return types.DoubleType
	case predeclared.UInt32, predeclared.UInt64, predeclared.Fixed32, predeclared.Fixed64:
		return types.UintType
	case predeclared.Int32, predeclared.Int64,
		predeclared.SInt32, predeclared.SInt64,
		predeclared.SFixed32, predeclared.SFixed64:
		return types.IntType
	default:
		return nil
	}
}

// celMapEntryToCELType converts a map-entry ir.Type to the corresponding CEL map type.
// Map fields in proto are represented internally as repeated MapEntry messages;
// this function extracts the key and value types to produce a CEL map type.
// Returns nil if irType is not a valid map entry.
func celMapEntryToCELType(irType ir.Type) *types.Type {
	if irType.IsZero() || !irType.IsMessage() || !irType.IsMapEntry() {
		return nil
	}
	keyMember, valueMember := irType.EntryFields()
	if keyMember.IsZero() || valueMember.IsZero() {
		return nil
	}
	keyType := celProtoTypeToExprType(keyMember.Element())
	valueType := celProtoTypeToExprType(valueMember.Element())
	if keyType == nil || valueType == nil {
		return nil
	}
	return types.NewMapType(keyType, valueType)
}

// celProtoFieldCompletionItems returns completion items for the fields of a proto message type.
func celProtoFieldCompletionItems(irType ir.Type) []protocol.CompletionItem {
	if irType.IsZero() || !irType.IsMessage() {
		return nil
	}
	var items []protocol.CompletionItem
	for member := range seq.Values(irType.Members()) {
		if member.IsSynthetic() || member.IsZero() {
			continue
		}
		fieldType := member.Element()
		detail := ""
		if !fieldType.IsZero() {
			detail = string(fieldType.FullName())
		}
		_, isDeprecated := member.Deprecated().AsBool()
		var documentation any
		if doc := leadingDocComments(member.AST()); doc != "" {
			documentation = &protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: doc,
			}
		}
		items = append(items, protocol.CompletionItem{
			Label:         member.Name(),
			Kind:          protocol.CompletionItemKindField,
			Detail:        detail,
			Documentation: documentation,
			Deprecated:    isDeprecated,
		})
	}
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(a.Label, b.Label)
	})
	return items
}

// celBinaryOperatorSymbols returns a map from display symbol (e.g. "&&") to
// cel-go internal name (e.g. "_&&_") for all binary operators in the environment.
func celBinaryOperatorSymbols(celEnv *cel.Env) map[string]string {
	result := make(map[string]string)
	for name := range celEnv.Functions() {
		display, ok := operators.FindReverseBinaryOperator(name)
		if ok && display != "" {
			if _, exists := result[display]; !exists {
				result[display] = name
			}
		}
	}
	return result
}

// celExpectedTypeAfterOperator checks if celContent ends with a binary operator and
// returns the expected type for the right-hand operand. Returns nil if no binary
// operator context is detected or the expected type cannot be determined.
func celExpectedTypeAfterOperator(celContent string, celEnv *cel.Env) *types.Type {
	before := strings.TrimRight(celContent, " \t\r\n")
	if before == "" {
		return nil
	}

	opSymbols := celBinaryOperatorSymbols(celEnv)

	// Sort display symbols longest-first so multi-character operators (e.g. "&&")
	// are matched before single-character ones (e.g. ">").
	symbols := make([]string, 0, len(opSymbols))
	for sym := range opSymbols {
		symbols = append(symbols, sym)
	}
	slices.SortFunc(symbols, func(a, b string) int {
		return cmp.Compare(len(b), len(a))
	})

	// Find the binary operator at the end of the trimmed content.
	var celOp, leftExpr string
	for _, sym := range symbols {
		if strings.HasSuffix(before, sym) {
			candidate := strings.TrimRight(before[:len(before)-len(sym)], " \t\r\n")
			if candidate == "" {
				continue
			}
			celOp = opSymbols[sym]
			leftExpr = candidate
			break
		}
	}
	if celOp == "" {
		return nil
	}

	// Compile the left-hand expression to determine its type.
	leftAst, iss := celEnv.Compile(leftExpr)
	if iss.Err() != nil {
		return nil
	}
	leftType := leftAst.OutputType()

	// Find operator overloads accepting leftType and collect the expected right-hand types.
	fn, ok := celEnv.Functions()[celOp]
	if !ok {
		return nil
	}

	rightTypes := make(map[string]*types.Type)
	for _, o := range fn.OverloadDecls() {
		args := o.ArgTypes()
		if len(args) != 2 {
			continue
		}
		if !args[0].IsAssignableType(leftType) && !leftType.IsAssignableType(args[0]) {
			continue
		}
		right := args[1]
		if celIsTypeParam(right) {
			right = leftType
		}
		rightTypes[right.String()] = right
	}

	// Only narrow the type if there is exactly one expected type.
	if len(rightTypes) == 1 {
		for _, t := range rightTypes {
			return t
		}
	}
	return nil
}

// celIsTypeParam returns true if t is a type parameter (e.g. <A>).
func celIsTypeParam(t *types.Type) bool {
	s := t.String()
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")
}

// celTypeMatches returns true if resultType is compatible with expectedType.
// If expectedType is nil, all types match.
func celTypeMatches(expectedType, resultType *types.Type) bool {
	if expectedType == nil {
		return true
	}
	return expectedType.IsAssignableType(resultType)
}

// celMemberCompletionItems returns completion items for member (receiver-style) functions.
// If receiverType is non-nil, only functions whose first argument is assignable from
// receiverType are included. If nil, all member functions are returned.
func celMemberCompletionItems(celEnv *cel.Env, receiverType *types.Type) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for name, fn := range celEnv.Functions() {
		if celIsOperatorOrInternal(name) {
			continue
		}

		var matchingOverloads []*decls.OverloadDecl
		for _, o := range fn.OverloadDecls() {
			if !o.IsMemberFunction() {
				continue
			}
			if receiverType != nil && len(o.ArgTypes()) > 0 {
				recvType := o.ArgTypes()[0]
				if !recvType.IsAssignableType(receiverType) {
					continue
				}
			}
			matchingOverloads = append(matchingOverloads, o)
		}
		if len(matchingOverloads) == 0 {
			continue
		}

		items = append(items, protocol.CompletionItem{
			Label:            name,
			Kind:             protocol.CompletionItemKindMethod,
			Detail:           celFormatOverloadSignature(fn.Name(), matchingOverloads[0]),
			Documentation:    fn.Description(),
			InsertText:       name + "($1)",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		})
	}
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(a.Label, b.Label)
	})
	return items
}

// celMemberMacroCompletionItems returns completion items for CEL comprehension macros
// (all, exists, filter, map, exists_one) invoked in member-access style on list and
// map receivers (e.g. myList.filter(x, pred)). These macros are not regular functions
// in cel-go's function registry, so celMemberCompletionItems does not include them.
func celMemberMacroCompletionItems() []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(celIteratorMacros))
	for _, name := range celIteratorMacros {
		items = append(items, protocol.CompletionItem{
			Label:            name,
			Kind:             protocol.CompletionItemKindMethod,
			Detail:           "macro",
			InsertText:       name + "($1, $2)",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		})
	}
	return items
}

// celGlobalCompletionItems returns completion items for global (non-receiver) functions
// and type conversions. If expectedType is non-nil, only functions that can return a
// compatible type are included.
func celGlobalCompletionItems(celEnv *cel.Env, expectedType *types.Type) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for name, fn := range celEnv.Functions() {
		if celIsOperatorOrInternal(name) {
			continue
		}

		hasMatchingGlobal := false
		for _, o := range fn.OverloadDecls() {
			if o.IsMemberFunction() {
				continue
			}
			if celTypeMatches(expectedType, o.ResultType()) {
				hasMatchingGlobal = true
				break
			}
		}
		if !hasMatchingGlobal {
			continue
		}

		items = append(items, protocol.CompletionItem{
			Label:            name,
			Kind:             protocol.CompletionItemKindFunction,
			Detail:           celGlobalFunctionDetail(fn),
			Documentation:    fn.Description(),
			InsertText:       name + "($1)",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		})
	}
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(a.Label, b.Label)
	})
	return items
}

// celMacroCompletionItems returns completion items for CEL macros derived from
// cel-go's env.Macros(). Macros are not filtered by expectedType since they do
// not carry return type information in cel-go.
func celMacroCompletionItems(celEnv *cel.Env) []protocol.CompletionItem {
	seen := make(map[string]bool)
	var items []protocol.CompletionItem
	for _, m := range celEnv.Macros() {
		name := m.Function()
		if seen[name] {
			continue
		}
		seen[name] = true

		items = append(items, protocol.CompletionItem{
			Label:            name,
			Kind:             protocol.CompletionItemKindFunction,
			Detail:           "macro",
			InsertText:       name + "($1)",
			InsertTextFormat: protocol.InsertTextFormatSnippet,
		})
	}
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(a.Label, b.Label)
	})
	return items
}

// celKeywordCompletionItems returns completion items for CEL literal keywords and
// protovalidate runtime variables ("this", "now"). If expectedType is non-nil, only
// keywords whose compiled type is compatible with it are included.
// "this" is always included when there is no expected type constraint.
// "now" is included when its timestamp type is compatible with expectedType.
func celKeywordCompletionItems(celEnv *cel.Env, expectedType *types.Type) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, kw := range celKeywords {
		if kw == "this" {
			// "this" is the protovalidate special variable bound at eval time;
			// it cannot be compiled in the static environment. Always include it
			// unless the caller has narrowed to a specific expected type (in that
			// case its type is unknown so we cannot confirm compatibility).
			if expectedType == nil {
				items = append(items, protocol.CompletionItem{
					Label:  kw,
					Kind:   protocol.CompletionItemKindVariable,
					Detail: "protovalidate: current message or field",
				})
			}
			continue
		}
		ast, iss := celEnv.Compile(kw)
		if iss.Err() != nil {
			continue
		}
		kwType := ast.OutputType()
		if !celTypeMatches(expectedType, kwType) {
			continue
		}
		kind := protocol.CompletionItemKindKeyword
		detail := kwType.String()
		if kw == "now" {
			// "now" is a protovalidate runtime variable, not a literal keyword.
			kind = protocol.CompletionItemKindVariable
			detail = "protovalidate: current evaluation timestamp"
		}
		items = append(items, protocol.CompletionItem{
			Label:  kw,
			Kind:   kind,
			Detail: detail,
		})
	}
	return items
}

// celIsOperatorOrInternal returns true if name represents an operator or internal
// function that should not appear as a user-visible completion item.
func celIsOperatorOrInternal(name string) bool {
	if _, ok := operators.FindReverse(name); ok {
		return true
	}
	return strings.HasPrefix(name, "@") || strings.HasPrefix(name, "_")
}

// celGlobalFunctionDetail returns a short detail string for the first global
// (non-member) overload of fn.
func celGlobalFunctionDetail(fn *decls.FunctionDecl) string {
	for _, o := range fn.OverloadDecls() {
		if o.IsMemberFunction() {
			continue
		}
		return celFormatOverloadSignature(fn.Name(), o)
	}
	return ""
}

// celFormatOverloadSignature formats a single overload as "name(args...) -> result".
// For member functions, the format is "receiver.name(args...) -> result".
func celFormatOverloadSignature(name string, o *decls.OverloadDecl) string {
	args := o.ArgTypes()
	var parts []string
	start := 0
	prefix := ""
	if o.IsMemberFunction() && len(args) > 0 {
		prefix = args[0].String() + "."
		start = 1
	}
	for _, a := range args[start:] {
		parts = append(parts, a.String())
	}
	result := o.ResultType().String()
	return fmt.Sprintf("%s%s(%s) -> %s", prefix, name, strings.Join(parts, ", "), result)
}
