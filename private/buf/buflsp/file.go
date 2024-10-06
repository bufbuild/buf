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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/google/uuid"
	"go.lsp.dev/protocol"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	// This is enforced by mutex.go.
	lock mutex

	text string
	// Version is an opaque version identifier given to us by the LSP client. This
	// is used in the protocol to disambiguate which version of a file e.g. publishing
	// diagnostics or symbols an operating refers to.
	version int32
	hasText bool // Whether this file has ever had text read into it.
	// Always set false->true. Once true, never becomes false again.

	workspace     bufworkspace.Workspace
	module        bufmodule.Module
	imageFileInfo bufimage.ImageFileInfo

	isWKT bool

	fileNode          *ast.FileNode
	packageNode       *ast.PackageNode
	diagnostics       []protocol.Diagnostic
	importableToImage map[string]bufimage.ImageFileInfo
	importToFile      map[string]*file
	symbols           []*symbol
	image             bufimage.Image
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
	f.lsp.logger.Debug(fmt.Sprintf("resetting file %v", f.uri))

	// Lock and unlock to acquire the import map, then nil everything out
	// This map is never mutated after being created, so we only
	// need to read the pointer.
	//
	// We need to lock and unlock because Close() will call Reset() on other
	// files, and this will deadlock if cyclic imports exist.
	f.lock.Lock(ctx)
	imports := f.importToFile

	f.fileNode = nil
	f.packageNode = nil
	f.diagnostics = nil
	f.importableToImage = nil
	f.importToFile = nil
	f.symbols = nil
	f.image = nil
	f.lock.Unlock(ctx)

	// Close all imported files while file.mu is not held.
	for _, imported := range imports {
		imported.Close(ctx)
	}
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

	f.lsp.logger.Info(fmt.Sprintf("new file version: %v, %v -> %v", f.uri, f.version, version))
	f.version = version
	f.text = text
	f.hasText = true
}

