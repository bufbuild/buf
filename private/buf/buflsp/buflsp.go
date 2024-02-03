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
	fileWatcher             *fsnotify.Watcher
	wellKnownTypesModuleSet bufmodule.ModuleSet
	wellKnownTypesResolver  moduleSetResolver
	// paths are normalized.
	pathToFileEntryCache map[string]*fileEntry
	// normalized.
	cacheDirPath string
	lock         sync.Mutex
}

func newServer(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	jsonrpc2Conn jsonrpc2.Conn,
	controller bufctl.Controller,
	cacheDirPath string,
) (*server, error) {
	fileWatcher, err := fsnotify.NewWatcher()
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
	server := &server{
		jsonrpc2Conn:            jsonrpc2Conn,
		logger:                  logger,
		tracer:                  tracer,
		controller:              controller,
		lintHandler:             buflint.NewHandler(logger, tracer),
		breakingHandler:         bufbreaking.NewHandler(logger, tracer),
		pathToFileEntryCache:    make(map[string]*fileEntry),
		fileWatcher:             fileWatcher,
		wellKnownTypesModuleSet: wellKnownTypesModuleSet,
		wellKnownTypesResolver:  wellKnownTypesResolver,
		cacheDirPath:            normalpath.Normalize(cacheDirPath),
	}
	go func() {
		for event := range server.fileWatcher.Events {
			if event.Op&fsnotify.Write == fsnotify.Write {
				server.lock.Lock()
				// TODO: How do we know that event.Name is normalized? Is it?
				if fileEntry, ok := server.getCachedFileEntryForPath(event.Name); ok {
					if err := server.refreshImage(ctx, fileEntry.resolver); err != nil {
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

	// Always load the descriptor.proto file
	if _, err := s.resolveImport(
		ctx,
		s.wellKnownTypesResolver,
		wellKnownTypesDescriptorProtoPath,
	); err != nil {
		return nil, err
	}

	// Reply with capabilities
	return &protocol.InitializeResult{
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
	}, nil
}

func (s *server) Shutdown(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.fileWatcher.Close()
}

func (s *server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	filePath := uriToPath(params.TextDocument.URI)

	// Check if it is already open.
	if fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI); ok {
		fileEntry.refCount++
		if _, err := fileEntry.updateText(ctx, params.TextDocument.Text); err != nil {
			return err
		}
		return nil
	}

	moduleSetResolver := s.getModuleSetResolverForFilePath(ctx, filePath)
	if _, err := s.createFileEntry(ctx, params.TextDocument, moduleSetResolver); err != nil {
		return err
	}
	return s.refreshImage(ctx, moduleSetResolver)
}

func (s *server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI)
	if !ok {
		return fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	matches, err := fileEntry.updateText(ctx, params.ContentChanges[0].Text)
	if err != nil {
		return err
	}
	if matches { // Same as on disk, so safe to refresh the image data.
		if err := s.refreshImage(ctx, fileEntry.resolver); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI); ok {
		s.decrementReferenceCount(fileEntry)
	}
	return nil
}

func (s *server) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	if fileEntry.hasParseError {
		return nil, nil
	}
	var fileData strings.Builder
	if err := bufformat.FormatFileNode(&fileData, fileEntry.fileNode); err != nil {
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
					Line:      uint32(len(fileEntry.lines)),
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

	fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	result := make([]interface{}, len(fileEntry.docSymbols))
	for i, symbol := range fileEntry.docSymbols {
		result[i] = symbol
	}
	return result, nil
}

func (s *server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := fileEntry.getSourcePos(params.Position)

	// Look for import completions.
	result, found, err := s.findImportCompletionsAt(ctx, fileEntry, pos)
	if err != nil {
		return nil, err
	} else if found {
		return completionsToCompletionList(result), nil
	}

	// Look for ast-based symbol completions.
	symbolScope := fileEntry.findSymbolScope(pos)
	if symbolScope != nil {
		if ref := s.findReferenceAt(ctx, fileEntry, symbolScope, pos); ref != nil {
			options := make(completionOptions)
			s.findRefCompletions(ref, options)
			return completionsToCompletionList(options), nil
		}
	}

	// Fallback on prefix-based completions.
	options := s.findPrefixCompletions(ctx, fileEntry, fileEntry.findScope(pos), fileEntry.codeAt(pos))
	return completionsToCompletionList(options), nil
}

func (s *server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location /* Definition | DefinitionLink[] | null */, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := fileEntry.getSourcePos(params.Position)

	var result []protocol.Location
	if importEntry := fileEntry.findImportEntry(pos); importEntry != nil {
		if importEntry.docURI == "" {
			return nil, nil
		}
		result = []protocol.Location{{URI: importEntry.docURI}}
	} else {
		result = s.findReferencedDefLoc(ctx, fileEntry, pos)
	}

	// Make sure all the files exist.
	for _, loc := range result {
		// TODO: This is not a great way to determine if the file is cached.
		if s.isFilePathCachedWellKnownType(loc.URI.Filename()) || s.isFilePathInCachedRemoteModule(loc.URI.Filename()) {
			// This is a temporary file, make sure it exists.
			localPath := normalpath.Unnormalize(loc.URI.Filename())
			if _, err := os.Stat(localPath); err != nil {
				tmpFileEntry, ok := s.getCachedFileEntryForURI(loc.URI)
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
				// TODO: check error
				defer tmpFile.Close()
				if _, err := tmpFile.WriteString(tmpFileEntry.document.Text); err != nil {
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

	fileEntry, ok := s.getCachedFileEntryForURI(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := fileEntry.getSourcePos(params.Position)

	symbols := s.findReferencedSymbols(ctx, fileEntry, pos)
	if len(symbols) == 0 {
		return nil, nil
	}
	symbol := symbols[0]
	refFileEntry, ok := s.getCachedFileEntryForURI(symbol.file)
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", symbol.file)
	}

	codeData, err := refFileEntry.genNodeSignature(symbol.node)
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

func (s *server) getModuleSetResolverForFilePath(ctx context.Context, filePath string) moduleSetResolver {
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
	if s.isFilePathCachedWellKnownType(filePath) {
		return s.wellKnownTypesResolver
	}
	if s.isFilePathInCachedRemoteModule(filePath) {
		return newModuleSetResolver(
			func() (bufmodule.ModuleSet, error) {
				// Normally, this case won't occur, because the file will already be in the cache.
				// This case occurs mainly when the LSP is started and a cache file is already open.
				// We need to recover the module key from the filename in this case.
				key, _, err := s.cachePathToModuleKey(filePath)
				if err != nil {
					return nil, err
				}
				return s.controller.GetWorkspace(ctx, key.String())
			},
		)
	}
	return newModuleSetResolver(
		func() (bufmodule.ModuleSet, error) {
			// TODO: This is returning a ProtoFileRef-based workspace, which is materially different
			// that one created for a ModuleKey-based workspace. In a ProtoFileRef-based workspace,
			// we get a ModuleSet with one target Module, with one target File. Everything else
			// is ignored for i.e. lint. We need to at least have "include_package_files=true", otherwise
			// the lint results will be different than they would be for a remote Module constructed
			// with the ModuleKey-based call above.
			workspace, err := s.controller.GetWorkspace(ctx, filePath)
			if err != nil {
				// TODO: Why are we logging here but not in the above if statement?
				s.logger.Sugar().Warnf(
					"No buf workspace found for %s: %s -- continuing with limited features.",
					filePath,
					err,
				)
			}
			return workspace, err
		},
	)
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
	if entry, ok := s.getCachedFileEntryForURI(uri); ok {
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
			if entry, ok := s.pathToFileEntryCache[fileInfo.ExternalPath()]; ok {
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
		if entry, ok := s.pathToFileEntryCache[externalPath]; ok {
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
		s.moduleCacheDirPath(),
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
	path = strings.TrimPrefix(path, s.moduleCacheDirPath())
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
			s.wellKnownTypesCacheDirPath(),
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

// Assumed to be called inside lock.
func (s *server) decrementReferenceCount(entry *fileEntry) {
	entry.refCount--
	// Doing <= instead of == for safety.
	if entry.refCount <= 0 {
		for _, importEntry := range entry.imports {
			// TODO: in what situations will this be empty? Is this an error?
			if importEntry.docURI != "" {
				if importFile, ok := s.getCachedFileEntryForURI(importEntry.docURI); ok {
					s.decrementReferenceCount(importFile)
				}
			}
		}
		if !entry.isRemote {
			if err := s.fileWatcher.Remove(entry.document.URI.Filename()); err != nil {
				s.logger.Sugar().Errorf("error removing file from watcher: %s", err)
			}
		}
		s.invalidateCachedFileEntryForURI(entry.document.URI)
	}
}

// Create a new file entry with the given contents and metadata.
func (s *server) createFileEntry(
	ctx context.Context,
	item protocol.TextDocumentItem,
	resolver moduleSetResolver,
) (*fileEntry, error) {
	filePath := item.URI.Filename()
	entry := newFileEntry(s, &item, resolver, filePath, s.isFilePathCachedWellKnownType(filePath) || s.isFilePathInCachedRemoteModule(filePath))
	s.pathToFileEntryCache[filePath] = entry
	if err := entry.processText(ctx); err != nil {
		return nil, err
	}
	if !entry.isRemote {
		if err := s.fileWatcher.Add(filePath); err != nil {
			return nil, err
		}
	}
	return entry, nil
}

func (s *server) isFilePathCachedWellKnownType(filePath string) bool {
	// TODO: We should not be using strings.HasPrefix for this, but it is difficult as
	// we do not know whether the path is relative or absolute (do we?).
	return strings.HasPrefix(filePath, s.wellKnownTypesCacheDirPath())
}

func (s *server) isFilePathInCachedRemoteModule(filePath string) bool {
	// TODO: We should not be using strings.HasPrefix for this, but it is difficult as
	// we do not know whether the path is relative or absolute (do we?).
	return strings.HasPrefix(filePath, s.moduleCacheDirPath())
}

func (s *server) getCachedFileEntryForURI(uri uri.URI) (*fileEntry, bool) {
	return s.getCachedFileEntryForPath(uriToPath(uri))
}

func (s *server) getCachedFileEntryForPath(filePath string) (*fileEntry, bool) {
	fileEntry, ok := s.pathToFileEntryCache[filePath]
	return fileEntry, ok
}

func (s *server) invalidateCachedFileEntryForURI(uri uri.URI) {
	s.invalidateCachedFileEntryForPath(uriToPath(uri))
}

func (s *server) invalidateCachedFileEntryForPath(filePath string) {
	delete(s.pathToFileEntryCache, filePath)
}

// wellKnownTypesCacheDirPath returns the full path the the well-known types cache directory.
func (s *server) wellKnownTypesCacheDirPath() string {
	return normalpath.Join(s.cacheDirPath, wellKnownTypesCacheRelDirPath)
}

// moduleCacheDirPath returns the full path to the module cache directory.
//
// This stores remote modules that were downloaded for the LSP.
func (s *server) moduleCacheDirPath() string {
	return normalpath.Join(s.cacheDirPath, moduleCacheRelDirPath)
}
