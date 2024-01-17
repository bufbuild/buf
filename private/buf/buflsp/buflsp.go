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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
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
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/fsnotify/fsnotify"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

var (
	// wellKnownTypesDescriptorProtoPath is the path of the descriptor.proto well-known types file.
	wellKnownTypesDescriptorProtoPath = normalpath.Join("google", "protobuf", "descriptor.proto")
	// lspWellKnownTypesCacheRelDirPath is the relative path to the cache directory for materialized
	// well-known types .proto files.
	lspWellKnownTypesCacheRelDirPath = normalpath.Join("v3", "lsp", "wkt")
	// v3CacheModuleRelDirPath is the relative path to the cache directory in its newest iteration.
	// NOTE: This needs to be kept in sync with module_data_provider.
	v3CacheModuleRelDirPath = normalpath.Join("v3", "modules")
	// v3CacheExternalModuleDataFilesDir is the subdirectory within a module commit's cache
	// directory where its external data (e.g. proto files) is stored.
	v3CacheExternalModuleDataFilesDir = "files"
)

// NewServer returns a new LSP server for the jsonrpc connection.
func NewServer(
	ctx context.Context,
	jsonrpc2Conn jsonrpc2.Conn,
	container appext.Container,
	controller bufctl.Controller,
) (protocol.Server, error) {
	return newServer(ctx, jsonrpc2Conn, container, controller)
}

// *** PRIVATE ***

type server struct {
	nopServer

	jsonrpc2Conn            jsonrpc2.Conn
	logger                  *zap.Logger
	container               appext.Container
	controller              bufctl.Controller
	lintHandler             buflint.Handler
	breakingHandler         bufbreaking.Handler
	fileCache               map[string]*fileEntry
	fileWatcher             *fsnotify.Watcher
	wellKnownTypesModuleSet bufmodule.ModuleSet
	wellKnownTypesResolver  moduleSetResolver

	folders   []protocol.WorkspaceFolder
	clientCap protocol.ClientCapabilities

	lock sync.Mutex
}

