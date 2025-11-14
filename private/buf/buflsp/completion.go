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
	"github.com/bufbuild/protocompile/experimental/id"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
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
	if file.ir.AST() == nil {
		file.lsp.logger.DebugContext(
			ctx,
			"no AST found for completion",
			slog.String("file", file.uri.Filename()),
		)
		return nil
	}

	// This grabs the contents of the file as the top-level [ast.DeclBody], see [ast.File].Decls()
	// for reference.
	offset := positionToOffset(file, position)
	declPath := getDeclForOffset(id.Wrap(file.ir.AST(), id.ID[ast.DeclBody](1)), offset)
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

	offset := positionToOffset(file, position)
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
	offset := positionToOffset(file, position)
	inBody := offsetInSpan(offset, def.Body().Span()) == 0

	// Parent declaration determines child completions.
	var parentDef ast.DeclDef
	if len(declPath) >= 2 {
		parentDef = declPath[len(declPath)-2].AsDef()
	}
	if inBody {
		parentDef = def
		def = ast.DeclDef{} // Mark current def as invalid.
	}

	file.lsp.logger.DebugContext(
		ctx,
		"completion: definition",
		slog.String("keyword", def.Keyword().String()),
		slog.String("type", def.Type().Span().Text()),
		slog.String("name", def.Name().Span().Text()),
		slog.String("kind", def.Classify().String()),
		slog.String("parent_kind", parentDef.Classify().String()),
		slog.Bool("in_body", inBody),
		slog.Int("path_depth", len(declPath)),
	)

	switch def.Classify() {
	case ast.DefKindMessage, ast.DefKindService, ast.DefKindEnum, ast.DefKindGroup:
		// Ignore these kinds as this will be a completion for the name of the declaration.
		return nil
	case ast.DefKindOption:
		return completionItemsForOptions(ctx, file, parentDef, def, offset)
	case ast.DefKindField, ast.DefKindMethod, ast.DefKindInvalid:
		// Use these kinds as completion starts.
		// An invalid kind is caused from partial values, which may be any kind.
	default:
		file.lsp.logger.DebugContext(ctx, "completion: unknown definition type", slog.String("kind", def.Classify().String()))
		return nil
	}

	// This checks for options declared within the declaration.
	if offsetInSpan(offset, def.Options().Span()) == 0 {
		return completionItemsForCompactOptions(ctx, file, def, offset)
	}

	tokenSpan := extractAroundOffset(file, offset, isTokenType, isTokenType)
	tokenPrefix, tokenSuffix := splitSpan(tokenSpan, offset)

	// Extract the full type and name declaration to compute the heuristics for completion.
	// Use the token stream to capture invalid declarations.
	beforeCount, afterCount := 0, 0
	hasBeforeGap, hasAfterGap := false, false
	hasStart := false // Start is a newline or open parenthesis for the start of a definition
	hasTypeModifier := false
	hasDeclaration := false
	typeSpan := extractAroundOffset(file, offset,
		func(tok token.Token) bool {
			if isTokenTypeDelimiter(tok) {
				hasStart = true
				return false
			}
			if hasBeforeGap {
				beforeCount += 1
				hasBeforeGap = false
				if kw := tok.Keyword(); kw != keyword.Unknown {
					_, isDeclaration := declarationSet[tok.Keyword()]
					hasDeclaration = hasDeclaration || isDeclaration
					_, isFieldModifier := typeModifierSet[tok.Keyword()]
					hasTypeModifier = hasTypeModifier || isFieldModifier
				}
			}
			if isTokenSpace(tok) {
				hasBeforeGap = true
				return true
			}
			return isTokenType(tok) || isTokenParen(tok)
		},
		func(tok token.Token) bool {
			if hasAfterGap {
				afterCount += 1
				hasAfterGap = false
			}
			if isTokenTypeDelimiter(tok) {
				return false
			}
			if isTokenSpace(tok) {
				hasAfterGap = true
				return true
			}
			return isTokenType(tok) || isTokenParen(tok)
		},
	)
	typePrefix, typeSuffix := splitSpan(typeSpan, offset)
	file.lsp.logger.DebugContext(
		ctx, "completion: definition value",
		slog.String("token", tokenSpan.Text()),
		slog.String("token_prefix", tokenPrefix),
		slog.String("token_suffix", tokenSuffix),
		slog.String("type", typeSpan.Text()),
		slog.String("type_prefix", typePrefix),
		slog.String("type_suffix", typeSuffix),
		slog.Int("before_count", beforeCount),
		slog.Int("after_count", afterCount),
		slog.Bool("has_start", hasStart),
		slog.Bool("has_field_modifier", hasTypeModifier),
		slog.Bool("has_declaration", hasDeclaration),
	)
	if !hasStart {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: ignoring definition type unable to find start",
			slog.String("kind", def.Classify().String()),
		)
		return nil
	}

	// This corrects for invalid syntax on option declarations with a corrupt declaration.
	// For example "option (extension)." will fail to be parsed as a declaration.
	if def.Keyword() == keyword.Option ||
		strings.HasPrefix(typeSpan.Text(), keyword.Option.String()+" ") ||
		strings.HasPrefix(def.Type().Span().Text(), keyword.Option.String()+" ") {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: identified option from declaration",
			slog.String("kind", def.Classify().String()),
			slog.String("text", typeSpan.Text()),
		)
		return completionItemsForOptions(ctx, file, parentDef, def, offset)
	}

	// If at the top level, and on the first item, return top level keywords.
	if parentDef.IsZero() {
		showKeywords := beforeCount == 0
		if showKeywords {
			file.lsp.logger.DebugContext(ctx, "completion: definition returning top-level keywords")
			return slices.Collect(keywordToCompletionItem(
				topLevelKeywords(),
				protocol.CompletionItemKindKeyword,
				tokenSpan,
				offset,
			))
		}
		return nil // unknown
	}

	var iters []iter.Seq[protocol.CompletionItem]
	switch parentDef.Classify() {
	case ast.DefKindMessage:
		// Limit completions based on the following heuristics:
		// - Show keywords for the first values
		// - Show types if no type declaration and at first, or second position with field modifier.
		showKeywords := beforeCount == 0
		showTypes := !hasDeclaration && (beforeCount == 0 || (hasTypeModifier && beforeCount == 1))
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
					true, // Allow enums.
				),
			)
		}
	case ast.DefKindService:
		// Method types are only shown within args of the method definition.
		// Use the type to handle invalid defs.
		defMethod := def.AsMethod()
		isRPC := defMethod.Keyword.Keyword() == keyword.RPC ||
			strings.HasPrefix(typeSpan.Text(), keyword.RPC.String())

		// Limit complitions based on the following heuristics:
		// - Show keywords for the first values.
		// - Show return keyword if an RPC method and at the third position.
		// - Show types if an RPC method and within the Input or Output args.
		showKeywords := beforeCount == 0
		showReturnKeyword := isRPC &&
			(beforeCount == 2 || offsetInSpan(offset, defMethod.Signature.Returns().Span()) == 0)
		showTypes := isRPC && !hasDeclaration &&
			(beforeCount == 0 || (hasTypeModifier && beforeCount == 1)) &&
			(offsetInSpan(offset, defMethod.Signature.Inputs().Span()) == 0 ||
				offsetInSpan(offset, defMethod.Signature.Outputs().Span()) == 0)

		if showKeywords {
			// If both showKeywords and showTypes is set, we are in a services method args.
			if showTypes {
				iters = append(iters,
					keywordToCompletionItem(
						methodArgLevelKeywords(),
						protocol.CompletionItemKindKeyword,
						tokenSpan,
						offset,
					),
				)
			} else {
				iters = append(iters,
					keywordToCompletionItem(
						serviceLevelKeywords(),
						protocol.CompletionItemKindKeyword,
						tokenSpan,
						offset,
					),
				)
			}
		} else if showReturnKeyword {
			iters = append(iters,
				keywordToCompletionItem(
					serviceReturnKeyword(),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
			)
		}
		if showTypes {
			iters = append(iters,
				typeReferencesToCompletionItems(
					file,
					"", // No parent type within a service declaration.
					tokenSpan,
					offset,
					false, // Disallow enums.
				),
			)
		}
	case ast.DefKindMethod:
		showKeywords := beforeCount == 0
		if showKeywords {
			iters = append(iters,
				keywordToCompletionItem(
					optionKeywords(),
					protocol.CompletionItemKindKeyword,
					tokenSpan,
					offset,
				),
			)
		}
	case ast.DefKindEnum:
		showKeywords := beforeCount == 0
		if showKeywords {
			iters = append(iters,
				keywordToCompletionItem(
					optionKeywords(),
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

	offset := positionToOffset(file, position)
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

// completionItemsForOptions returns completion items for an option declaration.
func completionItemsForOptions(
	ctx context.Context,
	file *file,
	parentDef ast.DeclDef,
	def ast.DeclDef,
	offset int,
) []protocol.CompletionItem {
	file.lsp.logger.DebugContext(ctx, "completion: options", slog.Int("offset", offset))
	optionSpan, optionSpanParts := parseOptionSpan(file, offset)
	if optionSpan.IsZero() {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: ignoring compact option unable to parse declaration",
			slog.String("kind", def.Classify().String()),
		)
		return nil
	}
	optionsTypeName, targetKind := defKindToOptionType(parentDef.Classify())
	if targetKind == ir.OptionTargetInvalid {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: unknown def kind for options",
			slog.String("kind", parentDef.Classify().String()),
		)
		return nil
	}
	// Find the options message type in the workspace by looking through all types.
	var optionsType ir.Type
	for _, importedFile := range file.workspace.PathToFile() {
		irSymbol := importedFile.ir.FindSymbol(optionsTypeName)
		optionsType = irSymbol.AsType()
		if !optionsType.IsZero() {
			break
		}
	}
	if optionsType.IsZero() {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: could not find options type in workspace",
			slog.String("options_type_name", string(optionsTypeName)),
		)
		return nil
	}
	// Complete options within the value or the path.
	if offsetInSpan(offset, def.Value().Span()) == 0 {
		var parentType ir.Type
		for irType := range seq.Values(file.ir.AllTypes()) {
			if irType.AST().Span() == parentDef.Span() {
				parentType = irType
				break
			}
		}
		optionType, isOptionType := getOptionValueType(file, ctx, parentType.Options(), offset)
		if !isOptionType {
			file.lsp.logger.DebugContext(
				ctx,
				"completion: could not find options type within value",
			)
			return nil
		}
		return slices.Collect(
			messageFieldCompletionItems(file, optionType, optionSpan, offset, true),
		)
	}
	return slices.Collect(
		optionNamesToCompletionItems(file, optionSpanParts, offset, optionsType, targetKind),
	)
}

// completionItemsForCompactOptions returns completion items for options within an options block.
func completionItemsForCompactOptions(
	ctx context.Context,
	file *file,
	def ast.DeclDef,
	offset int,
) []protocol.CompletionItem {
	file.lsp.logger.DebugContext(ctx, "completion: compact options", slog.Int("offset", offset))

	optionSpan, optionSpanParts := parseOptionSpan(file, offset)
	if optionSpan.IsZero() {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: ignoring compact option unable to parse declaration",
			slog.String("kind", def.Classify().String()),
		)
		return nil
	}
	// Find the parent containing type for the definition.
	optionsTypeName, targetKind := defKindToOptionType(def.Classify())
	if targetKind == ir.OptionTargetInvalid {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: unknown def kind for options",
			slog.String("kind", def.Classify().String()),
		)
		return nil
	}
	// Search for the option message in the IR.
	optionMessage := defToOptionMessage(file, def)
	if optionMessage.IsZero() {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: unable to find containing option message",
			slog.String("kind", def.Classify().String()),
		)
		return nil
	}
	// Check the position within the option value.
	if optionValueType, isOptionValue := getOptionValueType(file, ctx, optionMessage, offset); isOptionValue {
		if optionValueType.IsZero() {
			file.lsp.logger.DebugContext(ctx, "completion: unknown option value type")
			return nil
		}
		// Generate completions for fields in the options value at this position.
		return slices.Collect(messageFieldCompletionItems(file, optionValueType, optionSpan, offset, true))
	}
	// Find the options message type in the workspace by looking through all types.
	var optionsType ir.Type
	for _, importedFile := range file.workspace.PathToFile() {
		irSymbol := importedFile.ir.FindSymbol(optionsTypeName)
		optionsType = irSymbol.AsType()
		if !optionsType.IsZero() {
			break
		}
	}
	if optionsType.IsZero() {
		file.lsp.logger.DebugContext(
			ctx,
			"completion: could not find options type in workspace",
			slog.String("options_type_name", string(optionsTypeName)),
		)
		return nil
	}
	return slices.Collect(
		optionNamesToCompletionItems(file, optionSpanParts, offset, optionsType, targetKind),
	)
}

