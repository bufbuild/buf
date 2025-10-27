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
			slog.Any("decl_span_start", decl.Span().StartLoc()),
			slog.Any("decl_span_end", decl.Span().EndLoc()),
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
		return completionItemsForSyntax(ctx, file, decl.AsSyntax())
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
func completionItemsForSyntax(ctx context.Context, file *file, syntaxDecl ast.DeclSyntax) []protocol.CompletionItem {
	file.lsp.logger.DebugContext(ctx, "completion: syntax declaration", slog.Bool("is_edition", syntaxDecl.IsEdition()))

	var prefix string
	if syntaxDecl.Equals().IsZero() {
		prefix += "= "
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
		items = append(items, protocol.CompletionItem{
			Label: fmt.Sprintf("%s%q;", prefix, syntax),
			Kind:  protocol.CompletionItemKindValue,
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
		slog.String("def_kind", def.Classify().String()),
		slog.Int("path_depth", len(declPath)),
		slog.Any("decl_span_start", def.Span().StartLoc()),
		slog.Any("decl_span_end", def.Span().EndLoc()),
	)

	// Check if cursor is in the name of the definition, if so ignore.
	if !def.Name().IsZero() {
		nameSpan := def.Name().Span()
		within := reportSpanToProtocolRange(nameSpan)
		if positionInRange(position, within) == 0 {
			file.lsp.logger.DebugContext(
				ctx,
				"completion: ignoring definition name",
				slog.String("kind", def.Classify().String()),
			)
			return nil
		}
	}
	// Check if the cursor is outside the type, if so ignore.
	if !def.Type().IsZero() {
		typeSpan := def.Type().Span()
		within := reportSpanToProtocolRange(typeSpan)
		if positionInRange(position, within) > 0 {
			file.lsp.logger.DebugContext(
				ctx,
				"completion: ignoring definition passed type",
				slog.String("kind", def.Classify().String()),
			)
			return nil
		}
	}

	switch def.Classify() {
	case ast.DefKindMessage:
		return nil
	case ast.DefKindService:
		return nil
	case ast.DefKindEnum:
		return nil
	case ast.DefKindField:
		return completionItemsForField(ctx, file, declPath, def, position)
	case ast.DefKindInvalid:
		return completionItemsForKeyword(ctx, file, declPath, def, position)
	default:
		file.lsp.logger.DebugContext(ctx, "completion: unknown definition type", slog.String("kind", def.Classify().String()))
		return nil
	}
}

// completionItemsForKeyword returns completion items for a declaration expecting keywords.
func completionItemsForKeyword(ctx context.Context, file *file, declPath []ast.DeclAny, def ast.DeclDef, position protocol.Position) []protocol.CompletionItem {
	positionLocation := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	offset := positionLocation.Offset
	span := def.Span()

	// Check if at newline or end of span. Keywords are restricted to the first identifier.
	if !isNewlineOrEndOfSpan(span, offset) {
		if len(declPath) > 1 {
			return completionItemsForField(ctx, file, declPath, def, position)
		}
		file.lsp.logger.Debug("completion: keyword skip on span bounds", slog.String("span", span.Text()))
		return nil
	}

	tokenSpan := extractAroundToken(file, offset)
	file.lsp.logger.DebugContext(ctx, "completion: keyword items", slog.String("span", span.Text()))

	// If this is an invalid definition at the top level, return top-level keywords.
	if len(declPath) == 1 {
		file.lsp.logger.DebugContext(ctx, "completion: keyword returning top-level")
		return slices.Collect(keywordToCompletionItem(
			topLevelKeywords(),
			protocol.CompletionItemKindKeyword,
			tokenSpan,
			offset,
		))
	}

	parent := declPath[len(declPath)-2]
	file.lsp.logger.DebugContext(
		ctx, "completion: keyword nested definition",
		slog.String("kind", parent.Kind().String()),
	)
	if parent.Kind() != ast.DeclKindDef {
		return nil
	}

	var items iter.Seq[protocol.CompletionItem]
	parentDef := parent.AsDef()
	switch parentDef.Classify() {
	case ast.DefKindMessage:
		isProto2 := isProto2(file)
		items = joinSequences(
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
			keywordToCompletionItem(
				predeclaredTypeKeywords(),
				protocol.CompletionItemKindClass,
				tokenSpan,
				offset,
			),
			typeReferencesToCompletionItems(
				file,
				tokenSpan,
				offset,
			),
		)
	case ast.DefKindService:
		items = keywordToCompletionItem(
			serviceLevelKeywords(),
			protocol.CompletionItemKindKeyword,
			tokenSpan,
			offset,
		)
	case ast.DefKindEnum:
		items = keywordToCompletionItem(
			enumLevelKeywords(),
			protocol.CompletionItemKindKeyword,
			tokenSpan,
			offset,
		)
	default:
		return nil
	}
	return slices.Collect(items)
}

// completionItemsForField returns completion items for a field.
func completionItemsForField(ctx context.Context, file *file, declPath []ast.DeclAny, def ast.DeclDef, position protocol.Position) []protocol.CompletionItem {
	if len(declPath) == 1 {
		file.lsp.logger.DebugContext(ctx, "completion: field top level, going to keywords")
		return completionItemsForKeyword(ctx, file, declPath, def, position)
	}

	parent := declPath[len(declPath)-2]
	file.lsp.logger.DebugContext(
		ctx, "completion: field within definition",
		slog.String("kind", parent.Kind().String()),
	)
	if parent.Kind() != ast.DeclKindDef {
		return nil
	}

	positionLocation := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	offset := positionLocation.Offset

	// If on a newline, before the current field or at the end of the span return keywords.
	if isNewlineOrEndOfSpan(def.Span(), offset) {
		file.lsp.logger.DebugContext(ctx, "completion: field on newline, return keywords")
		return completionItemsForKeyword(ctx, file, declPath, def, position)
	}

	typeSpan := def.Type().Span()
	if offsetInSpan(typeSpan, offset) {
		file.lsp.logger.DebugContext(
			ctx, "completion: field outside definition",
			slog.String("kind", parent.Kind().String()),
		)
		return nil
	}

	// Resolve the token the cursor is under. We don't use the ast.FieldStruct as this doesn't
	// work with invalid path types. Instead resolve the token from the stream.
	tokenSpan := extractAroundToken(file, offset)
	tokenPrefix, tokenSuffix := splitSpan(tokenSpan, offset)
	typePrefix, typeSuffix := splitSpan(typeSpan, offset)
	prefixCount := 0
	for range strings.FieldsSeq(strings.TrimSuffix(typePrefix, tokenPrefix)) {
		prefixCount++
	}
	suffixCount := 0
	for range strings.FieldsSeq(strings.TrimPrefix(typeSuffix, tokenSuffix)) {
		suffixCount++
	}
	// Limit completions based on the following heuristic:
	// - Show modifiers for the first two types
	// - Only show types on the final type
	showModifiers := prefixCount <= 2
	showTypes := suffixCount == 0

	file.lsp.logger.DebugContext(
		ctx, "completion: got types",
		slog.String("kind", parent.Kind().String()),
		slog.String("type", typeSpan.Text()),
		slog.String("type_kind", def.Type().Kind().String()),
		slog.Int("start", typeSpan.Start),
		slog.Int("end", typeSpan.End),
		slog.Bool("show_modifiers", showModifiers),
		slog.Bool("show_types", showTypes),
		slog.String("token", tokenSpan.Text()),
	)

	var items iter.Seq[protocol.CompletionItem]
	parentDef := parent.AsDef()
	switch parentDef.Classify() {
	case ast.DefKindMessage:
		var iters []iter.Seq[protocol.CompletionItem]
		if showModifiers {
			iters = append(iters,
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
					protocol.CompletionItemKindClass,
					tokenSpan,
					offset,
				),
				typeReferencesToCompletionItems(
					file,
					tokenSpan,
					offset,
				),
			)
		}
		if len(iters) == 0 {
			return nil
		}
		items = joinSequences(iters...)
	default:
		return nil
	}
	return slices.Collect(items)
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
	for importPath := range file.workspace.PathToFile() {
		if importPath == file.objectInfo.LocalPath() {
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
		})
	}
	return items
}

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

