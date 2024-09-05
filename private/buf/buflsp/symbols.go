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
// In particular, this file handles semantic information in files that have been
// *opened by the editor*, and thus do not need references to Buf modules to find.
// See imports.go for that part of the LSP.

package buflsp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/protocompile/ast"
	"go.lsp.dev/protocol"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
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

	// The node containing the entity defined by this symbol, if it is a definition.
	// For example, if this symbol represents the definition site of a message, this
	// node is the *ast.MessageNode.
	defNode ast.Node

	// The full path of this symbol (as far as Protobuf's semantics are concerned),
	// not including the package.
	//
	// May be nil if the symbol does not have a path (such as an import).
	path []string

	// The file that this symbol is believed to be defined in. Nil if we don't know
	// where this symbol is defined.
	//
	// We do not store where in the file; that is only resolved when we do go-to-definition.
	// Why? Because the symbol may have moved due to edits in the file!
	// NOTE: We currently do not correctly handle symbols migrating between files.
	definedIn *file

	// Set if this symbol's "definition" is the language itself, i.e. this is something
	// like int32.
	isBuiltin bool
}

// IndexSymbols processes the AST of a file and generates symbols for each symbol in
// the document.
func (file *file) IndexSymbols(ctx context.Context) {
	_, span := file.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)

	// Throw away all the old symbols. Unlike other indexing functions, we rebuild
	// symbols unconditionally.
	file.symbols = nil

	// Generate new symbols.
	newWalker(file).Walk(file.ast)
	// Finally, sort the symbols in position order, with shorter symbols sorting smaller.
	slices.SortFunc(file.symbols, func(s1, s2 *symbol) int {
		diff := s1.info.Start().Offset - s2.info.Start().Offset
		if diff == 0 {
			return s1.info.End().Offset - s2.info.End().Offset
		}
		return diff
	})

	for _, symbol := range file.symbols {
		file.server.logger.Debug(
			"symbol",
			zap.String("uri", string(file.uri)),
			zap.Strings("path", symbol.path),
			zap.Reflect("start", symbol.info.Start()),
			zap.Reflect("end", symbol.info.End()),
		)
	}
}

// SymbolAt finds a symbol in this file at the given cursor position, if one exists.
//
// Returns nil if no symbol is found.
func (file *file) SymbolAt(ctx context.Context, cursor protocol.Position) *symbol {
	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)

	// Binary search for the symbol whose start is before or equal to cursor.
	idx, found := slices.BinarySearchFunc(file.symbols, cursor, func(sym *symbol, cursor protocol.Position) int {
		return comparePositions(sym.Range().Start, cursor)
	})
	file.server.logger.Sugar().Debugf("found symbol: %v, %v", idx, found)
	if !found {
		if idx == 0 {
			return nil
		}
		idx--
	}

	symbol := file.symbols[idx]

	// Check that cursor is before the end of the symbol.
	if comparePositions(symbol.Range().End, cursor) <= 0 {
		return nil
	}

	return symbol
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

func (walker *symbolWalker) Walk(node ast.Node) {
	// Save the stack depth on entry, so we can undo it on exit.
	top := len(walker.path)
	defer func() { walker.path = walker.path[:top] }()

	switch node := node.(type) {
	case *ast.FileNode:
		for _, decl := range node.Decls {
			walker.Walk(decl)
		}

	case *ast.ImportNode:
		// Generate a symbol for the import string. This symbol points to a file,
		// not another symbol.
		symbol := walker.newSymbol(nil, node.Name)
		if imported, ok := walker.file.imports[node]; ok {
			symbol.definedIn = imported
		}

	case *ast.MessageNode:
		walker.newSymbol(node, node.Name)
		walker.path = append(walker.path, node)
		for _, decl := range node.Decls {
			walker.Walk(decl)
		}

	case *ast.FieldNode:
		walker.newSymbol(node, node.Name)
		walker.newRef(node.FldType)

	case *ast.MapFieldNode:
		walker.newSymbol(node, node.Name)
		walker.newRef(node.MapType.KeyType)
		walker.newRef(node.MapType.ValueType)

	case *ast.OneofNode:
		walker.newSymbol(node, node.Name)
		// NOTE: oneof fields are not scoped to their oneof's name, so we can skip
		// pushing to walker.path.
		// walker.path = append(walker.path, node.Name.Val)
		for _, decl := range node.Decls {
			walker.Walk(decl)
		}

	case *ast.EnumNode:
		walker.newSymbol(node, node.Name)
		walker.path = append(walker.path, node)
		for _, decl := range node.Decls {
			walker.Walk(decl)
		}

	case *ast.EnumValueNode:
		walker.newSymbol(node, node.Name)

	case *ast.ServiceNode:
		walker.newSymbol(node, node.Name)
		walker.path = append(walker.path, node)
		for _, decl := range node.Decls {
			walker.Walk(decl)
		}

	case *ast.RPCNode:
		walker.newSymbol(node, node.Name)
		walker.newRef(node.Input.MessageType)
		walker.newRef(node.Output.MessageType)
	}
}

