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
	"log/slog"
	"unicode/utf16"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/protocompile/experimental/ast"
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
		// No AST available, return empty completions.
		file.lsp.logger.Debug(
			"no AST found for completion",
			slog.String("file", file.uri.Filename()),
		)
		return nil
	}

	// Convert LSP position (UTF-16 code units) to byte offset.
	offset := protocolPositionToOffset(file.text, position)

	// Find the smallest AST declaration containing this offset.
	decl := getDeclForPosition(file.ir.AST().DeclBody, offset)

	file.lsp.logger.Debug(
		"decl for completion",
		slog.String("decl_kind", decl.Kind().String()),
		slog.Any("decl_span_start", decl.Span().StartLoc()),
		slog.Any("decl_span_end", decl.Span().EndLoc()),
		slog.Int("cursor_offset", offset),
	)

	// Check if cursor is in a name position (where unique identifiers are defined).
	// We should not provide completions for names.
	if isInNamePosition(decl, offset) {
		file.lsp.logger.Debug("cursor in name position, skipping completions")
		return nil
	}

	// Return context-aware completions based on the declaration type.
	return completionItemsForDecl(file, decl)
}

// isInNamePosition checks if the cursor offset is in a position where a unique identifier
// is being defined (like a message name, field name, etc.). We should not provide completions
// for these positions.
func isInNamePosition(decl ast.DeclAny, offset int) bool {
	if decl.IsZero() {
		return false
	}

	switch decl.Kind() {
	case ast.DeclKindDef:
		def := decl.AsDef()
		// Check if cursor is in the name of the definition.
		if !def.Name().IsZero() {
			nameSpan := def.Name().Span()
			if offset >= nameSpan.Start && offset <= nameSpan.End {
				return true
			}
		}

		// For fields, also check the field name (not the type).
		switch def.Classify() {
		case ast.DefKindField:
			field := def.AsField()
			if !field.Name.IsZero() {
				nameSpan := field.Name.Span()
				if offset >= nameSpan.Start && offset <= nameSpan.End {
					return true
				}
			}
		case ast.DefKindEnumValue:
			enumValue := def.AsEnumValue()
			if !enumValue.Name.IsZero() {
				nameSpan := enumValue.Name.Span()
				if offset >= nameSpan.Start && offset <= nameSpan.End {
					return true
				}
			}
		case ast.DefKindMethod:
			method := def.AsMethod()
			if !method.Name.IsZero() {
				nameSpan := method.Name.Span()
				if offset >= nameSpan.Start && offset <= nameSpan.End {
					return true
				}
			}
		}

	case ast.DeclKindSyntax:
		// Cursor in syntax declaration - likely in the syntax value, not a good place for completions.
		return true

	case ast.DeclKindPackage:
		// Cursor in package declaration - package name is unique.
		return true
	}

	return false
}

// completionItemsForDecl returns completion items based on the AST declaration at the cursor.
func completionItemsForDecl(file *file, decl ast.DeclAny) []protocol.CompletionItem {
	if decl.IsZero() {
		// No declaration found, return top-level keywords.
		return topLevelCompletionItems()
	}

	// Return context-specific completions based on declaration type.
	switch decl.Kind() {
	case ast.DeclKindDef:
		return completionItemsForDef(file, decl.AsDef())
	case ast.DeclKindSyntax:
		// Inside syntax declaration - could suggest syntax values.
		return nil
	case ast.DeclKindPackage:
		// Inside package declaration - could suggest package name based on path.
		return nil
	case ast.DeclKindImport:
		// Inside import declaration - could suggest importable files.
		return completionItemsForImport(file)
	default:
		// For other declaration types, fall back to top-level keywords.
		return topLevelCompletionItems()
	}
}

