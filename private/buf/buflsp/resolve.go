// Copyright 2020-2023 Buf Technologies, Inc.
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
	"context"

	"github.com/bufbuild/protocompile/ast"
	"go.lsp.dev/protocol"
)

type symbolRef struct {
	entry   *fileEntry    // The file entry that contains the reference.
	refName symbolRefName // The reference itself
	scope   symbolName    // The scope containing the reference.

	isField  bool       // True if the reference is to a field.
	extendee *symbolRef // The extendee type, if the reference is an extension field.
}

// Resolve the name and scope to candidate symbol names, and then find the symbols
// reachable from the given file and it's direct imports.
func (b *BufLsp) findSymbols(ref *symbolRef) []*symbolEntry {
	for _, candidate := range findCandidates(ref.refName, ref.scope) {
		symbols := b.findSymbolsForCandidate(ref, candidate)
		if len(symbols) != 0 {
			return symbols
		}
	}
	return nil
}

// Searches for the given symbol by name, in the given file and its direct imports.
func (b *BufLsp) findSymbolsForCandidate(ref *symbolRef, candidate symbolName) []*symbolEntry {
	// Include local symbols
	result := ref.entry.findSymbols(candidate, ref.isField)

	// Include symbols from direct imports
	for _, importEntry := range ref.entry.imports {
		result = append(result, b.findImportedSymbolsForCandidate(ref, importEntry, candidate)...)
	}

	return result
}

func (b *BufLsp) findImportedSymbolsForCandidate(ref *symbolRef, importEntry *importEntry, candidate symbolName) []*symbolEntry {
	var result []*symbolEntry
	if importEntry.docURI != "" {
		if importFile, ok := b.fileCache[importEntry.docURI.Filename()]; ok {
			result = append(result, importFile.findSymbols(candidate, ref.isField)...)
			for _, subImportEntry := range importFile.imports {
				if subImportEntry.isPublic {
					// Include symbols from indirect public imports
					result = append(result, b.findImportedSymbolsForCandidate(ref, subImportEntry, candidate)...)
				}
			}
		}
	}
	return result
}

func (b *BufLsp) findReferencedSymbols(ctx context.Context, entry *fileEntry, pos ast.SourcePos) []*symbolEntry {
	symbolScope := entry.findSymbolScope(pos)
	if symbolScope == nil {
		return nil // No references exist in the top level scope.
	}

	result := b.findReferenceAt(ctx, entry, symbolScope, pos)
	if result == nil || len(result.refName) == 0 {
		return nil
	}
	return b.findSymbols(result)
}

func (b *BufLsp) findSymbolLocation(symbol *symbolEntry) protocol.Location {
	if symbolFile, ok := b.fileCache[symbol.file.Filename()]; ok {
		return protocol.Location{
			URI:   symbol.file,
			Range: symbolFile.nodeLocation(symbol.node),
		}
	}
	// Should never happen.
	return protocol.Location{}
}

// Find the location of any symbols that are referenced at the given position.
func (b *BufLsp) findReferencedDefLoc(ctx context.Context, entry *fileEntry, pos ast.SourcePos) []protocol.Location {
	symbols := b.findReferencedSymbols(ctx, entry, pos)
	result := make([]protocol.Location, len(symbols))
	for i, symbol := range symbols {
		result[i] = b.findSymbolLocation(symbol)
	}
	return result
}

func getRefName(identValue ast.IdentValueNode) symbolRefName {
	switch ident := identValue.(type) {
	case *ast.IdentNode:
		return []string{ident.Val}
	case *ast.CompoundIdentNode:
		result := []string{}
		if ident.LeadingDot != nil {
			result = append(result, "")
		}
		for _, component := range ident.Components {
			result = append(result, component.Val)
		}
		return result
	}
	return nil
}

func (b *BufLsp) findReferenceAt(ctx context.Context, entry *fileEntry, symbolScope *symbolScopeEntry, pos ast.SourcePos) *symbolRef {
	// Check if the position is in a type reference.
	for _, ref := range symbolScope.typeRefs {
		if !entry.containsPos(ref.node, pos) {
			continue
		}
		switch node := ref.node.(type) {
		case *ast.IdentNode:
			return &symbolRef{
				entry:   entry,
				refName: symbolRefName{node.Val},
				scope:   symbolScope.symbol.name(),
			}
		case *ast.CompoundIdentNode:
			result := &symbolRef{
				entry: entry,
				scope: symbolScope.symbol.name(),
			}
			for _, component := range node.Components {
				result.refName = append(result.refName, component.Val)
				if entry.containsOrPastPos(component, pos) {
					return result
				}
			}
			return result
		default:
			return nil
		}
	}

	// Check if the position is in an option.
	for _, option := range symbolScope.options {
		if !entry.containsPos(option.node, pos) {
			continue
		}
		result := b.resolveWellKnownExtendee(ctx, option.extendee)
		for _, field := range option.node.Name.Parts {
			var found bool
			result, found = b.resolveFieldRef(entry, result, field, pos)
			if found || result == nil {
				return result
			}
			result = b.resolveFieldType(result)
			if result == nil {
				return nil
			}
		}
		result, _ = b.findReferenceInValueNode(entry, result, option.node.Val, pos)
		return result
	}

	return nil
}

