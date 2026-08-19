// Copyright 2020-2026 Buf Technologies, Inc.
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

// This file defines the reverse index from symbol definitions to their references.

package buflsp

import (
	"cmp"
	"slices"

	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ir"
	"go.lsp.dev/protocol"
)

// referenceKey identifies a symbol definition site.
//
// The key is the absolute local path of the file declaring the symbol plus the symbol's
// fully-qualified name. Keying on the definition's path, rather than on a resolved
// [symbol] pointer, means a reference can be recorded before the declaring file has been
// indexed. Including the path, rather than keying on the full name alone, keeps two
// workspaces that legitimately declare the same fully-qualified name from cross-linking.
type referenceKey struct {
	path     string
	fullName ir.FullName
}

// newReferenceKey builds the key for the definition an unresolved symbol names.
//
// Returns false if the definition has no span to take a path from, which happens for a
// zero [ast.DeclDef].
func newReferenceKey(def ast.DeclDef, fullName ir.FullName) (referenceKey, bool) {
	path := def.Span().Path()
	if path == "" || fullName == "" {
		return referenceKey{}, false
	}
	return referenceKey{path: path, fullName: fullName}, true
}

// referenceIndex is a reverse index from a symbol definition to the symbols referencing it.
//
// The index is owned by the [lsp] rather than by a file or a workspace, so references
// resolve across every workspace the server has indexed. This matters for shared
// dependencies: a well-known type is reachable from many workspaces, but a file belongs to
// at most one, so a per-workspace index only ever sees a fraction of its references.
//
// References are grouped by the file containing them so that re-indexing a file replaces
// only that file's contribution, leaving every other file's references untouched. Indexing
// a file therefore costs time proportional to the references in that file alone.
//
// The index is not safe for concurrent use. It is protected by the [lsp] lock, which
// already serializes all request handling.
type referenceIndex struct {
	// keyToFileRefs maps a definition to the referencing symbols, grouped by containing file.
	keyToFileRefs map[referenceKey]map[protocol.URI][]*symbol
	// uriToKeys records which definitions each file contributes references to, so that a
	// file's contribution can be removed without scanning the whole index.
	uriToKeys map[protocol.URI][]referenceKey
}

// newReferenceIndex creates a new reference index.
func newReferenceIndex() *referenceIndex {
	return &referenceIndex{
		keyToFileRefs: make(map[referenceKey]map[protocol.URI][]*symbol),
		uriToKeys:     make(map[protocol.URI][]referenceKey),
	}
}

// SetFile replaces all references contributed by the file at the given URI.
func (i *referenceIndex) SetFile(uri protocol.URI, references map[referenceKey][]*symbol) {
	i.RemoveFile(uri)
	if len(references) == 0 {
		return
	}
	keys := make([]referenceKey, 0, len(references))
	for key, symbols := range references {
		fileRefs, ok := i.keyToFileRefs[key]
		if !ok {
			fileRefs = make(map[protocol.URI][]*symbol, 1)
			i.keyToFileRefs[key] = fileRefs
		}
		fileRefs[uri] = symbols
		keys = append(keys, key)
	}
	i.uriToKeys[uri] = keys
}

// RemoveFile drops all references contributed by the file at the given URI.
//
// This must be called when a file is evicted. The index holds [symbol] values that point
// back at their file, and an evicted file is zeroed, so entries left behind would resolve
// to locations with an empty URI.
func (i *referenceIndex) RemoveFile(uri protocol.URI) {
	for _, key := range i.uriToKeys[uri] {
		fileRefs, ok := i.keyToFileRefs[key]
		if !ok {
			continue
		}
		delete(fileRefs, uri)
		if len(fileRefs) == 0 {
			delete(i.keyToFileRefs, key)
		}
	}
	delete(i.uriToKeys, uri)
}

// References returns all symbols referencing the given definition.
//
// The result is ordered by file URI and then by position, so that results are stable
// across calls. The index is built from maps, whose iteration order is not.
func (i *referenceIndex) References(key referenceKey) []*symbol {
	fileRefs := i.keyToFileRefs[key]
	if len(fileRefs) == 0 {
		return nil
	}
	var symbols []*symbol
	for _, fileSymbols := range fileRefs {
		symbols = append(symbols, fileSymbols...)
	}
	slices.SortFunc(symbols, func(symbol1, symbol2 *symbol) int {
		return cmp.Or(
			cmp.Compare(symbol1.file.uri, symbol2.file.uri),
			cmp.Compare(symbol1.span.Start, symbol2.span.Start),
			cmp.Compare(symbol1.span.End, symbol2.span.End),
		)
	})
	return symbols
}
