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

package buflsp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/fsnotify/fsnotify"
	"github.com/gofrs/uuid/v5"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

var (
	// wellKnownTypesDescriptorProtoPath is the path of the descriptor.proto well-known types file.
	wellKnownTypesDescriptorProtoPath = normalpath.Join("google", "protobuf", "descriptor.proto")
	// wellKnownTypesCacheRelDirPath is the relative path to the cache directory for materialized
	// well-known types .proto files.
	wellKnownTypesCacheRelDirPath = "wkt"
	// moduleCacheRelDirPath is the relative path to the cache directory in its newest iteration.
	moduleCacheRelDirPath = "modules"
)

// NewServer returns a new LSP server for the jsonrpc connection.
func NewServer(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	jsonrpc2Conn jsonrpc2.Conn,
	controller bufctl.Controller,
	cacheDirPath string,
) (protocol.Server, error) {
	return newServer(ctx, logger, tracer, jsonrpc2Conn, controller, cacheDirPath)
}

// *** PRIVATE ***

type server struct {
	nopServer

	jsonrpc2Conn            jsonrpc2.Conn
	logger                  *zap.Logger
	tracer                  tracing.Tracer
	controller              bufctl.Controller
	lintHandler             buflint.Handler
	breakingHandler         bufbreaking.Handler
	fileCache               map[string]*fileEntry
	fileWatcher             *fsnotify.Watcher
	wellKnownTypesModuleSet bufmodule.ModuleSet
	wellKnownTypesResolver  moduleSetResolver

	cacheDirPath               string
	wellKnownTypesCacheDirPath string
	moduleCacheDirPath         string

	folders   []protocol.WorkspaceFolder
	clientCap protocol.ClientCapabilities

	lock sync.Mutex
}

func newServer(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	jsonrpc2Conn jsonrpc2.Conn,
	controller bufctl.Controller,
	cacheDirPath string,
) (*server, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	wellKnownTypesResolver := newModuleSetResolver(func() (bufmodule.ModuleSet, error) {
		moduleSetBuilder := bufmodule.NewModuleSetBuilder(
			ctx,
			tracer,
			bufmodule.NopModuleDataProvider,
			bufmodule.NopCommitProvider,
		)
		moduleSetBuilder.AddLocalModule(
			datawkt.ReadBucket,
			".",
			true,
		)
		return moduleSetBuilder.Build()
	})
	wellKnownTypesModuleSet, err := wellKnownTypesResolver.ModuleSet()
	if err != nil {
		return nil, err
	}
	wellKnownTypesCacheDirPath := normalpath.Join(
		cacheDirPath,
		wellKnownTypesCacheRelDirPath,
	)
	moduleCacheDirPath := normalpath.Join(
		cacheDirPath,
		moduleCacheRelDirPath,
	)
	server := &server{
		jsonrpc2Conn:               jsonrpc2Conn,
		logger:                     logger,
		tracer:                     tracer,
		controller:                 controller,
		lintHandler:                buflint.NewHandler(logger, tracer),
		breakingHandler:            bufbreaking.NewHandler(logger, tracer),
		fileCache:                  make(map[string]*fileEntry),
		fileWatcher:                watcher,
		wellKnownTypesModuleSet:    wellKnownTypesModuleSet,
		wellKnownTypesResolver:     wellKnownTypesResolver,
		cacheDirPath:               cacheDirPath,
		wellKnownTypesCacheDirPath: wellKnownTypesCacheDirPath,
		moduleCacheDirPath:         moduleCacheDirPath,
	}
	go func() {
		for event := range server.fileWatcher.Events {
			if event.Op&fsnotify.Write == fsnotify.Write {
				server.lock.Lock()
				if entry, ok := server.fileCache[event.Name]; ok {
					if err := server.refreshImage(ctx, entry.resolver); err != nil {
						server.logger.Sugar().Errorf("failed to build new image: %s", err)
					}
				}
				server.lock.Unlock()
			}
		}
	}()
	return server, nil
}

func (s *server) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Store client info
	s.folders = params.WorkspaceFolders
	s.clientCap = params.Capabilities

	// Always load the descriptor.proto file
	if _, err := s.resolveImport(
		ctx,
		s.wellKnownTypesResolver,
		wellKnownTypesDescriptorProtoPath,
	); err != nil {
		return nil, err
	}

	// Reply with capabilities
	initializeResult := &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"."},
			},
			DocumentFormattingProvider: true,
			DefinitionProvider:         true,
			DocumentSymbolProvider:     true,
			HoverProvider:              true,
			SemanticTokensProvider:     &protocol.SemanticTokensOptions{},
		},
	}
	return initializeResult, nil
}

