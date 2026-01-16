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

// This file implements the deprecation code action.

package buflsp

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token"
	"go.lsp.dev/protocol"
)

// CodeActionKindSourceDeprecate is the custom code action kind for deprecation.
const CodeActionKindSourceDeprecate protocol.CodeActionKind = "source.deprecate"

// getDeprecateCodeAction generates a code action for deprecating the symbol at the given range.
// It returns nil if no deprecation action is available for the position.
func (s *server) getDeprecateCodeAction(
	ctx context.Context,
	file *file,
	params *protocol.CodeActionParams,
) *protocol.CodeAction {
	if file.workspace == nil || file.ir == nil {
		return nil
	}

	// Get the symbol at the cursor position
	symbol := file.SymbolAt(ctx, params.Range.Start)
	if symbol == nil {
		return nil
	}

	// Determine the FQN prefix to deprecate based on symbol type
	fqnPrefix, title := getDeprecationTarget(symbol)
	if fqnPrefix == "" {
		return nil
	}

	// Generate workspace-wide edits for all types matching the FQN prefix
	checker := newDeprecationChecker([]string{string(fqnPrefix)})
	edits := make(map[protocol.DocumentURI][]protocol.TextEdit)

	for _, wsFile := range file.workspace.PathToFile() {
		if wsFile.ir == nil {
			continue
		}
		fileEdits := generateDeprecationEdits(wsFile, checker)
		if len(fileEdits) > 0 {
			edits[wsFile.uri] = fileEdits
		}
	}

	if len(edits) == 0 {
		return nil
	}

	return &protocol.CodeAction{
		Title: title,
		Kind:  CodeActionKindSourceDeprecate,
		Edit:  &protocol.WorkspaceEdit{Changes: edits},
	}
}

// getDeprecationTarget determines what FQN prefix to deprecate based on the symbol.
// Returns the FQN prefix and a human-readable title for the code action.
func getDeprecationTarget(symbol *symbol) (ir.FullName, string) {
	if symbol.ir.IsZero() {
		return "", ""
	}

	fqn := symbol.ir.FullName()
	if fqn == "" {
		return "", ""
	}

	switch symbol.ir.Kind() {
	case ir.SymbolKindMessage:
		return fqn, fmt.Sprintf("Deprecate message %s", fqn)
	case ir.SymbolKindEnum:
		return fqn, fmt.Sprintf("Deprecate enum %s", fqn)
	case ir.SymbolKindService:
		return fqn, fmt.Sprintf("Deprecate service %s", fqn)
	case ir.SymbolKindMethod:
		return fqn, fmt.Sprintf("Deprecate method %s", fqn)
	case ir.SymbolKindField:
		// Fields require exact match, so return the full FQN
		return fqn, fmt.Sprintf("Deprecate field %s", fqn)
	case ir.SymbolKindEnumValue:
		// Enum values require exact match
		return fqn, fmt.Sprintf("Deprecate enum value %s", fqn)
	case ir.SymbolKindOneof:
		// Oneofs can be deprecated like messages
		return fqn, fmt.Sprintf("Deprecate oneof %s", fqn)
	default:
		return "", ""
	}
}

// generateDeprecationEdits generates TextEdits to add deprecation options for all
// types in a file that match the deprecation checker's FQN prefixes.
func generateDeprecationEdits(file *file, checker *deprecationChecker) []protocol.TextEdit {
	var edits []protocol.TextEdit

	// Check file-level deprecation (for package-level deprecation)
	pkgFQN := getPackageFQN(file)
	if len(pkgFQN) > 0 && checker.shouldDeprecate(pkgFQN) {
		if edit := insertFileDeprecationOption(file); edit != nil {
			edits = append(edits, *edit)
		}
	}

	// Check messages and enums
	for typ := range seq.Values(file.ir.AllTypes()) {
		fqn := splitFQN(typ.FullName())
		if checker.shouldDeprecate(fqn) {
			if edit := insertTypeDeprecationOption(file, typ); edit != nil {
				edits = append(edits, *edit)
			}
		}
	}

	// Check services
	for svc := range seq.Values(file.ir.Services()) {
		fqn := splitFQN(svc.FullName())
		if checker.shouldDeprecate(fqn) {
			if edit := insertServiceDeprecationOption(file, svc); edit != nil {
				edits = append(edits, *edit)
			}
		}
		// Check methods
		for method := range seq.Values(svc.Methods()) {
			methodFQN := splitFQN(method.FullName())
			if checker.shouldDeprecate(methodFQN) {
				if edit := insertMethodDeprecationOption(file, method); edit != nil {
					edits = append(edits, *edit)
				}
			}
		}
	}

	// Check fields and enum values (exact match only)
	for member := range file.ir.AllMembers() {
		fqn := splitFQN(member.FullName())
		if checker.shouldDeprecateExact(fqn) {
			if edit := insertMemberDeprecationOption(file, member); edit != nil {
				edits = append(edits, *edit)
			}
		}
	}

	return edits
}

