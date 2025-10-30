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

// This file implements code completion support for the LSP.

package buflsp

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"math"
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ast/syntax"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"go.lsp.dev/protocol"
)

// getCompletionItems returns completion items for the given position in the file.
//
// This function is called by the Completion handler in server.go.
func getCompletionItems(
	ctx context.Context,
	file *file,
	position protocol.Position,
) []protocol.CompletionItem {
	if file.ir.AST().IsZero() {
		file.lsp.logger.DebugContext(
			ctx,
			"no AST found for completion",
			slog.String("file", file.uri.Filename()),
		)
		return nil
	}

	declPath := getDeclForPosition(file.ir.AST().DeclBody, position)
	if len(declPath) > 0 {
		decl := declPath[len(declPath)-1]
		file.lsp.logger.DebugContext(
			ctx,
			"completion: found declaration",
			slog.String("decl_kind", decl.Kind().String()),
			slog.Int("path_depth", len(declPath)),
		)
	} else {
		file.lsp.logger.DebugContext(ctx, "completion: no declaration found at position")
		return nil
	}

	// Return context-aware completions based on the declaration type.
	return completionItemsForDeclPath(ctx, file, declPath, position)
}

// completionItemsForDeclPath returns completion items based on the AST declaration path at the cursor.
// The declPath is a slice from parent to smallest declaration, where [0] is the top-level and [len-1] is the innermost.
func completionItemsForDeclPath(ctx context.Context, file *file, declPath []ast.DeclAny, position protocol.Position) []protocol.CompletionItem {
	if len(declPath) == 0 {
		return nil
	}

	// Get the innermost (smallest) declaration.
	decl := declPath[len(declPath)-1]
	file.lsp.logger.DebugContext(
		ctx,
		"completion: processing declaration",
		slog.String("kind", decl.Kind().String()),
	)

	// Return context-specific completions based on declaration type.
	switch decl.Kind() {
	case ast.DeclKindInvalid:
		file.lsp.logger.DebugContext(ctx, "completion: ignoring invalid declaration")
		return nil
	case ast.DeclKindDef:
		return completionItemsForDef(ctx, file, declPath, decl.AsDef(), position)
	case ast.DeclKindSyntax:
		return completionItemsForSyntax(ctx, file, decl.AsSyntax(), position)
	case ast.DeclKindPackage:
		return completionItemsForPackage(ctx, file, decl.AsPackage())
	case ast.DeclKindImport:
		return completionItemsForImport(ctx, file, decl.AsImport(), position)
	default:
		file.lsp.logger.DebugContext(ctx, "completion: unknown declaration type", slog.String("kind", decl.Kind().String()))
		return nil
	}
}

// completionItemsForSyntax returns the completion items for the files syntax.
func completionItemsForSyntax(ctx context.Context, file *file, syntaxDecl ast.DeclSyntax, position protocol.Position) []protocol.CompletionItem {
	file.lsp.logger.DebugContext(ctx, "completion: syntax declaration", slog.Bool("is_edition", syntaxDecl.IsEdition()))

	positionLocation := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	offset := positionLocation.Offset

	valueSpan := syntaxDecl.Value().Span()
	start, end := valueSpan.Start, valueSpan.End
	valueText := valueSpan.Text()

	// Break on newlines to split an unintended capture.
	if index := strings.IndexByte(valueText, '\n'); index != -1 {
		end = start + index
		valueText = valueText[:index]
	}

	if start > offset || offset > end {
		file.lsp.logger.DebugContext(
			ctx, "completion: syntax outside value",
			slog.String("value", valueSpan.Text()),
			slog.Int("start", start),
			slog.Int("offset", offset),
			slog.Int("end", end),
			slog.Int("line", int(position.Line)),
			slog.Int("character", int(position.Character)),
		)
		return nil // outside value
	}

	index := offset - start
	prefix := valueText[:index]
	suffix := valueText[index:]

	var additionalTextEdits []protocol.TextEdit
	if syntaxDecl.Equals().IsZero() {
		valueRange := reportSpanToProtocolRange(valueSpan)
		additionalTextEdits = append(additionalTextEdits, protocol.TextEdit{
			NewText: "= ",
			// Insert before value.
			Range: protocol.Range{
				Start: valueRange.Start,
				End:   valueRange.Start,
			},
		})
	}
	if syntaxDecl.Semicolon().IsZero() {
		additionalTextEdits = append(additionalTextEdits, protocol.TextEdit{
			NewText: ";",
			// End of line.
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      position.Line,
					Character: math.MaxInt32 - 1,
				},
				End: protocol.Position{
					Line:      position.Line,
					Character: math.MaxUint32,
				},
			},
		})
	}

	var syntaxes iter.Seq[syntax.Syntax]
	if syntaxDecl.IsEdition() {
		syntaxes = syntax.Editions()
	} else {
		syntaxes = func(yield func(syntax.Syntax) bool) {
			_ = yield(syntax.Proto2) &&
				yield(syntax.Proto3)
		}
	}
	var items []protocol.CompletionItem
	for syntax := range syntaxes {
		suggest := fmt.Sprintf("%q", syntax)
		if !strings.HasPrefix(suggest, prefix) || !strings.HasSuffix(suggest, suffix) {
			file.lsp.logger.Debug("completion: skipping on prefix/suffix",
				slog.String("value", valueSpan.Text()),
				slog.String("suggest", suggest),
				slog.String("prefix", prefix),
				slog.String("suffix", suffix),
			)
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label: syntax.String(),
			Kind:  protocol.CompletionItemKindValue,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: position,
					End:   position,
				},
				NewText: suggest[len(prefix) : len(suggest)-len(suffix)],
			},
			AdditionalTextEdits: additionalTextEdits,
		})
	}
	return items
}

