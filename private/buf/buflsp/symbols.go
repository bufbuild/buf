// Copyright 2023 Buf Technologies, Inc.
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
	"strings"

	"github.com/bufbuild/protocompile/ast"
	"go.lsp.dev/protocol"
)

var (
	// Well known options.
	fileOptionsRefName      = symbolRefName{"", "google", "protobuf", "FileOptions"}
	messageOptionsRefName   = symbolRefName{"", "google", "protobuf", "MessageOptions"}
	fieldOptionsRefName     = symbolRefName{"", "google", "protobuf", "FieldOptions"}
	enumOptionsRefName      = symbolRefName{"", "google", "protobuf", "EnumOptions"}
	enumValueOptionsRefName = symbolRefName{"", "google", "protobuf", "EnumValueOptions"}
	serviceOptionsRefName   = symbolRefName{"", "google", "protobuf", "ServiceOptions"}
	methodOptionsRefName    = symbolRefName{"", "google", "protobuf", "MethodOptions"}

	// All well known options.
	wellKnownOptions = []symbolRefName{
		fileOptionsRefName,
		messageOptionsRefName,
		fieldOptionsRefName,
		enumOptionsRefName,
		enumValueOptionsRefName,
		serviceOptionsRefName,
		methodOptionsRefName,
	}
)

// A reference to a symbol that may be absolute or relative.
// If the reference is absolute, the first element is the empty string.
// Never otherwise contains the empty string.
type symbolRefName []string

func (sr symbolRefName) isAbsolute() bool {
	return len(sr) > 0 && sr[0] == ""
}

// The fully qualified name of a symbol.
// Always absolute, but never starts with the empty string.
// Never contains the empty string.
type symbolName []string

// A import statement in a proto file.
type importEntry struct {
	docURI   protocol.DocumentURI
	node     *ast.ImportNode
	isPublic bool
}

// A symbol generated from a parsed proto file.
type symbolEntry struct {
	refName  symbolRefName
	node     ast.Node
	file     protocol.DocumentURI
	extendee ast.IdentValueNode
}

func (se symbolEntry) name() symbolName {
	return symbolName(se.refName[1:])
}

func (se symbolEntry) ref() symbolRefName {
	return se.refName
}

// The scope associated with a symbol entry.
type symbolScopeEntry struct {
	symbol   *symbolEntry
	children []*symbolScopeEntry
	typeRefs []typeRefEntry
	options  []optionEntry
}

type typeRefEntry struct {
	node ast.IdentValueNode
}

type optionEntry struct {
	node     *ast.OptionNode
	extendee symbolRefName
}

func getCompletionItem(symbol *symbolEntry) protocol.CompletionItem {
	result := protocol.CompletionItem{
		Label:  symbol.name()[len(symbol.name())-1],
		Detail: strings.Join(symbol.name()[:len(symbol.name())-1], "."),
		Kind:   protocol.CompletionItemKindModule,
	}
	switch symbol.node.(type) {
	case *ast.MessageNode:
		result.Kind = protocol.CompletionItemKindClass
	case *ast.EnumNode:
		result.Kind = protocol.CompletionItemKindEnum
	case *ast.FieldNode:
		result.Kind = protocol.CompletionItemKindField
	case *ast.MapFieldNode:
		result.Kind = protocol.CompletionItemKindField
	case *ast.ServiceNode:
		result.Kind = protocol.CompletionItemKindInterface
	case *ast.RPCNode:
		result.Kind = protocol.CompletionItemKindMethod
	default:
	}
	return result
}

type docSymbolGen struct {
	fileEntry    *fileEntry
	pkg          []string
	scope        []string
	symbolScopes []*symbolScopeEntry
}

func (g *docSymbolGen) getSymbolRefName(name string) symbolRefName {
	symbolRefName := make([]string, len(g.pkg)+len(g.scope)+2)
	symbolRefName[0] = ""
	copy(symbolRefName[1:], g.pkg)
	copy(symbolRefName[len(g.pkg)+1:], g.scope)
	symbolRefName[len(symbolRefName)-1] = name
	return symbolRefName
}

func (g *docSymbolGen) addTypeSymbol(name string, node ast.Node, file protocol.DocumentURI) *symbolEntry {
	result := &symbolEntry{
		file:    file,
		refName: g.getSymbolRefName(name),
		node:    node,
	}
	g.fileEntry.typeSymbols = append(g.fileEntry.typeSymbols, result)
	return result
}

func (g *docSymbolGen) addFieldSymbol(name string, node ast.Node, file protocol.DocumentURI, extendee ast.IdentValueNode) {
	g.fileEntry.fieldSymbols = append(g.fileEntry.fieldSymbols, &symbolEntry{
		file:     file,
		refName:  g.getSymbolRefName(name),
		node:     node,
		extendee: extendee,
	})
}

