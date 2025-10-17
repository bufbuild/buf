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
	"iter"
	"log/slog"
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ast/syntax"
	"github.com/bufbuild/protocompile/experimental/seq"
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

	decl := getDeclForPosition(file.ir.AST().DeclBody, position)
	file.lsp.logger.DebugContext(
		ctx,
		"completion: found declaration",
		slog.String("decl_kind", decl.Kind().String()),
		slog.Any("decl_span_start", decl.Span().StartLoc()),
		slog.Any("decl_span_end", decl.Span().EndLoc()),
	)

	// Return context-aware completions based on the declaration type.
	return completionItemsForDecl(file, decl, position)
}

// completionItemsForDecl returns completion items based on the AST declaration at the cursor.
func completionItemsForDecl(file *file, decl ast.DeclAny, position protocol.Position) []protocol.CompletionItem {
	if decl.IsZero() {
		file.lsp.logger.Debug("completion: no declaration found, returning top-level keywords")
		return slices.Collect(topLevelCompletionItems())
	}

	file.lsp.logger.Debug("completion: processing declaration", slog.String("kind", decl.Kind().String()))

	// Return context-specific completions based on declaration type.
	switch decl.Kind() {
	case ast.DeclKindInvalid:
		file.lsp.logger.Debug("completion: ignoring invalid declaration")
		return nil
	case ast.DeclKindDef:
		return completionItemsForDef(file, decl.AsDef(), position)
	case ast.DeclKindSyntax:
		return completionItemsForSyntax(file, decl.AsSyntax())
	case ast.DeclKindPackage:
		return completionItemsForPackage(file, decl.AsPackage())
	case ast.DeclKindImport:
		return completionItemsForImport(file)
	default:
		file.lsp.logger.Debug("completion: unknown declaration type, returning top-level keywords", slog.String("kind", decl.Kind().String()))
		return slices.Collect(topLevelCompletionItems())
	}
}

// completionItemsForSyntax returns the completion items for the files syntax.
func completionItemsForSyntax(file *file, syntaxDecl ast.DeclSyntax) []protocol.CompletionItem {
	file.lsp.logger.Debug("completion: syntax declaration", slog.Bool("is_edition", syntaxDecl.IsEdition()))

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
			Label: prefix + "\"" + syntax.String() + "\";",
			Kind:  protocol.CompletionItemKindValue,
		})
	}
	return items
}

// completionItemsForPackage returns the completion items for the package name.
//
// Suggest the package name based on the filepath.
func completionItemsForPackage(file *file, syntaxPackage ast.DeclPackage) []protocol.CompletionItem {
	components := normalpath.Components(file.objectInfo.Path())
	suggested := components[:len(components)-1] // Strip the filename.
	if len(suggested) == 0 {
		file.lsp.logger.Debug("completion: package at root, no suggestions")
		return nil // File is at root, return no suggestions.
	}
	file.lsp.logger.Debug("completion: package suggestion", slog.String("package", strings.Join(suggested, ".")))
	return []protocol.CompletionItem{{
		Label: strings.Join(suggested, ".") + ";",
		Kind:  protocol.CompletionItemKindSnippet,
	}}
}

// completionItemsForDef returns completion items for definition declarations (message, enum, service, etc.).
func completionItemsForDef(file *file, def ast.DeclDef, position protocol.Position) []protocol.CompletionItem {
	// Check if cursor is in the name of the definition, if so ignore.
	if !def.Name().IsZero() {
		nameSpan := def.Name().Span()
		within := reportSpanToProtocolRange(nameSpan)
		if positionInRange(position, within) == 0 {
			file.lsp.logger.Debug("completion: ignoring definition name",
				slog.String("def_kind", def.Classify().String()),
			)
			return nil
		}
	}

	file.lsp.logger.Debug("completion: definition", slog.String("def_kind", def.Classify().String()))

	switch def.Classify() {
	case ast.DefKindInvalid:
		file.lsp.logger.Debug("completion: ignoring invalid definition")
		return nil
	case ast.DefKindMessage:
		return completionItemsForMessage(file)
	case ast.DefKindService:
		return completionItemsForService(file, def.AsService(), position)
	case ast.DefKindEnum:
		return []protocol.CompletionItem{
			keywordToCompletionItem(keyword.Option),
		}
	case ast.DefKindField:
		return completionItemsForField(file, def.AsField(), position)
	default:
		file.lsp.logger.Debug("completion: unknown definition type, returning top-level keywords", slog.String("def_kind", def.Classify().String()))
		return slices.Collect(topLevelCompletionItems())
	}
}

// completionItemsForMessage returns completion items for inside a message definition.
func completionItemsForMessage(file *file) []protocol.CompletionItem {
	file.lsp.logger.Debug("completion: message body")

	items := []protocol.CompletionItem{
		keywordToCompletionItem(keyword.Message),
		keywordToCompletionItem(keyword.Enum),
		keywordToCompletionItem(keyword.Option),
		keywordToCompletionItem(keyword.Oneof),
		keywordToCompletionItem(keyword.Extensions),
		keywordToCompletionItem(keyword.Reserved),
		keywordToCompletionItem(keyword.Extend),
		// Field keyword
		keywordToCompletionItem(keyword.Repeated),
		keywordToCompletionItem(keyword.Optional),
		keywordToCompletionItem(keyword.Required),
	}
	// Add predeclared types (primitives).
	items = slices.AppendSeq(items, predeclaredTypeCompletionItems())
	// Add referenceable types (messages, enums) from this file and imports.
	items = slices.AppendSeq(items, referenceableTypeCompletionItems(file))
	return items
}