// completionItemsForPackage returns the completion items for the package name.
//
// Suggest the package name based on the filepath.
func completionItemsForPackage(ctx context.Context, file *file, syntaxPackage ast.DeclPackage) []protocol.CompletionItem {
	components := normalpath.Components(file.objectInfo.Path())
	suggested := components[:len(components)-1] // Strip the filename.
	if len(suggested) == 0 {
		file.lsp.logger.DebugContext(ctx, "completion: package at root, no suggestions")
		return nil // File is at root, return no suggestions.
	}
	file.lsp.logger.DebugContext(ctx, "completion: package suggestion", slog.String("package", strings.Join(suggested, ".")))
	return []protocol.CompletionItem{{
		Label: strings.Join(suggested, ".") + ";",
		Kind:  protocol.CompletionItemKindModule,
	}}
}

// completionItemsForDef returns completion items for definition declarations (message, enum, service, etc.).
func completionItemsForDef(ctx context.Context, file *file, declPath []ast.DeclAny, def ast.DeclDef, position protocol.Position) []protocol.CompletionItem {
	file.lsp.logger.DebugContext(
		ctx,
		"completion: definition",
		slog.String("type", def.Type().Span().String()),
		slog.String("name", def.Name().Span().String()),
		slog.String("kind", def.Classify().String()),
		slog.Int("path_depth", len(declPath)),
	)

	// Extract the span of the definition to include the type and name.
	// We only complete in types, but the type may be capture in the name for an unfinished def.
	span := def.Span()
	if !def.Type().IsZero() {
		span.Start = def.Type().Span().Start
	}
	if !def.Name().IsZero() {
		span.End = def.Name().Span().End
	}
	positionLocation := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	offset := positionLocation.Offset
	if !offsetInSpan(span, offset) {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: ignoring definition outside span",
			slog.String("kind", def.Classify().String()),
		)
		return nil
	}

	tokenSpan := extractAroundToken(file, offset)
	tokenPrefix, tokenSuffix := splitSpan(tokenSpan, offset)
	typeSpan := extractLine(span, offset)
	typePrefix, typeSuffix := splitSpan(typeSpan, offset)

	if !offsetInSpan(typeSpan, offset) {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: ignoring definition outside type and name",
			slog.String("kind", def.Classify().String()),
		)
		return nil
	}

	switch def.Classify() {
	case ast.DefKindMessage, ast.DefKindService, ast.DefKindEnum:
		return nil // ignored
	case ast.DefKindField, ast.DefKindInvalid:
		// Use DefKindField and DefKindInvalid as completion starts.
		// An invalid field is caused from partial values, with still invalid syntax.
	default:
		file.lsp.logger.DebugContext(ctx, "completion: unknown definition type", slog.String("kind", def.Classify().String()))
		return nil
	}

	// Compute the heuristics for completion. Use strings over the token stream for whitespace handling.
	prefixCount := 0
	hasDeclaration := false
	for value := range strings.FieldsSeq(strings.TrimSuffix(typePrefix, tokenPrefix)) {
		_, isDeclaration := declarationSet[value]
		hasDeclaration = hasDeclaration || isDeclaration
		prefixCount++
	}
	suffixCount := 0
	for range strings.FieldsSeq(strings.TrimPrefix(typeSuffix, tokenSuffix)) {
		suffixCount++
	}

	// If at the top level, and on the first item, return top level keywords.
	if len(declPath) == 1 && prefixCount == 0 {
		file.lsp.logger.DebugContext(ctx, "completion: definition returning top-level keywords")
		return slices.Collect(keywordToCompletionItem(
			topLevelKeywords(),
			protocol.CompletionItemKindKeyword,
			tokenSpan,
			offset,
		))
	}

	parent := declPath[len(declPath)-2]
	if parent.Kind() != ast.DeclKindDef {
		return nil
	}
	parentDef := parent.AsDef()
	file.lsp.logger.DebugContext(
		ctx, "completion: definition nested declaration",
		slog.String("kind", parentDef.Classify().String()),
	)

	// Limit completions based on the following heuristic:
	// - Show keywords for the first values
	// - Show types up until the last value, and no declaration keyword is present
	showKeywords := prefixCount == 0
	showTypes := prefixCount < 2 && suffixCount < 2 && !hasDeclaration

	var iters []iter.Seq[protocol.CompletionItem]
	switch parentDef.Classify() {
	case ast.DefKindMessage:
		if showKeywords {
			isProto2 := isProto2(file)
			iters = append(iters,
				keywordToCompletionItem(
					messageLevelKeywords(isProto2),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
				keywordToCompletionItem(
					messageLevelFieldKeywords(),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
			)
		}
		if showTypes {
			iters = append(iters,
				keywordToCompletionItem(
					predeclaredTypeKeywords(),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
				typeReferencesToCompletionItems(
					file,
					findTypeFullName(file, parentDef),
					tokenSpan,
					offset,
				),
			)
		}
	case ast.DefKindService:
		if showKeywords {
			iters = append(iters,
				keywordToCompletionItem(
					serviceLevelKeywords(),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
			)
		}
	case ast.DefKindMethod:
		if showKeywords {
			iters = append(iters,
				keywordToCompletionItem(
					methodLevelKeywords(),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
			)
		}
	case ast.DefKindEnum:
		if showKeywords {
			iters = append(iters,
				keywordToCompletionItem(
					enumLevelKeywords(),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
			)
		}
	default:
		return nil
	}
	if len(iters) == 0 {
		return nil
	}
	return slices.Collect(joinSequences(iters...))
}

// completionItemsForImport returns completion items for import declarations.
//
// Suggest all importable files.
func completionItemsForImport(ctx context.Context, file *file, declImport ast.DeclImport, position protocol.Position) []protocol.CompletionItem {
	file.lsp.logger.DebugContext(ctx, "completion: import declaration", slog.Int("importable_count", len(file.workspace.PathToFile())))

	positionLocation := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	offset := positionLocation.Offset

	importPathSpan := declImport.ImportPath().Span()
	start, end := importPathSpan.Start, importPathSpan.End
	importPathText := importPathSpan.Text()

	// Break on newlines to split an unintended capture.
	if index := strings.IndexByte(importPathText, '\n'); index != -1 {
		end = start + index
		importPathText = importPathText[:index]
	}

	if start > offset || offset > end {
		file.lsp.logger.Debug("completion: outside import expr range",
			slog.String("import", importPathText),
			slog.Int("start", start),
			slog.Int("offset", offset),
			slog.Int("end", end),
			slog.Int("line", int(position.Line)),
			slog.Int("character", int(position.Character)),
		)
		return nil
	}

	index := offset - start
	prefix := importPathText[:index]
	suffix := importPathText[index:]

	// We ought to also send along an AdditionalTextEdit for adding the `;` at the end of the
	// line, if it doesn't already exist.
	var additionalTextEdits []protocol.TextEdit
	if declImport.Semicolon().IsZero() {
		additionalTextEdits = append(additionalTextEdits, protocol.TextEdit{
			NewText: ";",
			// End of line.
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      position.Line,
					Character: math.MaxInt32 - 1,
				},
				End: protocol.Position{
					Line:      position.Line,
					Character: math.MaxUint32,
				},
			},
		})
	}

	var items []protocol.CompletionItem
	for importPath, importFile := range file.workspace.PathToFile() {
		if file == importFile {
			continue // ignore self
		}

		suggest := fmt.Sprintf("%q", importPath)
		if !strings.HasPrefix(suggest, prefix) || !strings.HasSuffix(suggest, suffix) {
			file.lsp.logger.Debug("completion: skipping on prefix/suffix",
				slog.String("import", importPathText),
				slog.String("suggest", suggest),
				slog.String("prefix", prefix),
				slog.String("suffix", suffix),
			)
			continue
		}

		var importFileIsDeprecated bool
		if _, ok := importFile.ir.Deprecated().AsBool(); ok {
			importFileIsDeprecated = true
		}
		items = append(items, protocol.CompletionItem{
			Label: importPath,
			Kind:  protocol.CompletionItemKindFile,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: position,
					End:   position,
				},
				NewText: suggest[len(prefix) : len(suggest)-len(suffix)],
			},
			AdditionalTextEdits: additionalTextEdits,
			Deprecated:          importFileIsDeprecated,
		})
	}
	return items
}

// declarationSet is the set of keywords that starts a declaration.
var declarationSet = func() map[string]struct{} {
	m := make(map[string]struct{})
	for keyword := range joinSequences(
		topLevelKeywords(),
		messageLevelKeywords(true),
	) {
		m[keyword.String()] = struct{}{}
	}
	return m
}()

// topLevelKeywords returns keywords for the top-level.
func topLevelKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Syntax) &&
			yield(keyword.Edition) &&
			yield(keyword.Import) &&
			yield(keyword.Package) &&
			yield(keyword.Message) &&
			yield(keyword.Service) &&
			yield(keyword.Option) &&
			yield(keyword.Enum) &&
			yield(keyword.Extend)
	}
}