func (b *BufLsp) findReferenceInValueNode(entry *fileEntry, ref *symbolRef, valueNode ast.ValueNode, pos ast.SourcePos) (*symbolRef, bool) {
	if ref == nil {
		return nil, false
	}
	if !entry.containsPos(valueNode, pos) {
		return nil, false
	}
	msgNode, ok := valueNode.(*ast.MessageLiteralNode)
	if !ok {
		return nil, false
	}

	result := &symbolRef{
		entry:   ref.entry,
		refName: ref.refName,
		isField: true,
	}
	for _, element := range msgNode.Elements {
		if !entry.containsPos(element, pos) {
			continue
		}
		var found bool
		result, found = b.resolveFieldRef(entry, result, element.Name, pos)
		if found || result == nil {
			return result, found
		}
		result = b.resolveFieldType(result)
		return b.findReferenceInValueNode(entry, result, element.Val, pos)
	}
	// We aren't on any field name, so assume we are starting a new field name.
	result.refName = append(result.refName, "")
	return result, true
}

// Resolve fieldRef (a field symbolRefName) in the context of messageRef (its containing message symbolRefName).
//
// Returns the resolved field and if the position is in the field name.
func (b *BufLsp) resolveFieldRef(entry *fileEntry, messageRef *symbolRef, fieldRefNode *ast.FieldReferenceNode, pos ast.SourcePos) (*symbolRef, bool) {
	if fieldRefNode.IsExtension() {
		return b.resolveExtField(entry, messageRef, fieldRefNode, pos)
	} else {
		return b.resolveNormalField(entry, messageRef, fieldRefNode, pos)
	}
}

func (b *BufLsp) resolveNormalField(entry *fileEntry, messageRef *symbolRef, fieldRefNode *ast.FieldReferenceNode, pos ast.SourcePos) (*symbolRef, bool) {
	if fieldName, ok := fieldRefNode.Name.(*ast.IdentNode); ok {
		fieldRef := &symbolRef{
			entry:   messageRef.entry,
			isField: true,
		}
		fieldRef.refName = append(fieldRef.refName, messageRef.refName...)
		fieldRef.refName = append(fieldRef.refName, fieldName.Val)
		return fieldRef, entry.containsOrPastPos(fieldName, pos)
	}
	// Unreachable
	return nil, false
}

// Returns the resolved field and if the position is in the field name.
func (b *BufLsp) resolveExtField(entry *fileEntry, messageRef *symbolRef, fieldRefNode *ast.FieldReferenceNode, pos ast.SourcePos) (*symbolRef, bool) {
	symbols := b.findSymbols(messageRef)
	if len(symbols) > 0 {
		messageRef.refName = symbols[0].ref()
	}

	switch name := fieldRefNode.Name.(type) {
	case *ast.IdentNode:
		result := &symbolRef{
			entry:    entry,
			refName:  symbolRefName{name.Val},
			scope:    entry.pkg,
			isField:  true,
			extendee: messageRef,
		}
		return result, entry.containsOrPastPos(name, pos)
	case *ast.CompoundIdentNode:
		result := &symbolRef{
			entry:    entry,
			scope:    entry.pkg,
			isField:  true,
			extendee: messageRef,
		}
		if name.LeadingDot != nil {
			result.refName = symbolRefName{""}
		}
		for _, component := range name.Components {
			result.refName = append(result.refName, component.Val)
			if entry.containsOrPastPos(component, pos) {
				return result, true
			}
		}
		return result, false
	}
	// Should be unreachable.
	return nil, false
}

func (b *BufLsp) resolveWellKnownExtendee(ctx context.Context, extendee symbolRefName) *symbolRef {
	// Check if it is a reference to a well known type Option type.
	if descEntry, err := b.loadWktFile(ctx, wktSourceDir+"descriptor.proto"); err == nil {
		defer func() { b.derefFileEntry(descEntry) }()
		return &symbolRef{
			entry:   descEntry,
			refName: extendee,
		}
	}
	return nil
}

func (b *BufLsp) resolveFieldType(ref *symbolRef) *symbolRef {
	symbols := b.findSymbols(ref)
	if len(symbols) == 0 {
		return nil
	}

	symbol := symbols[0]
	if symbolFile, ok := b.fileCache[symbol.file.Filename()]; ok {
		if node, ok := symbol.node.(*ast.FieldNode); ok {
			fieldRef := &symbolRef{
				entry:   symbolFile,
				refName: getRefName(node.FldType),
				scope:   symbol.name()[:len(symbol.name())-1],
				isField: true,
			}
			symbols := b.findSymbols(fieldRef)
			if len(symbols) == 0 {
				return nil
			}
			symbol = symbols[0]
			return &symbolRef{
				entry:   symbolFile,
				refName: symbol.ref(),
			}
		}
	}
	return nil
}
