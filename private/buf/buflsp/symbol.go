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

// This file defines all of the message handlers that involve symbols.
//
// In particular, this file handles semantic information in fileManager that have been
// *opened by the editor*, and thus do not need references to Buf modules to find.
// See imports.go for that part of the LSP.

package buflsp

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ast/predeclared"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"go.lsp.dev/protocol"
	"google.golang.org/protobuf/encoding/protowire"
)

// symbol represents a named symbol inside of a [file].
//
// For each symbol, we keep track of the location [report.Span] and file [*file] of the
// actual symbol and the definition symbol, if available.
//
// We also keep track of metadata for documentation rendering.
type symbol struct {
	ir ir.Symbol

	file *file
	def  *symbol
	span report.Span
	kind kind

	// isOption indicates whether this symbol represents an option.
	isOption bool
}

// kind is used to track the symbol kind and lets us resolve definitions to their symbol.
type kind interface {
	isSymbolKind()
}

type referenceable struct {
	ast        ast.DeclDef
	references []*symbol
}

type reference struct {
	def ast.DeclDef
}

type static struct {
	ast ast.DeclDef
}

type imported struct {
	file *file
}

type builtin struct {
	predeclared predeclared.Name
}

type tag struct{}

func (*referenceable) isSymbolKind() {}
func (*reference) isSymbolKind()     {}
func (*static) isSymbolKind()        {}
func (*imported) isSymbolKind()      {}
func (*builtin) isSymbolKind()       {}
func (*tag) isSymbolKind()           {}

// Range constructs an LSP protocol code range for this symbol.
func (s *symbol) Range() protocol.Range {
	return reportSpanToProtocolRange(s.span)
}

// IsBuiltIn checks if the symbol's type is a predeclared type. Predeclared type will
// not have a resolved definition symbol and the underlying type AST will be predeclared.
func (s *symbol) IsBuiltIn() bool {
	_, ok := s.kind.(*builtin)
	return ok
}

// Definition returns the location of the definition of the symbol.
func (s *symbol) Definition() protocol.Location {
	if imported, ok := s.kind.(*imported); ok {
		return protocol.Location{
			URI: imported.file.uri,
		}
	}
	if s.def == nil {
		// The definition does not have a span, so we just jump to the span of the symbol itself
		// as a fallback.
		return protocol.Location{
			URI:   s.file.uri,
			Range: s.Range(),
		}
	}
	return protocol.Location{
		URI:   s.def.file.uri,
		Range: s.def.Range(),
	}
}

// References returns the locations of references to the symbol, if applicable. Otherwise,
// it just returns the location of the symbol itself.
func (s *symbol) References() []protocol.Location {
	referenceable, ok := s.kind.(*referenceable)
	if !ok {
		return []protocol.Location{{
			URI:   s.file.uri,
			Range: s.Range(),
		}}
	}
	return xslices.Map(referenceable.references, func(sym *symbol) protocol.Location {
		return protocol.Location{
			URI:   sym.file.uri,
			Range: sym.Range(),
		}
	})
}

// LogValue provides the log value for a symbol.
func (s *symbol) LogValue() slog.Value {
	loc := func(loc report.Location) slog.Value {
		return slog.GroupValue(
			slog.Int("line", loc.Line),
			slog.Int("column", loc.Column),
		)
	}
	attrs := []slog.Attr{
		slog.String("file", s.span.Path()),
		slog.Any("start", loc(s.span.StartLoc())),
		slog.Any("end", loc(s.span.EndLoc())),
	}
	if imported, ok := s.kind.(*imported); ok {
		attrs = append(attrs, slog.String("imported", imported.file.uri.Filename()))
	} else if s.def != nil {
		attrs = append(attrs, slog.String("def_file", s.def.file.uri.Filename()))
		attrs = append(attrs, slog.Any("def_start", loc(s.def.span.StartLoc())))
		attrs = append(attrs, slog.Any("def_end", loc(s.def.span.EndLoc())))
	}
	return slog.GroupValue(attrs...)
}

