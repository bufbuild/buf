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

package buflsp

import (
	"slices"

	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"github.com/google/cel-go/cel"
	"go.lsp.dev/protocol"
)

// The subset of SemanticTokenTypes that we support.
// Must match the order of [semanticTypeLegend].
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#textDocument_semanticTokens
const (
	semanticTypeProperty = iota
	semanticTypeStruct
	semanticTypeVariable
	semanticTypeEnum
	semanticTypeEnumMember
	semanticTypeInterface
	semanticTypeMethod
	semanticTypeFunction
	semanticTypeDecorator
	semanticTypeMacro
	semanticTypeNamespace
	semanticTypeKeyword
	semanticTypeModifier
	semanticTypeComment
	semanticTypeString
	semanticTypeNumber
	semanticTypeType
	semanticTypeOperator
)

// The subset of SemanticTokenModifiers that we support.
// Must match the order of [semanticModifierLegend].
// Semantic modifiers are encoded as a bitset, hence the shifted iota.
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#textDocument_semanticTokens
const (
	semanticModifierDeprecated = 1 << iota
	semanticModifierDefaultLibrary
)

var (
	// These slices must match the order of the indices in the above const blocks.
	// We use protocol constants where available.
	semanticTypeLegend = []string{
		string(protocol.SemanticTokenProperty),
		string(protocol.SemanticTokenStruct),
		string(protocol.SemanticTokenVariable),
		string(protocol.SemanticTokenEnum),
		string(protocol.SemanticTokenEnumMember),
		string(protocol.SemanticTokenInterface),
		string(protocol.SemanticTokenMethod),
		string(protocol.SemanticTokenFunction),
		"decorator", // Added in LSP 3.17.0; not in our protocol library yet.
		string(protocol.SemanticTokenMacro),
		string(protocol.SemanticTokenNamespace),
		string(protocol.SemanticTokenKeyword),
		string(protocol.SemanticTokenModifier),
		string(protocol.SemanticTokenComment),
		string(protocol.SemanticTokenString),
		string(protocol.SemanticTokenNumber),
		string(protocol.SemanticTokenType),
		string(protocol.SemanticTokenOperator),
	}
	semanticModifierLegend = []string{
		string(protocol.SemanticTokenModifierDeprecated),
		string(protocol.SemanticTokenModifierDefaultLibrary),
	}
)

