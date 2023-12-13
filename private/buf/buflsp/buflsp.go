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
	"fmt"
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
	"go.uber.org/zap"
)

var (
	// wellKnownTypesDescriptorProtoPath is the path of the descriptor.proto well-known types file.
	wellKnownTypesDescriptorProtoPath = normalpath.Join("google", "protobuf", "descriptor.proto")
	// lspWellKnownTypesCacheRelDirPath is the relative path to the cache directory for materialized
	// well-known types .proto files.
	lspWellKnownTypesCacheRelDirPath = normalpath.Join("lsp", "wkt")
	// v3CacheModuleRelDirPath is the relative path to the cache directory in its newest iteration.
	// NOTE: This needs to be kept in sync with module_data_provider.
	v3CacheModuleRelDirPath = normalpath.Join("v3", "module")
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
	wellKnownTypesBucket    bufmodule.ModuleReadBucket

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
	wellKnownTypesModuleSet, err := wellKnownTypesModuleSet(ctx, logger)
	if err != nil {
		return nil, err
	}
	wellKnownTypesBucket := bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(
		wellKnownTypesModuleSet,
	)
	buflsp := &server{
		jsonrpc2Conn:            jsonrpc2Conn,
		logger:                  logger,
		container:               container,
		controller:              controller,
		lintHandler:             buflint.NewHandler(logger, tracer),
		breakingHandler:         bufbreaking.NewHandler(logger, tracer),
		fileCache:               make(map[string]*fileEntry),
		fileWatcher:             watcher,
		wellKnownTypesModuleSet: wellKnownTypesModuleSet,
		wellKnownTypesBucket:    wellKnownTypesBucket,
	}
	go func() {
		for event := range buflsp.fileWatcher.Events {
			if event.Op&fsnotify.Write == fsnotify.Write {
				buflsp.lock.Lock()
				if entry, ok := buflsp.fileCache[event.Name]; ok {
					if entry.moduleSet != nil {
						if err := buflsp.refreshImage(context.Background(), entry.moduleSet, entry.bucket); err != nil {
							buflsp.logger.Sugar().Errorf("refreshImage error: %s", err)
						}
					}
				}
				buflsp.lock.Unlock()
			}
		}
	}()
	return buflsp, nil
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
		s.wellKnownTypesModuleSet,
		s.wellKnownTypesBucket,
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

	// Check if it is already open.
	if entry, ok := s.fileCache[params.TextDocument.URI.Filename()]; ok {
		entry.refCount++
		if _, err := entry.updateText(ctx, s, params.TextDocument.Text); err != nil {
			return err
		}
		return nil
	}
	var moduleSet bufmodule.ModuleSet
	wellKnownTypesCachePath := normalpath.Join(
		s.container.CacheDirPath(),
		lspWellKnownTypesCacheRelDirPath,
	)
	if strings.HasPrefix(normalpath.Normalize(params.TextDocument.URI.Filename()), wellKnownTypesCachePath) {
		moduleSet = s.wellKnownTypesModuleSet
	} else {
		var err error
		workspace, err := s.controller.GetWorkspace(ctx, params.TextDocument.URI.Filename())
		if err != nil {
			s.logger.Warn("could not determine workspace", zap.Error(err))
			// Continue anyways if this fails.
		}
		if workspace != nil {
			moduleSet = workspace
		}
	}

	// Create a new file entry for the file
	entry, err := s.createFileEntry(ctx, params.TextDocument, moduleSet)
	if err != nil {
		return err
	}
	if entry.moduleSet != nil {
		if err := s.refreshImage(ctx, entry.moduleSet, entry.bucket); err != nil {
			return err
		}
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
		if entry.moduleSet != nil {
			if err := s.refreshImage(ctx, entry.moduleSet, entry.bucket); err != nil {
				return err
			}
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

func (s *server) resolveImport(ctx context.Context, workspace bufmodule.ModuleSet, bucket bufmodule.ModuleReadBucket, path string) (*fileEntry, error) {
	file, err := bucket.GetFile(ctx, path)
	if err != nil {
		workspace = s.wellKnownTypesModuleSet
		bucket = s.wellKnownTypesBucket
		file, err = bucket.GetFile(ctx, path)
		if err != nil {
			s.logger.Warn("could not resolve import", zap.String("path", path))
			return nil, nil
		}
	}
	localPath, err := s.localPathForImport(ctx, workspace, bucket, file)
	if err != nil {
		return nil, err
	}
	uri := makeFileURI(localPath)
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
	}, workspace)
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
func (s *server) createFileEntry(ctx context.Context, item protocol.TextDocumentItem, moduleSet bufmodule.ModuleSet) (*fileEntry, error) {
	entry := newFileEntry(&item, moduleSet, item.URI.Filename(), strings.HasPrefix(item.URI.Filename(), s.container.CacheDirPath()))
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

func (s *server) refreshImage(ctx context.Context, moduleSet bufmodule.ModuleSet, bucket bufmodule.ModuleReadBucket) error {
	err := bucket.WalkFileInfos(ctx, func(fileInfo bufmodule.FileInfo) error {
		if entry, ok := s.fileCache[fileInfo.ExternalPath()]; ok {
			entry.bufDiags = nil
		}
		return nil
	})
	if err != nil {
		return err
	}
	diagsByFile := make(map[string][]protocol.Diagnostic)
	image, buildAnnots, err := bufimage.BuildImage(ctx, tracing.NewTracer(s.container.Tracer()), bucket)
	if err != nil {
		return err
	}
	for _, annot := range buildAnnots {
		diagsByFile[annot.FileInfo().ExternalPath()] = append(diagsByFile[annot.FileInfo().ExternalPath()],
			annotationToDiagnostic(annot, protocol.DiagnosticSeverityError))
	}

	if workspace, ok := moduleSet.(bufworkspace.Workspace); ok && image != nil {
		for _, module := range moduleSet.Modules() {
			lintAnnots, err := s.lintHandler.Check(ctx, workspace.GetLintConfigForOpaqueID(module.OpaqueID()), image)
			if err != nil {
				return err
			}
			for _, annot := range lintAnnots {
				diagsByFile[annot.FileInfo().ExternalPath()] = append(diagsByFile[annot.FileInfo().ExternalPath()],
					annotationToDiagnostic(annot, protocol.DiagnosticSeverityWarning))
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
	workspace bufmodule.ModuleSet,
	bucket bufmodule.ModuleReadBucket,
	file bufmodule.File,
) (string, error) {
	if workspace == s.wellKnownTypesModuleSet {
		return normalpath.Join(s.container.CacheDirPath(), lspWellKnownTypesCacheRelDirPath, file.Path()), nil
	}
	module := file.Module()
	if module.IsLocal() {
		return file.ExternalPath(), nil
	}
	moduleFullName := module.ModuleFullName()
	if moduleFullName == nil {
		return "", syserror.Newf("remote module %q had nil ModuleFullName", module.OpaqueID())
	}
	digest, err := module.Digest()
	if err != nil {
		return "", err
	}
	return normalpath.Unnormalize(
		normalpath.Join(
			s.container.CacheDirPath(),
			v3CacheModuleRelDirPath,
			moduleFullName.Registry(),
			moduleFullName.Owner(),
			moduleFullName.Name(),
			digest.String(),
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
				s.logger.Sugar().Errorf("fileWatcher.Remove error: %s", err)
			}
		}
		delete(s.fileCache, entry.document.URI.Filename())
	}
}

func makeFileURI(path string) protocol.DocumentURI {
	return protocol.DocumentURI("file://" + path)
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

func wellKnownTypesModuleSet(
	ctx context.Context,
	logger *zap.Logger,
) (bufmodule.ModuleSet, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, bufmodule.NopModuleDataProvider)
	moduleSetBuilder.AddLocalModule(
		datawkt.ReadBucket,
		".",
		true,
	)
	return moduleSetBuilder.Build()
}
