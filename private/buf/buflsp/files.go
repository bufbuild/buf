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

// This file defines all of the document synchronization message handlers for buflsp.server.

package buflsp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/refcount"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
	"go.opentelemetry.io/otel/attribute"
)

// files is a manager for all files the LSP is currently handling.
type files struct {
	server *server
	table  refcount.Map[protocol.URI, file]
}

// newFiles creates a new file manager.
func newFiles(server *server) *files {
	return &files{server: server}
}

// Open finds a file with the given URI, or creates one.
func (files *files) Open(ctx context.Context, uri protocol.URI) *file {
	file, found := files.table.Insert(uri)
	if !found {
		file.server = files.server
		file.uri = uri
	}

	return file
}

// Get finds a file with the given URI, or returns nil.
func (files *files) Get(uri protocol.URI) *file {
	return files.table.Get(uri)
}

// Close marks a file as closed.
//
// This will not necessarily evict the file, since there may be more than one user
// for this file.
func (files *files) Close(ctx context.Context, uri protocol.URI) {
	if deleted := files.table.Delete(uri); deleted != nil {
		deleted.Reset(ctx)
	}
}

// file is a file that has been opened by the client.
//
// Mutating a file is thread-safe.
type file struct {
	// server and uri are not protected by file.lock; they are immutable after
	// file creation!
	server *server
	uri    protocol.URI

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

	// This is the module set that this file belongs to. It is used to search for
	// things like symbol definitions.
	moduleSet bufmodule.ModuleSet
}

// Owner returns the file manager that owns this file.
func (f *file) Owner() *files {
	return f.server.files
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
	f.server.logger.Sugar().Debugf("resetting file %v", f.uri)

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
	f.server.files.Close(ctx, f.uri)
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

	f.server.logger.Sugar().Infof("new file version: %v, %v -> %v", f.uri, f.version, version)
	f.version = version
	f.text = text
	f.hasText = true
}

func (f *file) FindWorkspace(ctx context.Context) {
	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)

	// Try to dig up a ModuleSet for this file.
	workspace, err := f.server.controller.GetWorkspace(ctx, f.uri.Filename(),
		bufctl.WithImageExcludeImports(false),
		bufctl.WithImageExcludeSourceInfo(false),
	)
	if err != nil {
		f.server.logger.Sugar().Warnf(
			"no Buf workspace found for %s, continuing with limited features; %s",
			f.uri.Filename(), err,
		)
	}

	f.moduleSet = workspace
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

	f.server.logger.Sugar().Infof("parsing AST for %v, %v", f.uri, f.version)
	parsed, err := parser.Parse(f.uri.Filename(), strings.NewReader(f.text), handler)
	if err == nil {
		// Throw away the error. It doesn't contain anything not in the diagnostic array.
		_, _ = parser.ResultFromAST(parsed, true, handler)
	}

	f.fileNode = parsed
	f.diagnostics = report
	f.server.logger.Sugar().Debugf("got %v diagnostic(s)", len(f.diagnostics))

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
	ctx, span := f.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(f.uri))))
	defer span.End()

	f.lock.Lock(ctx)
	defer f.lock.Unlock(ctx)

	if f.diagnostics == nil {
		return
	}

	// Publish the diagnostics. This error is automatically logged by the LSP framework.
	_ = f.server.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI: f.uri,
		// NOTE: For some reason, Version is int32 in the document struct, but uint32 here.
		// This seems like a bug in the LSP protocol package.
		Version:     uint32(f.version),
		Diagnostics: f.diagnostics,
	})
}