// messageLevelFieldKeywords returns keywords for messages.
func messageLevelKeywords(isProto2 bool) iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		ok := yield(keyword.Message) &&
			yield(keyword.Enum) &&
			yield(keyword.Option) &&
			yield(keyword.Extend) &&
			yield(keyword.Oneof) &&
			yield(keyword.Extensions) &&
			yield(keyword.Reserved)
		_ = ok && isProto2 &&
			yield(keyword.Group)
	}
}

// messageLevelFieldKeywords returns keywords for type modifiers.
func messageLevelFieldKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Repeated) &&
			yield(keyword.Optional) &&
			yield(keyword.Required)
	}
}

// predeclaredTypeKeywords returns keywords for all predeclared types.
func predeclaredTypeKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Int32) &&
			yield(keyword.Int64) &&
			yield(keyword.UInt32) &&
			yield(keyword.UInt64) &&
			yield(keyword.SInt32) &&
			yield(keyword.SInt64) &&
			yield(keyword.Fixed32) &&
			yield(keyword.Fixed64) &&
			yield(keyword.SFixed32) &&
			yield(keyword.SFixed64) &&
			yield(keyword.Float) &&
			yield(keyword.Double) &&
			yield(keyword.Bool) &&
			yield(keyword.String) &&
			yield(keyword.Bytes)
	}
}

