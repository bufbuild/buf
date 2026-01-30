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
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bufbuild/buf/private/pkg/diff/diffmyers"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/parser"
	"github.com/bufbuild/protocompile/experimental/printer"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"go.lsp.dev/protocol"
)

// getDeprecateCodeAction generates a code action for deprecating the symbol at the given range.
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

	fqn, title := getDeprecationTarget(ctx, file, params.Range.Start)
	if fqn == "" {
		s.logger.DebugContext(ctx, "deprecate: no deprecation target")
		return nil
	}
	s.logger.DebugContext(ctx, "deprecate: generating edits",
		slog.String("fqn", string(fqn)),
		slog.String("title", title),
	)

	// Generate workspace-wide edits for all types matching the FQN prefix.
	edits := make(map[protocol.DocumentURI][]protocol.TextEdit)
	for _, wsFile := range file.workspace.PathToFile() {
		if wsFile.ir == nil {
			continue
		}
		if fileEdits := generateDeprecationEdits(wsFile, fqn); len(fileEdits) > 0 {
			edits[wsFile.uri] = fileEdits
		}
	}
	if len(edits) == 0 {
		s.logger.DebugContext(ctx, "deprecate: no edits generated")
		return nil
	}
	s.logger.DebugContext(ctx, "deprecate: returning code action", slog.Int("edit_count", len(edits)))
	return &protocol.CodeAction{
		Title: title,
		Kind:  protocol.RefactorRewrite,
		Edit:  &protocol.WorkspaceEdit{Changes: edits},
	}
}

// getDeprecationTarget determines what FQN to deprecate based on the cursor position.
func getDeprecationTarget(ctx context.Context, file *file, position protocol.Position) (ir.FullName, string) {
	symbol := file.SymbolAt(ctx, position)
	if symbol == nil {
		return getPackageDeprecationTarget(file, position)
	}
	if symbol.ir.IsZero() {
		return "", ""
	}
	fqn := symbol.ir.FullName()
	if fqn == "" {
		return "", ""
	}
	var kind string
	switch symbol.ir.Kind() {
	case ir.SymbolKindMessage:
		kind = "message"
	case ir.SymbolKindEnum:
		kind = "enum"
	case ir.SymbolKindService:
		kind = "service"
	case ir.SymbolKindMethod:
		kind = "method"
	case ir.SymbolKindField:
		kind = "field"
	case ir.SymbolKindEnumValue:
		kind = "enum value"
	case ir.SymbolKindOneof:
		kind = "oneof"
	default:
		return "", ""
	}
	return fqn, fmt.Sprintf("Deprecate %s %s", kind, fqn)
}

// getPackageDeprecationTarget checks if the cursor is on a package declaration.
func getPackageDeprecationTarget(file *file, position protocol.Position) (ir.FullName, string) {
	astFile := file.ir.AST()
	if astFile == nil {
		return "", ""
	}
	pkgDecl := astFile.Package()
	if pkgDecl.IsZero() {
		return "", ""
	}
	pkgSpan := pkgDecl.Span()
	offset := positionToOffset(file, position)
	if offsetInSpan(offset, pkgSpan) != 0 {
		return "", ""
	}
	cursorLine := int(position.Line) + 1
	if cursorLine < pkgSpan.StartLoc().Line || cursorLine > pkgSpan.EndLoc().Line {
		return "", ""
	}
	pkg := file.ir.Package()
	return pkg, fmt.Sprintf("Deprecate package %s", pkg)
}

// generateDeprecationEdits generates TextEdits to add deprecation options for all
// types in a file that match the given FQN prefix.
func generateDeprecationEdits(file *file, fqnPrefix ir.FullName) []protocol.TextEdit {
	if file.file == nil {
		return nil
	}
	originalText := file.file.Text()
	if originalText == "" {
		return nil
	}

	// Parse fresh to get a mutable AST.
	errs := &report.Report{}
	astFile, _ := parser.Parse(file.file.Path(), source.NewFile(file.file.Path(), originalText), errs)
	if astFile == nil {
		return nil
	}

	// Collect what needs deprecation from IR analysis.
	toDeprecate := collectDeprecations(file.ir, fqnPrefix)
	if !toDeprecate.file && len(toDeprecate.fqns) == 0 {
		return nil
	}

	// Apply deprecations to the AST.
	if !applyDeprecations(astFile, toDeprecate) {
		return nil
	}

	// Diff to generate text edits.
	modifiedText := printer.PrintFile(astFile, printer.Options{})
	return diffToTextEdits(originalText, modifiedText)
}