// completionItemsForService returns completion items for inside a service definition.
func completionItemsForService(file *file, defService ast.DefService, position protocol.Position) []protocol.CompletionItem {
	file.lsp.logger.Debug("completion: service body")

	return []protocol.CompletionItem{
		keywordToCompletionItem(keyword.RPC),
		keywordToCompletionItem(keyword.Option),
	}
}

// completionItemsForImport returns completion items for import declarations.
//
// Suggest all importable files.
func completionItemsForImport(file *file) []protocol.CompletionItem {
	file.lsp.logger.Debug("completion: import declaration", slog.Int("importable_count", len(file.importToFile)))

	items := make([]protocol.CompletionItem, 0, len(file.importToFile))
	for importPath := range file.importToFile {
		items = append(items, protocol.CompletionItem{
			Label: " \"" + importPath + "\";",
			Kind:  protocol.CompletionItemKindFile,
		})
	}
	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})
	return items
}

// completionItemsForField returns the completion items for a field declaration.
func completionItemsForField(file *file, defField ast.DefField, position protocol.Position) []protocol.CompletionItem {
	// Check if cursor is within the field name, if so ignore.
	if !defField.Name.IsZero() {
		nameSpan := defField.Name.Span()
		within := reportSpanToProtocolRange(nameSpan)
		if positionInRange(position, within) == 0 {
			file.lsp.logger.Debug("completion: ignoring field name",
				slog.String("field_name", defField.Name.Text()),
			)
			return nil
		}
	}

	file.lsp.logger.Debug("completion: field type")

	items := slices.Collect(predeclaredTypeCompletionItems())
	items = slices.AppendSeq(items, referenceableTypeCompletionItems(file))
	return items
}

// predeclaredTypeCompletionItems returns completion items for predeclared (primitive) types.
func predeclaredTypeCompletionItems() iter.Seq[protocol.CompletionItem] {
	keywordToType := func(kw keyword.Keyword) protocol.CompletionItem {
		return protocol.CompletionItem{
			Label: kw.String(),
			Kind:  protocol.CompletionItemKindTypeParameter,
		}
	}
	return func(yield func(protocol.CompletionItem) bool) {
		_ = yield(keywordToType(keyword.Int32)) &&
			yield(keywordToType(keyword.Int64)) &&
			yield(keywordToType(keyword.UInt32)) &&
			yield(keywordToType(keyword.UInt64)) &&
			yield(keywordToType(keyword.SInt32)) &&
			yield(keywordToType(keyword.SInt64)) &&
			yield(keywordToType(keyword.Fixed32)) &&
			yield(keywordToType(keyword.Fixed64)) &&
			yield(keywordToType(keyword.SFixed32)) &&
			yield(keywordToType(keyword.SFixed64)) &&
			yield(keywordToType(keyword.Float)) &&
			yield(keywordToType(keyword.Double)) &&
			yield(keywordToType(keyword.Bool)) &&
			yield(keywordToType(keyword.String)) &&
			yield(keywordToType(keyword.Bytes))
	}
}

// referenceableTypeCompletionItems returns completion items for user-defined types (messages, enums).
func referenceableTypeCompletionItems(current *file) iter.Seq[protocol.CompletionItem] {
	fileSymobolTypesIter := func(yield func(*file, *symbol) bool) {
		for _, imported := range current.importToFile {
			for _, symbol := range imported.referenceableSymbols {
				if !symbol.ir.Kind().IsType() {
					continue
				}
				if !yield(imported, symbol) {
					return
				}
			}
		}
	}
	return func(yield func(protocol.CompletionItem) bool) {
		for file, symbol := range fileSymobolTypesIter {
			var label string
			if file.ir.Package() == current.ir.Package() {
				label = symbol.ir.FullName().Name()
			} else {
				label = string(symbol.ir.FullName())
			}
			item := protocol.CompletionItem{
				Label: label,
				Kind:  protocol.CompletionItemKindTypeParameter,
			}
			if !yield(item) {
				return
			}
		}
	}
}

// topLevelCompletionItems returns completion items for top-level proto keywords.
func topLevelCompletionItems() iter.Seq[protocol.CompletionItem] {
	return func(yield func(protocol.CompletionItem) bool) {
		_ = yield(keywordToCompletionItem(keyword.Syntax)) &&
			yield(keywordToCompletionItem(keyword.Edition)) &&
			yield(keywordToCompletionItem(keyword.Import)) &&
			yield(keywordToCompletionItem(keyword.Package)) &&
			yield(keywordToCompletionItem(keyword.Message)) &&
			yield(keywordToCompletionItem(keyword.Service)) &&
			yield(keywordToCompletionItem(keyword.Option)) &&
			yield(keywordToCompletionItem(keyword.Enum)) &&
			yield(keywordToCompletionItem(keyword.Extend))
	}
}

// keywordToCompletionItem converts a keyword to a completion item.
func keywordToCompletionItem(kw keyword.Keyword) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label: kw.String(),
		Kind:  protocol.CompletionItemKindKeyword,
	}
}

// getDeclForPosition finds the smallest AST declaration that contains the given protocol position.
func getDeclForPosition(body ast.DeclBody, position protocol.Position) ast.DeclAny {
	smallestSize := -1
	var smallest ast.DeclAny
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
			size := span.End - span.Start
			if smallestSize == -1 || size < smallestSize {
				smallest = decl
				smallestSize = size
			}
			// If this is a definition with a body, search recursively.
			if decl.Kind() == ast.DeclKindDef && !decl.AsDef().Body().IsZero() {
				child := getDeclForPosition(decl.AsDef().Body(), position)
				if !child.IsZero() {
					childSize := child.Span().End - child.Span().Start
					if childSize < smallestSize {
						smallest = child
						smallestSize = childSize
					}
				}
			}
		}
	}
	return smallest
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