// getPackageFQN extracts the package FQN components from a file.
func getPackageFQN(file *file) []string {
	if file.ir == nil {
		return nil
	}
	pkg := file.ir.Package()
	if pkg == "" {
		return nil
	}
	return strings.Split(string(pkg), ".")
}

// splitFQN splits a fully-qualified name into components.
func splitFQN(fqn ir.FullName) []string {
	if fqn == "" {
		return nil
	}
	return strings.Split(string(fqn), ".")
}

// insertFileDeprecationOption generates a TextEdit to insert "option deprecated = true;"
// at the file level (after the package declaration).
func insertFileDeprecationOption(file *file) *protocol.TextEdit {
	// Check if already deprecated
	if _, ok := file.ir.Deprecated().AsBool(); ok {
		return nil
	}

	astFile := file.ir.AST()
	if astFile == nil {
		return nil
	}

	// Find the insertion point after package declaration
	var insertLine int
	pkgDecl := astFile.Package()
	if !pkgDecl.IsZero() {
		insertLine = pkgDecl.Span().EndLoc().Line
	} else {
		syntaxDecl := astFile.Syntax()
		if !syntaxDecl.IsZero() {
			insertLine = syntaxDecl.Span().EndLoc().Line
		} else {
			insertLine = 1
		}
	}

	// Create the edit to insert the option
	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(insertLine), Character: 0},
			End:   protocol.Position{Line: uint32(insertLine), Character: 0},
		},
		NewText: "\noption deprecated = true;\n",
	}
}

// insertTypeDeprecationOption generates a TextEdit to insert "option deprecated = true;"
// inside a message or enum definition.
func insertTypeDeprecationOption(file *file, typ ir.Type) *protocol.TextEdit {
	// Check if already deprecated
	if _, ok := typ.Deprecated().AsBool(); ok {
		return nil
	}

	declDef := typ.AST()
	if declDef.IsZero() {
		return nil
	}

	return insertOptionAfterOpenBrace(file, declDef.Body())
}

// insertServiceDeprecationOption generates a TextEdit to insert "option deprecated = true;"
// inside a service definition.
func insertServiceDeprecationOption(file *file, svc ir.Service) *protocol.TextEdit {
	// Check if already deprecated
	if _, ok := svc.Deprecated().AsBool(); ok {
		return nil
	}

	declDef := svc.AST()
	if declDef.IsZero() {
		return nil
	}

	return insertOptionAfterOpenBrace(file, declDef.Body())
}

// insertMethodDeprecationOption generates a TextEdit to insert "option deprecated = true;"
// inside an RPC method definition.
func insertMethodDeprecationOption(file *file, method ir.Method) *protocol.TextEdit {
	// Check if already deprecated
	if _, ok := method.Deprecated().AsBool(); ok {
		return nil
	}

	declDef := method.AST()
	if declDef.IsZero() {
		return nil
	}

	body := declDef.Body()
	if body.IsZero() {
		// Method has no body (ends with semicolon), need to add braces
		return insertMethodBodyWithDeprecation(file, declDef)
	}

	return insertOptionAfterOpenBrace(file, body)
}

// insertMethodBodyWithDeprecation handles methods that don't have a body yet.
// Converts "rpc Foo(Req) returns (Resp);" to "rpc Foo(Req) returns (Resp) { option deprecated = true; }"
func insertMethodBodyWithDeprecation(file *file, declDef ast.DeclDef) *protocol.TextEdit {
	span := declDef.Span()

	// Find the semicolon at the end
	stream := declDef.Context().Stream()
	tok, _ := stream.Around(span.End)
	cursor := token.NewCursorAt(tok)

	// Walk backwards to find the semicolon
	for !tok.IsZero() && tok.Kind() != token.Keyword {
		tok = cursor.PrevSkippable()
	}

	if tok.IsZero() {
		return nil
	}

	// Replace the semicolon with the body
	semiSpan := tok.Span()
	return &protocol.TextEdit{
		Range:   reportSpanToProtocolRange(semiSpan),
		NewText: " {\n  option deprecated = true;\n}",
	}
}