// deprecations holds the set of FQNs to deprecate.
type deprecations struct {
	file bool                     // add file-level deprecated option
	fqns map[ir.FullName]struct{} // types, services, methods, and members
}

// collectDeprecations analyzes the IR to determine what needs deprecation.
func collectDeprecations(irFile *ir.File, fqnPrefix ir.FullName) deprecations {
	result := deprecations{fqns: make(map[ir.FullName]struct{})}

	// Check file-level deprecation (for package-level deprecation).
	pkg := irFile.Package()
	if len(pkg) > 0 && fqnHasPrefix(pkg, fqnPrefix) {
		if _, ok := irFile.Deprecated().AsBool(); !ok {
			result.file = true
		}
	}

	// Check messages and enums.
	for typ := range seq.Values(irFile.AllTypes()) {
		fqn := typ.FullName()
		if fqnHasPrefix(fqn, fqnPrefix) {
			if _, ok := typ.Deprecated().AsBool(); !ok {
				result.fqns[fqn] = struct{}{}
			}
		}
	}

	// Check services and methods.
	for svc := range seq.Values(irFile.Services()) {
		fqn := svc.FullName()
		if fqnHasPrefix(fqn, fqnPrefix) {
			if _, ok := svc.Deprecated().AsBool(); !ok {
				result.fqns[fqn] = struct{}{}
			}
		}
		for method := range seq.Values(svc.Methods()) {
			methodFQN := method.FullName()
			if fqnHasPrefix(methodFQN, fqnPrefix) {
				if _, ok := method.Deprecated().AsBool(); !ok {
					result.fqns[methodFQN] = struct{}{}
				}
			}
		}
	}

	// Check fields and enum values (exact match only).
	for member := range irFile.AllMembers() {
		fqn := member.FullName()
		if fqn == fqnPrefix {
			if _, ok := member.Deprecated().AsBool(); !ok {
				result.fqns[fqn] = struct{}{}
			}
		}
	}

	return result
}

// fqnHasPrefix returns true if fqn equals prefix or starts with "prefix.".
func fqnHasPrefix(fqn, prefix ir.FullName) bool {
	return fqn == prefix || strings.HasPrefix(string(fqn), string(prefix)+".")
}

// applyDeprecations modifies the AST to add deprecated options.
func applyDeprecations(astFile *ast.File, toDeprecate deprecations) bool {
	modified := false

	// Handle file-level deprecation.
	var pkgName string
	pkgDecl := astFile.Package()
	if !pkgDecl.IsZero() {
		pkgName = pkgDecl.Path().Canonicalized()
		if toDeprecate.file && !hasDeprecatedOptionDecl(astFile.Decls()) {
			addFileDeprecatedOption(astFile)
			modified = true
		}
	}

	// Walk declarations.
	modified = walkAndDeprecate(astFile, astFile.Decls(), pkgName, toDeprecate.fqns) || modified
	return modified
}

// walkAndDeprecate walks declarations and adds deprecated options where needed.
func walkAndDeprecate(
	astFile *ast.File,
	decls seq.Inserter[ast.DeclAny],
	parentFQN string,
	toDeprecate map[ir.FullName]struct{},
) bool {
	modified := false

	for i := range decls.Len() {
		def := decls.At(i).AsDef()
		if def.IsZero() {
			continue
		}
		name := defName(def)
		if name == "" {
			continue
		}
		fqn := ir.FullName(joinFQN(parentFQN, name))

		switch def.Classify() {
		case ast.DefKindMessage, ast.DefKindEnum:
			if _, ok := toDeprecate[fqn]; ok {
				if !hasDeprecatedOptionDecl(def.Body().Decls()) {
					addBodyDeprecatedOption(astFile, def.Body())
					modified = true
				}
			}
			// Enum values use parent FQN scope, not the enum's FQN.
			if def.Classify() == ast.DefKindEnum && !def.Body().IsZero() {
				modified = deprecateEnumValues(astFile, def.Body(), parentFQN, toDeprecate) || modified
			}
			// Recurse into nested types.
			if !def.Body().IsZero() {
				modified = walkAndDeprecate(astFile, def.Body().Decls(), string(fqn), toDeprecate) || modified
			}

		case ast.DefKindService:
			if _, ok := toDeprecate[fqn]; ok {
				if !hasDeprecatedOptionDecl(def.Body().Decls()) {
					addBodyDeprecatedOption(astFile, def.Body())
					modified = true
				}
			}
			if !def.Body().IsZero() {
				modified = deprecateMethods(astFile, def.Body(), string(fqn), toDeprecate) || modified
			}

		case ast.DefKindField:
			if _, ok := toDeprecate[fqn]; ok {
				if !hasDeprecatedCompactOption(def.Options()) {
					addCompactDeprecatedOption(astFile, def)
					modified = true
				}
			}

		case ast.DefKindOneof:
			if !def.Body().IsZero() {
				modified = walkAndDeprecate(astFile, def.Body().Decls(), string(fqn), toDeprecate) || modified
			}
		}
	}
	return modified
}

