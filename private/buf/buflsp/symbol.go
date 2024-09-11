// Copyright 2020-2024 Buf Technologies, Inc.
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
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/protocompile/ast"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// symbol represents a named symbol inside of a buflsp.file
type symbol struct {
	// The file this symbol sits in.
	file *file

	// The node containing the symbol's name.
	name ast.Node
	// Node info for the symbol itself. This specifies the region of the file
	// that contains this symbol.
	info ast.NodeInfo
	// What kind of symbol this is.
	kind symbolKind

	// Whether this symbol came from an option node.
	isOption bool
}

// symbolKind is a kind of symbol. It is implemented by *definition, *reference, and *import_.
type symbolKind interface {
	isSymbolKind()
}

// definition is a symbol that is a definition.
type definition struct {
	// The node of the overall definition. E.g. for a message this is the whole message node.
	node ast.Node
	// The fully qualified path of this symbol, not including its package (which is implicit from
	// its file.)
	path []string
}

// reference is a reference to a symbol in some other file.
type reference struct {
	// The file this symbol is defined in. Nil if this reference is unresolved.
	file *file
	// The fully qualified path of this symbol, not including its package (which is implicit from
	// its definition file.)
	path []string

	// If this is nonnil, this is a reference symbol to a field inside of an option path
	// or composite textproto literal. For example, consider the code
	//
	// [(foo.bar).baz = xyz]
	//
	// baz is a symbol, whose reference depends on the type of foo.bar, which depends on the
	// imports of the file foo.bar is defined in.
	seeTypeOf *symbol

	// If this is nonnil, this is a non-custom option reference defined in the given node.
	isNonCustomOptionIn ast.Node
}

// import_ is a symbol representing an import.
type import_ struct {
	// The imported file. Nil if this reference is unresolved.
	file *file
}

// builtin is a built-in symbol.
type builtin struct {
	name string
}

func (*definition) isSymbolKind() {}
func (*reference) isSymbolKind()  {}
func (*import_) isSymbolKind()    {}
func (*builtin) isSymbolKind()    {}

// Range constructs an LSP protocol code range for this symbol.
func (s *symbol) Range() protocol.Range {
	return infoToRange(s.info)
}

// Definition looks up the definition of this symbol, if known.
func (s *symbol) Definition(ctx context.Context) (*symbol, ast.Node) {
	switch kind := s.kind.(type) {
	case *definition:
		return s, kind.node
	case *reference:
		if kind.file == nil {
			return nil, nil
		}

		kind.file.lock.Lock(ctx)
		defer kind.file.lock.Unlock(ctx)
		for _, symbol := range kind.file.symbols {
			def, ok := symbol.kind.(*definition)
			if ok && slices.Equal(kind.path, def.path) {
				return symbol, def.node
			}
		}
	}

	return nil, nil
}

// ReferencePath returns the reference path of this string, i.e., the components of
// a path like foo.bar.Baz.
//
// Returns nil if the name of this symbol is not a path.
func (s *symbol) ReferencePath() (path []string, absolute bool) {
	switch name := s.name.(type) {
	case *ast.IdentNode:
		path = []string{name.Val}
	case *ast.CompoundIdentNode:
		path = slicesext.Map(name.Components, func(name *ast.IdentNode) string { return name.Val })
		absolute = name.LeadingDot != nil
	}
	return
}

