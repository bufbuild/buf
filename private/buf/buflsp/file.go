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
	"github.com/bufbuild/buf/private/pkg/storage"
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
	lsp *lsp
	uri protocol.URI

	text string
	// Version is an opaque version identifier given to us by the LSP client. This
	// is used in the protocol to disambiguate which version of a file e.g. publishing
	// diagnostics or symbols an operating refers to.
	version int32
	hasText bool // Whether this file has ever had text read into it.
	// Always set false->true. Once true, never becomes false again.

	workspace bufworkspace.Workspace
	module    bufmodule.Module

	objectInfo             storage.ObjectInfo
	importablePathToObject map[string]storage.ObjectInfo

	fileNode     *ast.FileNode
	packageNode  *ast.PackageNode
	diagnostics  []protocol.Diagnostic
	importToFile map[string]*file
	symbols      []*symbol
	image        bufimage.Image
}

// IsWKT returns whether this file corresponds to a well-known type.
func (f *file) IsWKT() bool {
	_, ok := f.objectInfo.(wktObjectInfo)
	return ok
}

// IsLocal returns whether this is a local file, i.e. a file that the editor
// is editing and not something from e.g. the BSR.
func (f *file) IsLocal() bool {
	if f.objectInfo == nil {
		return false
	}

	return f.objectInfo.LocalPath() == f.objectInfo.ExternalPath()
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

	f.fileNode = nil
	f.packageNode = nil
	f.diagnostics = nil
	f.importablePathToObject = nil
	f.importToFile = nil
	f.symbols = nil
	f.image = nil

	for _, imported := range f.importToFile {
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

// IsOpenInEditor returns whether this file was opened in the LSP client's
// editor.
//
// Some files may be opened as dependencies, so we want to avoid doing extra
// work like sending progress notifications.
func (f *file) IsOpenInEditor() bool {
	return f.version != -1 // See [file.ReadFromDisk].
}

// ReadFromDisk reads this file from disk if it has never had data loaded into it before.
//
// If it has been read from disk before, or has received updates from the LSP client, this
// function returns nil.
func (f *file) ReadFromDisk(ctx context.Context) (err error) {
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

	f.lsp.logger.Info(fmt.Sprintf("new file version: %v, %v -> %v", f.uri, f.version, version))
	f.version = version
	f.text = text
	f.hasText = true
}

// Refresh rebuilds all of a file's internal book-keeping.
//
// If deep is set, this will also load imports and refresh those, too.
func (f *file) Refresh(ctx context.Context) {
	var progress *progress
	if f.IsOpenInEditor() {
		// NOTE: Nil progress does nothing when methods are called. This helps
		// minimize RPC spam from the client when indexing lots of files.
		progress = newProgress(f.lsp)
	}
	progress.Begin(ctx, "Indexing")

	progress.Report(ctx, "Parsing AST", 1.0/6)
	f.RefreshAST(ctx)

	progress.Report(ctx, "Indexing Imports", 2.0/6)
	f.IndexImports(ctx)

	progress.Report(ctx, "Detecting Module", 3.0/6)
	f.FindModule(ctx)

	progress.Report(ctx, "Linking Descriptors", 4.0/6)
	f.BuildImage(ctx)
	f.RunLints(ctx)

	progress.Report(ctx, "Indexing Symbols", 5.0/6)
	f.IndexSymbols(ctx)

	progress.Done(ctx)

	// NOTE: Diagnostics are published unconditionally. This is necessary even
	// if we have zero diagnostics, so that the client correctly ticks over from
	// n > 0 diagnostics to 0 diagnostics.
	f.PublishDiagnostics(ctx)
}

// RefreshAST reparses the file and generates diagnostics if necessary.
//
// Returns whether a reparse was necessary.
func (f *file) RefreshAST(ctx context.Context) bool {
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
	if !f.IsOpenInEditor() {
		// If the file does get opened by the editor, the server will call
		// Refresh() and this function will retry sending diagnostics. Which is
		// to say: returning here does not result in stale diagnostics on the
		// client.
		return
	}

	defer slogext.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()

	// NOTE: We need to avoid sending a JSON null here, so we replace it with
	// a non-nil empty slice when the diagnostics slice is nil.
	diagnostics := f.diagnostics
	if f.diagnostics == nil {
		diagnostics = []protocol.Diagnostic{}
	}

	// Publish the diagnostics. This error is automatically logged by the LSP framework.
	_ = f.lsp.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI: f.uri,
		// NOTE: For some reason, Version is int32 in the document struct, but uint32 here.
		// This seems like a bug in the LSP protocol package.
		Version:     uint32(f.version),
		Diagnostics: diagnostics,
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

	f.workspace = workspace
	f.module = module
}

// IndexImports finds URIs for all of the files imported by this file.
func (f *file) IndexImports(ctx context.Context) {
	defer slogext.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()

	if f.fileNode == nil || f.importToFile != nil {
		return
	}

	importable, err := findImportable(ctx, f.uri, f.lsp)
	if err != nil {
		f.lsp.logger.Warn(fmt.Sprintf("could not compute importable files for %s: %s", f.uri, err))
		if f.importablePathToObject == nil {
			return
		}
	} else if f.importablePathToObject == nil {
		f.importablePathToObject = importable
	}

	// Find the FileInfo for this path. The crazy thing is that it may appear in importable
	// multiple times, with different path lengths! We want to pick the one with the longest path
	// length.
	for _, fileInfo := range importable {
		if fileInfo.LocalPath() == f.uri.Filename() {
			if f.objectInfo != nil && len(f.objectInfo.Path()) > len(fileInfo.Path()) {
				continue
			}
			f.objectInfo = fileInfo
		}
	}

	f.importToFile = make(map[string]*file)
	for _, decl := range f.fileNode.Decls {
		node, ok := decl.(*ast.ImportNode)
		if !ok {
			continue
		}

		// If this is an external file, it will be in the cache and therefore
		// finding imports via lsp.findImportable() will not work correctly:
		// the bucket for the workspace found for a dependency will have
		// truncated paths, and those workspace files will appear to be
		// local rather than external.
		//
		// Thus, we search for name and all of its path suffixes. This is not
		// ideal but is our only option in this case.
		var fileInfo storage.ObjectInfo
		var pathWasTruncated bool
		name := node.Name.AsString()
		for {
			fileInfo, ok = importable[name]
			if ok {
				break
			}

			idx := strings.Index(name, "/")
			if idx == -1 {
				break
			}

			name = name[idx+1:]
			pathWasTruncated = true
		}
		if fileInfo == nil {
			f.lsp.logger.Warn(fmt.Sprintf("could not find URI for import %q", node.Name.AsString()))
			continue
		}
		if pathWasTruncated && !strings.HasSuffix(fileInfo.LocalPath(), node.Name.AsString()) {
			// Verify that the file we found, with a potentially too-short path, does in fact have
			// the "correct" full path as a prefix. E.g., suppose we import a/b/c.proto. We find
			// c.proto in importable. Now, we look at the full local path, which we expect to be of
			// the form /home/blah/.cache/blah/a/b/c.proto or similar. If it does not contain
			// a/b/c.proto as a suffix, we didn't find our file.
			f.lsp.logger.Warn(fmt.Sprintf("could not find URI for import %q, but found same-suffix path %q", node.Name.AsString(), fileInfo.LocalPath()))
			continue
		}

		f.lsp.logger.Debug(
			"mapped import -> path",
			slog.String("import", name),
			slog.String("path", fileInfo.LocalPath()),
		)

		var imported *file
		if fileInfo.LocalPath() == f.uri.Filename() {
			imported = f
		} else {
			imported = f.Manager().Open(ctx, protocol.URI("file://"+fileInfo.LocalPath()))
		}

		imported.objectInfo = fileInfo
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
			imported.objectInfo = descriptorFile
			f.importToFile[descriptorPath] = imported
		}
	}

	// FIXME: This algorithm is not correct: it does not account for `import public`.
	fileImports := f.importToFile

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
	importable := f.importablePathToObject
	fileInfo := f.objectInfo

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
		f.diagnostics = report.diagnostics
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

	f.image = image
}

// RunLints runs linting on this file. Returns whether any lints failed.
//
// This operation requires BuildImage().
func (f *file) RunLints(ctx context.Context) bool {
	if f.IsWKT() {
		// Well-known types are not linted.
		return false
	}

	workspace := f.workspace
	module := f.module
	image := f.image

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
			Source:   serverName,
			Message:  annotation.Message(),
		})
	}
	return true
}

