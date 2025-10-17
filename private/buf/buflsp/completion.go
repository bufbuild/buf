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

	"buf.build/go/standard/xslices"
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

	// Find the smallest AST declaration containing this offset.
	decl := getDeclForPosition(file.ir.AST().DeclBody, position)

	file.lsp.logger.DebugContext(
		ctx,
		"decl for completion",
		slog.String("decl_kind", decl.Kind().String()),
		slog.Any("decl_span_start", decl.Span().StartLoc()),
		slog.Any("decl_span_end", decl.Span().EndLoc()),
		//slog.Int("cursor_offset", offset),
		//slog.String("prefix", prefix),
	)

	prefix := ""

	// Return context-aware completions based on the declaration type.
	return completionItemsForDecl(file, decl, prefix)
}

// completionItemsForDecl returns completion items based on the AST declaration at the cursor.
func completionItemsForDecl(file *file, decl ast.DeclAny, prefix string) []protocol.CompletionItem {
	if decl.IsZero() {
		// No declaration found, return top-level keywords.
		return topLevelCompletionItems()
	}

	// Return context-specific completions based on declaration type.
	switch decl.Kind() {
	case ast.DeclKindDef:
		return completionItemsForDef(file, decl.AsDef(), prefix)
	case ast.DeclKindSyntax:
		return completionItemsForSyntax(file, decl.AsSyntax(), prefix)
	case ast.DeclKindPackage:
		return completionItemsForPackage(file, decl.AsPackage(), prefix)
	case ast.DeclKindImport:
		return completionItemsForImport(file)
	default:
		// For other declaration types, fall back to top-level keywords.
		return topLevelCompletionItems()
	}
}