func (s *server) Shutdown(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.fileWatcher.Close()
}

func (s *server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	filename := normalpath.Normalize(params.TextDocument.URI.Filename())

	// Check if it is already open.
	if entry, ok := s.fileCache[filename]; ok {
		entry.refCount++
		if _, err := entry.updateText(ctx, params.TextDocument.Text); err != nil {
			return err
		}
		return nil
	}
	// What is going on (from a GitHub comment):
	//
	// - `DidOpen` is called when we get a [`textDocument/didOpen`](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_didOpen) event from the LSP host.
	// - This signals that a new document window was opened in the editor.
	// - The LSP server will create a new file entry (`createFileEntry`) and build an image (`refreshImage`).
	// - Some (but not all) LSP operations require a module set to function properly.
	// - In order to avoid storing and carefully checking for `nil` throughout the codebase, which proved to be error-prone, module set resolution is deferred until it is needed and the result is memoized using `syncext.OnceValues`. That's the ceremony re: "resolving" the module set.
	// - The 3 conditional branches here decide how the deferred module set should be resolved:
	//   - Materialized well-known types files will be provided by the built-in well-known types module set resolver.
	//   - Materialized remote dependency files will be provided by parsing the module key out of the path.
	//   - For all other files, we need to find the appropriate workspace.
	//
	// tl;dr: We need a way to get the enclosing workspace for a file.
	var resolver moduleSetResolver
	if strings.HasPrefix(filename, s.wellKnownTypesCacheDirPath) {
		resolver = s.wellKnownTypesResolver
	} else if strings.HasPrefix(filename, s.moduleCacheDirPath) {
		resolver = newModuleSetResolver(func() (bufmodule.ModuleSet, error) {
			// Normally, this case won't occur, because the file will already be in the cache.
			// This case occurs mainly when the LSP is started and a cache file is already open.
			// We need to recover the module key from the filename in this case.
			key, _, err := s.cachePathToModuleKey(filename)
			if err != nil {
				return nil, err
			}
			return s.controller.GetWorkspace(ctx, key.String())
		})
	} else {
		resolver = newModuleSetResolver(func() (bufmodule.ModuleSet, error) {
			// TODO: This is returning a ProtoFileRef-based workspace, which is materially different
			// that one created for a ModuleKey-based workspace. In a ProtoFileRef-based workspace,
			// we get a ModuleSet with one target Module, with one target File. Everything else
			// is ignored for i.e. lint. We need to at least have "include_package_files=true", otherwise
			// the lint results will be different than they would be for a remote Module constructed
			// with the ModuleKey-based call above.
			workspace, err := s.controller.GetWorkspace(ctx, filename)
			if err != nil {
				s.logger.Sugar().Warnf("No buf workspace found for %s: %s -- continuing with limited features.", filename, err)
			}
			return workspace, err
		})
	}

	// Create a new file entry for the file
	if _, err := s.createFileEntry(ctx, params.TextDocument, resolver); err != nil {
		return err
	}
	if err := s.refreshImage(ctx, resolver); err != nil {
		return err
	}
	return nil
}

func (s *server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	entry, ok := s.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	matches, err := entry.updateText(ctx, params.ContentChanges[0].Text)
	if err != nil {
		return err
	}
	if matches { // Same as on disk, so safe to refresh the image data.
		if err := s.refreshImage(ctx, entry.resolver); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if entry, ok := s.fileCache[params.TextDocument.URI.Filename()]; ok {
		s.decrementReferenceCount(entry)
	}
	return nil
}

func (s *server) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	entry, ok := s.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	if entry.hasParseError {
		return nil, nil
	}
	fileData := strings.Builder{}
	if err := bufformat.FormatFileNode(&fileData, entry.fileNode); err != nil {
		return nil, err
	}
	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      0,
					Character: 0,
				},
				End: protocol.Position{
					Line:      uint32(len(entry.lines)),
					Character: 0,
				},
			},
			NewText: fileData.String(),
		},
	}, nil
}