// completionItemsForDef returns completion items for definition declarations (message, enum, service, etc.).
func completionItemsForDef(file *file, def ast.DeclDef) []protocol.CompletionItem {
	switch def.Classify() {
	case ast.DefKindMessage:
		// Inside a message - suggest field keywords, types, and nested declarations.
		return completionItemsForMessage(file)
	case ast.DefKindService:
		// Inside a service - suggest rpc and option keywords.
		return completionItemsForService()
	case ast.DefKindEnum:
		// Inside an enum - suggest option keyword.
		return []protocol.CompletionItem{
			keywordToCompletionItem(keyword.Option),
		}
	default:
		// For other definitions, return top-level keywords.
		return topLevelCompletionItems()
	}
}

// completionItemsForMessage returns completion items for inside a message definition.
func completionItemsForMessage(file *file) []protocol.CompletionItem {
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
	items = append(items, referenceableTypeCompletionItems(file)...)

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
func completionItemsForImport(file *file) []protocol.CompletionItem {
	// Suggest importable file paths.
	paths := xslices.MapKeysToSlice(file.importToFile)
	items := make([]protocol.CompletionItem, 0, len(paths))
	for _, path := range paths {
		items = append(items, protocol.CompletionItem{
			Label: path,
			Kind:  protocol.CompletionItemKindFile,
		})
	}
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
func referenceableTypeCompletionItems(file *file) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Add types from the current file.
	for _, symbol := range file.referenceableSymbols {
		if symbol.ir.Kind().IsType() {
			items = append(items, protocol.CompletionItem{
				Label: symbol.ir.FullName().Name(),
				Kind:  protocol.CompletionItemKindTypeParameter,
			})
		}
	}

	// Add types from imported files.
	for _, imported := range file.importToFile {
		for _, symbol := range imported.referenceableSymbols {
			if symbol.ir.Kind().IsType() {
				// Use full name if from a different package.
				label := symbol.ir.FullName().Name()
				if imported.ir.Package() != file.ir.Package() {
					label = string(symbol.ir.FullName())
				}
				items = append(items, protocol.CompletionItem{
					Label: label,
					Kind:  protocol.CompletionItemKindTypeParameter,
				})
			}
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

// protocolPositionToOffset converts an LSP protocol position (0-based line, 0-based UTF-16 character)
// to a byte offset in the file text.
func protocolPositionToOffset(text string, position protocol.Position) int {
	targetLine := int(position.Line)
	targetChar := int(position.Character)

	currentLine := 0
	byteOffset := 0

	// Find the start of the target line.
	for i, r := range text {
		if currentLine == targetLine {
			byteOffset = i
			break
		}
		if r == '\n' {
			currentLine++
		}
	}

	// If we didn't find the target line, return end of file.
	if currentLine < targetLine {
		return len(text)
	}

	// Now count UTF-16 code units from the start of the line to find target character.
	utf16Col := 0
	for i, r := range text[byteOffset:] {
		if r == '\n' {
			// Reached end of line before target character.
			break
		}
		if utf16Col >= targetChar {
			return byteOffset + i
		}
		utf16Col += utf16.RuneLen(r)
	}

	// If we reached here, return the end of the line.
	for i, r := range text[byteOffset:] {
		if r == '\n' {
			return byteOffset + i
		}
	}

	// End of file.
	return len(text)
}

// getDeclForPosition finds the smallest AST declaration that contains the given byte offset.
func getDeclForPosition(body ast.DeclBody, offset int) ast.DeclAny {
	return findSmallestDecl(body, offset)
}

// findSmallestDecl recursively searches for the smallest declaration containing the offset.
func findSmallestDecl(body ast.DeclBody, offset int) ast.DeclAny {
	var smallest ast.DeclAny
	smallestSize := -1

	for _, decl := range seq.All(body.Decls()) {
		if decl.IsZero() {
			continue
		}

		span := decl.Span()
		if span.IsZero() {
			continue
		}

		// Check if the offset is within this declaration's span.
		if offset >= span.Start && offset <= span.End {
			size := span.End - span.Start
			if smallestSize == -1 || size < smallestSize {
				smallest = decl
				smallestSize = size
			}

			// If this is a definition with a body, search recursively.
			if decl.Kind() == ast.DeclKindDef && !decl.AsDef().Body().IsZero() {
				child := findSmallestDecl(decl.AsDef().Body(), offset)
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