// serviceLevelKeywords returns keywords for services.
func serviceLevelKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.RPC) &&
			yield(keyword.Option)
	}
}

// methodLevelKeywords returns keywords for methods
func methodLevelKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Option)
	}
}

// enumLevelKeywords returns keywords for enums.
func enumLevelKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Option)
	}
}

// keywordToCompletionItem converts a keyword to a completion item.
func keywordToCompletionItem(
	keywords iter.Seq[keyword.Keyword],
	kind protocol.CompletionItemKind,
	span report.Span,
	offset int,
) iter.Seq[protocol.CompletionItem] {
	return func(yield func(protocol.CompletionItem) bool) {
		editRange := reportSpanToProtocolRange(span)
		prefix, suffix := splitSpan(span, offset)
		for keyword := range keywords {
			suggest := keyword.String()
			if !strings.HasPrefix(suggest, prefix) || !strings.HasSuffix(suggest, suffix) {
				continue
			}
			if !yield(protocol.CompletionItem{
				Label: suggest,
				Kind:  kind,
				TextEdit: &protocol.TextEdit{
					Range:   editRange,
					NewText: suggest,
				},
			}) {
				break
			}
		}
	}
}

// typeReferencesToCompletionItems returns completion items for user-defined types (messages, enums, etc).
func typeReferencesToCompletionItems(
	current *file,
	parentFullName ir.FullName,
	span report.Span,
	offset int,
) iter.Seq[protocol.CompletionItem] {
	fileSymbolTypesIter := func(yield func(*file, *symbol) bool) {
		for _, imported := range current.workspace.PathToFile() {
			for _, symbol := range imported.referenceableSymbols {
				if !yield(imported, symbol) {
					return
				}
			}
		}
	}
	var lastImportLine int64
	currentImportPaths := map[string]struct{}{}
	for currentFileImport := range seq.Values(current.ir.Imports()) {
		lastImportLine = max(lastImportLine, int64(currentFileImport.Decl.Span().EndLoc().Line))
		currentImportPaths[currentFileImport.Path()] = struct{}{}
	}
	if lastImportLine == 0 {
		// If lastImportLine is 0, we have no imports in this file; put it after `package`, if
		// package exists. Otherwise, after `syntax`, if it exists. Otherwise, balk.
		// NOTE: We simply want to add the import on the next line (which may not be how `buf
		// format` would format the file); we leave the overall file formatting to `buf format`.
		switch {
		case !current.ir.AST().Package().IsZero():
			lastImportLine = int64(current.ir.AST().Package().Span().EndLoc().Line)
		case !current.ir.AST().Syntax().IsZero():
			lastImportLine = int64(current.ir.AST().Syntax().Span().EndLoc().Line)
		}
	}
	if lastImportLine < 0 || lastImportLine > math.MaxUint32 {
		lastImportLine = 0 // Default to insert at top of page.
	}
	importInsertPosition := protocol.Position{
		Line:      uint32(lastImportLine),
		Character: 0,
	}
	parentPrefix := string(parentFullName) + "."
	packagePrefix := string(current.ir.Package()) + "."
	return func(yield func(protocol.CompletionItem) bool) {
		editRange := reportSpanToProtocolRange(span)
		prefix, _ := splitSpan(span, offset)
		// Prefix filter on the trigger character '.', if present.
		prefix = prefix[:strings.LastIndexByte(prefix, '.')+1]
		for _, symbol := range fileSymbolTypesIter {
			// We only support types in this completion instance, and not scalar values, which leaves us
			// with messages and enums.
			var kind protocol.CompletionItemKind
			switch symbol.ir.Kind() {
			case ir.SymbolKindMessage:
				kind = protocol.CompletionItemKindClass // Messages are like classes
			case ir.SymbolKindEnum:
				kind = protocol.CompletionItemKindEnum
			default:
				continue // Unsupported kind, skip it.
			}
			label := string(symbol.ir.FullName())
			if strings.HasPrefix(label, parentPrefix) {
				label = label[len(parentPrefix):]
			} else if strings.HasPrefix(label, packagePrefix) {
				label = label[len(packagePrefix):]
			}
			if !strings.HasPrefix(label, prefix) {
				continue
			}
			var isDeprecated bool
			if _, ok := symbol.ir.Deprecated().AsBool(); ok {
				isDeprecated = true
			}
			symbolFile := symbol.ir.File().Path()
			_, hasImport := currentImportPaths[symbolFile]
			var additionalTextEdits []protocol.TextEdit
			if !hasImport && symbolFile != current.ir.Path() {
				additionalTextEdits = append(additionalTextEdits, protocol.TextEdit{
					NewText: "import " + `"` + symbolFile + `";` + "\n",
					Range: protocol.Range{
						Start: importInsertPosition,
						End:   importInsertPosition,
					},
				})
			}
			if !yield(protocol.CompletionItem{
				Label: label,
				Kind:  kind,
				TextEdit: &protocol.TextEdit{
					Range:   editRange,
					NewText: label,
				},
				Deprecated:          isDeprecated,
				Documentation:       symbol.FormatDocs(),
				AdditionalTextEdits: additionalTextEdits,
			}) {
				break
			}
		}
	}
}