// completionItemsForSyntax returns the completion items for the files syntax.
func completionItemsForSyntax(file *file, syntaxDecl ast.DeclSyntax, _ string) []protocol.CompletionItem {
	var prefix string
	if syntaxDecl.KeywordToken().IsZero() {
		if syntaxDecl.IsEdition() {
			prefix += "edition"
		} else {
			prefix += "syntax"
		}
	}
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
func completionItemsForPackage(file *file, syntaxPackage ast.DeclPackage, _ string) []protocol.CompletionItem {
	components := normalpath.Components(file.objectInfo.Path())
	suggested := components[:len(components)-1] // Strip the filename.
	if len(suggested) == 0 {
		return nil // File is at root, return no suggestions.
	}
	return []protocol.CompletionItem{{
		Label: strings.Join(suggested, ".") + ";",
		Kind:  protocol.CompletionItemKindSnippet,
	}}
}

// completionItemsForDef returns completion items for definition declarations (message, enum, service, etc.).
func completionItemsForDef(file *file, def ast.DeclDef, prefix string) []protocol.CompletionItem {
	switch def.Classify() {
	case ast.DefKindMessage:
		return completionItemsForMessage(file, prefix)
	case ast.DefKindService:
		return completionItemsForService()
	case ast.DefKindEnum:
		return []protocol.CompletionItem{
			keywordToCompletionItem(keyword.Option),
		}
	default:
		// TODO: limit where we return keywords.
		return topLevelCompletionItems()
	}
}

// completionItemsForMessage returns completion items for inside a message definition.
func completionItemsForMessage(file *file, prefix string) []protocol.CompletionItem {
	// Keywords for message body.
	messageKeywords := []keyword.Keyword{
		keyword.Message,
		keyword.Enum,
		keyword.Option,
		keyword.Oneof,
		keyword.Extensions,
		keyword.Reserved,
		keyword.Extend,
		// Field keywords
		keyword.Repeated,
		keyword.Optional,
		keyword.Required,
	}

	items := xslices.Map(messageKeywords, keywordToCompletionItem)

	// Add predeclared types (primitives).
	items = append(items, predeclaredTypeCompletionItems()...)

	// Add referenceable types (messages, enums) from this file and imports.
	items = append(items, referenceableTypeCompletionItems(file, prefix)...)

	return items
}

// completionItemsForService returns completion items for inside a service definition.
func completionItemsForService() []protocol.CompletionItem {
	serviceKeywords := []keyword.Keyword{
		keyword.RPC,
		keyword.Option,
	}
	return xslices.Map(serviceKeywords, keywordToCompletionItem)
}

// completionItemsForImport returns completion items for import declarations.
//
// Suggest all importable files.
func completionItemsForImport(file *file) []protocol.CompletionItem {
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

// predeclaredTypeCompletionItems returns completion items for predeclared (primitive) types.
func predeclaredTypeCompletionItems() []protocol.CompletionItem {
	predeclaredTypes := []keyword.Keyword{
		keyword.Int32,
		keyword.Int64,
		keyword.UInt32,
		keyword.UInt64,
		keyword.SInt32,
		keyword.SInt64,
		keyword.Fixed32,
		keyword.Fixed64,
		keyword.SFixed32,
		keyword.SFixed64,
		keyword.Float,
		keyword.Double,
		keyword.Bool,
		keyword.String,
		keyword.Bytes,
	}
	return xslices.Map(predeclaredTypes, func(kw keyword.Keyword) protocol.CompletionItem {
		return protocol.CompletionItem{
			Label: kw.String(),
			Kind:  protocol.CompletionItemKindTypeParameter,
		}
	})
}

// referenceableTypeCompletionItems returns completion items for user-defined types (messages, enums).
// The prefix parameter filters types and determines what to insert (suffix after prefix).
func referenceableTypeCompletionItems(file *file, prefix string) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Add types from the current file.
	for _, symbol := range file.referenceableSymbols {
		if !symbol.ir.Kind().IsType() {
			continue
		}

		// For same-file types, use short name.
		fullName := string(symbol.ir.FullName())
		shortName := symbol.ir.FullName().Name()

		// Filter by prefix if provided.
		if prefix != "" {
			// Check if full name or short name matches the prefix.
			if !strings.HasPrefix(fullName, prefix) && !strings.HasPrefix(shortName, prefix) {
				continue
			}
		}

		// Determine label and insert text.
		label := shortName
		insertText := shortName
		if prefix != "" {
			// If user typed a prefix, show full context but insert only suffix.
			if strings.HasPrefix(fullName, prefix) {
				label = fullName
				insertText = strings.TrimPrefix(fullName, prefix)
			} else if strings.HasPrefix(shortName, prefix) {
				insertText = strings.TrimPrefix(shortName, prefix)
			}
		}

		items = append(items, protocol.CompletionItem{
			Label:      label,
			InsertText: insertText,
			Kind:       protocol.CompletionItemKindTypeParameter,
		})
	}

	// Add types from imported files.
	for _, imported := range file.importToFile {
		for _, symbol := range imported.referenceableSymbols {
			if !symbol.ir.Kind().IsType() {
				continue
			}

			fullName := string(symbol.ir.FullName())
			shortName := symbol.ir.FullName().Name()

			// Filter by prefix if provided.
			if prefix != "" {
				// Check if full name or short name matches the prefix.
				if !strings.HasPrefix(fullName, prefix) && !strings.HasPrefix(shortName, prefix) {
					continue
				}
			}

			// Determine label and insert text based on package.
			label := shortName
			insertText := shortName
			if imported.ir.Package() != file.ir.Package() {
				// Different package - use fully qualified name.
				label = fullName
				insertText = fullName
			}

			// If user typed a prefix, adjust insert text to be just the suffix.
			if prefix != "" {
				if strings.HasPrefix(fullName, prefix) {
					label = fullName
					insertText = strings.TrimPrefix(fullName, prefix)
				} else if strings.HasPrefix(shortName, prefix) {
					insertText = strings.TrimPrefix(shortName, prefix)
				}
			}

			items = append(items, protocol.CompletionItem{
				Label:      label,
				InsertText: insertText,
				Kind:       protocol.CompletionItemKindTypeParameter,
			})
		}
	}

	return items
}

// topLevelCompletionItems returns completion items for top-level proto keywords.
func topLevelCompletionItems() []protocol.CompletionItem {
	keywords := []keyword.Keyword{
		keyword.Syntax,
		keyword.Edition,
		keyword.Import,
		keyword.Package,
		keyword.Message,
		keyword.Service,
		keyword.Option,
		keyword.Enum,
		keyword.Extend,
	}

	return xslices.Map(keywords, keywordToCompletionItem)
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
	for _, decl := range seq.All(body.Decls()) {
		if decl.IsZero() {
			continue
		}
		span := decl.Span()
		if span.IsZero() {
			continue
		}

		// Check if the position is within this declaration's span.
		protocolRange := reportSpanToProtocolRange(span)
		if comparePositions(position, protocolRange.Start) >= 0 &&
			comparePositions(position, protocolRange.End) <= 0 {
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