// newSymbol creates a new symbol and adds it to the running list.
//
// If node is nil, this is a symbol for a reference; otherwise, it is a symbol for a
// definition, meaning that this symbol defines itself.
//
// name is the node representing the name of the symbol that can be go-to-definition'd.
// If name is an *ast.IdentNode, it will be used for constructing this symbol's path;
// otherwise path will be left nil.
func (walker *symbolWalker) newSymbol(node, name ast.Node) *symbol {
	symbol := &symbol{
		file:    walker.file,
		name:    name,
		info:    walker.file.ast.NodeInfo(name),
		defNode: node,
	}
	if node != nil {
		symbol.definedIn = symbol.file
	}

	if ident, ok := name.(*ast.IdentNode); ok {
		symbol.path = append(makeNestingPath(walker.path), ident.Val)
	}

	walker.file.symbols = append(walker.file.symbols, symbol)
	return symbol
}

// newRef performs Protobuf name resolution. It searches for a partial package
// name in each enclosing scope (per walker.path) and then searches the imports.
//
// Returns a new symbol for that reference.
func (walker *symbolWalker) newRef(name ast.IdentValueNode) *symbol {
	var (
		components []string
		isAbsolute bool
	)
	switch name := name.(type) {
	case *ast.IdentNode:
		components = []string{name.Val}
	case *ast.CompoundIdentNode:
		components = slicesext.Map(name.Components, func(name *ast.IdentNode) string { return name.Val })
		isAbsolute = name.LeadingDot != nil
	}

	symbol := walker.newSymbol(nil, name)

	// Handle the built-in types.
	if !isAbsolute && len(components) == 1 {
		switch components[0] {
		case "int32", "int64", "uint32", "uint64", "sint32", "sint64",
			"fixed32", "fixed64", "sfixed32", "sfixed64",
			"float", "double", "string", "bytes":
			symbol.path = components
			symbol.isBuiltin = true
			return symbol
		}
	}

	// First, search the containing messages.
	if !isAbsolute {
		for i := len(walker.path) - 1; i >= 0; i-- {
			message, ok := walker.path[i].(*ast.MessageNode)
			if !ok {
				continue
			}

			if findDeclByPath(message.Decls, components) != nil {
				symbol.path = append(makeNestingPath(walker.path[:i+1]), components...)
				symbol.definedIn = walker.file
				return symbol
			}
		}
	}

	// If we couldn't find it within a nested message, we now try to find it at the top level.
	if findDeclByPath(walker.file.ast.Decls, components) != nil {
		symbol.path = components
		symbol.definedIn = walker.file
		return symbol
	}

	// Also try with the package removed.
	if path, ok := slicesext.TrimPrefix(components, symbol.file.Package()); ok {
		if findDeclByPath(walker.file.ast.Decls, path) != nil {
			symbol.path = path
			symbol.definedIn = walker.file
			return symbol
		}
	}

	// If that didn't work, we search the imports.
	if walker.file.imports != nil {
		for _, imported := range walker.file.imports {
			// Remove a leading pkg from components.
			path, ok := slicesext.TrimPrefix(components, imported.Package())
			if !ok {
				continue
			}

			if findDeclByPath(imported.ast.Decls, path) != nil {
				symbol.path = path
				symbol.definedIn = imported
				return symbol
			}
		}
	}

	// If we couldn't resolve the symbol, symbol.definedIn will be nil.
	// However, for hover, it's necessary to still remember the components.
	symbol.path = components

	return symbol
}

