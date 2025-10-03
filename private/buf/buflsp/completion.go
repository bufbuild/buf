package buflsp

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoversion"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ast/syntax"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"go.lsp.dev/protocol"
)

var topLevelKeywords = []keyword.Keyword{
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

func getCompletionItems(
	ctx context.Context,
	file *file,
	position protocol.Position,
) []protocol.CompletionItem {
	if file.ir.AST().IsZero() {
		// No AST available, we just return the top-level keywords
		// This shouldn't really happen, so we log a warning.
		file.lsp.logger.Debug(
			"no AST found for completion",
			slog.String("file", file.uri.Filename()),
		)
		return topLevelCompletionItems()
	}

	decl := getDeclForPosition(
		file.ir.AST().DeclBody,
		int(position.Line)+1,      // Use 1-indexed line number
		int(position.Character)+1, // Use 1-indexed character count
		0,
		math.MaxInt,
	)

	file.lsp.logger.Debug(
		"decl for completion",
		slog.String("decl_kind", decl.Kind().String()),
		slog.Any("decl_span_start", decl.Span().StartLoc()),
		slog.Any("decl_span_end", decl.Span().EndLoc()),
		slog.Int("cursor_line", int(position.Line)),
		slog.Int("cursor_char", int(position.Character)),
	)

	return completionItemsForDecl(file, decl, position)
}

// TODO: resolutiuon behaviour:
func resolveCompletionItem(
	ctx context.Context,
	completionItem *protocol.CompletionItem,
) (*protocol.CompletionItem, error) {
	return completionItem, nil
}

// getDeclForPosition takes the AST body and a cursor position with a 1-indexed line number
// and character, and returns the "smallest" [ast.DeclAny] that intersects with the cursor.
func getDeclForPosition(body ast.DeclBody, line, char, min, max int) ast.DeclAny {
	var ret ast.DeclAny
	for _, decl := range seq.All(body.Decls()) {
		if decl.Span().StartLoc().Line > min &&
			decl.Span().StartLoc().Line <= line &&
			decl.Span().EndLoc().Line < max &&
			decl.Span().EndLoc().Line >= int(line) {
			if decl.Span().EndLoc().Line == int(line) &&
				decl.Span().EndLoc().Column < int(char) {
				continue
			}
			min = decl.Span().StartLoc().Line
			max = decl.Span().EndLoc().Line
			ret = decl
			if decl.Kind() == ast.DeclKindDef && !decl.AsDef().Body().IsZero() {
				child := getDeclForPosition(decl.AsDef().Body(), line, char, min, max)
				if !child.IsZero() {
					ret = child
				}
			}
		}
	}
	return ret
}

func completionItemsForDecl(file *file, decl ast.DeclAny, position protocol.Position) []protocol.CompletionItem {
	switch decl.Kind() {
	case ast.DeclKindSyntax:
		return syntaxToCompletionItems(decl.AsSyntax())
	case ast.DeclKindPackage:
		// In the case of the package declaration, we should return the suggested package name
		// based on the path.
		var suggested []string
		for _, component := range normalpath.Components(file.objectInfo.Path()) {
			suggested = append(suggested, component)
			_, isVersion := protoversion.NewPackageVersionForComponent(component)
			if isVersion {
				break
			}
		}
		return []protocol.CompletionItem{{
			Label: fmt.Sprintf("%s;", strings.Join(suggested, ".")),
			Kind:  protocol.CompletionItemKindSnippet,
		}}
	case ast.DeclKindImport:
		// In the case where we are part of an import declaration, we should return suggested
		// import paths based on importable file paths.
		return importsToCompletionItems(xslices.MapKeysToSlice(file.importToFile))
	case ast.DeclKindDef:
		return defToCompletionItems(decl.AsDef(), file, position)
	case ast.DeclKindRange:
	}
	// Invalid or empty, return top-level keywords.
	return topLevelCompletionItems()
}

func defToCompletionItems(def ast.DeclDef, file *file, position protocol.Position) []protocol.CompletionItem {
	switch def.Classify() {
	case ast.DefKindMessage:
		suggestions := xslices.Map([]keyword.Keyword{
			keyword.Message,
			keyword.Enum,
			keyword.Option,
			keyword.Group, // TODO: proto2 only
			keyword.Extend,
			keyword.Oneof,
			keyword.Extensions,
			keyword.Reserved,

			keyword.Repeated,
			keyword.Optional,
			keyword.Required,
		}, keywordToCompletionItem)

		suggestions = append(suggestions, xslices.Map([]keyword.Keyword{
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
		}, predeclaredToCompletionItem)...)

		suggestions = append(suggestions, completionItemsForReferenceableSymbols(file)...)
		return suggestions
	case ast.DefKindService:
		return xslices.Map([]keyword.Keyword{
			keyword.Option,
			keyword.RPC,
		}, keywordToCompletionItem)
	case ast.DefKindMethod:
		return completionItemsForMethod(def.AsMethod(), file, position)
	case ast.DefKindField:
	case ast.DefKindOneof:
	case ast.DefKindGroup:
	case ast.DefKindEnumValue:
	case ast.DefKindOption:
	}
	// All other types definitions, we provide option as a suggestion
	return []protocol.CompletionItem{
		keywordToCompletionItem(keyword.Option),
	}
}

func completionItemsForMethod(method ast.DefMethod, file *file, position protocol.Position) []protocol.CompletionItem {
	line := int(position.Line) + 1
	char := int(position.Character) + 1
	isInSpan := func(span report.Span, line, char int) bool {
		if span.StartLoc().Line <= line &&
			span.EndLoc().Line >= line {
			return span.StartLoc().Column <= char && span.EndLoc().Column > char
		}
		return false
	}
	if isInSpan(method.Signature.Span(), line, char) {
		if !isInSpan(method.Signature.Returns().Span(), line, char) {
			// In a method signature and not in the returns keyword, suggestion input and output
			// message types.
			return completionItemsForReferenceableMessages(file)
		}
	}
	// No suggestions outside of defining inputs and outputs
	return []protocol.CompletionItem{}
}

func completionItemsForReferenceableMessages(file *file) []protocol.CompletionItem {
	messageSymbolsToCompletionItems := func(
		referenceableSymbols map[string]*symbol,
		mapper func(*symbol) protocol.CompletionItem,
	) []protocol.CompletionItem {
		var suggestions []protocol.CompletionItem
		for _, sym := range referenceableSymbols {
			if sym.ir.Kind() == ir.SymbolKindMessage {
				suggestions = append(suggestions, mapper(sym))
			}
		}
		return suggestions
	}
	suggestions := messageSymbolsToCompletionItems(file.referenceableSymbols, symbolToCompletionItemLocalName)
	for _, imported := range file.importToFile {
		if imported.ir.Package() == file.ir.Package() {
			suggestions = append(suggestions, messageSymbolsToCompletionItems(imported.referenceableSymbols, symbolToCompletionItemLocalName)...)
		} else {
			suggestions = append(suggestions, messageSymbolsToCompletionItems(imported.referenceableSymbols, symbolToCompletionItemFullName)...)
		}
	}
	return suggestions
}

func completionItemsForReferenceableSymbols(file *file) []protocol.CompletionItem {
	typeSymbolsToCompletionItems := func(
		referenceableSymbols map[string]*symbol,
		mapper func(*symbol) protocol.CompletionItem,
	) []protocol.CompletionItem {
		var suggestions []protocol.CompletionItem
		for _, symbol := range referenceableSymbols {
			if symbol.ir.Kind().IsType() {
				suggestions = append(suggestions, mapper(symbol))
			}
		}
		return suggestions
	}
	suggestions := typeSymbolsToCompletionItems(file.referenceableSymbols, symbolToCompletionItemLocalName)
	for _, imported := range file.importToFile {
		if imported.ir.Package() == file.ir.Package() {
			suggestions = append(suggestions, typeSymbolsToCompletionItems(imported.referenceableSymbols, symbolToCompletionItemLocalName)...)
		} else {
			suggestions = append(suggestions, typeSymbolsToCompletionItems(imported.referenceableSymbols, symbolToCompletionItemFullName)...)
		}
	}
	return suggestions
}

func symbolToCompletionItemFullName(symbol *symbol) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label: string(symbol.ir.FullName()),
		Kind:  protocol.CompletionItemKindTypeParameter,
	}
}

