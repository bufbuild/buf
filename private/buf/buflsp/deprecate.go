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

// This file implements the deprecation code action using direct text insertion.

package buflsp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
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
	s.logger.DebugContext(ctx, "deprecate: checking code action",
		slog.String("uri", string(params.TextDocument.URI)),
		slog.Uint64("line", uint64(params.Range.Start.Line)),
		slog.Uint64("char", uint64(params.Range.Start.Character)),
	)
	if file.workspace == nil || file.ir == nil {
		s.logger.DebugContext(ctx, "deprecate: no workspace or IR")
		return nil
	}

	fqnPrefix, title := getDeprecationTarget(ctx, file, params.Range.Start)
	if fqnPrefix == "" {
		s.logger.DebugContext(ctx, "deprecate: no deprecation target")
		return nil
	}
	s.logger.DebugContext(
		ctx, "deprecate: generating edits",
		slog.String("fqn_prefix", string(fqnPrefix)),
		slog.String("title", title),
	)

	// Generate workspace-wide edits for all types matching the FQN prefix.
	checker := newFullNameMatcher(fqnPrefix)
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
		s.logger.DebugContext(ctx, "deprecate: no edits generated")
		return nil
	}
	s.logger.DebugContext(
		ctx, "deprecate: returning code action",
		slog.Int("edit_count", len(edits)),
	)
	return &protocol.CodeAction{
		Title: title,
		Kind:  CodeActionKindSourceDeprecate,
		Edit:  &protocol.WorkspaceEdit{Changes: edits},
	}
}

// getDeprecationTarget determines what FQN prefix to deprecate based on the position.
// Returns the FQN prefix and a human-readable title for the code action.
func getDeprecationTarget(ctx context.Context, file *file, position protocol.Position) (ir.FullName, string) {
	// Get the symbol at the cursor position.
	symbol := file.SymbolAt(ctx, position)
	if symbol == nil {
		astFile := file.ir.AST()
		if astFile == nil {
			return "", ""
		}
		pkgDecl := astFile.Package()
		if pkgDecl.IsZero() {
			return "", ""
		}
		// Check if cursor is within the package declaration span.
		pkgSpan := pkgDecl.Span()
		offset := positionToOffset(file, position)
		if offsetInSpan(offset, pkgSpan) != 0 {
			return "", ""
		}
		cursorLine := int(position.Line) + 1 // Convert 0-indexed to 1-indexed
		if cursorLine < pkgSpan.StartLoc().Line || cursorLine > pkgSpan.EndLoc().Line {
			return "", ""
		}
		pkg := file.ir.Package()
		return pkg, fmt.Sprintf("Deprecate package %s", pkg)
	}
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
		return fqn, fmt.Sprintf("Deprecate field %s", fqn)
	case ir.SymbolKindEnumValue:
		return fqn, fmt.Sprintf("Deprecate enum value %s", fqn)
	case ir.SymbolKindOneof:
		return fqn, fmt.Sprintf("Deprecate oneof %s", fqn)
	default:
		return "", ""
	}
}

// generateDeprecationEdits generates TextEdits to add deprecation options for all
// types in a file that match the deprecation checker's FQN prefixes.
func generateDeprecationEdits(file *file, checker *fullNameMatcher) []protocol.TextEdit {
	var edits []protocol.TextEdit
	// Check file-level deprecation (for package-level deprecation)
	pkgFQN := getPackageFQN(file)
	if len(pkgFQN) > 0 && checker.matchesPrefix(pkgFQN) {
		if edit := deprecateFile(file); edit != nil {
			edits = append(edits, *edit)
		}
	}
	// Check messages and enums
	for typ := range seq.Values(file.ir.AllTypes()) {
		fqn := typ.FullName()
		if checker.matchesPrefix(fqn) {
			if edit := deprecateType(file, typ); edit != nil {
				edits = append(edits, *edit)
			}
		}
	}
	// Check services
	for svc := range seq.Values(file.ir.Services()) {
		fqn := svc.FullName()
		if checker.matchesPrefix(fqn) {
			if edit := deprecateService(file, svc); edit != nil {
				edits = append(edits, *edit)
			}
		}
		// Check methods
		for method := range seq.Values(svc.Methods()) {
			methodFQN := method.FullName()
			if checker.matchesPrefix(methodFQN) {
				if edit := deprecateMethod(file, method); edit != nil {
					edits = append(edits, *edit)
				}
			}
		}
	}
	// Check fields and enum values (exact match only)
	for member := range file.ir.AllMembers() {
		fqn := member.FullName()
		if checker.matchesExact(fqn) {
			if edit := deprecateMember(file, member); edit != nil {
				edits = append(edits, *edit)
			}
		}
	}
	return edits
}