func (g *docSymbolGen) addServiceSymbol(name string, node ast.Node, file protocol.DocumentURI) *symbolEntry {
	result := &symbolEntry{
		file:    file,
		refName: g.getSymbolRefName(name),
		node:    node,
	}
	return result
}

func (g *docSymbolGen) startSymbolScope(symbol *symbolEntry) {
	entry := &symbolScopeEntry{
		symbol: symbol,
	}
	if len(g.symbolScopes) > 0 {
		g.symbolScopes[len(g.symbolScopes)-1].children = append(g.symbolScopes[len(g.symbolScopes)-1].children, entry)
	} else {
		g.fileEntry.symbolScopes = append(g.fileEntry.symbolScopes, entry)
	}
	g.symbolScopes = append(g.symbolScopes, entry)
}

func (g *docSymbolGen) endSymbolScope() {
	g.symbolScopes = g.symbolScopes[:len(g.symbolScopes)-1]
}

func (g *docSymbolGen) addTypeRef(node ast.IdentValueNode) {
	if len(g.symbolScopes) == 0 {
		panic("no code pos entry")
	}
	curPos := g.symbolScopes[len(g.symbolScopes)-1]
	curPos.typeRefs = append(curPos.typeRefs, typeRefEntry{
		node: node,
	})
}

func (g *docSymbolGen) getFileSymbols(node *ast.FileNode) []protocol.DocumentSymbol {
	var symbols []protocol.DocumentSymbol
	for _, elem := range node.Decls {
		switch elem := elem.(type) {
		case *ast.PackageNode:
			switch ident := elem.Name.(type) {
			case *ast.IdentNode:
				g.pkg = []string{ident.Val}
			case *ast.CompoundIdentNode:
				g.pkg = []string{}
				for _, component := range ident.Components {
					g.pkg = append(g.pkg, component.Val)
				}
			}
		case *ast.ImportNode:
			g.fileEntry.imports = append(g.fileEntry.imports, &importEntry{
				node:     elem,
				isPublic: elem.Public != nil,
			})
		case *ast.ServiceNode:
			symbols = append(symbols, g.getServiceSymbols(elem))
		case *ast.MessageNode:
			symbols = append(symbols, g.getMessageSymbols(elem))
		case *ast.EnumNode:
			symbols = append(symbols, g.getEnumSymbols(elem))
		case *ast.ExtendNode:
			for _, decl := range elem.Decls {
				switch declNode := decl.(type) {
				case *ast.FieldNode:
					g.addFieldSymbol(declNode.Name.Val, declNode, g.fileEntry.document.URI, elem.Extendee)
				case *ast.GroupNode:
					g.addTypeSymbol(declNode.Name.Val, declNode, g.fileEntry.document.URI)
				}
			}
		}
	}
	return symbols
}

func (g *docSymbolGen) getServiceSymbols(node *ast.ServiceNode) protocol.DocumentSymbol {
	g.startSymbolScope(g.addServiceSymbol(node.Name.Val, node, g.fileEntry.document.URI))
	defer g.endSymbolScope()
	g.scope = append(g.scope, node.Name.Val)
	result := protocol.DocumentSymbol{
		Name:           node.Name.Val,
		Kind:           protocol.SymbolKindInterface,
		Range:          g.fileEntry.extentRange(node.Start(), node.End()),
		SelectionRange: g.fileEntry.tokenRange(node.Name.Token()),
	}
	for _, elem := range node.Decls {
		switch elem := elem.(type) {
		case *ast.OptionNode:
			g.addOption(elem, serviceOptionsRefName)
		case *ast.RPCNode:
			result.Children = append(result.Children, g.getRPCSymbols(elem))
		}
	}
	g.scope = g.scope[:len(g.scope)-1]
	return result
}

func (g *docSymbolGen) getRPCSymbols(node *ast.RPCNode) protocol.DocumentSymbol {
	g.addTypeRef(node.Input.MessageType)
	g.addTypeRef(node.Output.MessageType)
	result := protocol.DocumentSymbol{
		Name:           node.Name.Val,
		Kind:           protocol.SymbolKindFunction,
		Range:          g.fileEntry.extentRange(node.Start(), node.End()),
		SelectionRange: g.fileEntry.tokenRange(node.Name.Token()),
	}
	for _, elem := range node.Decls {
		if elem, ok := elem.(*ast.OptionNode); ok {
			g.addOption(elem, methodOptionsRefName)
		}
	}
	return result
}

