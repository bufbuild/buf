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

// This file defines file manipulation operations.

package buflsp

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const descriptorPath = "google/protobuf/descriptor.proto"

// file is a file that has been opened by the client.
//
// Mutating a file is thread-safe.
type file struct {
	// lsp and uri are not protected by file.lock; they are immutable after
	// file creation!
	lsp *lsp
	uri protocol.URI

	// All variables after this lock variables are protected by file.lock.
	//
	// NOTE: this package must NEVER attempt to acquire a lock on a file while
	// holding a lock on another file. This guarantees that any concurrent operations
	// on distinct files can always make forward progress, even if the information they
	// have is incomplete. This trades off up-to-date accuracy for responsiveness.
	//
	// For example, suppose g1 locks a.proto, and then attempts to lock b.proto
	// because it followed a pointer in importMap. However, in the meantime, g2
	// has acquired b.proto's lock already, and attempts to acquire a lock to a.proto,
	// again because of a pointer in importMap. This will deadlock, and it will
	// deadlock in such a way that will be undetectable to the Go scheduler, so the
	// LSP will hang forever.
	//
	// This seems like a contrived scenario, but it can happen if a user creates two
	// mutually-recursive Protobuf files. Although this is not permitted by Protobuf,
	// the LSP must handle this invalid state gracefully.
	//
	// TODO(mcy): enforce this somehow.
	lock mutex

	text    string
	version int32
	hasText bool // Whether this file has ever had text read into it.
	// Always set false->true. Once true, never becomes false again.

	fileNode    *ast.FileNode
	packageNode *ast.PackageNode
	diagnostics []protocol.Diagnostic
	imports     map[string]*file
	symbols     []*symbol
}

// Manager returns the file manager that owns this file.
func (f *file) Manager() *fileManager {
	return f.lsp.fileManager
}

// Package returns the package of this file, if known.
func (f *file) Package() []string {
	if f.packageNode == nil {
		return nil
	}

	return strings.Split(string(f.packageNode.Name.AsIdentifier()), ".")
}

// Reset clears all bookkeeping information on this file.
func (f *file) Reset(ctx context.Context) {
	f.lsp.logger.Sugar().Debugf("resetting file %v", f.uri)

	// Lock and unlock to acquire the import map.
	// This map is never mutated after being created, so we only
	// need to read the pointer.
	//
	// We need to lock and unlock because Close() will call Reset() on other
	// files, and this will deadlock if cyclic imports exist.
	f.lock.Lock(ctx)
	imports := f.imports
	f.lock.Unlock(ctx)

	// Close all imported files while file.mu is not held.
	for _, imported := range imports {
		imported.Close(ctx)
	}

	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)

	f.fileNode = nil
	f.packageNode = nil
	f.diagnostics = nil
	f.imports = nil
	f.symbols = nil
}

// Close marks a file as closed.
//
// This will not necessarily evict the file, since there may be more than one user
// for this file.
func (f *file) Close(ctx context.Context) {
	f.lsp.fileManager.Close(ctx, f.uri)
}

// ReadFromDisk reads this file from disk if it has never had data loaded into it before.
//
// If it has been read from disk before, or has received updates from the LSP client, this
// function returns nil.
func (f *file) ReadFromDisk(ctx context.Context) (err error) {
	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)
	if f.hasText {
		return nil
	}

	data, err := os.ReadFile(f.uri.Filename())
	if err != nil {
		return fmt.Errorf("could not read file %q from disk: %w", f.uri, err)
	}

	f.version = -1
	f.text = string(data)
	return nil
}

// Update updates the contents of this file with the given text received from
// the LSP client.
func (f *file) Update(ctx context.Context, version int32, text string) {
	f.Reset(ctx)

	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)

	f.lsp.logger.Sugar().Infof("new file version: %v, %v -> %v", f.uri, f.version, version)
	f.version = version
	f.text = text
	f.hasText = true
}

// Refresh rebuilds all of a file's internal book-keeping.
//
// If deep is set, this will also load imports and refresh those, too.
func (f *file) Refresh(ctx context.Context) {
	if f.RefreshAST(ctx) {
		f.PublishDiagnostics(ctx)
	}
	f.IndexImports(ctx)
	f.IndexSymbols(ctx)
}

// RefreshAST reparses the file and generates diagnostics if necessary.
//
// Returns whether a reparse was necessary.
func (f *file) RefreshAST(ctx context.Context) bool {
	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)
	if f.fileNode != nil {
		return false
	}

	// NOTE: We intentionally do not use var report report here, because we need
	// report to be non-nil when empty; this is because if it is nil, when calling
	// PublishDiagnostics() below it will be serialized as JSON null.
	report := report{}
	handler := reporter.NewHandler(&report)

	f.lsp.logger.Sugar().Infof("parsing AST for %v, %v", f.uri, f.version)
	parsed, err := parser.Parse(f.uri.Filename(), strings.NewReader(f.text), handler)
	if err == nil {
		// Throw away the error. It doesn't contain anything not in the diagnostic array.
		_, _ = parser.ResultFromAST(parsed, true, handler)
	}

	f.fileNode = parsed
	f.diagnostics = report
	f.lsp.logger.Sugar().Debugf("got %v diagnostic(s)", len(f.diagnostics))

	// Search for a potential package node.
	if f.fileNode != nil {
		for _, decl := range f.fileNode.Decls {
			if pkg, ok := decl.(*ast.PackageNode); ok {
				f.packageNode = pkg
				break
			}
		}
	}

	return true
}