// Resolve attempts to resolve an unresolved reference across fileManager.
func (s *symbol) ResolveCrossFile(ctx context.Context) {
	switch kind := s.kind.(type) {
	case *definition:
	case *builtin:
	case *import_:
		// These symbols do not require resolution.

	case *reference:
		if kind.file != nil {
			// Already resolved, not our problem!
			return
		}

		components, _ := s.ReferencePath()

		// This is a field of some foreign type. We need to track down where this is.
		if kind.seeTypeOf != nil {
			ref, ok := kind.seeTypeOf.kind.(*reference)
			if !ok || ref.file == nil {
				s.file.lsp.logger.Debug(
					"unexpected unresolved or non-reference symbol for seeTypeOf",
					zap.Object("symbol", s))
				return
			}

			// Fully index the file this reference is in, if different from the current.
			if s.file != ref.file {
				ref.file.Refresh(ctx)
			}

			// Find the definition that contains the type we want.
			def, node := kind.seeTypeOf.Definition(ctx)
			if def == nil {
				s.file.lsp.logger.Debug(
					"could not resolve dependent symbol definition",
					zap.Object("symbol", s),
					zap.Object("dep", kind.seeTypeOf))
				return
			}

			// Node here should be some kind of field.
			// TODO: Support more exotic field types.
			field, ok := node.(*ast.FieldNode)
			if !ok {
				s.file.lsp.logger.Debug(
					"dependent symbol definition was not a field",
					zap.Object("symbol", s),
					zap.Object("dep", kind.seeTypeOf),
					zap.Object("def", def))
				return
			}

			// Now, find the symbol for the field's type in the file's symbol table.
			// Searching by offset is faster.
			info := def.file.fileNode.NodeInfo(field.FldType)
			ty := def.file.SymbolAt(ctx, protocol.Position{
				Line:      uint32(info.Start().Line) - 1,
				Character: uint32(info.Start().Col) - 1,
			})
			if ty == nil {
				s.file.lsp.logger.Debug(
					"dependent symbol's field type didn't resolve",
					zap.Object("symbol", s),
					zap.Object("dep", kind.seeTypeOf),
					zap.Object("def", def))
				return
			}

			// This will give us enough information to figure out the path of this
			// symbol, namely, the name of the thing the symbol is inside of. We don't
			// actually validate if the dependent symbol exists, because that will happen for us
			// when we go to hover over the symbol.
			ref, ok = ty.kind.(*reference)
			if !ok || ty.file == nil {
				s.file.lsp.logger.Debug(
					"dependent symbol's field type didn't resolve to a reference",
					zap.Object("symbol", s),
					zap.Object("dep", kind.seeTypeOf),
					zap.Object("def", def),
					zap.Object("resolved", ty))
				return
			}

			// Done.
			kind.file = def.file
			kind.path = append(slicesext.Copy(ref.path), components...)
			return
		}

		if kind.isNonCustomOptionIn != nil {
			var optionsType []string
			switch kind.isNonCustomOptionIn.(type) {
			case *ast.FileNode:
				optionsType = []string{"FileOptions"}
			case *ast.MessageNode:
				optionsType = []string{"MessageOptions"}
			case *ast.FieldNode, *ast.MapFieldNode:
				optionsType = []string{"FieldOptions"}
			case *ast.OneofNode:
				optionsType = []string{"OneofOptions"}
			case *ast.EnumNode:
				optionsType = []string{"EnumOptions"}
			case *ast.EnumValueNode:
				optionsType = []string{"EnumValueOptions"}
			case *ast.ServiceNode:
				optionsType = []string{"ServiceOptions"}
			case *ast.RPCNode:
				optionsType = []string{"MethodOptions"}
			case *ast.ExtensionRangeNode:
				optionsType = []string{"DescriptorProto", "ExtensionRangeOptions"}
			default:
				// This node cannot contain options.
				return
			}

			fieldPath := append(optionsType, kind.path...)

			// Make a copy of the import table pointer and then drop the lock,
			// since searching inside of the imports will need to acquire other
			// fileManager' locks.
			s.file.lock.Lock(ctx)
			descriptorProto := s.file.imports[descriptorPath]
			s.file.lock.Unlock(ctx)

			if descriptorProto == nil {
				return
			}

			// Look for a symbol with this exact path in descriptor proto.

			descriptorProto.lock.Lock(ctx)
			defer descriptorProto.lock.Unlock(ctx)

			var fieldSymbol *symbol
			for _, symbol := range descriptorProto.symbols {
				if def, ok := symbol.kind.(*definition); ok && slices.Equal(def.path, fieldPath) {
					fieldSymbol = symbol
					break
				}
			}
			if fieldSymbol == nil {
				return
			}

			kind.file = descriptorProto
			kind.path = fieldPath
			return
		}

		// Make a copy of the import table pointer and then drop the lock,
		// since searching inside of the imports will need to acquire other
		// fileManager' locks.
		s.file.lock.Lock(ctx)
		imports := s.file.imports
		s.file.lock.Unlock(ctx)

		if imports == nil {
			// Hopeless. We'll have to try again once we have imports!
			return
		}

		for _, imported := range imports {
			// Remove a leading pkg from components.
			path, ok := slicesext.TrimPrefix(components, imported.Package())
			if !ok {
				continue
			}

			if findDeclByPath(imported.fileNode.Decls, path) != nil {
				kind.file = imported
				kind.path = path
				break
			}
		}
	}
}