// joinSequences returns a sequence of sequences.
func joinSequences[T any](itemIters ...iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, items := range itemIters {
			for item := range items {
				if !yield(item) {
					break
				}
			}
		}
	}
}

// getDeclForPosition finds the path of AST declarations from parent to smallest that contains the given protocol position.
// Returns a slice where [0] is the top-level declaration and [len-1] is the smallest/innermost declaration.
// Returns nil if no declaration contains the position.
func getDeclForPosition(body ast.DeclBody, position protocol.Position) []ast.DeclAny {
	return getDeclForPositionHelper(body, position, nil)
}

// getDeclForPositionHelper is the recursive helper for getDeclForPosition.
func getDeclForPositionHelper(body ast.DeclBody, position protocol.Position, path []ast.DeclAny) []ast.DeclAny {
	smallestSize := -1
	var bestPath []ast.DeclAny

	for decl := range seq.Values(body.Decls()) {
		if decl.IsZero() {
			continue
		}
		span := decl.Span()
		if span.IsZero() {
			continue
		}

		// Check if the position is within this declaration's span.
		within := reportSpanToProtocolRange(span)
		if positionInRange(position, within) == 0 {
			// Build the new path including this declaration.
			newPath := append(append([]ast.DeclAny(nil), path...), decl)
			size := span.End - span.Start
			if smallestSize == -1 || size < smallestSize {
				bestPath = newPath
				smallestSize = size
			}

			// If this is a definition with a body, search recursively.
			if decl.Kind() == ast.DeclKindDef && !decl.AsDef().Body().IsZero() {
				childPath := getDeclForPositionHelper(decl.AsDef().Body(), position, newPath)
				if len(childPath) > 0 {
					childSize := childPath[len(childPath)-1].Span().End - childPath[len(childPath)-1].Span().Start
					if childSize < smallestSize {
						bestPath = childPath
						smallestSize = childSize
					}
				}
			}
		}
	}
	return bestPath
}