// findDeclByPath searches for a declaration node that the given path names that is nested
// among decls. This is, in effect, Protobuf name resolution within a file.
//
// Currently, this will only find *ast.MessageNode and *ast.EnumNode values.
func findDeclByPath[Decl ast.Node](decls []Decl, path []string) ast.Node {
	var msgDecls []ast.MessageElement
	for _, decl := range decls {
		if decl, ok := ast.Node(decl).(ast.MessageElement); ok {
			msgDecls = append(msgDecls, decl)
		}
	}

	var node ast.Node
outer:
	for i, component := range path {
		for _, decl := range msgDecls {
			// Looking for either a message or an enum with the name component.
			switch decl := ast.Node(decl).(type) {
			case *ast.MessageNode:
				if decl.Name.Val == component {
					msgDecls = decl.Decls
					node = decl
					continue outer
				}
			case *ast.EnumNode:
				// This must be the last element in component, because enums can't
				// have nested things (yet).
				if i == len(path)-1 && decl.Name.Val == component {
					node = decl
					continue outer
				}
			}
		}

		// If we made it here, we found no decl in msgDecls that matches component,
		// so we failed.
		return nil
	}

	return node
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

// Range constructs an LSP protocol code range for this symbol.
func (symbol *symbol) Range() protocol.Range {
	return protocol.Range{
		// NOTE: protocompile uses 1-indexed lines and columns (as most compilers do) but bizarrely
		// the LSP protocol wants 0-indexed lines and columns, which is a little weird.
		//319
		// FIXME: the LSP protocol defines positions in terms of UTF-16, so we will need
		// to sort that out at some point.
		Start: protocol.Position{
			Line:      uint32(symbol.info.Start().Line) - 1,
			Character: uint32(symbol.info.Start().Col) - 1,
		},
		End: protocol.Position{
			Line:      uint32(symbol.info.End().Line) - 1,
			Character: uint32(symbol.info.End().Col) - 1,
		},
	}
}

// Definition looks up the definition of this symbol, if known.
func (symbol *symbol) Definition() *symbol {
	if symbol.isBuiltin || symbol.definedIn == nil {
		return nil
	}

	for _, def := range symbol.definedIn.symbols {
		if def.defNode == nil || !slices.Equal(symbol.path, def.path) {
			continue
		}
		return def
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

// Definition is the entry point for hover inlays.
func (server *server) Hover(
	ctx context.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	if err := server.checkInit(); err != nil {
		return nil, err
	}

	file := server.files.FindOrCreate(ctx, params.TextDocument.URI, false)
	symbol := file.SymbolAt(ctx, params.Position)
	if symbol == nil || symbol.path == nil {
		return nil, nil
	}

	var tooltip strings.Builder
	if symbol.isBuiltin {
		fmt.Fprintf(&tooltip, "```proto\nbuiltin %s\n```\n", symbol.path[0])
		fmt.Fprintf(&tooltip, "Builtin Protobuf type.")
	} else if def := symbol.Definition(); def != nil {
		pkg := "<empty>"
		if node := def.file.pkg; node != nil {
			pkg = string(node.Name.AsIdentifier())
		}

		fmt.Fprintf(&tooltip, "```proto\n%s.%s\n```\n", pkg, strings.Join(symbol.path, "."))

		info := def.file.ast.NodeInfo(def.defNode)
		allComments := []ast.Comments{info.LeadingComments(), info.TrailingComments()}
		for _, comments := range allComments {
			for i := 0; i < comments.Len(); i++ {
				comment := comments.Index(i).RawText()
				if strings.HasPrefix(comment, "//") {
					comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))
				} else {
					comment = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(comment, "/*"), "*/"))
				}
				// No need to process Markdown in comment; this Just Works!
				fmt.Fprintln(&tooltip, comment)
			}
		}
	} else {
		fmt.Fprintf(&tooltip, "```proto\n<unknown>.%s\n```\n", strings.Join(symbol.path, "."))
		fmt.Fprintf(&tooltip, "*could not resolve type*")
	}

	// Need to spill this here because Hover.Range is a pointer...
	range_ := symbol.Range()
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: tooltip.String(),
		},
		Range: &range_,
	}, nil
}

// Definition is the entry point for go-to-definition.
func (server *server) Definition(
	ctx context.Context,
	params *protocol.DefinitionParams,
) ([]protocol.Location, error) {
	if err := server.checkInit(); err != nil {
		return nil, err
	}

	file := server.files.FindOrCreate(ctx, params.TextDocument.URI, false)
	symbol := file.SymbolAt(ctx, params.Position)
	if symbol == nil {
		return nil, nil
	}

	if symbol.path == nil && symbol.definedIn != nil {
		// This is an import, we just want to jump to the file.
		return []protocol.Location{{
			URI: symbol.definedIn.uri,
		}}, nil
	}

	def := symbol.Definition()
	if def == nil {
		return nil, nil
	}

	return []protocol.Location{{
		URI:   def.file.uri,
		Range: def.Range(),
	}}, nil
}