func symbolToCompletionItemLocalName(symbol *symbol) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label: symbol.ir.FullName().Name(),
		Kind:  protocol.CompletionItemKindTypeParameter,
	}
}

func syntaxToCompletionItems(syntaxDecl ast.DeclSyntax) []protocol.CompletionItem {
	var prefix string
	if syntaxDecl.IsEdition() {
		if syntaxDecl.KeywordToken().IsZero() {
			prefix += "edition"
		}
		if syntaxDecl.Equals().IsZero() {
			prefix += "= "
		}
		return xslices.Map(slices.Collect(syntax.Editions()), func(edition syntax.Syntax) protocol.CompletionItem {
			return protocol.CompletionItem{
				Label: prefix + edition.String() + ";",
				Kind:  protocol.CompletionItemKindValue,
			}
		})
	}
	if syntaxDecl.KeywordToken().IsZero() {
		prefix += "syntax"
	}
	if syntaxDecl.Equals().IsZero() {
		prefix += "= "
	}
	return xslices.Map([]syntax.Syntax{syntax.Proto2, syntax.Proto3}, func(syntax syntax.Syntax) protocol.CompletionItem {
		return protocol.CompletionItem{
			Label: prefix + syntax.String() + ";",
			Kind:  protocol.CompletionItemKindValue,
		}
	})
}

func importsToCompletionItems(imports []string) []protocol.CompletionItem {
	return xslices.Map(imports, func(path string) protocol.CompletionItem {
		return protocol.CompletionItem{
			Label: fmt.Sprintf(`"%s";`, path),
			Kind:  protocol.CompletionItemKindFile,
		}
	})
}

// topLevelCompletionItems returns all the viable top-level keywords as completion items.
func topLevelCompletionItems() []protocol.CompletionItem {
	return xslices.Map(topLevelKeywords, keywordToCompletionItem)
}

func predeclaredToCompletionItem(predeclared keyword.Keyword) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label: predeclared.String(),
		Kind:  protocol.CompletionItemKindTypeParameter,
	}
}

func keywordToCompletionItem(keyword keyword.Keyword) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label: keyword.String(),
		Kind:  protocol.CompletionItemKindKeyword,
	}
}