// extractAroundToken extracts the value around the offset by querying the token stream.
func extractAroundToken(file *file, offset int) report.Span {
	if file.ir.AST().IsZero() {
		return report.Span{}
	}
	stream := file.ir.AST().Context().Stream()
	if stream == nil {
		return report.Span{}
	}
	before, after := stream.Around(offset)
	if before.IsZero() && after.IsZero() {
		return report.Span{}
	}

	isToken := func(tok token.Token) bool {
		switch kind := tok.Kind(); kind {
		case token.Ident, token.Punct:
			return true
		default:
			return false
		}
	}
	span := report.Span{
		File:  file.file,
		Start: offset,
		End:   offset,
	}
	if !before.IsZero() {
		cursor := token.NewCursorAt(before)
		for tok := cursor.PrevSkippable(); isToken(tok); tok = cursor.PrevSkippable() {
			before = tok
		}
		if isToken(before) {
			span.Start = before.Span().Start
		}
	}
	if !after.IsZero() {
		cursor := token.NewCursorAt(after)
		for tok := cursor.NextSkippable(); isToken(tok); tok = cursor.NextSkippable() {
			after = tok
		}
		if isToken(after) {
			span.End = after.Span().End
		}
	}
	return span
}

// extractLine extracts the line around the offset.
func extractLine(span report.Span, offset int) report.Span {
	text := span.Text()
	index := offset - span.Start
	if newLine := strings.IndexByte(text[index:], '\n'); newLine >= 0 {
		span.End = span.Start + index + newLine
	}
	span.Start += strings.LastIndexByte(text[:index], '\n') + 1
	return span
}

func splitSpan(span report.Span, offset int) (prefix string, suffix string) {
	if !offsetInSpan(span, offset) {
		return "", ""
	}
	index := offset - span.Start
	text := span.Text()
	return text[:index], text[index:]
}

func offsetInSpan(span report.Span, offset int) bool {
	return span.Start <= offset && offset <= span.End // End is inclusive for completions_
}

// isProto2 returns true if the file has a syntax declaration of proto2.
func isProto2(file *file) bool {
	return file.ir.Syntax() == syntax.Proto2
}

// findTypeFullName simply loops through and finds the type definition name.
func findTypeFullName(file *file, declDef ast.DeclDef) ir.FullName {
	declDefSpan := declDef.Span()
	if declDefSpan.IsZero() {
		return ""
	}
	for irType := range seq.Values(file.ir.AllTypes()) {
		typeSpan := irType.AST().Span()
		if typeSpan.Start == declDefSpan.Start && typeSpan.End == declDefSpan.End {
			file.lsp.logger.Debug("completion: found parent type", slog.String("parent", string(irType.FullName())))
			return irType.FullName()
		}
	}
	return ""
}

// resolveCompletionItem resolves additional details for a completion item.
//
// This function is called by the CompletionResolve handler in server.go.
func resolveCompletionItem(
	ctx context.Context,
	item *protocol.CompletionItem,
) (*protocol.CompletionItem, error) {
	// TODO: Implement completion resolution logic.
	// For now, just return the item unchanged.
	return item, nil
}