func (s *server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{} /* []SymbolInformation | []DocumentSymbol */, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	entry, ok := s.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	result := make([]interface{}, len(entry.docSymbols))
	for i, symbol := range entry.docSymbols {
		result[i] = symbol
	}
	return result, nil
}

func (s *server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	entry, ok := s.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := entry.getSourcePos(params.Position)

	// Look for import completions.
	{
		result, found, err := s.findImportCompletionsAt(ctx, entry, pos)
		if err != nil {
			return nil, err
		} else if found {
			return completionsToCompletionList(result), nil
		}
	}

	// Look for ast-based symbol completions.
	sybmolScope := entry.findSymbolScope(pos)
	if sybmolScope != nil {
		if ref := s.findReferenceAt(ctx, entry, sybmolScope, pos); ref != nil {
			options := make(completionOptions)
			s.findRefCompletions(ref, options)
			return completionsToCompletionList(options), nil
		}
	}

	// Fallback on prefix-based completions.
	options := s.findPrefixCompletions(ctx, entry, entry.findScope(pos), entry.codeAt(pos))
	return completionsToCompletionList(options), nil
}

func (s *server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location /* Definition | DefinitionLink[] | null */, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	entry, ok := s.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := entry.getSourcePos(params.Position)

	var result []protocol.Location
	if importEntry := entry.findImportEntry(pos); importEntry != nil {
		if importEntry.docURI == "" {
			return nil, nil
		}
		result = []protocol.Location{{URI: importEntry.docURI}}
	} else {
		result = s.findReferencedDefLoc(ctx, entry, pos)
	}

	// Make sure all the files exist.
	for _, loc := range result {
		if strings.HasPrefix(loc.URI.Filename(), s.cacheDirPath) {
			// This is a temporary file, make sure it exists.
			localPath := loc.URI.Filename()
			if _, err := os.Stat(localPath); err != nil {
				tmpEntry, ok := s.fileCache[loc.URI.Filename()]
				if !ok {
					return nil, fmt.Errorf("unknown file: %s", loc.URI)
				}

				// Create the file.
				if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
					return nil, err
				}
				tmpFile, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0444)
				if err != nil {
					return nil, err
				}
				defer tmpFile.Close()
				if _, err := tmpFile.WriteString(tmpEntry.document.Text); err != nil {
					return nil, err
				}
			}
		}
	}

	return result, nil
}

func (s *server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	entry, ok := s.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := entry.getSourcePos(params.Position)

	symbols := s.findReferencedSymbols(ctx, entry, pos)
	if len(symbols) == 0 {
		return nil, nil
	}
	symbol := symbols[0]
	refEntry, ok := s.fileCache[symbol.file.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", symbol.file)
	}

	codeData, err := refEntry.genNodeSignature(symbol.node)
	if err != nil {
		return nil, err
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: fmt.Sprintf("### %s\n```proto\n%s\n```", strings.Join(symbol.name(), "."), codeData),
		},
	}, nil
}

