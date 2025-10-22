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
		Kind:  protocol.CompletionItemKindSnippet,
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
				slog.String("def_kind", def.Classify().String()),
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
				slog.String("def_kind", def.Classify().String()),
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
	default:
		// If this is an invalid definition at the top level, return top-level keywords.
		if len(declPath) == 1 {
			file.lsp.logger.DebugContext(ctx, "completion: unknown definition at top level, returning top-level keywords")
			return slices.Collect(topLevelCompletionItems())
		}
		file.lsp.logger.DebugContext(ctx, "completion: unknown definition type (not at top level)")
		return nil
	}
}

// completionItemsForImport returns completion items for import declarations.
//
// Suggest all importable files.
func completionItemsForImport(ctx context.Context, file *file, declImport ast.DeclImport, position protocol.Position) []protocol.CompletionItem {
	file.lsp.logger.DebugContext(ctx, "completion: import declaration", slog.Int("importable_count", len(file.importToFile)))

	currentImportPathText := declImport.ImportPath().AsLiteral().Text()
	items := make([]protocol.CompletionItem, 0, len(file.importToFile))
	for importPath := range file.importToFile {
		suggestedImportPath := importPath
		if currentImportPathText != "" {
			// If there is already text in the import path, only suggest import paths with the given
			// prefix.
			currentImportPathWithoutSurroundingQuotes := strings.TrimSuffix(strings.TrimPrefix(currentImportPathText, `"`), `"`)
			if !strings.HasPrefix(importPath, currentImportPathWithoutSurroundingQuotes) {
				continue
			}
			suggestedImportPath = strings.TrimPrefix(suggestedImportPath, currentImportPathWithoutSurroundingQuotes)
		}
		var newText strings.Builder
		if afterImport := declImport.KeywordToken().Span().After(); len(afterImport) != 0 && afterImport[0] != ' ' {
			// If we have a literal `import` on the line and are asking for completion, we want to add
			// a space after the `import` keyword.
			_, _ = newText.WriteString(` `)
		}
		if !strings.HasPrefix(currentImportPathText, `"`) {
			_, _ = newText.WriteString(`"`)
		}
		newText.WriteString(suggestedImportPath)
		var additionalTextEdits []protocol.TextEdit
		if !strings.HasSuffix(currentImportPathText, `"`) {
			_, _ = newText.WriteString(`"`)
			// Only suggest a finishing `;` character if one doesn't already exist, and we're writing out
			// a `"` before it.
			if declImport.Semicolon().IsZero() {
				_, _ = newText.WriteString(";")
			}
		} else {
			// We're currently ending in a `"`, which means we're only going to suggest the file path.
			// e.g., if we're doing:
			//   import "‸"
			// We ought to also send along an AdditionalTextEdit for adding the `;` at the end of the
			// line, if it doesn't already exist.
			if declImport.Semicolon().IsZero() {
				additionalTextEdits = append(additionalTextEdits, protocol.TextEdit{
					NewText: ";",
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      position.Line,
							Character: 100, // End of line.
						},
						End: protocol.Position{
							Line:      position.Line,
							Character: 101,
						},
					},
				})
			}
		}
		items = append(items, protocol.CompletionItem{
			// Show the whole import path in the label.
			Label: importPath,
			Kind:  protocol.CompletionItemKindFile,
			TextEdit: &protocol.TextEdit{
				NewText: newText.String(),
				Range: protocol.Range{
					Start: position,
					End:   position,
				},
			},
			AdditionalTextEdits: additionalTextEdits,
		})
	}
	return items
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
