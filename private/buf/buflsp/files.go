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
	"sync/atomic"

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
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

	files map[protocol.URI]*file
	mu    mutex
}

// newFiles creates a new file manager.
func newFiles(server *server) *files {
	return &files{
		server: server,
		files:  make(map[protocol.URI]*file),
	}
}

// FindOrCreate finds a file with the given URI, or creates one.
//
// This function grabs the files lock; this lock must be released by calling the returned
// function, usually in a defer.
//
// opening indicates that this find is opening a file in a new window.
func (files *files) FindOrCreate(ctx context.Context, uri protocol.URI, opening bool) *file {
	ctx, span := files.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(uri))))
	defer span.End()
	_ = ctx

	files.mu.Lock(ctx)
	defer files.mu.Unlock(ctx)

	// Look up or create the file in the file manager.
	var entry *file
	if found, ok := files.files[uri]; ok {
		entry = found
	} else {
		entry = &file{
			server: files.server,
			uri:    uri,
		}

		if !opening {
			files.server.logger.Sugar().Warnf(
				"Find() called with opening set to false on a file that doesn't exist", uri)

			// Presumably this file has been opened at least once?
			entry.refs.Add(1)
		}

		files.files[uri] = entry
	}

	if opening {
		entry.refs.Add(1)
	}

	return entry
}

// file is a file that has been opened by the client.
//
// Mutating a file is thread-safe.
type file struct {
	// server and uri are not protected by file.mu; they are immutable after
	// file creation!
	server *server
	uri    protocol.URI

	// This is the number of users of this file. This can either be windows
	// in the editor, or other files that import it and need it for symbol lookup.
	refs atomic.Int32

	// All variables after this lock variables are protected by file.mu.
	mu mutex

	text    string
	version int32
	hasText bool // Whether this file has ever had text read into it.
	// Always set false->true. Once true, never becomes false again.

	ast         *ast.FileNode
	pkg         *ast.PackageNode
	diagnostics []protocol.Diagnostic
	imports     importMap
	symbols     []*symbol

	// This is the module set that this file belongs to. It is used to search for
	// things like symbol definitions.
	moduleSet bufmodule.ModuleSet
}

type importMap map[*ast.ImportNode]*file

// Owner returns the file manager that owns this file.
func (file *file) Owner() *files {
	return file.server.files
}

// Package returns the package of this file, if known.
func (file *file) Package() []string {
	if file.pkg == nil {
		return nil
	}

	return strings.Split(string(file.pkg.Name.AsIdentifier()), ".")
}

// Reset clears all bookkeeping information on this file.
func (file *file) Reset(ctx context.Context) {
	ctx, span := file.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)

	for _, imported := range file.imports {
		imported.Close(ctx)
	}

	file.ast = nil
	file.pkg = nil
	file.diagnostics = nil
	file.imports = nil
	file.symbols = nil
}

// Close marks a file as closed.
//
// This will not necessarily evict the file, since there may be more than one user
// for this file.
func (file *file) Close(ctx context.Context) {
	if file.refs.Load() == 0 {
		// This is a re-entrant call to Close() through Reset(), so we just return.
		return
	}

	if file.refs.Add(-1) > 0 {
		// There still exist other copies of this file elsewhere.
		return
	}

	ctx, span := file.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)
	file.Reset(ctx)

	file.Owner().mu.Lock(ctx)
	defer file.Owner().mu.Unlock(ctx)
	delete(file.Owner().files, file.uri)
}

// ReadFromDisk reads this file from disk if it has never had data loaded into it before.
//
// If it has been read from disk before, or has received updates from the LSP client, this
// function returns nil.
func (file *file) ReadFromDisk(ctx context.Context) (err error) {
	ctx, span := file.server.tracer.Start(ctx,
		tracing.WithErr(&err),
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()
	_ = ctx

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)
	if file.hasText {
		return nil
	}

	data, err := os.ReadFile(file.uri.Filename())
	if err != nil {
		return fmt.Errorf("could not read file %q from disk: %w", file.uri, err)
	}

	file.version = -1
	file.text = string(data)
	return nil
}

// Update updates the contents of this file with the given text received from
// the LSP client.
func (file *file) Update(ctx context.Context, version int32, text string) {
	ctx, span := file.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)

	file.server.logger.Sugar().Infof("new file version: %v, %v", file.uri, file.version)
	file.Reset(ctx)
	file.version = version
	file.text = text
	file.hasText = true
}

func (file *file) FindWorkspace(ctx context.Context) {
	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)

	// Try to dig up a ModuleSet for this file.
	workspace, err := file.server.controller.GetWorkspace(ctx, file.uri.Filename(),
		bufctl.WithImageExcludeImports(false),
		bufctl.WithImageExcludeSourceInfo(false),
	)
	if err != nil {
		file.server.logger.Sugar().Warnf(
			"no Buf workspace found for %s, continuing with limited features; %s",
			file.uri.Filename(), err,
		)
	}

	file.moduleSet = workspace
}