func newServer(
	ctx context.Context,
	jsonrpc2Conn jsonrpc2.Conn,
	container appext.Container,
	controller bufctl.Controller,
) (*server, error) {
	logger := container.Logger()
	tracer := tracing.NewTracer(container.Tracer())
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	wellKnownTypesResolver := newModuleSetResolver(func() (bufmodule.ModuleSet, error) {
		moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, bufmodule.NopModuleDataProvider)
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
		container:               container,
		controller:              controller,
		lintHandler:             buflint.NewHandler(logger, tracer),
		breakingHandler:         bufbreaking.NewHandler(logger, tracer),
		fileCache:               make(map[string]*fileEntry),
		fileWatcher:             watcher,
		wellKnownTypesModuleSet: wellKnownTypesModuleSet,
		wellKnownTypesResolver:  wellKnownTypesResolver,
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

	filename := params.TextDocument.URI.Filename()

	// Check if it is already open.
	if entry, ok := s.fileCache[filename]; ok {
		entry.refCount++
		if _, err := entry.updateText(ctx, s, params.TextDocument.Text); err != nil {
			return err
		}
		return nil
	}
	var resolver moduleSetResolver
	wellKnownTypesCachePath := normalpath.Join(
		s.container.CacheDirPath(),
		lspWellKnownTypesCacheRelDirPath,
	)
	if strings.HasPrefix(normalpath.Normalize(filename), wellKnownTypesCachePath) {
		resolver = s.wellKnownTypesResolver
	} else {
		resolver = newModuleSetResolver(func() (bufmodule.ModuleSet, error) {
			workspace, err := s.controller.GetWorkspace(ctx, filename)
			if err != nil {
				s.logger.Sugar().Warnf("No buf workspace found for %s: %s -- continuing with limited features.", filename, err)
			}
			return workspace, err
		})
	}

	// Create a new file entry for the file
	_, err := s.createFileEntry(ctx, params.TextDocument, resolver)
	if err != nil {
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
	matches, err := entry.updateText(ctx, s, params.ContentChanges[0].Text)
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
		if strings.HasPrefix(loc.URI.Filename(), s.container.CacheDirPath()) {
			// This is a temporary file, make sure it exists.
			localPath := loc.URI.Filename()
			if _, err := os.Stat(localPath); err != nil {
				tmpEntry, ok := s.fileCache[loc.URI.Filename()]
				if !ok {
					return nil, fmt.Errorf("unknown file: %s", loc.URI)
				}

				// Create the file.
				if err := os.MkdirAll(path.Dir(localPath), 0755); err != nil {
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
	bucket, err := resolver.Bucket()
	if err != nil {
		return nil, nil, err
	}
	file, err := bucket.GetFile(ctx, path)
	if err != nil {
		return nil, nil, err
	}
	return bucket, file, nil
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
	bucket, file, err := s.findImport(ctx, resolver, path)
	if err != nil {
		return nil, err
	}
	localPath, err := s.localPathForImport(ctx, bucket, file)
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
	return s.createFileEntry(ctx, protocol.TextDocumentItem{
		URI:  uri,
		Text: string(fileData),
	}, newModuleSetResolver(func() (bufmodule.ModuleSet, error) {
		return file.Module().ModuleSet(), nil
	}))
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
	entry := newFileEntry(&item, resolver, item.URI.Filename(), strings.HasPrefix(item.URI.Filename(), s.container.CacheDirPath()))
	s.fileCache[item.URI.Filename()] = entry
	if err := entry.processText(ctx, s); err != nil {
		return nil, err
	}
	if !entry.isRemote {
		if err := s.fileWatcher.Add(item.URI.Filename()); err != nil {
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
	bucket, err := resolver.Bucket()
	if err != nil {
		return err
	}
	err = bucket.WalkFileInfos(ctx, func(fileInfo bufmodule.FileInfo) error {
		if entry, ok := s.fileCache[fileInfo.ExternalPath()]; ok {
			entry.bufDiags = nil
		}
		return nil
	})
	if err != nil {
		return err
	}
	diagsByFile := make(map[string][]protocol.Diagnostic)
	image, err := bufimage.BuildImage(ctx, tracing.NewTracer(s.container.Tracer()), bucket)
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

// localPathForImport determines the local path on-disk that corresponds to the import path.
// Note that this is merely a hueristic and not applicable in all scenarios. It is only intended
// to be used in the LSP code.
func (s *server) localPathForImport(
	ctx context.Context,
	bucket bufmodule.ModuleReadBucket,
	file bufmodule.File,
) (string, error) {
	module := file.Module()
	if module.ModuleSet() == s.wellKnownTypesModuleSet {
		digest, err := file.Module().Digest(bufmodule.DigestTypeB5)
		if err != nil {
			return "", err
		}
		return normalpath.Join(s.container.CacheDirPath(), lspWellKnownTypesCacheRelDirPath, digest.String(), file.Path()), nil
	}
	if module.IsLocal() {
		return file.ExternalPath(), nil
	}
	moduleFullName := module.ModuleFullName()
	if moduleFullName == nil {
		return "", syserror.Newf("remote module %q had nil ModuleFullName", module.OpaqueID())
	}
	return normalpath.Unnormalize(
		normalpath.Join(
			s.container.CacheDirPath(),
			v3CacheModuleRelDirPath,
			moduleFullName.Registry(),
			moduleFullName.Owner(),
			moduleFullName.Name(),
			module.CommitID(),
			v3CacheExternalModuleDataFilesDir,
			file.Path(),
		),
	), nil
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

func completionsToCompletionList(
	options map[string]protocol.CompletionItem,
) *protocol.CompletionList {
	result := &protocol.CompletionList{}
	for _, item := range options {
		result.Items = append(result.Items, item)
	}
	return result
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