// IndexSymbols processes the AST of a file and generates symbols for each symbol in
// the document.
func (f *file) IndexSymbols(ctx context.Context) {
	defer slogext.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()

	// Throw away all the old symbols. Unlike other indexing functions, we rebuild
	// symbols unconditionally. This is because if this file depends on a file
	// that has since been modified, we may need to update references.
	f.symbols = nil

	// Generate new symbols.
	walker := newWalker(f)
	walker.Walk(f.fileNode, f.fileNode)
	f.symbols = walker.symbols

	// Finally, sort the symbols in position order, with shorter symbols sorting smaller.
	slices.SortFunc(f.symbols, func(s1, s2 *symbol) int {
		diff := s1.info.Start().Offset - s2.info.Start().Offset
		if diff == 0 {
			return s1.info.End().Offset - s2.info.End().Offset
		}
		return diff
	})

	symbols := f.symbols
	for _, symbol := range symbols {
		symbol.ResolveCrossFile(ctx)
	}

	f.lsp.logger.DebugContext(ctx, fmt.Sprintf("symbol indexing complete %s", f.uri))
}

// SymbolAt finds a symbol in this file at the given cursor position, if one exists.
//
// Returns nil if no symbol is found.
func (f *file) SymbolAt(ctx context.Context, cursor protocol.Position) *symbol {
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

// findImportable finds all files that can potentially be imported by the proto file at
// uri. This returns a map from potential Protobuf import path to the URI of the file it would import.
//
// Note that this performs no validation on these files, because those files might be open in the
// editor and might contain invalid syntax at the moment. We only want to get their paths and nothing
// more.
func findImportable(
	ctx context.Context,
	uri protocol.URI,
	lsp *lsp,
) (map[string]storage.ObjectInfo, error) {
	// This does not use Controller.GetImportableImageFileInfos because:
	//
	// 1. That function throws away Module/ModuleSet information, because it
	//    converts the module contents into ImageFileInfos.
	//
	// 2. That function does not specify which of the files it returns are
	//    well-known imports. Previously, we were making an educated guess about
	//    which files were well-known, but this resulted in subtle classification
	//    bugs.
	//
	// Doing the file walk here manually helps us retain some control over what
	// data is discarded.
	workspace, err := lsp.controller.GetWorkspace(ctx, uri.Filename())
	if err != nil {
		return nil, err
	}

	imports := make(map[string]storage.ObjectInfo)
	for _, module := range workspace.Modules() {
		err = module.WalkFileInfos(ctx, func(fileInfo bufmodule.FileInfo) error {
			if fileInfo.FileType() != bufmodule.FileTypeProto {
				return nil
			}

			imports[fileInfo.Path()] = fileInfo

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	err = lsp.wktBucket.Walk(ctx, "", func(object storage.ObjectInfo) error {
		imports[object.Path()] = wktObjectInfo{object}
		return nil
	})
	if err != nil {
		return nil, err
	}

	lsp.logger.Debug(fmt.Sprintf("found imports for %q: %#v", uri, imports))

	return imports, nil
}

// wktObjectInfo is a concrete type to help us identify WKTs among the
// importable files.
type wktObjectInfo struct {
	storage.ObjectInfo
}