func (g *docSymbolGen) getMessageSymbols(node *ast.MessageNode) protocol.DocumentSymbol {
	g.startSymbolScope(g.addTypeSymbol(node.Name.Val, node, g.fileEntry.document.URI))
	defer g.endSymbolScope()
	g.scope = append(g.scope, node.Name.Val)
	result := protocol.DocumentSymbol{
		Name:           node.Name.Val,
		Kind:           protocol.SymbolKindClass,
		Range:          g.fileEntry.extentRange(node.Start(), node.End()),
		SelectionRange: g.fileEntry.tokenRange(node.Name.Token()),
	}
	for _, elem := range node.Decls {
		switch elem := elem.(type) {
		case *ast.MessageNode:
			result.Children = append(result.Children, g.getMessageSymbols(elem))
		case *ast.EnumNode:
			result.Children = append(result.Children, g.getEnumSymbols(elem))
		case *ast.OptionNode:
			g.addOption(elem, messageOptionsRefName)
		case *ast.OneofNode:
			for _, subelem := range elem.Decls {
				if subelem, ok := subelem.(*ast.FieldNode); ok {
					result.Children = append(result.Children, g.getFieldSymbols(subelem))
				}
			}
		case *ast.FieldNode:
			result.Children = append(result.Children, g.getFieldSymbols(elem))
		case *ast.MapFieldNode:
			result.Children = append(result.Children, g.getMapFieldSymbols(elem))
		}
	}

	g.scope = g.scope[:len(g.scope)-1]
	return result
}

func (g *docSymbolGen) getFieldSymbols(node *ast.FieldNode) protocol.DocumentSymbol {
	g.addFieldSymbol(node.Name.Val, node, g.fileEntry.document.URI, nil)
	g.addTypeRef(node.FldType)
	result := protocol.DocumentSymbol{
		Name:           node.Name.Val,
		Kind:           protocol.SymbolKindField,
		Range:          g.fileEntry.extentRange(node.Start(), node.End()),
		SelectionRange: g.fileEntry.tokenRange(node.Name.Token()),
	}
	if node.Options != nil {
		for _, option := range node.Options.Options {
			g.addOption(option, fieldOptionsRefName)
		}
	}
	return result
}

func (g *docSymbolGen) getMapFieldSymbols(node *ast.MapFieldNode) protocol.DocumentSymbol {
	g.addFieldSymbol(node.Name.Val, node, g.fileEntry.document.URI, nil)
	g.addTypeRef(node.MapType.KeyType)
	g.addTypeRef(node.MapType.ValueType)
	result := protocol.DocumentSymbol{
		Name:           node.Name.Val,
		Kind:           protocol.SymbolKindField,
		Range:          g.fileEntry.extentRange(node.Start(), node.End()),
		SelectionRange: g.fileEntry.tokenRange(node.Name.Token()),
	}
	if node.Options != nil {
		for _, option := range node.Options.Options {
			g.addOption(option, fieldOptionsRefName)
		}
	}
	return result
}

func (g *docSymbolGen) getEnumSymbols(node *ast.EnumNode) protocol.DocumentSymbol {
	g.startSymbolScope(g.addTypeSymbol(node.Name.Val, node, g.fileEntry.document.URI))
	defer g.endSymbolScope()
	g.scope = append(g.scope, node.Name.Val)
	result := protocol.DocumentSymbol{
		Name:           node.Name.Val,
		Kind:           protocol.SymbolKindEnum,
		Range:          g.fileEntry.extentRange(node.Start(), node.End()),
		SelectionRange: g.fileEntry.tokenRange(node.Name.Token()),
	}
	for _, elem := range node.Decls {
		switch elem := elem.(type) {
		case *ast.OptionNode:
			g.addOption(elem, enumOptionsRefName)
		case *ast.EnumValueNode:
			result.Children = append(result.Children, g.getEnumValueSymbols(elem))
		}
	}
	g.scope = g.scope[:len(g.scope)-1]
	return result
}

func (g *docSymbolGen) getEnumValueSymbols(node *ast.EnumValueNode) protocol.DocumentSymbol {
	if node.Options != nil {
		for _, option := range node.Options.Options {
			g.addOption(option, enumValueOptionsRefName)
		}
	}
	return protocol.DocumentSymbol{
		Name:           node.Name.Val,
		Kind:           protocol.SymbolKindEnumMember,
		Range:          g.fileEntry.extentRange(node.Start(), node.End()),
		SelectionRange: g.fileEntry.tokenRange(node.Name.Token()),
	}
}

func (g *docSymbolGen) addOption(option *ast.OptionNode, extendee []string) {
	if len(g.symbolScopes) == 0 {
		panic("no code pos entry")
	}
	curSymbolScope := g.symbolScopes[len(g.symbolScopes)-1]
	curSymbolScope.options = append(curSymbolScope.options, optionEntry{
		node:     option,
		extendee: extendee,
	})
}
