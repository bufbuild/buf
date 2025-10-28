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
	"fmt"
	"log/slog"
	"slices"
	"strings"

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

// References returns the locations of references to the symbol (including the definition), if
// applicable. Otherwise, it just returns the location of the symbol itself.
func (s *symbol) References() []protocol.Location {
	var references []protocol.Location
	referenceableKind, ok := s.kind.(*referenceable)
	if !ok && s.def != nil {
		// If the symbol isn't referenceable itself, but has a referenceable definition, use the
		// definition for the references.
		referenceableKind, ok = s.def.kind.(*referenceable)
	}
	if ok {
		for _, reference := range referenceableKind.references {
			references = append(references, protocol.Location{
				URI:   reference.file.uri,
				Range: reference.Range(),
			})
		}
	} else {
		// No referenceable kind; add the location of the symbol itself.
		references = append(references, protocol.Location{
			URI:   s.file.uri,
			Range: s.Range(),
		})
	}
	// Add the definition of the symbol to the list of references.
	if s.def != nil {
		references = append(references, protocol.Location{
			URI:   s.def.file.uri,
			Range: s.def.Range(),
		})
	}
	return references
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
		slog.String("path", s.span.Path()),
		slog.Any("start", loc(s.span.StartLoc())),
		slog.Any("end", loc(s.span.EndLoc())),
	}
	if imported, ok := s.kind.(*imported); ok {
		attrs = append(attrs, slog.String("imported", imported.file.uri.Filename()))
	} else if s.def != nil {
		attrs = append(attrs,
			slog.String("uri", s.def.file.uri.Filename()),
			slog.Any("start", loc(s.span.StartLoc())),
			slog.Any("end", loc(s.span.EndLoc())),
		)
	}
	return slog.GroupValue(attrs...)
}

// FormatDocs finds appropriate documentation for the given s and constructs a Markdown
// string for showing to the client.
//
// Returns the empty string if no docs are available.
func (s *symbol) FormatDocs() string {
	var def ast.DeclDef
	switch s.kind.(type) {
	case *imported:
		imported, _ := s.kind.(*imported)
		// Show the path to the file on disk, which is similar to how other LSP clients treat hovering
		// on an import file.
		return imported.file.file.Path()
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
		return ""
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
	return getCommentsFromDef(def)
}

// GetSymbolInformation returns the protocol symbol information for the symbol.
func (s *symbol) GetSymbolInformation() protocol.SymbolInformation {
	if s.ir.IsZero() {
		return protocol.SymbolInformation{}
	}

	fullName := s.ir.FullName()
	name := fullName.Name()
	if name == "" {
		return protocol.SymbolInformation{}
	}
	parentFullName := fullName.Parent()
	containerName := string(parentFullName)

	location := protocol.Location{
		URI:   s.file.uri,
		Range: s.Range(),
	}

	// Determine the symbol kind for LSP.
	var kind protocol.SymbolKind
	switch s.ir.Kind() {
	case ir.SymbolKindMessage:
		kind = protocol.SymbolKindClass // Messages are like classes
	case ir.SymbolKindEnum:
		kind = protocol.SymbolKindEnum
	case ir.SymbolKindEnumValue:
		kind = protocol.SymbolKindEnumMember
	case ir.SymbolKindField:
		kind = protocol.SymbolKindField
	case ir.SymbolKindExtension:
		kind = protocol.SymbolKindField
	case ir.SymbolKindOneof:
		kind = protocol.SymbolKindClass // Oneof are like classes
	case ir.SymbolKindService:
		kind = protocol.SymbolKindInterface // Services are like interfaces
	case ir.SymbolKindMethod:
		kind = protocol.SymbolKindMethod
	default:
		kind = protocol.SymbolKindVariable
	}
	var isDeprecated bool
	if _, ok := s.ir.Deprecated().AsBool(); ok {
		isDeprecated = true
	}
	return protocol.SymbolInformation{
		Name:          name,
		Kind:          kind,
		Location:      location,
		ContainerName: containerName,
		// TODO: Use Tags with a protocol.CompletionItemTagDeprecated if the client supports tags.
		Deprecated: isDeprecated,
	}
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
	_, start := def.Context().Stream().Around(def.Span().Start)
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

// comparePositions compares two positions for lexicographic ordering.
func comparePositions(a, b protocol.Position) int {
	diff := int(a.Line) - int(b.Line)
	if diff == 0 {
		return int(a.Character) - int(b.Character)
	}
	return diff
}

// positionInRange returns 0 if a position is within the range, else returns -1 before or 1 after.
func positionInRange(position protocol.Position, within protocol.Range) int {
	if comparePositions(position, within.Start) < 0 {
		return -1
	}
	if comparePositions(position, within.End) > 0 {
		return 1
	}
	return 0
}

func reportSpanToProtocolRange(span report.Span) protocol.Range {
	startLocation := span.File.Location(span.Start, positionalEncoding)
	endLocation := span.File.Location(span.End, positionalEncoding)
	return reportLocationsToProtocolRange(startLocation, endLocation)
}

func reportLocationsToProtocolRange(startLocation, endLocation report.Location) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(startLocation.Line - 1),
			Character: uint32(startLocation.Column - 1),
		},
		End: protocol.Position{
			Line:      uint32(endLocation.Line - 1),
			Character: uint32(endLocation.Column - 1),
		},
	}
}