// Refresh rebuilds all of a file's internal book-keeping.
//
// If deep is set, this will also load imports and refresh those, too.
func (f *file) Refresh(ctx context.Context) {
	progress := newProgress(f.lsp)
	progress.Begin(ctx, "Indexing")

	progress.Report(ctx, "Parsing AST", 1.0/6)
	hasReport := f.RefreshAST(ctx)

	progress.Report(ctx, "Indexing Imports", 2.0/6)
	f.IndexImports(ctx)

	progress.Report(ctx, "Detecting Module", 3.0/6)
	f.FindModule(ctx)

	progress.Report(ctx, "Linking Descriptors", 4.0/6)
	f.BuildImage(ctx)
	hasReport = f.RunLints(ctx) || hasReport // Avoid short-circuit here.

	progress.Report(ctx, "Indexing Symbols", 5.0/6)
	f.IndexSymbols(ctx)

	progress.Done(ctx)
	if hasReport {
		f.PublishDiagnostics(ctx)
	}
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

	f.lsp.logger.Info(fmt.Sprintf("parsing AST for %v, %v", f.uri, f.version))
	parsed, err := parser.Parse(f.uri.Filename(), strings.NewReader(f.text), handler)
	if err == nil {
		// Throw away the error. It doesn't contain anything not in the diagnostic array.
		_, _ = parser.ResultFromAST(parsed, true, handler)
	}

	f.fileNode = parsed
	f.diagnostics = report.diagnostics
	f.lsp.logger.Debug(fmt.Sprintf("got %v diagnostic(s)", len(f.diagnostics)))

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
	defer slogext.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()

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

// FindModule finds the Buf module for this file.
func (f *file) FindModule(ctx context.Context) {
	workspace, err := f.lsp.controller.GetWorkspace(ctx, f.uri.Filename())
	if err != nil {
		f.lsp.logger.Warn("could not load workspace", slog.String("uri", string(f.uri)), slogext.ErrorAttr(err))
		return
	}

	// Figure out which module this file belongs to.
	var module bufmodule.Module
	for _, mod := range workspace.Modules() {
		// We do not care about this error, so we discard it.
		_ = mod.WalkFileInfos(ctx, func(fileInfo bufmodule.FileInfo) error {
			if fileInfo.LocalPath() == f.uri.Filename() {
				module = mod
			}
			return nil
		})
		if module != nil {
			break
		}
	}
	if module == nil {
		f.lsp.logger.Warn(fmt.Sprintf("could not find module for %q", f.uri))
	}

	// Determine if this is the WKT module. We do so by checking if this module contains
	// descriptor.proto.
	file, err := module.GetFile(ctx, descriptorPath)
	if err == nil {
		defer file.Close()
	}

	f.lock.Lock(ctx)
	f.workspace = workspace
	f.module = module
	f.lock.Unlock(ctx)
}

// IndexImports finds URIs for all of the files imported by this file.
func (f *file) IndexImports(ctx context.Context) {
	defer slogext.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()

	unlock := f.lock.Lock(ctx)
	defer unlock()

	if f.fileNode == nil || f.importToFile != nil {
		return
	}

	importable, err := f.lsp.findImportable(ctx, f.uri)
	if err != nil {
		f.lsp.logger.Warn(fmt.Sprintf("could not compute importable files for %s: %s", f.uri, err))
		return
	}
	f.importableToImage = importable

	// Find the FileInfo for this path. The crazy thing is that it may appear in importable
	// multiple times, with different path lengths! We want to pick the one with the longest path
	// length.
	for _, fileInfo := range importable {
		if fileInfo.LocalPath() == f.uri.Filename() {
			if f.imageFileInfo != nil && len(f.imageFileInfo.Path()) > len(fileInfo.Path()) {
				continue
			}
			f.imageFileInfo = fileInfo
		}
	}

	f.importToFile = make(map[string]*file)
	for _, decl := range f.fileNode.Decls {
		node, ok := decl.(*ast.ImportNode)
		if !ok {
			continue
		}

		name := node.Name.AsString()
		fileInfo, ok := importable[name]
		if !ok {
			f.lsp.logger.Warn(fmt.Sprintf("could not find URI for import %q", name))
			continue
		}

		var imported *file
		if fileInfo.LocalPath() == f.uri.Filename() {
			imported = f
		} else {
			imported = f.Manager().Open(ctx, protocol.URI("file://"+fileInfo.LocalPath()))
		}

		imported.imageFileInfo = fileInfo
		f.isWKT = strings.HasPrefix("google/protobuf/", fileInfo.Path())
		f.importToFile[node.Name.AsString()] = imported
	}

	// descriptor.proto is always implicitly imported.
	if _, ok := f.importToFile[descriptorPath]; !ok {
		descriptorFile := importable[descriptorPath]
		descriptorURI := protocol.URI("file://" + descriptorFile.LocalPath())
		if f.uri == descriptorURI {
			f.importToFile[descriptorPath] = f
		} else {
			imported := f.Manager().Open(ctx, descriptorURI)
			imported.imageFileInfo = descriptorFile
			f.importToFile[descriptorPath] = imported
		}
		f.isWKT = true
	}

	// FIXME: This algorithm is not correct: it does not account for `import public`.

	// Drop the lock after copying the pointer to the imports map. This
	// particular map will not be mutated further, and since we're going to grab the lock of
	// other files, we need to drop the currently held lock.
	fileImports := f.importToFile
	unlock()

	for _, file := range fileImports {
		if err := file.ReadFromDisk(ctx); err != nil {
			file.lsp.logger.Warn(fmt.Sprintf("could not load import import %q from disk: %s",
				file.uri, err.Error()))
			continue
		}

		// Parse the imported file and find all symbols in it, but do not
		// index symbols in the import's imports, otherwise we will recursively
		// index the universe and that would be quite slow.
		file.RefreshAST(ctx)
		file.IndexSymbols(ctx)
	}
}

// BuildImage builds a Buf Image for this file. This does not use the controller to build
// the image, because we need delicate control over the input files: namely, for the case
// when we depend on a file that has been opened and modified in the editor.
//
// This operation requires IndexImports().
func (f *file) BuildImage(ctx context.Context) {
	f.lock.Lock(ctx)
	importable := f.importableToImage
	fileInfo := f.imageFileInfo
	f.lock.Unlock(ctx)

	if importable == nil || fileInfo == nil {
		return
	}

	var report report
	var symbols linker.Symbols
	compiler := protocompile.Compiler{
		SourceInfoMode: protocompile.SourceInfoExtraOptionLocations,
		Resolver: &protocompile.SourceResolver{
			Accessor: func(path string) (io.ReadCloser, error) {
				var uri protocol.URI
				fileInfo, ok := importable[path]
				if ok {
					uri = protocol.URI("file://" + fileInfo.LocalPath())
				} else {
					uri = protocol.URI("file://" + path)
				}

				if file := f.Manager().Get(uri); file != nil {
					return ioext.CompositeReadCloser(strings.NewReader(file.text), ioext.NopCloser), nil
				} else if !ok {
					return nil, os.ErrNotExist
				}

				return os.Open(fileInfo.LocalPath())
			},
		},
		Symbols:  &symbols,
		Reporter: &report,
	}

	compiled, err := compiler.Compile(ctx, fileInfo.Path())
	if err != nil {
		f.lock.Lock(ctx)
		f.diagnostics = report.diagnostics
		f.lock.Unlock(ctx)
	}
	if compiled[0] == nil {
		return
	}

	var imageFiles []bufimage.ImageFile
	seen := map[string]bool{}

	queue := []protoreflect.FileDescriptor{compiled[0]}
	for len(queue) > 0 {
		descriptor := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		if seen[descriptor.Path()] {
			continue
		}
		seen[descriptor.Path()] = true

		unused, ok := report.pathToUnusedImports[descriptor.Path()]
		var unusedIndices []int32
		if ok {
			unusedIndices = make([]int32, 0, len(unused))
		}

		imports := descriptor.Imports()
		for i := 0; i < imports.Len(); i++ {
			dep := imports.Get(i).FileDescriptor
			if dep == nil {
				f.lsp.logger.Warn(fmt.Sprintf("found nil FileDescriptor for import %s", imports.Get(i).Path()))
				continue
			}

			queue = append(queue, dep)

			if unused != nil {
				if _, ok := unused[dep.Path()]; ok {
					unusedIndices = append(unusedIndices, int32(i))
				}
			}
		}

		descriptorProto := protoutil.ProtoFromFileDescriptor(descriptor)
		if descriptorProto == nil {
			err = fmt.Errorf("protoutil.ProtoFromFileDescriptor() returned nil for %q", descriptor.Path())
			break
		}

		var imageFile bufimage.ImageFile
		imageFile, err = bufimage.NewImageFile(
			descriptorProto,
			nil,
			uuid.UUID{},
			"",
			descriptor.Path(),
			descriptor.Path() != fileInfo.Path(),
			report.syntaxMissing[descriptor.Path()],
			unusedIndices,
		)
		if err != nil {
			break
		}

		imageFiles = append(imageFiles, imageFile)
		f.lsp.logger.Debug(fmt.Sprintf("added image file for %s", descriptor.Path()))
	}

	if err != nil {
		f.lsp.logger.Warn("could not build image", slog.String("uri", string(f.uri)), slogext.ErrorAttr(err))
		return
	}

	image, err := bufimage.NewImage(imageFiles)
	if err != nil {
		f.lsp.logger.Warn("could not build image", slog.String("uri", string(f.uri)), slogext.ErrorAttr(err))
		return
	}

	f.lock.Lock(ctx)
	f.image = image
	f.lock.Unlock(ctx)
}

// RunLints runs linting on this file. Returns whether any lints failed.
//
// This operation requires BuildImage().
func (f *file) RunLints(ctx context.Context) bool {
	if f.isWKT {
		// Well-known types are not linted.
		return false
	}

	f.lock.Lock(ctx)
	workspace := f.workspace
	module := f.module
	image := f.image
	f.lock.Unlock(ctx)

	if module == nil || image == nil {
		f.lsp.logger.Warn(fmt.Sprintf("could not find image for %q", f.uri))
		return false
	}

	f.lsp.logger.Debug(fmt.Sprintf("running lint for %q in %v", f.uri, module.ModuleFullName()))

	lintConfig := workspace.GetLintConfigForOpaqueID(module.OpaqueID())
	err := f.lsp.checkClient.Lint(
		ctx,
		lintConfig,
		image,
		bufcheck.WithPluginConfigs(workspace.PluginConfigs()...),
	)

	if err == nil {
		f.lsp.logger.Warn(fmt.Sprintf("lint generated no errors for %s", f.uri))
		return false
	}

	var annotations bufanalysis.FileAnnotationSet
	if !errors.As(err, &annotations) {
		f.lsp.logger.Warn("error while linting", slog.String("uri", string(f.uri)), slogext.ErrorAttr(err))
		return false
	}

	f.lsp.logger.Warn(fmt.Sprintf("lint generated %d error(s) for %s", len(annotations.FileAnnotations()), f.uri))

	f.lock.Lock(ctx)
	f.lock.Unlock(ctx)
	for _, annotation := range annotations.FileAnnotations() {
		f.lsp.logger.Info(annotation.FileInfo().Path(), " ", annotation.FileInfo().ExternalPath())

		f.diagnostics = append(f.diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(annotation.StartLine()) - 1,
					Character: uint32(annotation.StartColumn()) - 1,
				},
				End: protocol.Position{
					Line:      uint32(annotation.EndLine()) - 1,
					Character: uint32(annotation.EndColumn()) - 1,
				},
			},
			Code:     annotation.Type(),
			Severity: protocol.DiagnosticSeverityError,
			Source:   "buf lint",
			Message:  annotation.Message(),
		})
	}
	return true
}

// IndexSymbols processes the AST of a file and generates symbols for each symbol in
// the document.
func (f *file) IndexSymbols(ctx context.Context) {
	defer slogext.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()

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

	f.lsp.logger.DebugContext(ctx, fmt.Sprintf("symbol indexing complete %s", f.uri))
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
	f.lsp.logger.DebugContext(ctx, "found symbol", slog.Any("symbol", symbol))

	// Check that cursor is before the end of the symbol.
	if comparePositions(symbol.Range().End, cursor) <= 0 {
		return nil
	}

	return symbol
}