// getPackageFQN extracts the package FQN components from a file.
func getPackageFQN(file *file) ir.FullName {
	if file.ir == nil {
		return ""
	}
	return file.ir.Package()
}

// getSpanIndentation returns the leading whitespace of the first line of a span.
func getSpanIndentation(span source.Span) string {
	loc := span.StartLoc()
	if loc.Line < 1 {
		return ""
	}
	text := span.File.Text()
	lines := strings.Split(text, "\n")
	if loc.Line > len(lines) {
		return ""
	}
	line := lines[loc.Line-1]
	var indent strings.Builder
	for _, ch := range line {
		if ch == ' ' || ch == '\t' {
			indent.WriteRune(ch)
		} else {
			break
		}
	}
	return indent.String()
}

// deprecateFile generates a TextEdit to insert "option deprecated = true;"
// at the file level (after the package declaration).
func deprecateFile(file *file) *protocol.TextEdit {
	// Check semantic value first
	if deprecated, ok := file.ir.Deprecated().AsBool(); ok && deprecated {
		return nil
	}
	astFile := file.ir.AST()
	if astFile == nil {
		return nil
	}

	// Check AST for existing file-level deprecated option
	if hasDeprecatedOption(astFile.Decls()) {
		return nil
	}

	// Find the insertion point after package or syntax declaration
	var insertSpan source.Span
	pkgDecl := astFile.Package()
	if !pkgDecl.IsZero() {
		insertSpan = pkgDecl.Span()
	} else {
		syntaxDecl := astFile.Syntax()
		if !syntaxDecl.IsZero() {
			insertSpan = syntaxDecl.Span()
		} else {
			// No package or syntax - insert at start of file
			return &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				NewText: "option deprecated = true;\n",
			}
		}
	}
	// Insert after the end of the declaration (with blank line for readability)
	insertLocation := insertSpan.File.Location(insertSpan.End, positionalEncoding)
	return &protocol.TextEdit{
		Range:   reportLocationsToProtocolRange(insertLocation, insertLocation),
		NewText: "\n\noption deprecated = true;",
	}
}

// deprecateType generates a TextEdit to add "option deprecated = true;"
// inside a message or enum definition.
func deprecateType(file *file, typ ir.Type) *protocol.TextEdit {
	// Check semantic value first (handles deprecated = true)
	if deprecated, ok := typ.Deprecated().AsBool(); ok && deprecated {
		return nil
	}
	declDef := typ.AST()
	if declDef.IsZero() {
		return nil
	}
	// Also check AST for any existing deprecated option (handles deprecated = false)
	body := declDef.Body()
	if !body.IsZero() && hasDeprecatedOption(body.Decls()) {
		return nil
	}
	return deprecateDeclWithBody(file, declDef)
}

// deprecateService generates a TextEdit to add "option deprecated = true;"
// inside a service definition.
func deprecateService(file *file, svc ir.Service) *protocol.TextEdit {
	if deprecated, ok := svc.Deprecated().AsBool(); ok && deprecated {
		return nil
	}

	declDef := svc.AST()
	if declDef.IsZero() {
		return nil
	}

	body := declDef.Body()
	if !body.IsZero() && hasDeprecatedOption(body.Decls()) {
		return nil
	}

	return deprecateDeclWithBody(file, declDef)
}