// PublishDiagnostics publishes all of this file's diagnostics to the LSP client.
func (f *file) PublishDiagnostics(ctx context.Context) {
	ctx, span := f.lsp.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(f.uri))))
	defer span.End()

	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)

	if f.diagnostics == nil {
		return
	}

	// Publish the diagnostics. This error is automatically logged by the LSP framework.
	_ = f.lsp.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI: f.uri,
		// NOTE: For some reason, Version is int32 in the document struct, but uint32 here.
		// This seems like a bug in the LSP protocol package.
		Version:     uint32(f.version),
		Diagnostics: f.diagnostics,
	})
}

// IndexImports finds URIs for all of the files imported by this file.
func (f *file) IndexImports(ctx context.Context) {
	ctx, span := f.lsp.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(f.uri))))
	defer span.End()

	unlock := f.lock.Lock(ctx)
	defer unlock()

	if f.fileNode == nil || f.imports != nil {
		return
	}

	imports, err := f.lsp.findImportable(ctx, f.uri)
	if err != nil {
		f.lsp.logger.Sugar().Warnf("could not compute importable files for %s: %s", f.uri, err)
		return
	}

	f.imports = make(map[string]*file)
	for _, decl := range f.fileNode.Decls {
		node, ok := decl.(*ast.ImportNode)
		if !ok {
			continue
		}

		name := node.Name.AsString()
		uri, ok := imports[name]
		if !ok {
			f.lsp.logger.Sugar().Warnf("could not find URI for import %q", name)
			continue
		}

		imported := f.Manager().Open(ctx, uri)
		f.imports[node.Name.AsString()] = imported
	}

	if _, ok := f.imports[descriptorPath]; !ok {
		descriptorURI := imports[descriptorPath]
		if f.uri == descriptorURI {
			f.imports[descriptorPath] = f
		} else {
			imported := f.Manager().Open(ctx, descriptorURI)
			f.imports[descriptorPath] = imported
		}
	}

	// FIXME: This algorithm is not correct: it does not account for `import public`.

	// Drop the lock after copying the pointer to the imports map. This
	// particular map will not be mutated further, and since we're going to grab the lock of
	// other files, we need to drop the currently held lock.
	fileImports := f.imports
	unlock()

	for _, file := range fileImports {
		if err := file.ReadFromDisk(ctx); err != nil {
			file.lsp.logger.Sugar().Warnf("could not load import import %q from disk: %w",
				file.uri, err)
			continue
		}

		// Parse the imported file and find all symbols in it, but do not
		// index symbols in the import's imports, otherwise we will recursively
		// index the universe and that would be quite slow.
		file.RefreshAST(ctx)
		file.IndexSymbols(ctx)
	}
}

// IndexSymbols processes the AST of a file and generates symbols for each symbol in
// the document.
func (f *file) IndexSymbols(ctx context.Context) {
	_, span := f.lsp.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(f.uri))))
	defer span.End()

	unlock := f.lock.Lock(ctx)
	defer unlock()

	// Throw away all the old symbols. Unlike other indexing functions, we rebuild
	// symbols unconditionally.
	f.symbols = nil

	// Generate new symbols.
	newWalker(f).Walk(f.fileNode, f.fileNode)

	// Finally, sort the symbols in position order, with shorter symbols sorting smaller.
	slices.SortFunc(f.symbols, func(s1, s2 *symbol) int {
		diff := s1.info.Start().Offset - s2.info.Start().Offset
		if diff == 0 {
			return s1.info.End().Offset - s2.info.End().Offset
		}
		return diff
	})

	// Now we can drop the lock and search for cross-file references.
	symbols := f.symbols
	unlock()
	for _, symbol := range symbols {
		symbol.ResolveCrossFile(ctx)
	}

	f.lsp.logger.Sugar().Debugf("symbol indexing complete %s", f.uri)
}

// SymbolAt finds a symbol in this file at the given cursor position, if one exists.
//
// Returns nil if no symbol is found.
func (f *file) SymbolAt(ctx context.Context, cursor protocol.Position) *symbol {
	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)

	// Binary search for the symbol whose start is before or equal to cursor.
	idx, found := slices.BinarySearchFunc(f.symbols, cursor, func(sym *symbol, cursor protocol.Position) int {
		return comparePositions(sym.Range().Start, cursor)
	})
	if !found {
		if idx == 0 {
			return nil
		}
		idx--
	}

	symbol := f.symbols[idx]
	f.lsp.logger.Debug("found symbol", zap.Object("symbol", symbol))

	// Check that cursor is before the end of the symbol.
	if comparePositions(symbol.Range().End, cursor) <= 0 {
		return nil
	}

	return symbol
}