func (symbol *symbol) MarshalLogObject(enc zapcore.ObjectEncoder) (err error) {
	enc.AddString("file", symbol.file.uri.Filename())

	// zapPos converts an ast.SourcePos into a zap marshaller.
	zapPos := func(pos ast.SourcePos) zapcore.ObjectMarshalerFunc {
		return func(enc zapcore.ObjectEncoder) error {
			enc.AddInt("offset", pos.Offset)
			enc.AddInt("line", pos.Line)
			enc.AddInt("col", pos.Col)
			return nil
		}
	}

	err = enc.AddObject("start", zapPos(symbol.info.Start()))
	if err != nil {
		return err
	}

	err = enc.AddObject("end", zapPos(symbol.info.End()))
	if err != nil {
		return err
	}

	switch kind := symbol.kind.(type) {
	case *builtin:
		enc.AddString("builtin", kind.name)

	case *import_:
		if kind.file != nil {
			enc.AddString("imports", kind.file.uri.Filename())
		}

	case *definition:
		enc.AddString("defines", strings.Join(kind.path, "."))

	case *reference:
		if kind.file != nil {
			enc.AddString("imports", kind.file.uri.Filename())
		}
		if kind.path != nil {
			enc.AddString("references", strings.Join(kind.path, "."))
		}
		if kind.seeTypeOf != nil {
			err = enc.AddObject("see_type_of", kind.seeTypeOf)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// FormatDocs finds appropriate  documentation for the given symbol and constructs a Markdown
// string for showing to the client.
//
// Returns the empty string if no docs are available.
func (s *symbol) FormatDocs(ctx context.Context) string {
	var (
		tooltip strings.Builder
		def     *symbol
		node    ast.Node
		path    []string
	)

	switch kind := s.kind.(type) {
	case *builtin:
		fmt.Fprintf(&tooltip, "```proto\nbuiltin %s\n```\n", kind.name)
		for _, line := range builtinDocs[kind.name] {
			fmt.Fprintln(&tooltip, line)
		}

		fmt.Fprintln(&tooltip)
		fmt.Fprintf(
			&tooltip,
			"This is a built-in Protobuf type. [Learn more on protobuf.com.](https://protobuf.com/docs/language-spec#field-types)",
		)
		return tooltip.String()

	case *reference:
		def, node = s.Definition(ctx)
		path = kind.path

	case *definition:
		def = s
		node = kind.node
		path = kind.path

	default:
		return ""
	}

	pkg := "<empty>"
	if pkgNode := def.file.packageNode; pkgNode != nil {
		pkg = string(pkgNode.Name.AsIdentifier())
	}

	what := "unresolved"
	switch node.(type) {
	case *ast.FileNode:
		what = "file"
	case *ast.MessageNode:
		what = "message"
	case *ast.FieldNode, *ast.MapFieldNode:
		what = "field"
	case *ast.GroupNode:
		what = "group"
	case *ast.OneofNode:
		what = "oneof"
	case *ast.EnumNode:
		what = "enum"
	case *ast.EnumValueNode:
		what = "enum value"
	case *ast.ServiceNode:
		what = "service"
	case *ast.RPCNode:
		what = "rpc"
	}

	fmt.Fprintf(&tooltip, "```proto-decl\n%s %s.%s\n```\n", what, pkg, strings.Join(path, "."))

	if node == nil {
		fmt.Fprintln(&tooltip, "<could not resolve type>")
		return tooltip.String()
	}

	// Dump all of the comments into the tooltip. These will be rendered as Markdown automatically
	// by the client.
	info := def.file.fileNode.NodeInfo(node)
	allComments := []ast.Comments{info.LeadingComments(), info.TrailingComments()}
	var printed bool
	for _, comments := range allComments {
		for i := 0; i < comments.Len(); i++ {
			comment := comments.Index(i).RawText()

			// The compiler does not currently provide comments without their
			// delimited removed, so we have to do this ourselves.
			if strings.HasPrefix(comment, "//") {
				comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))
			} else {
				comment = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(comment, "/*"), "*/"))
			}

			if comment != "" {
				printed = true
			}

			// No need to process Markdown in comment; this Just Works!
			fmt.Fprintln(&tooltip, comment)
		}
	}

	if !printed {
		fmt.Fprintln(&tooltip, "<missing docs>")
	}

	return tooltip.String()
}

// symbolWalker is an AST walker that generates the symbol table for a file in IndexSymbols().
type symbolWalker struct {
	file *file

	// This is the set of *ast.MessageNode, *ast.EnumNode, and *ast.ServiceNode that
	// we have traversed. They are used for same-file symbol resolution, and for constructing
	// the full paths of symbols.
	path []ast.Node

	// This is a prefix sum of the length of each line in file.text. This is
	// necessary for mapping a line+col value in a source position to byte offsets.
	//
	// lineSum[n] is the number of bytes on every line up to line n, including the \n
	// byte on the current line.
	lineSum []int
}