func (s *server) findImportWithResolver(
	ctx context.Context,
	resolver moduleSetResolver,
	path string,
) (bufmodule.ModuleReadBucket, bufmodule.File, error) {
	moduleReadBucket, err := resolver.ModuleReadBucket()
	if err != nil {
		return nil, nil, err
	}
	file, err := moduleReadBucket.GetFile(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	return moduleReadBucket, file, nil
}

func (s *server) findImport(
	ctx context.Context,
	resolver moduleSetResolver,
	path string,
) (bufmodule.ModuleReadBucket, bufmodule.File, error) {
	bucket, file, err := s.findImportWithResolver(ctx, resolver, path)
	if err == nil {
		return bucket, file, nil
	}
	bucket, file, err2 := s.findImportWithResolver(ctx, s.wellKnownTypesResolver, path)
	if err2 == nil {
		return bucket, file, nil
	}
	err = fmt.Errorf("could not resolve import in workspace: %w", err)
	if !errors.Is(err2, fs.ErrNotExist) {
		err = fmt.Errorf("%w (additionally, an unexpected error occurred trying to resolve the import from well-known types: %v)", err, err2)
	}
	return nil, nil, err
}

func (s *server) resolveImport(ctx context.Context, resolver moduleSetResolver, path string) (*fileEntry, error) {
	_, file, err := s.findImport(ctx, resolver, path)
	if err != nil {
		return nil, err
	}
	localPath, err := s.localPathForImport(ctx, file)
	if err != nil {
		return nil, err
	}
	uri := uri.File(localPath)
	if entry, ok := s.fileCache[uri.Filename()]; ok {
		entry.refCount++
		return entry, nil
	}
	fileData, err := ioext.ReadAllAndClose(file)
	if err != nil {
		return nil, err
	}
	return s.createFileEntry(
		ctx,
		protocol.TextDocumentItem{
			URI:  uri,
			Text: string(fileData),
		},
		newModuleSetResolver(
			func() (bufmodule.ModuleSet, error) {
				return file.Module().ModuleSet(), nil
			},
		),
	)
}

// Refresh the results for the given file entry.
func (s *server) updateDiagnostics(ctx context.Context, entry *fileEntry) error {
	if s.jsonrpc2Conn == nil {
		return nil
	}
	diagParams := protocol.PublishDiagnosticsParams{
		URI: entry.document.URI,
	}
	switch {
	case entry.bufDiags != nil:
		diagParams.Diagnostics = entry.bufDiags
	case entry.parseDiags != nil:
		diagParams.Diagnostics = entry.parseDiags
	default:
		diagParams.Diagnostics = []protocol.Diagnostic{}
	}
	return s.jsonrpc2Conn.Notify(ctx, "textDocument/publishDiagnostics", diagParams)
}

// Create a new file entry with the given contents and metadata.
func (s *server) createFileEntry(ctx context.Context, item protocol.TextDocumentItem, resolver moduleSetResolver) (*fileEntry, error) {
	filename := item.URI.Filename()
	// TODO: The strings.HasPrefix call signifying remote seems super prone to error. At the minimum,
	// we need to factor that out into a function, and explain why this denotes that this is remote.
	entry := newFileEntry(s, &item, resolver, filename, strings.HasPrefix(filename, s.cacheDirPath))
	s.fileCache[filename] = entry
	if err := entry.processText(ctx); err != nil {
		return nil, err
	}
	if !entry.isRemote {
		if err := s.fileWatcher.Add(filename); err != nil {
			return nil, err
		}
	}
	return entry, nil
}

func (s *server) refreshImage(ctx context.Context, resolver moduleSetResolver) error {
	moduleSet, err := resolver.ModuleSet()
	if err != nil {
		return err
	}
	moduleReadBucket, err := resolver.ModuleReadBucket()
	if err != nil {
		return err
	}
	if err := moduleReadBucket.WalkFileInfos(
		ctx,
		func(fileInfo bufmodule.FileInfo) error {
			if entry, ok := s.fileCache[fileInfo.ExternalPath()]; ok {
				entry.bufDiags = nil
			}
			return nil
		},
	); err != nil {
		return err
	}
	// TODO: diagsByFile is flawed; it maps on ExternalPath, which may or may not be meaningful.
	// The LSP should probably instead map OS paths to modules on its own and use this to tie files
	// to diagnostics and etc.
	diagsByFile := make(map[string][]protocol.Diagnostic)
	image, err := bufimage.BuildImage(ctx, s.tracer, moduleReadBucket)
	if err != nil {
		var fileAnnotationSet bufanalysis.FileAnnotationSet
		if !errors.As(err, &fileAnnotationSet) {
			return err
		}
		for _, annot := range fileAnnotationSet.FileAnnotations() {
			diagsByFile[annot.FileInfo().ExternalPath()] = append(diagsByFile[annot.FileInfo().ExternalPath()],
				annotationToDiagnostic(annot, protocol.DiagnosticSeverityError))
		}
	}
	// TODO: This should never need to happen, why are we upcasting?
	if workspace, ok := moduleSet.(bufworkspace.Workspace); ok && image != nil {
		for _, module := range moduleSet.Modules() {
			if err := s.lintHandler.Check(ctx, workspace.GetLintConfigForOpaqueID(module.OpaqueID()), image); err != nil {
				var fileAnnotationSet bufanalysis.FileAnnotationSet
				if !errors.As(err, &fileAnnotationSet) {
					return err
				}
				for _, annot := range fileAnnotationSet.FileAnnotations() {
					diagsByFile[annot.FileInfo().ExternalPath()] = append(diagsByFile[annot.FileInfo().ExternalPath()],
						annotationToDiagnostic(annot, protocol.DiagnosticSeverityWarning))
				}
			}
		}
	}
	for externalPath, diags := range diagsByFile {
		if entry, ok := s.fileCache[externalPath]; ok {
			entry.bufDiags = diags
			if err := s.updateDiagnostics(ctx, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

// Creates a cache path for a module key and module file path. This cache is local to the LSP.
// The format of the cache path is:
// <remote>/<owner>/<repository>/<commit id>/<module file path>
func (s *server) moduleKeyToCachePath(
	key bufmodule.ModuleKey,
	moduleFilePath string,
) (string, error) {
	return normalpath.Join(
		s.moduleCacheDirPath,
		key.ModuleFullName().Registry(),
		key.ModuleFullName().Owner(),
		key.ModuleFullName().Name(),
		key.CommitID().String(),
		moduleFilePath,
	), nil
}

// Parses a module key out of the cache path. This is the inverse of moduleKeyToCachePath.
// Returns both the module key and the module file path that the cache path represents.
func (s *server) cachePathToModuleKey(path string) (bufmodule.ModuleKey, string, error) {
	path = strings.TrimPrefix(path, s.moduleCacheDirPath)
	normalpath.Components(path)
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return nil, "", fmt.Errorf("invalid temporary file path: %s", path)
	}
	registry, owner, name, commitIDString := parts[0], parts[1], parts[2], parts[3]
	moduleFilePath := normalpath.Join(parts[4:]...)
	moduleFullName, err := bufmodule.NewModuleFullName(registry, owner, name)
	if err != nil {
		return nil, "", err
	}
	commitID, err := uuid.FromString(commitIDString)
	if err != nil {
		return nil, "", err
	}
	key, err := bufmodule.NewModuleKey(moduleFullName, commitID, func() (bufmodule.Digest, error) {
		return nil, errors.New("no digest available")
	})
	if err != nil {
		return nil, "", err
	}
	return key, moduleFilePath, nil
}

// localPathForImport determines the local path on-disk that corresponds to the import path.
func (s *server) localPathForImport(
	ctx context.Context,
	file bufmodule.File,
) (string, error) {
	module := file.Module()
	// TODO: This is doing a pointer comparison, this is extremely unsafe.
	// We should instead be able to tell what is going on here based on the file path.
	isWellKnownTypesModule := module.ModuleSet() == s.wellKnownTypesModuleSet
	if !isWellKnownTypesModule && module.IsLocal() {
		return file.ExternalPath(), nil
	}
	digest, err := module.Digest(bufmodule.DigestTypeB5)
	if err != nil {
		return "", err
	}
	if isWellKnownTypesModule {
		return normalpath.Join(
			s.wellKnownTypesCacheDirPath,
			digest.Type().String(),
			// TODO: We should not be using digests as part of these paths.
			hex.EncodeToString(digest.Value()),
			file.Path(),
		), nil
	}
	moduleFullName := module.ModuleFullName()
	if moduleFullName == nil {
		// TODO: How do you have a guarantee in this function that the module is remote?
		return "", syserror.Newf("remote module %q had nil ModuleFullName", module.OpaqueID())
	}
	key, err := bufmodule.NewModuleKey(module.ModuleFullName(), module.CommitID(), func() (bufmodule.Digest, error) {
		return digest, nil
	})
	if err != nil {
		return "", err
	}
	return s.moduleKeyToCachePath(key, file.Path())
}

func (s *server) decrementReferenceCount(entry *fileEntry) {
	entry.refCount--
	if entry.refCount == 0 {
		for _, importEntry := range entry.imports {
			if importEntry.docURI != "" {
				if importFile, ok := s.fileCache[importEntry.docURI.Filename()]; ok {
					s.decrementReferenceCount(importFile)
				}
			}
		}
		if !entry.isRemote {
			if err := s.fileWatcher.Remove(entry.document.URI.Filename()); err != nil {
				s.logger.Sugar().Errorf("error removing file from watcher: %s", err)
			}
		}
		delete(s.fileCache, entry.document.URI.Filename())
	}
}

func annotationToDiagnostic(
	annotation bufanalysis.FileAnnotation,
	severity protocol.DiagnosticSeverity,
) protocol.Diagnostic {
	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(annotation.StartLine() - 1),
				Character: uint32(annotation.StartColumn() - 1),
			},
			End: protocol.Position{
				Line:      uint32(annotation.EndLine() - 1),
				Character: uint32(annotation.EndColumn() - 1),
			},
		},
		Severity: severity,
		Message:  fmt.Sprintf("%s (%s)", annotation.Message(), annotation.Type()),
	}
}
