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
	"unicode"

	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ast/predeclared"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"go.lsp.dev/protocol"
	"google.golang.org/protobuf/encoding/protowire"
)

// symbol represents a named symbol inside of a [file].
//
// For each symbol, we keep track of the location [source.Span] and file [*file] of the
// actual symbol and the definition symbol, if available.
//
// We also keep track of metadata for documentation rendering.
type symbol struct {
	ir ir.Symbol

	file    *file
	def     *symbol
	typeDef *symbol // Empty for non-option symbols
	span    source.Span
	kind    kind
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
	def      ast.DeclDef
	fullName ir.FullName
}

type option struct {
	def             ast.DeclDef
	defFullName     ir.FullName
	typeDef         ast.DeclDef
	typeDefFullName ir.FullName
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
func (*option) isSymbolKind()        {}
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

// TypeDefinition returns the location of the type definition of the symbol.
func (s *symbol) TypeDefinition() protocol.Location {
	// For non-option symbols, this is the same as the definition, so we return that.
	if _, ok := s.kind.(*option); !ok {
		return s.Definition()
	}
	if s.typeDef == nil {
		// The type definition does not have a span, so we just jump to the span of the symbol
		// itself as a fallback.
		return protocol.Location{
			URI:   s.file.uri,
			Range: s.Range(),
		}
	}
	return protocol.Location{
		URI:   s.typeDef.file.uri,
		Range: s.typeDef.Range(),
	}
}

// References returns the locations of references to the symbol (including the definition), if
// applicable. Otherwise, it just returns the location of the symbol itself.
// It also accepts the includeDeclaration param from the client - if true, the declaration
// of the symbol is included as a reference.
func (s *symbol) References(includeDeclaration bool) []protocol.Location {
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
	if includeDeclaration {
		// Add the definition of the symbol to the list of references.
		if s.def != nil {
			references = append(references, protocol.Location{
				URI:   s.def.file.uri,
				Range: s.def.Range(),
			})
		}
	}
	return references
}

// DocumentHighlights returns document highlights for the symbol within the current file.
// This includes the definition (if in the same file) and all references in the same file.
// All highlights use the [protocol.DocumentHighlightKindText] kind.
func (s *symbol) DocumentHighlights() []protocol.DocumentHighlight {
	// Don't highlight static symbols (services, methods, enum values)
	if _, ok := s.kind.(*static); ok {
		return nil
	}

	// Get the referenceable kind to find all references
	referenceableKind, ok := s.kind.(*referenceable)
	if !ok && s.def != nil {
		// If the symbol isn't referenceable itself, but has a referenceable definition, use the
		// definition for the references.
		referenceableKind, ok = s.def.kind.(*referenceable)
	}
	if !ok {
		return nil
	}

	// Don't highlight field names. Field names have referenceable kind directly on the symbol,
	// whereas field type references have reference kind and reference a referenceable definition.
	// Both have ir.SymbolKindField, so we distinguish by checking if s.kind is referenceable.
	if _, isRefKind := s.kind.(*referenceable); isRefKind && s.ir.Kind() == ir.SymbolKindField {
		return nil
	}

	var highlights []protocol.DocumentHighlight
	// Add all references in the same file
	for _, reference := range referenceableKind.references {
		if reference.file.uri == s.file.uri {
			highlights = append(highlights, protocol.DocumentHighlight{
				Range: reference.Range(),
				Kind:  protocol.DocumentHighlightKindText,
			})
		}
	}

	// Add the definition if it's in the same file
	if s.def != nil && s.def.file.uri == s.file.uri {
		highlights = append(highlights, protocol.DocumentHighlight{
			Range: s.def.Range(),
			Kind:  protocol.DocumentHighlightKindText,
		})
	} else if s.def == nil {
		// If there's no separate definition, the symbol itself is the definition
		highlights = append(highlights, protocol.DocumentHighlight{
			Range: s.Range(),
			Kind:  protocol.DocumentHighlightKindText,
		})
	}

	return highlights
}

// LogValue provides the log value for a symbol.
func (s *symbol) LogValue() slog.Value {
	if s == nil {
		return slog.AnyValue(nil)
	}
	loc := func(loc source.Location) slog.Value {
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
	switch s.kind.(type) {
	case *imported:
		imported, _ := s.kind.(*imported)
		// Show the path to the file on disk, which is similar to how other LSP clients treat hovering
		// on an import file.
		return imported.file.file.Path()
	case *tag:
		return irMemberDoc(s.ir.AsMember())
	case *builtin:
		builtin, _ := s.kind.(*builtin)
		comments, ok := builtinDocs[builtin.predeclared.String()]
		if ok {
			// Use specific anchor for map, generic anchor for other builtins
			anchor := "field-types"
			if builtin.predeclared.String() == "map" {
				anchor = "maps"
			}
			comments = append(
				comments,
				"",
				fmt.Sprintf(
					"`%s` is a Protobuf builtin. [Learn more on protobuf.com.](https://protobuf.com/docs/language-spec#%s)",
					builtin.predeclared,
					anchor,
				),
			)
			return strings.Join(comments, "\n")
		}
		return ""
	case *referenceable, *static, *reference, *option:
		return s.getDocsFromComments()
	}
	return ""
}

// GetSymbolInformation returns the protocol symbol information for the symbol.
func (s *symbol) GetSymbolInformation() protocol.SymbolInformation {
	if s.ir.IsZero() {
		return protocol.SymbolInformation{}
	}

	name := s.ir.FullName()
	if name == "" {
		return protocol.SymbolInformation{}
	}
	parentFullName := name.Parent()
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
	case ir.SymbolKindScalar:
		kind = protocol.SymbolKindConstant
	default:
		kind = protocol.SymbolKindVariable
	}
	var isDeprecated bool
	if _, ok := s.ir.Deprecated().AsBool(); ok {
		isDeprecated = true
	}
	return protocol.SymbolInformation{
		Name:          string(name),
		Kind:          kind,
		Location:      location,
		ContainerName: containerName,
		Deprecated:    isDeprecated,
		Tags: []protocol.SymbolTag{
			protocol.SymbolTagDeprecated,
		},
	}
}

// Rename returns the [protocol.WorkspaceEdit] for renaming the symbol.
func (s *symbol) Rename(newName string) (*protocol.WorkspaceEdit, error) {
	var edits protocol.WorkspaceEdit
	switch s.kind.(type) {
	case *referenceable:
		if err := checkRenameConflicts(s, newName); err != nil {
			return nil, err
		}
		changes, err := renameChangesForReferenceableSymbol(s, newName)
		if err != nil {
			return nil, err
		}
		edits.Changes = changes
	case *static:
		if err := checkRenameConflicts(s, newName); err != nil {
			return nil, err
		}
		edits.Changes = map[protocol.DocumentURI][]protocol.TextEdit{
			s.file.uri: {{
				Range:   reportSpanToProtocolRange(s.span),
				NewText: newName,
			}},
		}
	case *reference:
		// For references, we attempt to rename the definition symbol, if resolved. This would
		// include this reference symbol.
		if s.def != nil {
			if err := checkRenameConflicts(s.def, newName); err != nil {
				return nil, err
			}
			changes, err := renameChangesForReferenceableSymbol(s.def, newName)
			if err != nil {
				return nil, err
			}
			edits.Changes = changes
		}
	}
	// All other symbol types (options, imports, built-ins, and tags) cannot be renamed.
	return &edits, nil
}

// renameChangesForReferenceableSymbol is a helper for getting all rename changes for the
// given referenceable symbol.
func renameChangesForReferenceableSymbol(s *symbol, newName string) (map[protocol.DocumentURI][]protocol.TextEdit, error) {
	// At minimum, we would rename the symbol itself.
	changes := map[protocol.DocumentURI][]protocol.TextEdit{
		s.file.uri: {{
			Range:   reportSpanToProtocolRange(s.span),
			NewText: newName,
		}},
	}
	// Get the referenceable kind to find all references
	referenceableKind, ok := s.kind.(*referenceable)
	if !ok && s.def != nil {
		// If the symbol isn't referenceable itself, but has a referenceable definition, use the
		// definition for the references.
		referenceableKind, ok = s.def.kind.(*referenceable)
	}
	if ok {
		for _, reference := range referenceableKind.references {
			newText := newName
			// For option references (extension usages), preserve package qualification and parentheses.
			// e.g., if renaming "(subpkg.testing)" to "validated", result should be "(subpkg.validated)"
			if _, isOption := reference.kind.(*option); isOption {
				spanText := reference.span.Text()
				// Extract components: prefix (opening paren + package), suffix (closing paren)
				prefix := ""
				suffix := ""
				nameOnly := spanText

				// Check for opening parenthesis
				if strings.HasPrefix(spanText, "(") {
					prefix = "("
					nameOnly = nameOnly[1:]
				}
				// Check for closing parenthesis
				if strings.HasSuffix(nameOnly, ")") {
					suffix = ")"
					nameOnly = nameOnly[:len(nameOnly)-1]
				}
				// Check if there's package qualification (contains a dot)
				if lastDot := strings.LastIndex(nameOnly, "."); lastDot != -1 {
					// Preserve the package qualification
					packageQualification := nameOnly[:lastDot+1]
					newText = prefix + packageQualification + newName + suffix
				} else {
					// No package qualification, just preserve parens
					newText = prefix + newName + suffix
				}
			}
			changes[reference.file.uri] = append(changes[reference.file.uri], protocol.TextEdit{
				Range:   reportSpanToProtocolRange(reference.span),
				NewText: newText,
			})
		}
	} else {
		return nil, fmt.Errorf("attempting to rename a non-referenceble symbol as a referenceable symbol: %v", s)
	}
	return changes, nil
}

// checkRenameConflicts takes the symbol and desired new name and checks if this conflicts
// with an existing symbol in the same scope and returns an error if a conflict is found.
func checkRenameConflicts(target *symbol, newName string) error {
	parent := target.ir.FullName().Parent()
	if parent != "" {
		var existing source.Span
		newFullName := parent.Append(newName)
		containsFunc := func(s *symbol) bool {
			existing = s.span
			return s.ir.FullName() == newFullName
		}
		// We check all files in the workspace for a conflict, since a package can span an arbitrary
		// number of files.
		// We first check the current symbol's file.
		if slices.ContainsFunc(target.file.symbols, containsFunc) {
			return fmt.Errorf(
				"Renaming %q to %q would conflict with existing symbol at %s:%d:%d",
				target.ir.FullName().Name(),
				newName,
				target.file.ir.Path(),
				existing.StartLoc().Line,
				existing.StartLoc().Column,
			)
		}
		for _, file := range target.file.workspace.PathToFile() {
			if slices.ContainsFunc(file.symbols, containsFunc) {
				return fmt.Errorf(
					"Renaming %q to %q would conflict with existing symbol at %s:%d:%d",
					target.ir.FullName().Name(),
					newName,
					file.ir.Path(),
					existing.StartLoc().Line,
					existing.StartLoc().Column,
				)
			}
		}
	}
	return nil
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

// getDocsFromComments is a helper function that gets the doc string from the comments from
// the definition AST, if available.
// This helper function expects that imports, tags, and predeclared (builtin) types are
// already handled, since those types currently do not get docs from their comments.
func (s *symbol) getDocsFromComments() string {
	if s.def == nil {
		return ""
	}
	var def ast.DeclDef
	switch s.kind.(type) {
	case *referenceable:
		referenceable, _ := s.kind.(*referenceable)
		def = referenceable.ast
	case *static:
		static, _ := s.kind.(*static)
		def = static.ast
	case *reference:
		reference, _ := s.kind.(*reference)
		def = reference.def
	case *option:
		option, _ := s.kind.(*option)
		def = option.def
	}
	if def.IsZero() {
		return ""
	}

	var comments []string
	// We drop the other side of "Around" because we only care about the beginning -- we're
	// traversing backwards for leading comments only.
	tok, _ := def.Context().Stream().Around(def.Span().Start)
	cursor := token.NewCursorAt(tok)
	// Count consecutive newlines. If we accumulate 2+ without encountering a comment,
	// there's a blank line separating the comments from the symbol.
	newlinesSeen := 0
	if tok.Kind() == token.Space {
		newlinesSeen = strings.Count(tok.Text(), "\n")
		if newlinesSeen >= 2 {
			return ""
		}
	}
	for {
		t := cursor.PrevSkippable()
		if t.Kind() == token.Comment {
			if isTrailingComment(t) {
				break
			}
			text := commentToMarkdown(t.Text()) + "\n"
			newlinesSeen = 0
			comments = append(comments, text)
		} else if t.Kind() == token.Space {
			newlinesSeen += strings.Count(t.Text(), "\n")
			if newlinesSeen >= 2 {
				break
			}
		} else {
			break
		}
	}
	comments = lineUpComments(comments)
	// Reverse the list and return joined.
	slices.Reverse(comments)

	var docs strings.Builder
	for _, comment := range comments {
		docs.WriteString(comment)
	}

	// If the file is a remote dependency, link to BSR docs.
	if s.def != nil && s.def.file != nil && !s.def.file.IsLocal() {
		// In the BSR, messages, enums, and service definitions support anchor tags in the link.
		// Otherwise, we use the anchor for the parent type.
		var hasAnchor, isExtension bool
		switch def.Classify() {
		case ast.DefKindMessage, ast.DefKindEnum, ast.DefKindService:
			hasAnchor = true
		case ast.DefKindExtend:
			isExtension = !def.AsExtend().Extendee.IsZero()
			hasAnchor = isExtension
		}

		var module bufmodule.Module
		var bsrHost string
		if s.def.file.IsWKT() {
			bsrHost = bufconnect.DefaultRemote + "/protocolbuffers/wellknowntypes"
		} else if fileInfo, ok := s.def.file.objectInfo.(bufmodule.FileInfo); ok {
			module = fileInfo.Module()
			bsrHost = module.FullName().String()
		}

		defFullName := s.def.ir.FullName()
		if !hasAnchor {
			defFullName = defFullName.Parent()
		}
		bsrAnchor := string(defFullName)
		// For extensions, we use the anchor for the extensions section in the BSR docs.
		if isExtension {
			bsrAnchor = "extensions"
		}

		if bsrHost != "" {
			packageName := string(s.def.file.ir.Package())
			var url string
			if s.def.file.IsWKT() {
				// WKT uses special bsrHost format
				url = "https://" + bsrHost + "/docs/main:" + packageName
				if bsrAnchor != "" {
					url += "#" + bsrAnchor
				}
			} else {
				// Use bsrURL for non-WKT modules
				url = bsrURL(module, packageName, bsrAnchor, bsrTabTypeDocs)
			}
			if url != "" {
				fmt.Fprintf(
					&docs,
					"\n[`%s` on the Buf Schema Registry](%s)\n",
					defFullName,
					url,
				)
			}
		}
	}
	return docs.String()
}

// commentToMarkdown processes comment strings and formats them for markdown display.
func commentToMarkdown(comment string) string {
	if after, ok := strings.CutPrefix(comment, "//"); ok {
		// NOTE: We do not trim the space here, because indentation is
		// significant for Markdown code fences, and if every line
		// starts with a space, Markdown will trim it for us, even off
		// of code blocks.
		return after
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

// lineUpComments is a helper function for lining up the comments for docs, since some users
// may start their comments with spaces, e.g.
//
//	// Foo is a ...
//	//
//	// Foo is used for ...
//	message Foo { ... }
//
// vs.
//
//	//Foo is a ...
//	//
//	//Foo is used for ...
//	message Foo { ... }
//
// When different LSP clients render these docs, they treat leading spaces differently. To
// help mitigate this, we use the following heuristic to line up the comments:
//
// If all lines containing one or more non-space characters all start with a single space
// character, we trim the space character for each of these lines. Otherwise, do nothing.
func lineUpComments(comments []string) []string {
	linedUp := make([]string, len(comments))
	for i, comment := range comments {
		if strings.ContainsFunc(comment, func(r rune) bool {
			return !unicode.IsSpace(r)
		}) {
			// We are only checking for " " and do not count nbsp's.
			if !strings.HasPrefix(comment, " ") {
				return comments
			}
			linedUp[i] = strings.TrimPrefix(comment, " ")
		} else {
			linedUp[i] = comment
		}
	}
	return linedUp
}

// irMemberDoc returns the documentation for a message field, enum value or extension field.
func irMemberDoc(irMember ir.Member) string {
	number := irMember.Number()
	if irMember.IsEnumValue() {
		varint := protowire.AppendVarint(nil, uint64(number))
		return fmt.Sprintf(
			"`0x%x`, `0b%b`\n\nencoded (hex): `%X` (%d byte%s)",
			number,
			number,
			varint,
			len(varint),
			plural(len(varint)),
		)
	}

	var (
		builder strings.Builder
		ty      protowire.Type
		packed  bool
	)
	typeAST := irMember.TypeAST()
	if irMember.IsGroup() {
		ty = protowire.StartGroupType
	} else if typeAST.Kind() == ast.TypeKindPrefixed {
		prefixed := typeAST.AsPrefixed()
		prefixedType := prefixed.Type()
		if prefixedType.Kind() == ast.TypeKindPath {
			ty = protowireTypeForPredeclared(prefixedType.AsPath().AsPredeclared())
			if ty != protowire.BytesType {
				packed = prefixed.Prefix() == keyword.Repeated
			}
		} else {
			ty = protowire.BytesType
		}
	} else if typeAST.Kind() == ast.TypeKindPath {
		ty = protowireTypeForPredeclared(typeAST.AsPath().AsPredeclared())
	} else {
		// All other cases, use protowire.BytesType
		ty = protowire.BytesType
	}
	varint := protowire.AppendTag(nil, protowire.Number(number), ty)
	fmt.Fprintf(
		&builder,
		"encoded (hex): `%X` (%d byte%s)",
		varint, len(varint), plural(len(varint)),
	)
	if packed {
		packed := protowire.AppendTag(nil, protowire.Number(number), protowire.BytesType)
		fmt.Fprintf(
			&builder,
			"\n\npacked (hex): `%X` (%d byte%s)",
			packed, len(packed), plural(len(varint)),
		)
		return builder.String()
	}
	return builder.String()
}

func reportSpanToProtocolRange(span source.Span) protocol.Range {
	startLocation := span.File.Location(span.Start, positionalEncoding)
	endLocation := span.File.Location(span.End, positionalEncoding)
	return reportLocationsToProtocolRange(startLocation, endLocation)
}

func reportLocationsToProtocolRange(startLocation, endLocation source.Location) protocol.Range {
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

func plural(i int) string {
	if i == 1 {
		return ""
	}
	return "s"
}

// isTrailingComment returns true if the comment has code on the same line before it.
func isTrailingComment(t token.Token) bool {
	if t.Kind() != token.Comment {
		return false
	}
	for c := token.NewCursorAt(t); ; {
		p := c.PrevSkippable()
		if p.IsZero() || strings.Contains(p.Text(), "\n") {
			return false
		}
		if p.Kind() != token.Space {
			return true
		}
	}
}