// deprecateMethod generates a TextEdit to add "option deprecated = true;"
// inside an RPC method definition.
func deprecateMethod(file *file, method ir.Method) *protocol.TextEdit {
	if deprecated, ok := method.Deprecated().AsBool(); ok && deprecated {
		return nil
	}

	declDef := method.AST()
	if declDef.IsZero() {
		return nil
	}

	body := declDef.Body()
	if body.IsZero() {
		// Method has no body (ends with semicolon), need to add braces and option
		return deprecateMethodWithoutBody(file, declDef)
	}

	if hasDeprecatedOption(body.Decls()) {
		return nil
	}

	return deprecateDeclWithBody(file, declDef)
}

// hasDeprecatedOption checks if declarations contain an "option deprecated" declaration.
func hasDeprecatedOption(decls seq.Inserter[ast.DeclAny]) bool {
	for decl := range seq.Values(decls) {
		def := decl.AsDef()
		if def.IsZero() {
			continue
		}
		// Check if this is an "option" declaration with name "deprecated"
		typePath := def.Type().AsPath()
		if typePath.IsZero() {
			continue
		}
		if !typePath.Path.IsIdents("option") {
			continue
		}
		// Check if the option name is "deprecated"
		namePath := def.Name()
		if namePath.IsZero() {
			continue
		}
		if namePath.IsIdents("deprecated") {
			return true
		}
	}
	return false
}

// deprecateDeclWithBody generates a TextEdit for a declaration that has a body.
// It inserts "option deprecated = true;" right after the opening brace.
func deprecateDeclWithBody(file *file, declDef ast.DeclDef) *protocol.TextEdit {
	body := declDef.Body()
	if body.IsZero() {
		return nil
	}

	// Get the opening brace position (Braces() returns a fused token pair)
	braceToken := body.Braces()
	if braceToken.IsZero() {
		return nil
	}

	// Get the opening and closing braces
	openBrace, closeBrace := braceToken.StartEnd()
	if openBrace.IsZero() {
		return nil
	}

	// Calculate indent: declaration indent + one level (2 spaces)
	declIndent := getSpanIndentation(declDef.Span())
	bodyIndent := declIndent + "  "
	openBraceSpan := openBrace.LeafSpan()

	// Check if braces are on the same line (empty or single-line body like `{}`)
	// If so, we need to add a trailing newline + indent for the closing brace
	var newText string
	if !closeBrace.IsZero() {
		closeBraceSpan := closeBrace.LeafSpan()
		if closeBraceSpan.StartLoc().Line == openBraceSpan.EndLoc().Line {
			// Same line: insert option with newlines to properly format
			newText = "\n" + bodyIndent + "option deprecated = true;\n" + declIndent
		} else {
			// Different lines: just insert the option
			newText = "\n" + bodyIndent + "option deprecated = true;"
		}
	} else {
		newText = "\n" + bodyIndent + "option deprecated = true;"
	}

	insertLocation := openBraceSpan.File.Location(openBraceSpan.End, positionalEncoding)
	return &protocol.TextEdit{
		Range:   reportLocationsToProtocolRange(insertLocation, insertLocation),
		NewText: newText,
	}
}

// deprecateMethodWithoutBody handles methods that don't have a body yet.
// It replaces the trailing semicolon with a body containing the deprecation option.
func deprecateMethodWithoutBody(file *file, declDef ast.DeclDef) *protocol.TextEdit {
	// Find the semicolon at the end of the declaration
	semi := declDef.Semicolon()
	if semi.IsZero() {
		return nil
	}

	// Calculate indent
	declIndent := getSpanIndentation(declDef.Span())
	bodyIndent := declIndent + "  "

	// Replace the semicolon with a body block
	semiSpan := semi.Span()
	return &protocol.TextEdit{
		Range:   reportSpanToProtocolRange(semiSpan),
		NewText: " {\n" + bodyIndent + "option deprecated = true;\n" + declIndent + "}",
	}
}