// deprecateEnumValues adds deprecated options to enum values.
func deprecateEnumValues(
	astFile *ast.File,
	body ast.DeclBody,
	parentFQN string,
	toDeprecate map[ir.FullName]struct{},
) bool {
	modified := false
	for i := range body.Decls().Len() {
		def := body.Decls().At(i).AsDef()
		if def.IsZero() || def.Classify() != ast.DefKindEnumValue {
			continue
		}
		fqn := ir.FullName(joinFQN(parentFQN, defName(def)))
		if _, ok := toDeprecate[fqn]; ok {
			if !hasDeprecatedCompactOption(def.Options()) {
				addCompactDeprecatedOption(astFile, def)
				modified = true
			}
		}
	}
	return modified
}

// deprecateMethods adds deprecated options to service methods.
func deprecateMethods(
	astFile *ast.File,
	body ast.DeclBody,
	parentFQN string,
	toDeprecate map[ir.FullName]struct{},
) bool {
	modified := false
	for i := range body.Decls().Len() {
		def := body.Decls().At(i).AsDef()
		if def.IsZero() || def.Classify() != ast.DefKindMethod {
			continue
		}
		fqn := ir.FullName(joinFQN(parentFQN, defName(def)))
		if _, ok := toDeprecate[fqn]; ok {
			if def.Body().IsZero() {
				addMethodBodyWithDeprecatedOption(astFile, def)
				modified = true
			} else if !hasDeprecatedOptionDecl(def.Body().Decls()) {
				addBodyDeprecatedOption(astFile, def.Body())
				modified = true
			}
		}
	}
	return modified
}

// defName extracts the name from a definition.
func defName(def ast.DeclDef) string {
	namePath := def.Name()
	if namePath.IsZero() {
		return ""
	}
	ident := namePath.AsIdent()
	if ident.IsZero() {
		return ""
	}
	return ident.Text()
}