// FormatDocs finds appropriate documentation for the given s and constructs a Markdown
// string for showing to the client.
//
// Returns the empty string if no docs are available.
func (s *symbol) FormatDocs(ctx context.Context) string {
	missingDocs := "<missing docs>"
	var tooltip strings.Builder
	var def ast.DeclDef
	switch s.kind.(type) {
	case *imported:
		imported, _ := s.kind.(*imported)
		// Provide a preview of the imported file.
		return fmt.Sprintf("```proto\n%s\n```", imported.file.text)
	case *tag:
		plural := func(i int) string {
			if i == 1 {
				return ""
			}
			return "s"
		}
		number := s.ir.AsMember().Number()
		var ty protowire.Type
		var packed bool
		switch s.ir.Kind() {
		case ir.SymbolKindEnumValue:
			varint := protowire.AppendVarint(nil, uint64(number))
			return fmt.Sprintf(
				"`0x%x`, `0b%b`\n\nencoded (hex): `%X` (%d byte%s)",
				number,
				number,
				varint,
				len(varint),
				plural(len(varint)),
			)
		case ir.SymbolKindField:
			typ := s.ir.AsMember().TypeAST()
			if s.ir.AsMember().IsGroup() {
				ty = protowire.StartGroupType
			} else if s.ir.AsMember().TypeAST().Kind() == ast.TypeKindPrefixed {
				prefixed := typ.AsPrefixed()
				prefixedType := prefixed.Type()
				if prefixedType.Kind() == ast.TypeKindPath {
					ty = protowireTypeForPredeclared(prefixedType.AsPath().AsPredeclared())
					if ty != protowire.BytesType {
						packed = prefixed.Prefix() == keyword.Repeated
					}
				} else {
					ty = protowire.BytesType
				}
			} else if s.ir.AsMember().TypeAST().Kind() == ast.TypeKindPath {
				ty = protowireTypeForPredeclared(typ.AsPath().AsPredeclared())
			} else {
				// All other cases, use protowire.BytesType
				ty = protowire.BytesType
			}
			varint := protowire.AppendTag(nil, protowire.Number(number), ty)
			doc := fmt.Sprintf(
				"encoded (hex): `%X` (%d byte%s)",
				varint, len(varint), plural(len(varint)),
			)
			if packed {
				packed := protowire.AppendTag(nil, protowire.Number(number), protowire.BytesType)
				return doc + fmt.Sprintf(
					"\n\npacked (hex): `%X` (%d byte%s)",
					packed, len(packed), plural(len(varint)),
				)
			}
			return doc
		}
	case *builtin:
		builtin, _ := s.kind.(*builtin)
		comments, ok := builtinDocs[builtin.predeclared.String()]
		if ok {
			return strings.Join(comments, "\n")
		}
		return missingDocs
	case *referenceable:
		referenceable, _ := s.kind.(*referenceable)
		def = referenceable.ast
	case *static:
		static, _ := s.kind.(*static)
		def = static.ast
	case *reference:
		reference, _ := s.kind.(*reference)
		def = reference.def
	}
	docs := getCommentsFromDef(def)
	if docs != "" {
		fmt.Fprintln(&tooltip, docs)
	} else {
		fmt.Fprintln(&tooltip, missingDocs)
	}
	return tooltip.String()
}

func protowireTypeForPredeclared(name predeclared.Name) protowire.Type {
	switch name {
	case predeclared.Bool, predeclared.Int32, predeclared.Int64, predeclared.UInt32,
		predeclared.UInt64, predeclared.SInt32, predeclared.SInt64:
		return protowire.VarintType
	case predeclared.Fixed32, predeclared.SFixed32, predeclared.Float:
		return protowire.Fixed32Type
	case predeclared.Fixed64, predeclared.SFixed64, predeclared.Double:
		return protowire.Fixed64Type
	}
	return protowire.BytesType
}

func getCommentsFromDef(def ast.DeclDef) string {
	if def.IsZero() {
		return ""
	}
	var comments []string
	// We drop the other side of "Around" because we only care about the beginning -- we're
	// traversing backwards for leading comemnts only.
	_, start := def.Context().Stream().Around(def.Span().StartLoc().Offset)
	cursor := token.NewCursorAt(start)
	t := cursor.PrevSkippable()
	for !t.IsZero() {
		switch t.Kind() {
		case token.Comment:
			comments = append(comments, commentToMarkdown(t.Text()))
		}
		if !cursor.PeekPrevSkippable().Kind().IsSkippable() {
			break
		}
		t = cursor.PrevSkippable()
	}
	// Reverse the list and return joined.
	slices.Reverse(comments)
	return strings.Join(comments, "")
}

// commentToMarkdown processes comment strings and formats them for markdown display.
func commentToMarkdown(comment string) string {
	if strings.HasPrefix(comment, "//") {
		// NOTE: We do not trim the space here, because indentation is
		// significant for Markdown code fences, and if every line
		// starts with a space, Markdown will trim it for us, even off
		// of code blocks.
		return strings.TrimPrefix(comment, "//")
	}
	if strings.HasPrefix(comment, "/**") && !strings.HasPrefix(comment, "/**/") {
		// NOTE: Doxygen-style comments (/** ... */) to Markdown format
		// by removing comment delimiters and formatting the content.
		//
		// Example:
		// /**
		//  * This is a Doxygen comment
		//  * with multiple lines
		//  */
		comment = strings.TrimSuffix(strings.TrimPrefix(comment, "/**"), "*/")
		lines := strings.Split(strings.TrimSpace(comment), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "*")
			lines[i] = line
		}
		return strings.Join(lines, "\n")
	}
	// Handle standard multi-line comments (/* ... */)
	return strings.TrimSuffix(strings.TrimPrefix(comment, "/*"), "*/")
}

// compareRanges compares two ranges for lexicographic ordering.
func comparePositions(a, b protocol.Position) int {
	diff := int(a.Line) - int(b.Line)
	if diff == 0 {
		return int(a.Character) - int(b.Character)
	}
	return diff
}

func reportSpanToProtocolRange(span report.Span) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(span.StartLoc().Line) - 1,
			Character: uint32(column(span.File, span.StartLoc())),
		},
		End: protocol.Position{
			Line:      uint32(span.EndLoc().Line) - 1,
			Character: uint32(column(span.File, span.EndLoc())),
		},
	}
}