// deprecateMember generates a TextEdit to add "[deprecated = true]"
// to a field or enum value using compact options.
func deprecateMember(file *file, member ir.Member) *protocol.TextEdit {
	if _, ok := member.Deprecated().AsBool(); ok {
		return nil
	}
	declDef := member.AST()
	if declDef.IsZero() {
		return nil
	}

	// Check if there are existing compact options
	opts := declDef.Options()
	if !opts.IsZero() {
		openBracket, closeBracket := opts.Brackets().StartEnd()
		openBracketSpan := openBracket.LeafSpan()
		closeBracketSpan := closeBracket.LeafSpan()

		// Check if options span multiple lines
		isMultiLine := openBracketSpan.StartLoc().Line != closeBracketSpan.StartLoc().Line

		if isMultiLine {
			// Multi-line options: insert after the last entry with proper formatting
			entries := opts.Entries()
			if entries.Len() > 0 {
				lastIdx := entries.Len() - 1
				lastEntry := entries.At(lastIdx)
				lastEntrySpan := lastEntry.Span()

				// Get indentation from the last entry
				indent := getSpanIndentation(lastEntrySpan)

				// Check if there's already a comma after the last entry
				existingComma := entries.Comma(lastIdx)
				var newText string
				var insertSpan source.Span

				if existingComma.IsZero() {
					// No comma - insert after the last entry value
					insertSpan = lastEntrySpan
					newText = ",\n" + indent + "deprecated = true"
				} else {
					// Has comma - insert after the comma
					insertSpan = existingComma.LeafSpan()
					newText = "\n" + indent + "deprecated = true"
				}

				insertLocation := insertSpan.File.Location(insertSpan.End, positionalEncoding)
				return &protocol.TextEdit{
					Range:   reportLocationsToProtocolRange(insertLocation, insertLocation),
					NewText: newText,
				}
			}
		}

		// Single-line options: insert ", deprecated = true" before the closing bracket
		insertLocation := closeBracketSpan.File.Location(closeBracketSpan.Start, positionalEncoding)
		return &protocol.TextEdit{
			Range:   reportLocationsToProtocolRange(insertLocation, insertLocation),
			NewText: ", deprecated = true",
		}
	}

	// No existing options - insert "[deprecated = true]" before the semicolon
	semi := declDef.Semicolon()
	if semi.IsZero() {
		return nil
	}
	semiSpan := semi.Span()
	insertLocation := semiSpan.File.Location(semiSpan.Start, positionalEncoding)
	return &protocol.TextEdit{
		Range:   reportLocationsToProtocolRange(insertLocation, insertLocation),
		NewText: " [deprecated = true]",
	}
}

// fullNameMatcher determines which types should have deprecated options added.
type fullNameMatcher struct {
	prefixes []ir.FullName
}

// newFullNameMatcher creates a new matcher for the given FQN prefixes.
func newFullNameMatcher(fqnPrefixes ...ir.FullName) *fullNameMatcher {
	return &fullNameMatcher{prefixes: fqnPrefixes}
}

// matchesPrefix returns true if the given FQN matches using prefix matching.
func (d *fullNameMatcher) matchesPrefix(fqn ir.FullName) bool {
	for _, prefix := range d.prefixes {
		if fqnMatchesPrefix(fqn, prefix) {
			return true
		}
	}
	return false
}

// matchesExact returns true if the given FQN matches exactly.
func (d *fullNameMatcher) matchesExact(fqn ir.FullName) bool {
	for _, prefix := range d.prefixes {
		if fqn == prefix {
			return true
		}
	}
	return false
}

// fqnMatchesPrefix returns true if fqn starts with prefix using component-based matching.
func fqnMatchesPrefix(fqn, prefix ir.FullName) bool {
	if len(prefix) > len(fqn) {
		return false
	}
	if len(prefix) == len(fqn) {
		return fqn == prefix
	}
	return strings.HasPrefix(string(fqn), string(prefix)+".")
}