// newWalker constructs a new walker from a file, constructing any necessary book-keeping.
func newWalker(file *file) *symbolWalker {
	walker := &symbolWalker{
		file: file,
	}

	// NOTE: Don't use range here, that produces runes, not bytes.
	for i := 0; i < len(file.text); i++ {
		if file.text[i] == '\n' {
			walker.lineSum = append(walker.lineSum, i+1)
		}
	}
	walker.lineSum = append(walker.lineSum, len(file.text))

	return walker
}

func (walker *symbolWalker) Walk(node, parent ast.Node) {
	// Save the stack depth on entry, so we can undo it on exit.
	top := len(walker.path)
	defer func() { walker.path = walker.path[:top] }()

	switch node := node.(type) {
	case *ast.FileNode:
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.ImportNode:
		// Generate a symbol for the import string. This symbol points to a file,
		// not another symbol.
		symbol := walker.newSymbol(node.Name)
		import_ := new(import_)
		symbol.kind = import_
		if imported, ok := walker.file.imports[node.Name.AsString()]; ok {
			import_.file = imported
		}

	case *ast.MessageNode:
		walker.newDef(node, node.Name)
		walker.path = append(walker.path, node)
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.ExtendNode:
		walker.newRef(node.Extendee)
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.GroupNode:
		walker.newDef(node, node.Name)
		// TODO: also do the name of the generated field.
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.FieldNode:
		walker.newDef(node, node.Name)
		walker.newRef(node.FldType)
		if node.Options != nil {
			for _, option := range node.Options.Options {
				walker.Walk(option, node)
			}
		}

	case *ast.MapFieldNode:
		walker.newDef(node, node.Name)
		walker.newRef(node.MapType.KeyType)
		walker.newRef(node.MapType.ValueType)
		if node.Options != nil {
			for _, option := range node.Options.Options {
				walker.Walk(option, node)
			}
		}

	case *ast.OneofNode:
		walker.newDef(node, node.Name)
		// NOTE: oneof fields are not scoped to their oneof's name, so we can skip
		// pushing to walker.path.
		// walker.path = append(walker.path, node.Name.Val)
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.EnumNode:
		walker.newDef(node, node.Name)
		walker.path = append(walker.path, node)
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.EnumValueNode:
		walker.newDef(node, node.Name)

	case *ast.ServiceNode:
		walker.newDef(node, node.Name)
		walker.path = append(walker.path, node)
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.RPCNode:
		walker.newDef(node, node.Name)
		walker.newRef(node.Input.MessageType)
		walker.newRef(node.Output.MessageType)
		for _, decl := range node.Decls {
			walker.Walk(decl, node)
		}

	case *ast.OptionNode:
		for i, part := range node.Name.Parts {
			var next *symbol
			if part.IsExtension() {
				next = walker.newRef(part.Name)
			} else if i == 0 {
				// This lies in descriptor.proto and has to wait until we're resolving
				// cross-file references.
				next = walker.newSymbol(part.Name)
				next.kind = &reference{
					path:                []string{part.Value()},
					isNonCustomOptionIn: parent}
			} else {
				// This depends on the type of the previous symbol.
				prev := walker.file.symbols[len(walker.file.symbols)-1]
				next = walker.newSymbol(part.Name)
				next.kind = &reference{seeTypeOf: prev}
			}
			next.isOption = true
		}

		// TODO: node.Val
	}
}

// newSymbol creates a new symbol and adds it to the running list.
//
// name is the node representing the name of the symbol that can be go-to-definition'd.
func (walker *symbolWalker) newSymbol(name ast.Node) *symbol {
	symbol := &symbol{
		file: walker.file,
		name: name,
		info: walker.file.fileNode.NodeInfo(name),
	}

	walker.file.symbols = append(walker.file.symbols, symbol)
	return symbol
}

// newDef creates a new symbol for a definition, and adds it to the running list.
//
// Returns a new symbol for that definition.
func (walker *symbolWalker) newDef(node ast.Node, name *ast.IdentNode) *symbol {
	symbol := walker.newSymbol(name)
	symbol.kind = &definition{
		node: node,
		path: append(makeNestingPath(walker.path), name.Val),
	}
	return symbol
}