// insertOptionAfterOpenBrace generates a TextEdit to insert "option deprecated = true;"
// after the opening brace of a body.
func insertOptionAfterOpenBrace(file *file, body ast.DeclBody) *protocol.TextEdit {
	if body.IsZero() {
		return nil
	}

	braces := body.Braces()
	if braces.IsZero() {
		return nil
	}

	// Get the position right after the opening brace
	braceSpan := braces.Span()
	insertPos := source.Location{
		Line:   braceSpan.StartLoc().Line,
		Column: braceSpan.StartLoc().Column + 1, // After the '{'
	}

	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(insertPos.Line - 1),
				Character: uint32(insertPos.Column - 1),
			},
			End: protocol.Position{
				Line:      uint32(insertPos.Line - 1),
				Character: uint32(insertPos.Column - 1),
			},
		},
		NewText: "\n  option deprecated = true;",
	}
}

// insertMemberDeprecationOption generates a TextEdit to add "[deprecated = true]"
// to a field or enum value.
func insertMemberDeprecationOption(file *file, member ir.Member) *protocol.TextEdit {
	// Check if already deprecated
	if _, ok := member.Deprecated().AsBool(); ok {
		return nil
	}

	declDef := member.AST()
	if declDef.IsZero() {
		return nil
	}

	// For fields and enum values, we add compact options before the semicolon
	// or append to existing compact options
	span := declDef.Span()

	// Find the semicolon
	stream := declDef.Context().Stream()
	tok, _ := stream.Around(span.End)
	cursor := token.NewCursorAt(tok)

	// Walk backwards to find the semicolon
	for !tok.IsZero() && tok.Kind() != token.Keyword {
		tok = cursor.PrevSkippable()
	}

	if tok.IsZero() {
		return nil
	}

	// Check if there are existing compact options by looking for ']' before semicolon
	prevTok := cursor.PrevSkippable()
	for !prevTok.IsZero() && prevTok.Kind() == token.Space {
		prevTok = cursor.PrevSkippable()
	}

	if !prevTok.IsZero() && prevTok.Span().Text() == "]" {
		// Existing compact options, insert before the closing bracket
		return &protocol.TextEdit{
			Range:   reportSpanToProtocolRange(prevTok.Span()),
			NewText: ", deprecated = true]",
		}
	}

	// No existing compact options, insert before the semicolon
	semiSpan := tok.Span()
	return &protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(semiSpan.StartLoc().Line - 1),
				Character: uint32(semiSpan.StartLoc().Column - 1),
			},
			End: protocol.Position{
				Line:      uint32(semiSpan.StartLoc().Line - 1),
				Character: uint32(semiSpan.StartLoc().Column - 1),
			},
		},
		NewText: " [deprecated = true]",
	}
}

// deprecationChecker determines which types should have deprecated options added.
type deprecationChecker struct {
	prefixes [][]string // FQN prefixes split into components
}

// newDeprecationChecker creates a new deprecationChecker for the given FQN prefixes.
func newDeprecationChecker(fqnPrefixes []string) *deprecationChecker {
	prefixes := make([][]string, 0, len(fqnPrefixes))
	for _, prefix := range fqnPrefixes {
		if prefix != "" {
			prefixes = append(prefixes, strings.Split(prefix, "."))
		}
	}
	return &deprecationChecker{prefixes: prefixes}
}

// shouldDeprecate returns true if the given FQN should be deprecated using prefix matching.
func (d *deprecationChecker) shouldDeprecate(fqn []string) bool {
	for _, prefix := range d.prefixes {
		if fqnMatchesPrefix(fqn, prefix) {
			return true
		}
	}
	return false
}

// shouldDeprecateExact returns true if the given FQN matches exactly.
// This is used for fields and enum values which are only deprecated on exact match.
func (d *deprecationChecker) shouldDeprecateExact(fqn []string) bool {
	for _, prefix := range d.prefixes {
		if fqnMatchesExact(fqn, prefix) {
			return true
		}
	}
	return false
}

// fqnMatchesPrefix returns true if fqn starts with prefix using component-based matching.
func fqnMatchesPrefix(fqn, prefix []string) bool {
	if len(prefix) > len(fqn) {
		return false
	}
	for i, p := range prefix {
		if fqn[i] != p {
			return false
		}
	}
	return true
}

// fqnMatchesExact returns true if fqn exactly equals prefix.
func fqnMatchesExact(fqn, prefix []string) bool {
	if len(fqn) != len(prefix) {
		return false
	}
	for i, p := range prefix {
		if fqn[i] != p {
			return false
		}
	}
	return true
}