// serviceLevelKeywords returns keywords for service.
func serviceLevelKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.RPC) &&
			yield(keyword.Option)
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
	span report.Span,
	offset int,
) iter.Seq[protocol.CompletionItem] {
	fileSymbolTypesIter := func(yield func(*file, *symbol) bool) {
		for _, imported := range current.workspace.PathToFile() {
			if imported == current {
				continue
			}
			for _, symbol := range imported.referenceableSymbols {
				if !yield(imported, symbol) {
					return
				}
			}
		}
	}
	return func(yield func(protocol.CompletionItem) bool) {
		editRange := reportSpanToProtocolRange(span)
		prefix, suffix := splitSpan(span, offset)
		for file, symbol := range fileSymbolTypesIter {
			if !symbol.ir.Kind().IsType() {
				continue
			}
			var (
				label string
				kind  protocol.CompletionItemKind
			)
			if file.ir.Package() == current.ir.Package() {
				label = symbol.ir.FullName().Name()
			} else {
				label = string(symbol.ir.FullName())
			}
			if !strings.HasPrefix(label, prefix) || !strings.HasSuffix(label, suffix) {
				continue
			}
			switch symbol.ir.Kind() {
			case ir.SymbolKindMessage:
				kind = protocol.CompletionItemKindStruct
			case ir.SymbolKindEnum:
				kind = protocol.CompletionItemKindEnum
			case ir.SymbolKindService:
				kind = protocol.CompletionItemKindInterface
			case ir.SymbolKindScalar:
				kind = protocol.CompletionItemKindClass
			case ir.SymbolKindPackage:
				kind = protocol.CompletionItemKindModule
			case ir.SymbolKindField, ir.SymbolKindEnumValue, ir.SymbolKindExtension, ir.SymbolKindOneof, ir.SymbolKindMethod:
				// These should be skipped by IsType() filter.
			}
			if kind == 0 {
				continue // Unsupported kind, skip it.
			}
			var isDeprecated bool
			if _, ok := symbol.ir.Deprecated().AsBool(); ok {
				isDeprecated = true
			}
			if !yield(protocol.CompletionItem{
				Label: label,
				Kind:  kind,
				TextEdit: &protocol.TextEdit{
					Range:   editRange,
					NewText: label,
				},
				Deprecated: isDeprecated,
				// TODO: If this type's file is not currently imported add an additional edit.
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

func splitSpan(span report.Span, offset int) (prefix string, suffix string) {
	if !offsetInSpan(span, offset) {
		return "", ""
	}
	index := offset - span.Start
	text := span.Text()
	return text[:index], text[index:]
}

func offsetInSpan(span report.Span, offset int) bool {
	return span.Start > offset || offset > span.End
}

// isNewlineOrEndOfSpan returns true if this offset is separated be a newline or at the end of the span.
// This most likely means we are at the start of a new declaration.
func isNewlineOrEndOfSpan(span report.Span, offset int) bool {
	if offset == span.End {
		return true
	}
	text := span.Text()
	index := offset - span.Start
	if newLine := strings.IndexByte(text[index:], '\n'); newLine >= 0 {
		// Newline separates the end, check theres no dangling content after us on this line.
		after := text[index : index+newLine]
		return len(strings.TrimSpace(after)) == 0
	}
	return false
}

// isProto2 returns true if the file has a syntax declaration of proto2.
func isProto2(file *file) bool {
	body := file.ir.AST().DeclBody
	for decl := range seq.Values(body.Decls()) {
		if decl.IsZero() {
			continue
		}
		if kind := decl.Kind(); kind == ast.DeclKindSyntax {
			declSyntax := decl.AsSyntax()
			return declSyntax.IsSyntax() && declSyntax.Value().Span().Text() == "proto2"
		}
	}
	return false
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