// joinFQN joins parent and name with a dot.
func joinFQN(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

// hasDeprecatedOptionDecl checks if declarations contain "option deprecated = true;".
func hasDeprecatedOptionDecl(decls seq.Inserter[ast.DeclAny]) bool {
	for i := range decls.Len() {
		def := decls.At(i).AsDef()
		if def.IsZero() {
			continue
		}
		typePath := def.Type().AsPath()
		if typePath.IsZero() || !typePath.Path.IsIdents(keyword.Option.String()) {
			continue
		}
		if def.Name().IsIdents("deprecated") {
			return true
		}
	}
	return false
}

// hasDeprecatedCompactOption checks if compact options contain deprecated.
func hasDeprecatedCompactOption(opts ast.CompactOptions) bool {
	if opts.IsZero() {
		return false
	}
	for i := range opts.Entries().Len() {
		if opts.Entries().At(i).Path.IsIdents("deprecated") {
			return true
		}
	}
	return false
}

// addFileDeprecatedOption adds "option deprecated = true;" after package/imports.
func addFileDeprecatedOption(astFile *ast.File) {
	optionDecl := newDeprecatedOptionDecl(astFile.Stream(), astFile.Nodes())

	// Find position after syntax, package, and imports.
	insertPos := 0
	for i := range astFile.Decls().Len() {
		decl := astFile.Decls().At(i)
		if !decl.AsSyntax().IsZero() || !decl.AsPackage().IsZero() || !decl.AsImport().IsZero() {
			insertPos = i + 1
		} else {
			break
		}
	}
	astFile.Decls().Insert(insertPos, optionDecl.AsAny())
}

// addBodyDeprecatedOption adds "option deprecated = true;" to a body.
func addBodyDeprecatedOption(astFile *ast.File, body ast.DeclBody) {
	optionDecl := newDeprecatedOptionDecl(astFile.Stream(), astFile.Nodes())

	// Insert after any existing options.
	insertPos := 0
	for i := range body.Decls().Len() {
		def := body.Decls().At(i).AsDef()
		if !def.IsZero() && def.Classify() == ast.DefKindOption {
			insertPos = i + 1
		} else {
			break
		}
	}
	body.Decls().Insert(insertPos, optionDecl.AsAny())
}

// addMethodBodyWithDeprecatedOption adds a body with deprecated option to a method.
func addMethodBodyWithDeprecatedOption(astFile *ast.File, def ast.DeclDef) {
	stream := astFile.Stream()
	nodes := astFile.Nodes()

	openBrace := stream.NewPunct(keyword.LBrace.String())
	closeBrace := stream.NewPunct(keyword.RBrace.String())
	stream.NewFused(openBrace, closeBrace)

	body := nodes.NewDeclBody(openBrace)
	seq.Append(body.Decls(), newDeprecatedOptionDecl(stream, nodes).AsAny())
	def.SetBody(body)
}

// newDeprecatedOptionDecl creates "option deprecated = true;".
func newDeprecatedOptionDecl(stream *token.Stream, nodes *ast.Nodes) ast.DeclDef {
	return nodes.NewDeclDef(ast.DeclDefArgs{
		Type:      ast.TypePath{Path: nodes.NewPath(nodes.NewPathComponent(token.Zero, stream.NewIdent(keyword.Option.String())))}.AsAny(),
		Name:      nodes.NewPath(nodes.NewPathComponent(token.Zero, stream.NewIdent("deprecated"))),
		Equals:    stream.NewPunct(keyword.Assign.String()),
		Value:     ast.ExprPath{Path: nodes.NewPath(nodes.NewPathComponent(token.Zero, stream.NewIdent(keyword.True.String())))}.AsAny(),
		Semicolon: stream.NewPunct(keyword.Semi.String()),
	})
}

// addCompactDeprecatedOption adds "[deprecated = true]" to a field or enum value.
func addCompactDeprecatedOption(astFile *ast.File, def ast.DeclDef) {
	stream := astFile.Stream()
	nodes := astFile.Nodes()

	opt := ast.Option{
		Path:   nodes.NewPath(nodes.NewPathComponent(token.Zero, stream.NewIdent("deprecated"))),
		Equals: stream.NewPunct(keyword.Assign.String()),
		Value:  ast.ExprPath{Path: nodes.NewPath(nodes.NewPathComponent(token.Zero, stream.NewIdent(keyword.True.String())))}.AsAny(),
	}

	opts := def.Options()
	if opts.IsZero() {
		openBracket := stream.NewPunct(keyword.LBracket.String())
		closeBracket := stream.NewPunct(keyword.RBracket.String())
		stream.NewFused(openBracket, closeBracket)
		opts = nodes.NewCompactOptions(openBracket)
		def.SetOptions(opts)
	}

	// Add comma after existing entries if needed.
	entries := opts.Entries()
	if entries.Len() > 0 {
		lastIdx := entries.Len() - 1
		lastEntry := entries.At(lastIdx)
		entries.Delete(lastIdx)
		entries.InsertComma(lastIdx, lastEntry, stream.NewPunct(keyword.Comma.String()))
	}
	seq.Append(entries, opt)
}

// diffToTextEdits converts a diff between original and modified text to LSP text edits.
func diffToTextEdits(original, modified string) []protocol.TextEdit {
	if original == modified {
		return nil
	}

	// Convert to [][]byte for diffmyers.
	var fromBytes [][]byte
	for line := range strings.Lines(original) {
		fromBytes = append(fromBytes, []byte(line))
	}
	var toBytes [][]byte
	for line := range strings.Lines(modified) {
		toBytes = append(toBytes, []byte(line))
	}

	edits := diffmyers.Diff(fromBytes, toBytes)
	if len(edits) == 0 {
		return nil
	}

	// Convert to LSP text edits.
	var lspEdits []protocol.TextEdit
	i := 0
	for i < len(edits) {
		fromPos := edits[i].FromPosition
		var insertText bytes.Buffer
		deleteCount := 0

		// Collect inserts at this position.
		for i < len(edits) && edits[i].Kind == diffmyers.EditKindInsert && edits[i].FromPosition == fromPos {
			insertText.Write(toBytes[edits[i].ToPosition])
			i++
		}
		// Collect deletes at this position.
		deleteStart := fromPos
		for i < len(edits) && edits[i].Kind == diffmyers.EditKindDelete && edits[i].FromPosition == deleteStart+deleteCount {
			deleteCount++
			i++
		}
		// Collect inserts after deletes.
		for i < len(edits) && edits[i].Kind == diffmyers.EditKindInsert && edits[i].FromPosition == deleteStart+deleteCount {
			insertText.Write(toBytes[edits[i].ToPosition])
			i++
		}

		if deleteCount > 0 || insertText.Len() > 0 {
			lspEdits = append(lspEdits, protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(fromPos), Character: 0},
					End:   protocol.Position{Line: uint32(fromPos + deleteCount), Character: 0},
				},
				NewText: insertText.String(),
			})
		}
	}
	return lspEdits
}