// declarationSet is the set of keywords that starts a declaration.
var declarationSet = func() map[keyword.Keyword]struct{} {
	m := make(map[keyword.Keyword]struct{})
	for keyword := range joinSequences(
		topLevelKeywords(),
		messageLevelKeywords(true),
		serviceLevelKeywords(),
	) {
		m[keyword] = struct{}{}
	}
	return m
}()

// typeModifierSet is the set of keywords for type modifiers.
var typeModifierSet = func() map[keyword.Keyword]struct{} {
	m := make(map[keyword.Keyword]struct{})
	for keyword := range joinSequences(
		messageLevelFieldKeywords(),
		methodArgLevelKeywords(),
	) {
		m[keyword] = struct{}{}
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

// serviceReturnKeyword returns keyword for service "return" value.
func serviceReturnKeyword() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Returns)
	}
}

// methodArgLevelKeywords returns keyword for methods.
func methodArgLevelKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Stream)
	}
}

// optionKeywords returns the option keywords for methods and enums.
func optionKeywords() iter.Seq[keyword.Keyword] {
	return func(yield func(keyword.Keyword) bool) {
		_ = yield(keyword.Option)
	}
}

// keywordToCompletionItem converts a keyword to a completion item.
func keywordToCompletionItem(
	keywords iter.Seq[keyword.Keyword],
	kind protocol.CompletionItemKind,
	span source.Span,
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
	span source.Span,
	offset int,
	allowEnums bool,
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
				if !allowEnums {
					continue
				}
				kind = protocol.CompletionItemKindEnum
			default:
				continue // Unsupported kind, skip it.
			}
			label := string(symbol.ir.FullName())
			if len(parentFullName) > 0 && strings.HasPrefix(label, parentPrefix) {
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
			symbolFile := symbol.ir.Context().Path()
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

// optionToCompletionItems returns completion items for options.
func optionToCompletionItems(
	current *file,
	span source.Span,
	offset int,
	optionsType ir.Type,
	targetKind ir.OptionTarget,
) iter.Seq[protocol.CompletionItem] {
	prefix, _ := splitSpan(span, offset)
	return func(yield func(protocol.CompletionItem) bool) {
		// Add standard option fields (non-extensions) from the options type
		// Filter by option targets to only show options that can be applied to this symbol kind
		for member := range seq.Values(optionsType.Members()) {
			if member.IsExtension() {
				continue
			}
			// Check if this option can target this symbol kind
			if !member.CanTarget(targetKind) {
				current.lsp.logger.Debug(
					"completion: skipping option due to target mismatch",
					slog.String("option", member.Name()),
					slog.String("target_kind", targetKind.String()),
				)
				continue
			}
			label := member.Name()
			if !strings.HasPrefix(label, prefix) {
				continue
			}
			var isDeprecated bool
			if _, ok := member.Deprecated().AsBool(); ok {
				isDeprecated = true
			}
			var detail string
			fieldType := member.Element()
			if !fieldType.IsZero() {
				if fieldType.IsPredeclared() {
					detail = fieldType.Name()
				} else {
					detail = string(fieldType.FullName())
				}
			}
			item := protocol.CompletionItem{
				Label: label,
				Kind:  protocol.CompletionItemKindField,
				TextEdit: &protocol.TextEdit{
					Range:   reportSpanToProtocolRange(span),
					NewText: label,
				},
				Deprecated:    isDeprecated,
				Documentation: irMemberDoc(member),
				Detail:        detail,
			}
			if !yield(item) {
				break
			}
		}
	}
}

// extensionToCompletionItems returns completion items for options from extensions.
func extensionToCompletionItems(
	current *file,
	span source.Span,
	offset int,
	optionsType ir.Type,
	targetKind ir.OptionTarget,
) iter.Seq[protocol.CompletionItem] {
	prefix, _ := splitSpan(span, offset)
	return func(yield func(protocol.CompletionItem) bool) {
		// Find all extensions in the workspace that extend the options type.
		for _, importedFile := range current.workspace.PathToFile() {
			for extension := range seq.Values(importedFile.ir.AllExtensions()) {
				if !extension.IsExtension() {
					continue
				}
				// Ensure the extension extends the type we are looking for.
				container := extension.Container()
				if container.IsZero() {
					continue
				}
				if container.FullName() != optionsType.FullName() {
					continue
				}
				// Check if this extension can target this symbol kind
				if !extension.CanTarget(targetKind) {
					current.lsp.logger.Debug(
						"completion: skipping extension due to target mismatch",
						slog.String("extension", string(extension.FullName())),
						slog.String("target_kind", targetKind.String()),
					)
					continue
				}
				// Extensions are referenced with parentheses, e.g., (my.extension)
				label := "(" + string(extension.FullName()) + ")"
				if !strings.HasPrefix(label, prefix) {
					continue
				}
				var isDeprecated bool
				if _, ok := extension.Deprecated().AsBool(); ok {
					isDeprecated = true
				}
				var detail string
				fieldType := extension.Element()
				if !fieldType.IsZero() {
					if fieldType.IsPredeclared() {
						detail = fieldType.Name()
					} else {
						detail = string(fieldType.FullName())
					}
				}
				item := protocol.CompletionItem{
					Label: label,
					Kind:  protocol.CompletionItemKindProperty,
					TextEdit: &protocol.TextEdit{
						Range:   reportSpanToProtocolRange(span),
						NewText: label,
					},
					Deprecated:    isDeprecated,
					Documentation: irMemberDoc(extension),
					Detail:        detail,
				}
				if !yield(item) {
					break
				}
			}
		}
	}
}

// optionNamesToCompletionItems completes nested option field paths like: option field.nested.sub = value
// It walks through the path segments to find the current message type and suggests valid fields.
func optionNamesToCompletionItems(
	current *file,
	pathSpans []source.Span,
	offset int,
	optionType ir.Type,
	targetKind ir.OptionTarget,
) iter.Seq[protocol.CompletionItem] {
	if len(pathSpans) == 0 {
		return func(yield func(protocol.CompletionItem) bool) {}
	}

	rootSpan := pathSpans[0]
	rootText := rootSpan.Text()
	isExtension := strings.HasPrefix(rootText, "(") && strings.HasSuffix(rootText, ")")
	rootFieldName := rootText
	if isExtension {
		// Remove parentheses: "(foo.bar.Extension)" -> "foo.bar.Extension"
		rootFieldName = rootText[1 : len(rootText)-1]
	}
	if len(pathSpans) == 1 {
		if isExtension {
			return extensionToCompletionItems(current, rootSpan, offset, optionType, targetKind)
		}
		return optionToCompletionItems(current, rootSpan, offset, optionType, targetKind)
	}
	tokenSpan := pathSpans[len(pathSpans)-1]
	pathSpans = pathSpans[1 : len(pathSpans)-1]

	var currentField ir.Member
	if isExtension {
		// Search for extension across all files in workspace
		for _, importedFile := range current.workspace.PathToFile() {
			for extension := range seq.Values(importedFile.ir.AllExtensions()) {
				if !extension.IsExtension() {
					continue
				}
				container := extension.Container()
				if container.IsZero() || container.FullName() != optionType.FullName() {
					continue
				}
				if string(extension.FullName()) == rootFieldName && extension.CanTarget(targetKind) {
					currentField = extension
					break
				}
			}
			if !currentField.IsZero() {
				break
			}
		}
	} else {
		// Search for regular field in optionsType members.
		for member := range seq.Values(optionType.Members()) {
			if member.IsExtension() {
				continue
			}
			if member.Name() == rootFieldName && member.CanTarget(targetKind) {
				currentField = member
				break
			}
		}
	}
	if currentField.IsZero() {
		current.lsp.logger.Debug(
			"completion: could not find root field",
			slog.String("root", rootFieldName),
		)
	}
	// Walk through each path segment to navigate nested messages.
	currentType := currentField.Element()
	for i, pathSegment := range pathSpans {
		segmentName := pathSegment.Text()
		current.lsp.logger.Debug(
			"completion: walking path segment",
			slog.Int("index", i),
			slog.String("segment", segmentName),
			slog.String("current_type", string(currentType.FullName())),
		)
		// The current field should be a message type to have nested fields
		if !currentType.IsMessage() || currentType.IsZero() {
			current.lsp.logger.Debug(
				"completion: current type is not a message, cannot continue path",
				slog.String("segment", segmentName),
			)
			break
		}
		// Find the field with this name in the current message
		var found bool
		for member := range seq.Values(currentType.Members()) {
			if member.Name() == segmentName {
				currentField = member
				currentType = member.Element()
				found = true
				break
			}
		}
		if !found {
			current.lsp.logger.Debug(
				"completion: could not find field in path",
				slog.String("segment", segmentName),
				slog.String("type", string(currentType.FullName())),
			)
			currentType = ir.Type{}
			break
		}
	}
	return messageFieldCompletionItems(current, currentType, tokenSpan, offset, false)
}

// messageFieldCompletionItems generates completion items for fields in a message type.
// This is used for both option field paths and option value field completion.
func messageFieldCompletionItems(
	current *file,
	messageType ir.Type,
	tokenSpan source.Span,
	offset int,
	isValueType bool,
) iter.Seq[protocol.CompletionItem] {
	return func(yield func(protocol.CompletionItem) bool) {
		if messageType.IsZero() || !messageType.IsMessage() {
			current.lsp.logger.Debug(
				"completion: final type is not a message, no completions",
				slog.String("type", string(messageType.FullName())),
			)
			return
		}

		prefix, _ := splitSpan(tokenSpan, offset)
		editRange := reportSpanToProtocolRange(tokenSpan)

		current.lsp.logger.Debug(
			"completion: generating message field completions",
			slog.String("type", string(messageType.FullName())),
			slog.String("prefix", prefix),
		)

		// Generate completion items for all fields in the message
		for member := range seq.Values(messageType.Members()) {
			label := member.Name()
			if !strings.HasPrefix(label, prefix) {
				continue
			}
			if member.IsExtension() {
				label = "(" + label + ")"
			}
			newText := label
			if isValueType {
				newText += ":"
			}
			var isDeprecated bool
			if _, ok := member.Deprecated().AsBool(); ok {
				isDeprecated = true
			}
			// Add detail string based on field type for better context
			var detail string
			fieldType := member.Element()
			if !fieldType.IsZero() {
				if fieldType.IsPredeclared() {
					detail = fieldType.Name()
				} else {
					detail = string(fieldType.FullName())
				}
			}
			item := protocol.CompletionItem{
				Label: label,
				Kind:  protocol.CompletionItemKindField,
				TextEdit: &protocol.TextEdit{
					Range:   editRange,
					NewText: newText,
				},
				Deprecated:    isDeprecated,
				Documentation: irMemberDoc(member),
				Detail:        detail,
			}
			if !yield(item) {
				return
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

// getDeclForOffset finds the path of AST declarations from parent to smallest that contains the given offset.
// Returns a slice where [0] is the top-level declaration and [len-1] is the smallest/innermost declaration.
// Returns nil if no declaration contains the offset.
func getDeclForOffset(body ast.DeclBody, offset int) []ast.DeclAny {
	return getDeclForOffsetHelper(body, offset, nil)
}

// getDeclForOffsetHelper is the recursive helper for getDeclForOffset.
func getDeclForOffsetHelper(body ast.DeclBody, offset int, path []ast.DeclAny) []ast.DeclAny {
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
		if offsetInSpan(offset, span) == 0 {
			// Build the new path including this declaration.
			newPath := append(append([]ast.DeclAny(nil), path...), decl)
			size := span.End - span.Start
			if smallestSize == -1 || size < smallestSize {
				bestPath = newPath
				smallestSize = size
			}

			// If this is a definition with a body, search recursively.
			if decl.Kind() == ast.DeclKindDef && !decl.AsDef().Body().IsZero() {
				childPath := getDeclForOffsetHelper(decl.AsDef().Body(), offset, newPath)
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

// parseOptionSpan returns the span associated with the option declaration and
// the fields up until the offset. This handles invalid declarations from
// partial syntax.
func parseOptionSpan(file *file, offset int) (source.Span, []source.Span) {
	hasStart, hasGap := false, false
	var tokens []token.Token
	typeSpan := extractAroundOffset(
		file, offset,
		func(tok token.Token) bool {
			// A gap is only allowed if the preceding token is the "option" token.
			// This is the start of an option declaration.
			if hasGap {
				hasStart = tok.Keyword() == keyword.Option
				return false
			}
			if isTokenSpace(tok) {
				hasGap = true
				return true
			}
			if isTokenTypeDelimiter(tok) {
				hasStart = true
				return false
			}
			tokens = append(tokens, tok)
			return isTokenType(tok) || isTokenParen(tok)
		},
		isTokenType,
	)
	if !hasStart {
		return source.Span{}, nil
	}
	// If no tokens were found, return an empty span at the offset.
	if len(tokens) == 0 {
		emptySpan := source.Span{File: file.file, Start: offset, End: offset}
		return emptySpan, []source.Span{emptySpan}
	}
	slices.Reverse(tokens)
	if typeSpan.Start > 0 && file.file.Text()[typeSpan.Start-1] == '(' {
		// Within parens, caputure the type as the full path. One root span.
		// The offset is within the child span. We expand the span to capture the
		// extension.
		typeSpan.Start -= 1
		if strings.HasPrefix(file.file.Text()[typeSpan.End:], ")") {
			typeSpan.End += 1
		}
		return typeSpan, []source.Span{typeSpan}
	}
	pathSpans := []source.Span{tokens[0].Span()}
	for i := 1; i < len(tokens)-1; i += 2 {
		dotToken := tokens[i]
		identToken := tokens[i+1]
		if dotToken.Text() != "." || identToken.Kind() != token.Ident {
			return source.Span{}, nil
		}
		pathSpans = append(pathSpans, identToken.Span())
	}
	// Append an empty span on trailing "." token at it's position.
	if (len(tokens)-1)%2 != 0 {
		lastToken := tokens[len(tokens)-1]
		if lastToken.Kind() == token.Punct && lastToken.Text() == "." {
			lastSpan := lastToken.Span()
			lastSpan.Start = lastSpan.End
			pathSpans = append(pathSpans, lastSpan)
		}
	}
	return typeSpan, pathSpans
}

// defKindToOptionType returns the option type associated with the decl.
func defKindToOptionType(kind ast.DefKind) (ir.FullName, ir.OptionTarget) {
	switch kind {
	case ast.DefKindMessage:
		return "google.protobuf.MessageOptions", ir.OptionTargetMessage
	case ast.DefKindEnum:
		return "google.protobuf.EnumOptions", ir.OptionTargetEnum
	case ast.DefKindField:
		return "google.protobuf.FieldOptions", ir.OptionTargetField
	case ast.DefKindEnumValue:
		return "google.protobuf.EnumValueOptions", ir.OptionTargetEnumValue
	case ast.DefKindService:
		return "google.protobuf.ServiceOptions", ir.OptionTargetService
	case ast.DefKindMethod:
		return "google.protobuf.MethodOptions", ir.OptionTargetMethod
	case ast.DefKindOneof:
		return "google.protobuf.OneofOptions", ir.OptionTargetOneof
	default:
		return "", ir.OptionTargetInvalid
	}
}

// defToOptionMessage returns the option message associated with the decl.
func defToOptionMessage(file *file, def ast.DeclDef) ir.MessageValue {
	defSpan := def.Span()
	switch kind := def.Classify(); kind {
	case ast.DefKindMessage:
		for irType := range seq.Values(file.ir.AllTypes()) {
			if irType.AST().Span() == defSpan {
				return irType.Options()
			}
		}
	case ast.DefKindEnum:
		for irType := range seq.Values(file.ir.AllTypes()) {
			if irType.AST().Span() == defSpan {
				return irType.Options()
			}
		}
	case ast.DefKindField:
		for irType := range seq.Values(file.ir.AllTypes()) {
			for member := range seq.Values(irType.Members()) {
				if member.AST().Span() == defSpan {
					return member.Options()
				}
			}
		}
		for extension := range seq.Values(file.ir.AllExtensions()) {
			if extension.AST().Span() == defSpan {
				return extension.Options()
			}
		}
	case ast.DefKindEnumValue:
		for irType := range seq.Values(file.ir.AllTypes()) {
			for member := range seq.Values(irType.Members()) {
				if member.AST().Span() == defSpan {
					return member.Options()
				}
			}
		}
	case ast.DefKindService:
		for service := range seq.Values(file.ir.Services()) {
			if service.AST().Span() == defSpan {
				return service.Options()
			}
		}
	case ast.DefKindMethod:
		for service := range seq.Values(file.ir.Services()) {
			for method := range seq.Values(service.Methods()) {
				if method.AST().Span() == defSpan {
					return method.Options()
				}
			}
		}
	case ast.DefKindOneof:
		for irType := range seq.Values(file.ir.AllTypes()) {
			for oneof := range seq.Values(irType.Oneofs()) {
				if oneof.AST().Span() == defSpan {
					return oneof.Options()
				}
			}
		}
	}
	return ir.MessageValue{}
}

func isTokenType(tok token.Token) bool {
	kind := tok.Kind()
	return kind == token.Ident || (kind == token.Punct && tok.Text() == ".")
}

func isTokenSpace(tok token.Token) bool {
	return tok.Kind() == token.Space && strings.IndexByte(tok.Text(), '\n') == -1
}

func isTokenParen(tok token.Token) bool {
	return tok.Kind() == token.Punct &&
		(strings.HasPrefix(tok.Text(), "(") ||
			strings.HasSuffix(tok.Text(), ")"))
}

func isTokenTypeDelimiter(tok token.Token) bool {
	kind := tok.Kind()
	return (kind == token.Unrecognized && tok.IsZero()) ||
		(kind == token.Space && strings.IndexByte(tok.Text(), '\n') != -1) ||
		(kind == token.Comment)
}

// extractAroundOffset extracts the value around the offset by querying the token stream.
func extractAroundOffset(file *file, offset int, isTokenBefore, isTokenAfter func(token.Token) bool) source.Span {
	if file.ir.AST() == nil {
		return source.Span{}
	}
	stream := file.ir.AST().Stream()
	if stream == nil {
		return source.Span{}
	}
	before, after := stream.Around(offset)
	if before.IsZero() && after.IsZero() {
		return source.Span{}
	}

	span := source.Span{
		File:  file.file,
		Start: offset,
		End:   offset,
	}
	if !before.IsZero() && isTokenBefore != nil {
		var firstToken token.Token
		for cursor := token.NewCursorAt(before); isTokenBefore(before); before = cursor.PrevSkippable() {
			firstToken = before
			span.Start = firstToken.Span().Start
		}
	}
	if !after.IsZero() && isTokenAfter != nil {
		var lastToken token.Token
		for cursor := token.NewCursorAt(after); isTokenAfter(after); after = cursor.NextSkippable() {
			lastToken = after
		}
		if !lastToken.IsZero() {
			span.End = lastToken.Span().End
		}
	}
	return span
}

func splitSpan(span source.Span, offset int) (prefix string, suffix string) {
	if offsetInSpan(offset, span) != 0 {
		return "", ""
	}
	index := offset - span.Start
	text := span.Text()
	return text[:index], text[index:]
}

func offsetInSpan(offset int, span source.Span) int {
	if offset < span.Start {
		return -1
	} else if offset > span.End {
		// End is inclusive for completions	_
		return 1
	}
	return 0
}

// positionToOffset returns the offset from the protocol position.
func positionToOffset(file *file, position protocol.Position) int {
	positionLocation := file.file.InverseLocation(int(position.Line)+1, int(position.Character)+1, positionalEncoding)
	return positionLocation.Offset
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

// getOptionValueType finds the completion position within nested option message values.
// It returns the message type containing the current offset if found.
func getOptionValueType(file *file, ctx context.Context, optionValue ir.MessageValue, offset int) (
	optionType ir.Type,
	isOptionValue bool,
) {
	for field := range optionValue.Fields() {
		keySpan := field.KeyAST().Span()
		valueSpan := field.ValueAST().Span()
		for element := range seq.Values(field.Elements()) {
			if msg := element.AsMessage(); !msg.IsZero() {
				if optionType, isOptionValue := getOptionValueType(file, ctx, msg, offset); isOptionValue {
					return optionType, isOptionValue
				}
				// Option value must be different to the key, otherwise in type declaration.
				isOptionValue = isOptionValue ||
					(offsetInSpan(offset, element.AST().Span()) == 0 && keySpan != valueSpan)
				if isOptionValue {
					return msg.Type(), isOptionValue
				}
			}
		}
	}
	return ir.Type{}, false
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