// newDef creates a new symbol for a name reference, and adds it to the running list.
//
// newRef performs same-file Protobuf name resolution. It searches for a partial package
// name in each enclosing scope (per walker.path). Cross-file resolution is done by
// ResolveCrossFile().
//
// Returns a new symbol for that reference.
func (walker *symbolWalker) newRef(name ast.IdentValueNode) *symbol {
	symbol := walker.newSymbol(name)
	components, absolute := symbol.ReferencePath()

	// Handle the built-in types.
	if !absolute && len(components) == 1 {
		switch components[0] {
		case "int32", "int64", "uint32", "uint64", "sint32", "sint64",
			"fixed32", "fixed64", "sfixed32", "sfixed64",
			"float", "double", "string", "bytes":
			symbol.kind = &builtin{components[0]}
			return symbol
		}
	}

	ref := new(reference)
	symbol.kind = ref

	// First, search the containing messages.
	if !absolute {
		for i := len(walker.path) - 1; i >= 0; i-- {
			message, ok := walker.path[i].(*ast.MessageNode)
			if !ok {
				continue
			}

			if findDeclByPath(message.Decls, components) != nil {
				ref.file = walker.file
				ref.path = append(makeNestingPath(walker.path[:i+1]), components...)
				return symbol
			}
		}
	}

	// If we couldn't find it within a nested message, we now try to find it at the top level.
	if !absolute && findDeclByPath(walker.file.fileNode.Decls, components) != nil {
		ref.file = walker.file
		ref.path = components
		return symbol
	}

	// Also try with the package removed.
	if path, ok := slicesext.TrimPrefix(components, symbol.file.Package()); ok {
		if findDeclByPath(walker.file.fileNode.Decls, path) != nil {
			ref.file = walker.file
			ref.path = path
			return symbol
		}
	}

	// NOTE: cross-file resolution happens elsewhere, after we have walked the whole
	// ast and dropped this file's lock.

	// If we couldn't resolve the symbol, symbol.definedIn will be nil.
	// However, for hover, it's necessary to still remember the components.
	ref.path = components
	return symbol
}

// findDeclByPath searches for a declaration node that the given path names that is nested
// among decls. This is, in effect, Protobuf name resolution within a file.
//
// Currently, this will only find *ast.MessageNode and *ast.EnumNode values.
func findDeclByPath[N ast.Node](nodes []N, path []string) ast.Node {
	if len(path) == 0 {
		return nil
	}

	for _, node := range nodes {
		switch node := ast.Node(node).(type) {
		case *ast.MessageNode:
			if node.Name.Val == path[0] {
				if len(path) == 1 {
					return node
				}
				return findDeclByPath(node.Decls, path[1:])
			}
		case *ast.GroupNode:
			// TODO: This is incorrect. The name to compare with should have
			// its first letter lowercased.
			if len(path) == 1 && node.Name.Val == path[0] {
				return node
			}

			msg := node.AsMessage()
			if msg.Name.Val == path[0] {
				if len(path) == 1 {
					return msg
				}
				return findDeclByPath(msg.Decls, path[1:])
			}

		case *ast.ExtendNode:
			if found := findDeclByPath(node.Decls, path); found != nil {
				return found
			}
		case *ast.OneofNode:
			if found := findDeclByPath(node.Decls, path); found != nil {
				return found
			}

		case *ast.EnumNode:
			if len(path) == 1 && node.Name.Val == path[0] {
				return node
			}
		case *ast.FieldNode:
			if len(path) == 1 && node.Name.Val == path[0] {
				return node
			}
		case *ast.MapFieldNode:
			if len(path) == 1 && node.Name.Val == path[0] {
				return node
			}
		}
	}

	return nil
}

// compareRanges compares two ranges for lexicographic ordering.
func comparePositions(a, b protocol.Position) int {
	diff := int(a.Line) - int(b.Line)
	if diff == 0 {
		return int(a.Character) - int(b.Character)
	}
	return diff
}

// makeNestingPath converts a path composed of messages, enums, and services into a path
// composed of their names.
func makeNestingPath(path []ast.Node) []string {
	return slicesext.Map(path, func(node ast.Node) string {
		switch node := node.(type) {
		case *ast.MessageNode:
			return node.Name.Val
		case *ast.EnumNode:
			return node.Name.Val
		case *ast.ServiceNode:
			return node.Name.Val
		default:
			return "<error>"
		}
	})
}

func infoToRange(info ast.NodeInfo) protocol.Range {
	return protocol.Range{
		// NOTE: protocompile uses 1-indexed lines and columns (as most compilers do) but bizarrely
		// the LSP protocol wants 0-indexed lines and columns, which is a little weird.
		//319
		// FIXME: the LSP protocol defines positions in terms of UTF-16, so we will need
		// to sort that out at some point.
		Start: protocol.Position{
			Line:      uint32(info.Start().Line) - 1,
			Character: uint32(info.Start().Col) - 1,
		},
		End: protocol.Position{
			Line:      uint32(info.End().Line) - 1,
			Character: uint32(info.End().Col) - 1,
		},
	}
}