// RefreshAST reparses the file and generates diagnostics if necessary.
//
// Returns whether a reparse was necessary.
func (file *file) RefreshAST(ctx context.Context) bool {
	ctx, span := file.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()
	_ = ctx

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)
	if file.ast != nil {
		return false
	}

	// NOTE: We intentionally do not use var report report here, because we need
	// report to be non-nil when empty; this is because if it is nil, when calling
	// PublishDiagnostics() below it will be serialized as JSON null.
	report := report{}
	handler := reporter.NewHandler(&report)

	file.server.logger.Sugar().Infof("parsing AST for %v, %v", file.uri, file.version)
	parsed, err := parser.Parse(file.uri.Filename(), strings.NewReader(file.text), handler)
	if err == nil {
		// Throw away the error. It doesn't contain anything not in the diagnostic array.
		_, _ = parser.ResultFromAST(parsed, true, handler)
	}

	file.ast = parsed
	file.diagnostics = report

	// Search for a potential package node.
	if file.ast != nil {
		for _, decl := range file.ast.Decls {
			if pkg, ok := decl.(*ast.PackageNode); ok {
				file.pkg = pkg
				break
			}
		}
	}

	return true
}

// PublishDiagnostics publishes all of this file's diagnostics to the LSP client.
func (file *file) PublishDiagnostics(ctx context.Context) {
	ctx, span := file.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)

	if file.diagnostics == nil {
		return
	}

	// Publish the diagnostics. We can discard this error, it gets logged automatically
	// by the loggingStream.
	_ = file.server.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI: file.uri,
		// NOTE: For some reason, Version is int32 in the document struct, but uint32 here.
		// This seems like a bug in the LSP protocol package.
		Version:     uint32(file.version),
		Diagnostics: file.diagnostics,
	})
}

// IndexImports finds URIs for all of the files imported by this file.
func (file *file) IndexImports(ctx context.Context) {
	ctx, span := file.server.tracer.Start(ctx,
		tracing.WithAttributes(attribute.String("uri", string(file.uri))))
	defer span.End()

	file.mu.Lock(ctx)
	defer file.mu.Unlock(ctx)

	if file.ast == nil || file.imports != nil {
		return
	}

	imports, err := findImportable(ctx, file.server.rootBucket, "/", file.server.logger, file.uri)
	if err != nil {
		file.server.logger.Sugar().Warnf("could not compute importable files for %s: %s", file.uri, err)
		return
	}

	file.imports = make(importMap)
	for _, decl := range file.ast.Decls {
		node, ok := decl.(*ast.ImportNode)
		if !ok {
			continue
		}

		name := node.Name.AsString()
		uri, ok := imports[name]
		if !ok {
			file.server.logger.Sugar().Warnf("could not find URI for import %q", name)
			continue
		}

		imported := file.Owner().FindOrCreate(ctx, uri, true)
		file.imports[node] = imported

		if err := imported.ReadFromDisk(ctx); err != nil {
			file.server.logger.Sugar().Warnf("could not load import import %q from disk: %w", imported.uri, err)
			continue
		}

		// Parse the imported file and find all symbols in it, but do not
		// index symbols in the import's imports, otherwise we will recursively
		// index the universe and that would be quite slow.
		imported.RefreshAST(ctx)
		imported.IndexSymbols(ctx)
	}
}

// report is a reporter.Reporter that captures diagnostic events as
// protocol.Diagnostic values.
type report []protocol.Diagnostic

// Error implements reporter.Handler for *diagnostics.
func (diagnostics *report) Error(err reporter.ErrorWithPos) error {
	*diagnostics = append(*diagnostics, error2diagnostic(err, false))
	return nil
}

// Error implements reporter.Handler for *diagnostics.
func (diagnostics *report) Warning(err reporter.ErrorWithPos) {
	*diagnostics = append(*diagnostics, error2diagnostic(err, true))
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
	if err := server.checkInit(); err != nil {
		return err
	}

	file := server.files.FindOrCreate(ctx, params.TextDocument.URI, true)
	file.Update(ctx, params.TextDocument.Version, params.TextDocument.Text)
	if file.RefreshAST(ctx) {
		file.PublishDiagnostics(ctx)
	}
	file.IndexImports(ctx)
	file.IndexSymbols(ctx)

	return nil
}

// DidOpen is called whenever the client opens a document. This is our signal to parse
// the file.
func (server *server) DidChange(
	ctx context.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {
	if err := server.checkInit(); err != nil {
		return err
	}

	file := server.files.FindOrCreate(ctx, params.TextDocument.URI, true)
	file.Update(ctx, params.TextDocument.Version, params.ContentChanges[0].Text)
	if file.RefreshAST(ctx) {
		file.PublishDiagnostics(ctx)
	}
	file.IndexImports(ctx)
	file.IndexSymbols(ctx)

	return nil
}

// DidOpen is called whenever the client opens a document. This is our signal to parse
// the file.
func (server *server) DidClose(
	ctx context.Context,
	params *protocol.DidCloseTextDocumentParams,
) error {
	if err := server.checkInit(); err != nil {
		return err
	}

	server.files.FindOrCreate(ctx, params.TextDocument.URI, false).Close(ctx)
	return nil
}