// IndexImports finds URIs for all of the files imported by this file.
func (f *file) IndexImports(ctx context.Context) {
	ctx, span := f.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(f.uri))))
	defer span.End()

	unlock := f.lock.Lock(ctx)
	defer unlock()

	if f.fileNode == nil || f.imports != nil {
		return
	}

	imports, err := findImportable(ctx, f.server.rootBucket, "/", f.server.logger, f.uri)
	if err != nil {
		f.server.logger.Sugar().Warnf("could not compute importable files for %s: %s", f.uri, err)
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
			f.server.logger.Sugar().Warnf("could not find URI for import %q", name)
			continue
		}

		imported := f.Owner().Open(ctx, uri)
		f.imports[node.Name.AsString()] = imported
	}

	// Drop the lock after copying the pointer to the imports map. This
	// particular map will not be mutated further, and since we're going to grab the lock of
	// other files, we need to drop the currently held lock.
	fileImports := f.imports
	unlock()

	for _, file := range fileImports {
		if err := file.ReadFromDisk(ctx); err != nil {
			file.server.logger.Sugar().Warnf("could not load import import %q from disk: %w",
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

// report is a reporter.Reporter that captures diagnostic events as
// protocol.Diagnostic values.
type report []protocol.Diagnostic

// Error implements reporter.Handler for *diagnostics.
func (r *report) Error(err reporter.ErrorWithPos) error {
	*r = append(*r, error2diagnostic(err, false))
	return nil
}

// Error implements reporter.Handler for *diagnostics.
func (r *report) Warning(err reporter.ErrorWithPos) {
	*r = append(*r, error2diagnostic(err, true))
}

// error2diagnostic converts a protocompile error into a diagnostic.
//
// Unfortunately, protocompile's errors are currently too meagre to provide full code
// spans; that will require a fix in the compiler.
func error2diagnostic(err reporter.ErrorWithPos, isWarning bool) protocol.Diagnostic {
	pos := protocol.Position{
		Line:      uint32(err.GetPosition().Line - 1),
		Character: uint32(err.GetPosition().Col - 1),
	}

	diagnostic := protocol.Diagnostic{
		// TODO: The compiler currently does not record spans for diagnostics. This is
		// essentially a bug that will result in worse diagnostics until fixed.
		Range:    protocol.Range{Start: pos, End: pos},
		Severity: protocol.DiagnosticSeverityError,
		Message:  err.Unwrap().Error(),
	}

	if isWarning {
		diagnostic.Severity = protocol.DiagnosticSeverityWarning
	}

	return diagnostic
}

// DidOpen is called whenever the client opens a document. This is our signal to parse
// the file.
func (server *server) DidOpen(
	ctx context.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	file := server.files.Open(ctx, params.TextDocument.URI)
	file.Update(ctx, params.TextDocument.Version, params.TextDocument.Text)
	go file.Refresh(context.WithoutCancel(ctx))
	return nil
}

// DidOpen is called whenever the client opens a document. This is our signal to parse
// the file.
func (server *server) DidChange(
	ctx context.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {
	file := server.files.Get(params.TextDocument.URI)
	if file == nil {
		// Update for a file we don't know about? Seems bad!
		return fmt.Errorf("received update for file that was not open: %q", params.TextDocument.URI)
	}

	file.Update(ctx, params.TextDocument.Version, params.ContentChanges[0].Text)
	go file.Refresh(context.WithoutCancel(ctx))
	return nil
}

// Formatting is called whenever the user explicitly requests formatting.
func (server *server) Formatting(
	ctx context.Context,
	params *protocol.DocumentFormattingParams,
) ([]protocol.TextEdit, error) {
	file := server.files.Get(params.TextDocument.URI)
	if file == nil {
		// Format for a file we don't know about? Seems bad!
		return nil, fmt.Errorf("received update for file that was not open: %q", params.TextDocument.URI)
	}

	// Currently we have no way to honor any of the parameters.
	_ = params

	file.lock.Lock(ctx)
	defer file.lock.Unlock(ctx)
	if file.fileNode == nil {
		return nil, nil
	}

	var out strings.Builder
	if err := bufformat.FormatFileNode(&out, file.fileNode); err != nil {
		return nil, err
	}

	return []protocol.TextEdit{
		{
			Range:   infoToRange(file.fileNode.NodeInfo(file.fileNode)),
			NewText: out.String(),
		},
	}, nil
}

// DidOpen is called whenever the client opens a document. This is our signal to parse
// the file.
func (server *server) DidClose(
	ctx context.Context,
	params *protocol.DidCloseTextDocumentParams,
) error {
	server.files.Close(ctx, params.TextDocument.URI)
	return nil
}
