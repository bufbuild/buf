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

// This file defines all of the document syncrhonization message handlers for buflsp.server.

package buflsp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// files is a manager for all files the LSP is currently handling.
//
// This type is responsible for syncrhonization and for scheduling parsing and resolution
// jobs.
type files struct {
	server *server

	// Channel for the offline parser. Send freshly updated files here to
	// have them parsed and diagnostics published.
	parser chan *file
	done   chan struct{}

	files map[protocol.URI]*file
	mu    sync.Mutex
}

// newFiles creates a new file manager.
func newFiles(ctx context.Context, server *server) *files {
	files := &files{
		server: server,

		parser: make(chan *file),
		done:   make(chan struct{}, 1),
		files:  make(map[protocol.URI]*file),
	}

	go files.startParser(ctx)
	return files
}

// updateFile updates the contents of some file.
//
// If the file is not already tracked, it is created. Calling this function will
// schedule this file for parsing.
//
// isOpen is whether this is a freshly opened file. This is necessary to accurately
// track the number of instances of this particular file that are currently open.
func (files *files) updateFile(doc *protocol.TextDocumentItem, isOpen bool) {
	files.mu.Lock()
	defer files.mu.Unlock()

	// Look up or create the file in the file manager.
	var entry *file
	if found, ok := files.files[doc.URI]; ok {
		entry = found
	} else {
		entry = &file{
			server: files.server,
			doc:    doc,
		}
		files.files[doc.URI] = entry
	}

	if isOpen {
		entry.timesOpened++
	}

	files.parser <- entry
}

// closeFile closes a particular file.
//
// This will not necessarily evict the file from the file table, since the file
// may have been opened more than once.
func (files *files) closeFile(uri protocol.URI) error {
	files.mu.Lock()
	defer files.mu.Unlock()

	file, ok := files.files[uri]
	if !ok {
		return fmt.Errorf("attempted to close unopened file with URI %v", uri)
	}

	if file.timesOpened--; file.timesOpened == 0 {
		delete(files.files, uri)
	}
	return nil
}

// shutdown shuts down any goroutines spawned by this file manager.
//
// Blocks until everything is done shutting down.
func (files *files) shutdown() {
	close(files.parser)
	<-files.done
}

func (files *files) startParser(ctx context.Context) {
	for file := range files.parser {
		file.parse()
		file.publishDiagnostics(ctx)
	}
	close(files.done)
}

// file is a file that has been opened by the client.
//
// Mutating a file is thread-safe.
type file struct {
	server *server
	doc    *protocol.TextDocumentItem

	// This is the number of "windows" that have opened this
	// file on the client. This is necessary so that we can GC unused files
	// as they are closed.
	timesOpened int

	ast         *ast.FileNode
	diagnostics []protocol.Diagnostic

	mu sync.Mutex
}

func (file *file) parse() {
	file.mu.Lock()
	defer file.mu.Unlock()

	report := new(report)
	handler := reporter.NewHandler(report)

	ast, err := parser.Parse(file.doc.URI.Filename(), strings.NewReader(file.doc.Text), handler)
	if err == nil {
		_, err = parser.ResultFromAST(ast, true, handler)
	}

	file.ast = ast
	file.diagnostics = []protocol.Diagnostic(*report)
	_ = err // Discard the error; we only care about the diagnostics.
}

func (file *file) publishDiagnostics(ctx context.Context) {
	file.mu.Lock()
	defer file.mu.Unlock()

	if len(file.diagnostics) == 0 {
		return
	}

	err := file.server.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI: file.doc.URI,
		// NB: For some reason, Version is int32 in the document struct, but uint32 here.
		// This seems like a bug in the LSP protocol package.
		Version:     uint32(file.doc.Version),
		Diagnostics: file.diagnostics,
	})

	if err != nil {
		file.server.logger.Error(
			"error while publishing diagnostics",
			zap.Error(err),
		)
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

	server.files.updateFile(&params.TextDocument /*isOpen=*/, true)
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

	server.files.updateFile(&protocol.TextDocumentItem{
		URI:     params.TextDocument.URI,
		Version: params.TextDocument.Version,
		Text:    params.ContentChanges[0].Text,
	}, /*isOpen=*/ true)
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

	return server.files.closeFile(params.TextDocument.URI)
}