func semanticTokensFull(file *file, celEnv *cel.Env) (*protocol.SemanticTokens, error) {
	if file == nil {
		return nil, nil
	}
	// In the case where there are no symbols for the file, we return nil for SemanticTokensFull.
	// This is based on the specification for the method textDocument/semanticTokens/full,
	// the expected response is the union type `SemanticTokens | null`.
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
	if len(file.symbols) == 0 {
		return nil, nil
	}
	// Semantic tokens are encoded using a delta encoding scheme described here:
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_semanticTokens
	//
	// We use a three-phase approach:
	// 1. Collect all tokens (can be in any order)
	// 2. Sort tokens by position (line, then column)
	// 3. Delta-encode the sorted tokens
	// tokenInfo holds information about a single semantic token before encoding.
	type tokenInfo struct {
		span    source.Span
		semType uint32
		semMod  uint32
		keyword keyword.Keyword
	}
	var tokens []tokenInfo
	// collectToken adds a token to the collection.
	collectToken := func(span source.Span, semanticType, semanticModifier uint32, kw keyword.Keyword) {
		if span.IsZero() {
			return
		}
		tokens = append(tokens, tokenInfo{
			span:    span,
			semType: semanticType,
			semMod:  semanticModifier,
			keyword: kw,
		})
	}
	// Phase 1: Collect all tokens
	astFile := file.ir.AST()
	if astFile == nil {
		return nil, nil
	}
	// Collect all comments and certain keywords that can't be fetched in the IR from token stream
	for tok := range astFile.Stream().All() {
		if tok.Kind() == token.Comment {
			collectToken(tok.Span(), semanticTypeComment, 0, keyword.Unknown)
		}
		kw := tok.Keyword()
		switch kw {
		// These keywords seemingly are not easy to reach via the IR.
		case keyword.Option, keyword.Reserved, keyword.To, keyword.Returns, keyword.Extend, keyword.Extensions:
			collectToken(tok.Span(), semanticTypeKeyword, 0, kw)
		}
	}
	// Collect syntax/edition declaration
	syntax := astFile.Syntax()
	if kwTok := syntax.KeywordToken(); !kwTok.IsZero() {
		collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
	}
	if value := syntax.Value(); !value.Span().IsZero() {
		collectToken(value.Span(), semanticTypeString, 0, keyword.Unknown)
	}
	// Collect package declaration
	pkg := astFile.Package()
	if kwTok := pkg.KeywordToken(); !kwTok.IsZero() {
		collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
	}
	if path := pkg.Path(); !path.Span().IsZero() {
		collectToken(path.Span(), semanticTypeNamespace, 0, keyword.Unknown)
	}
	// Collect import declarations
	for imp := range seq.Values(file.ir.Imports()) {
		if imp.Decl.Span().IsZero() {
			continue
		}
		// Collect import keyword
		if kwTok := imp.Decl.KeywordToken(); !kwTok.IsZero() {
			collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
		}
		// Collect modifier keywords (weak, public)
		modifierTokens := imp.Decl.ModifierTokens()
		for modToken := range seq.Values(modifierTokens) {
			if !modToken.IsZero() {
				collectToken(modToken.Span(), semanticTypeModifier, 0, keyword.Unknown)
			}
		}
		// Collect import path string
		if importPath := imp.Decl.ImportPath(); !importPath.Span().IsZero() {
			collectToken(importPath.Span(), semanticTypeString, 0, keyword.Unknown)
		}
	}
	// Collect extend declarations - specifically the extendee type reference
	for decl := range seq.Values(astFile.Decls()) {
		if decl.Kind() == ast.DeclKindDef {
			def := decl.AsDef()
			if def.Classify() == ast.DefKindExtend {
				extend := def.AsExtend()
				// Collect the extendee type reference (e.g., "Foo" in "extend Foo {}")
				if !extend.Extendee.Span().IsZero() {
					// Look up what the extendee resolves to determine the semantic type
					// For now, we'll mark it as struct since extends can only extend messages
					collectToken(extend.Extendee.Span(), semanticTypeStruct, 0, keyword.Unknown)
				}
			}
		}
	}
	// Collect symbol tokens (identifiers, keywords for declarations)
	for _, symbol := range file.symbols {
		var semanticType uint32
		var semanticModifier uint32
		switch kind := symbol.kind.(type) {
		case *option:
			// Options like [deprecated = true]
			// Note: "option" keyword is collected from the token stream scan
			// since options don't have associated ir.Symbol
			semanticType = semanticTypeDecorator
		case *reference:
			// Type references are highlighted based on what they reference
			switch kind.def.Classify() {
			case ast.DefKindMessage:
				semanticType = semanticTypeStruct
			case ast.DefKindEnum:
				semanticType = semanticTypeEnum
			}
		default:
			// Declaration symbols
			switch symbol.ir.Kind() {
			case ir.SymbolKindPackage:
				semanticType = semanticTypeNamespace
			case ir.SymbolKindMessage:
				// Collect "message" keyword
				if kwTok := symbol.ir.AsType().AST().KeywordToken(); !kwTok.IsZero() {
					collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
				}
				semanticType = semanticTypeStruct
			case ir.SymbolKindEnum:
				// Collect "enum" keyword
				if kwTok := symbol.ir.AsType().AST().KeywordToken(); !kwTok.IsZero() {
					collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
				}
				semanticType = semanticTypeEnum
			case ir.SymbolKindField:
				fieldDef := symbol.ir.AsMember().AST().AsField()
				// Collect field modifiers (repeated, optional, required)
				for prefix := range fieldDef.Type.Prefixes() {
					if prefixTok := prefix.PrefixToken(); !prefixTok.IsZero() {
						collectToken(prefixTok.Span(), semanticTypeModifier, 0, keyword.Unknown)
					}
				}
				// Collect the field tag number
				if !fieldDef.Tag.Span().IsZero() {
					collectToken(fieldDef.Tag.Span(), semanticTypeNumber, 0, keyword.Unknown)
				}
				if symbol.IsBuiltIn() {
					semanticType = semanticTypeType
					semanticModifier += semanticModifierDefaultLibrary
				} else {
					semanticType = semanticTypeProperty
				}
			case ir.SymbolKindEnumValue:
				enumValueDef := symbol.ir.AsMember().AST().AsEnumValue()
				// Collect the enum value number
				if !enumValueDef.Tag.Span().IsZero() {
					collectToken(enumValueDef.Tag.Span(), semanticTypeNumber, 0, keyword.Unknown)
				}
				semanticType = semanticTypeEnumMember
			case ir.SymbolKindExtension:
				fieldDef := symbol.ir.AsMember().AST().AsField()
				// Collect field modifiers (repeated, optional, required)
				for prefix := range fieldDef.Type.Prefixes() {
					if prefixTok := prefix.PrefixToken(); !prefixTok.IsZero() {
						collectToken(prefixTok.Span(), semanticTypeModifier, 0, keyword.Unknown)
					}
				}
				// Collect the field tag number
				if !fieldDef.Tag.Span().IsZero() {
					collectToken(fieldDef.Tag.Span(), semanticTypeNumber, 0, keyword.Unknown)
				}
				semanticType = semanticTypeVariable
			case ir.SymbolKindScalar:
				// Scalars are built-in types like int32, string, etc.
				semanticType = semanticTypeType
				semanticModifier += semanticModifierDefaultLibrary
			case ir.SymbolKindService:
				// Collect "service" keyword
				if kwTok := symbol.ir.AsService().AST().KeywordToken(); !kwTok.IsZero() {
					collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
				}
				semanticType = semanticTypeInterface
			case ir.SymbolKindOneof:
				// Collect "oneof" keyword
				if kwTok := symbol.ir.AsOneof().AST().KeywordToken(); !kwTok.IsZero() {
					collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
				}
				semanticType = semanticTypeProperty
			case ir.SymbolKindMethod:
				declDef := symbol.ir.AsMethod().AST()
				// Collect "rpc" keyword
				if kwTok := declDef.KeywordToken(); !kwTok.IsZero() {
					collectToken(kwTok.Span(), semanticTypeKeyword, 0, kwTok.Keyword())
				}
				methodDef := declDef.AsMethod()
				// Collect "stream" modifiers for input/output types
				if inputs := methodDef.Signature.Inputs(); inputs.Len() == 1 {
					for prefix := range inputs.At(0).Prefixes() {
						if prefixTok := prefix.PrefixToken(); !prefixTok.IsZero() {
							collectToken(prefixTok.Span(), semanticTypeModifier, 0, keyword.Unknown)
						}
					}
				}
				if outputs := methodDef.Signature.Outputs(); outputs.Len() == 1 {
					for prefix := range outputs.At(0).Prefixes() {
						if prefixTok := prefix.PrefixToken(); !prefixTok.IsZero() {
							collectToken(prefixTok.Span(), semanticTypeModifier, 0, keyword.Unknown)
						}
					}
				}
				// Note: "returns" keyword is collected from the token stream scan
				semanticType = semanticTypeMethod
			default:
				continue
			}
		}
		if _, ok := symbol.ir.Deprecated().AsBool(); ok {
			semanticModifier += semanticModifierDeprecated
		}

		collectToken(symbol.span, semanticType, semanticModifier, keyword.Unknown)
	}

	// Collect CEL tokens from protovalidate expressions
	for _, symbol := range file.symbols {
		// Skip option symbols themselves - we want the symbols that HAVE options
		if _, isOption := symbol.kind.(*option); isOption {
			continue
		}

		// Extract CEL expressions from buf.validate options if present
		celExprs := extractCELExpressions(file, symbol)
		for _, celExpr := range celExprs {
			collectCELTokens(celEnv, celExpr, collectToken)
		}
	}

	// When multiple tokens share the same span, prefer more specific types over generic keywords.
	// For example, "string" appears as both a keyword and a built-in type; we want the type.
	seen := make(map[source.Span]int)
	dedupedTokens := tokens[:0]
	for _, tok := range tokens {
		if idx, exists := seen[tok.span]; exists {
			existingTok := dedupedTokens[idx]
			if existingTok.semType == semanticTypeKeyword && tok.semType != semanticTypeKeyword {
				dedupedTokens[idx] = tok
			}
		} else {
			seen[tok.span] = len(dedupedTokens)
			dedupedTokens = append(dedupedTokens, tok)
		}
	}
	tokens = dedupedTokens
	// Phase 2: Sort tokens by position (line, then column)
	slices.SortFunc(tokens, func(a, b tokenInfo) int {
		aLoc := a.span.StartLoc()
		bLoc := b.span.StartLoc()
		if aLoc.Line != bLoc.Line {
			return aLoc.Line - bLoc.Line
		}
		return aLoc.Column - bLoc.Column
	})
	// Phase 3: Delta-encode the sorted tokens
	// The encoding represents each token as:
	// [deltaLine, deltaStartChar, length, tokenType, tokenModifiers]
	var (
		encoded           []uint32
		prevLine, prevCol uint32
	)
	for _, tok := range tokens {
		start := tok.span.StartLoc()
		end := tok.span.EndLoc()
		// Skip multi-line tokens
		if start.Line != end.Line {
			continue
		}
		currentLine := uint32(start.Line - 1) // Convert to 0-indexed
		startCol := uint32(start.Column - 1)  // Convert to 0-indexed
		deltaCol := startCol
		// If on the same line as previous token, make column delta relative
		if prevLine == currentLine {
			deltaCol -= prevCol
		}
		tokenLen := uint32(end.Column - start.Column)
		// Append: [deltaLine, deltaCol, length, type, modifiers]
		encoded = append(encoded, currentLine-prevLine, deltaCol, tokenLen, tok.semType, tok.semMod)
		// Update state for next iteration
		prevLine = currentLine
		prevCol = startCol
	}
	if len(encoded) == 0 {
		return nil, nil
	}
	return &protocol.SemanticTokens{Data: encoded}, nil
}
