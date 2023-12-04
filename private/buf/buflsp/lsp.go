// Copyright 2023 Buf Technologies, Inc.
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
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufformat"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/fsnotify/fsnotify"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

const (
	// The cache directory for materialized wll known .proto files.
	wktCacheDir = "/v2/wkt/"
	// The directory that contains the well known .proto files.
	wktSourceDir = "google/protobuf/"

	// The cache directory for materialized dependency .proto files.
	depCacheDir = "/v2/files/"
)

type BufLsp struct {
	noopServer

	jconn     jsonrpc2.Conn
	logger    *zap.Logger
	container appflag.Container

	moduleReader       bufmodule.ModuleReader
	moduleConfigReader bufwire.ModuleConfigReader
	imageBuilder       bufimagebuild.Builder
	lintHandler        buflint.Handler
	breakingHandler    bufbreaking.Handler

	mutex       sync.Mutex
	fileCache   map[string]*fileEntry
	fileWatcher *fsnotify.Watcher

	folders   []protocol.WorkspaceFolder
	clientCap protocol.ClientCapabilities
}

func NewBufLsp(
	ctx context.Context,
	jconn jsonrpc2.Conn,
	logger *zap.Logger,
	container appflag.Container,
) (*BufLsp, error) {
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	moduleReader, err := bufcli.NewModuleReaderAndCreateCacheDirs(
		container,
		clientConfig,
	)
	if err != nil {
		return nil, err
	}

	runner := command.NewRunner()
	storageosProvider := bufcli.NewStorageosProvider(true)
	moduleConfigReader, err := bufcli.NewWireModuleConfigReaderForModuleReader(
		container,
		storageosProvider,
		runner,
		clientConfig,
		moduleReader,
	)
	if err != nil {
		return nil, err
	}

	// Create a file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	buflsp := &BufLsp{
		jconn:     jconn,
		logger:    logger,
		container: container,

		moduleReader:       moduleReader,
		moduleConfigReader: moduleConfigReader,
		imageBuilder:       bufimagebuild.NewBuilder(logger, moduleReader),
		lintHandler:        buflint.NewHandler(logger),
		breakingHandler:    bufbreaking.NewHandler(logger),

		fileCache:   make(map[string]*fileEntry),
		fileWatcher: watcher,
	}

	go func() {
		for event := range buflsp.fileWatcher.Events {
			if event.Op&fsnotify.Write == fsnotify.Write {
				buflsp.mutex.Lock()
				if entry, ok := buflsp.fileCache[event.Name]; ok {
					if entry.module != nil {
						if err := buflsp.refreshImage(context.Background(), entry.module); err != nil {
							buflsp.logger.Sugar().Errorf("refreshImage error: %s", err)
						}
					}
				}
				buflsp.mutex.Unlock()
			}
		}
	}()

	return buflsp, nil
}

func (b *BufLsp) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	// Store client info
	b.folders = params.WorkspaceFolders
	b.clientCap = params.Capabilities

	// Always load the descriptor.proto file
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if _, err := b.loadWktFile(ctx, wktSourceDir+"descriptor.proto"); err != nil {
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

func (b *BufLsp) Shutdown(ctx context.Context) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.fileWatcher.Close()
}

func (b *BufLsp) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Check if it is already open.
	if entry, ok := b.fileCache[params.TextDocument.URI.Filename()]; ok {
		entry.refCount++
		if _, err := entry.updateText(ctx, b, params.TextDocument.Text); err != nil {
			return err
		}
		return nil
	}

	// Check if it is a temporary file from the cache.
	if strings.HasPrefix(params.TextDocument.URI.Filename(), path.Join(b.container.CacheDirPath())) {
		path := strings.TrimPrefix(params.TextDocument.URI.Filename(), b.container.CacheDirPath())
		return b.restoreCacheFile(ctx, path, params.TextDocument)
	}

	// Create a new file entry for the file
	entry, err := b.createFileEntry(ctx, params.TextDocument, "", nil)
	if err != nil {
		return err
	}
	if entry.module != nil {
		if err := b.refreshImage(ctx, entry.module); err != nil {
			return err
		}
	}
	return nil
}

func (b *BufLsp) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	entry, ok := b.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	matches, err := entry.updateText(ctx, b, params.ContentChanges[0].Text)
	if err != nil {
		return err
	}
	if matches { // Same as on disk, so safe to refresh the image data.
		if err := b.refreshImage(ctx, entry.module); err != nil {
			return err
		}
	}
	return nil
}

func (b *BufLsp) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if entry, ok := b.fileCache[params.TextDocument.URI.Filename()]; ok {
		b.derefFileEntry(entry)
	}
	return nil
}

func (b *BufLsp) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	entry, ok := b.fileCache[params.TextDocument.URI.Filename()]
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

func (b *BufLsp) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{} /* []SymbolInformation | []DocumentSymbol */, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	entry, ok := b.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	result := make([]interface{}, len(entry.docSymbols))
	for i, symbol := range entry.docSymbols {
		result[i] = symbol
	}
	return result, nil
}

func (b *BufLsp) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	entry, ok := b.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := entry.getSourcePos(params.Position)

	// Look for import completions.
	{
		result, found, err := b.findImportCompletionsAt(ctx, entry, pos)
		if err != nil {
			return nil, err
		} else if found {
			return completionsToCompletionList(result), nil
		}
	}

	// Look for ast-based symbol completions.
	sybmolScope := entry.findSymbolScope(pos)
	if sybmolScope != nil {
		if ref := b.findReferenceAt(ctx, entry, sybmolScope, pos); ref != nil {
			options := make(completionOptions)
			b.findRefCompletions(ref, options)
			return completionsToCompletionList(options), nil
		}
	}

	// Fallback on prefix-based completions.
	options := b.findPrefixCompletions(ctx, entry, entry.findScope(pos), entry.codeAt(pos))
	return completionsToCompletionList(options), nil
}

func (b *BufLsp) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location /* Definition | DefinitionLink[] | null */, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	entry, ok := b.fileCache[params.TextDocument.URI.Filename()]
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
		result = b.findReferencedDefLoc(ctx, entry, pos)
	}

	// Make sure all the files exist.
	for _, loc := range result {
		if strings.HasPrefix(loc.URI.Filename(), b.container.CacheDirPath()) {
			// This is a temporary file, make sure it exists.
			localPath := loc.URI.Filename()
			if _, err := os.Stat(localPath); err != nil {
				tmpEntry, ok := b.fileCache[loc.URI.Filename()]
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

func (b *BufLsp) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	entry, ok := b.fileCache[params.TextDocument.URI.Filename()]
	if !ok {
		return nil, fmt.Errorf("unknown file: %s", params.TextDocument.URI)
	}
	pos := entry.getSourcePos(params.Position)

	symbols := b.findReferencedSymbols(ctx, entry, pos)
	if len(symbols) == 0 {
		return nil, nil
	}
	symbol := symbols[0]
	refEntry, ok := b.fileCache[symbol.file.Filename()]
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

func completionsToCompletionList(options map[string]protocol.CompletionItem) *protocol.CompletionList {
	result := &protocol.CompletionList{}
	for _, item := range options {
		result.Items = append(result.Items, item)
	}
	return result
}

func (b *BufLsp) resolveImport(ctx context.Context, entry *fileEntry, path string) (*fileEntry, error) {
	// Check in the module
	if file, err := entry.module.GetModuleFile(ctx, path); err == nil {
		return b.ensureLoaded(ctx, file, entry.modulePin)
	}

	// Check well known types
	if strings.HasPrefix(path, wktSourceDir) {
		return b.loadWktFile(ctx, path)
	}

	// Check in the dependencies
	for _, dep := range entry.module.DependencyModulePins() {
		mod, err := b.moduleReader.GetModule(ctx, dep)
		if err != nil {
			return nil, err
		}

		if file, err := mod.GetModuleFile(ctx, path); err == nil {
			return b.ensureLoaded(ctx, file, dep)
		}
	}
	return nil, nil
}

// Refresh the results for the given file entry.
func (b *BufLsp) updateDiagnostics(ctx context.Context, entry *fileEntry) error {
	if err := b.updateDiags(ctx, entry); err != nil {
		return err
	}
	return nil
}

// Create a new file entry with the given contents and metadata.
func (b *BufLsp) createFileEntry(ctx context.Context, item protocol.TextDocumentItem, externalPath string, pin bufmoduleref.ModulePin) (*fileEntry, error) {
	if externalPath == "" {
		externalPath = item.URI.Filename()
	}

	var module bufmodule.Module
	var path string
	if pin != nil {
		var err error
		path = externalPath
		module, err = b.moduleReader.GetModule(ctx, pin)
		if err != nil {
			return nil, err
		}
	} else {
		moduleConfig, filePath, err := b.getModuleConfig(ctx, externalPath)
		if err == nil {
			path = filePath
			module = moduleConfig.Module()
		} else {
			path = externalPath
			module = nil
		}
	}

	entry := newFileEntry(&item, module, pin, externalPath, path, strings.HasPrefix(item.URI.Filename(), b.container.CacheDirPath()))
	b.fileCache[item.URI.Filename()] = entry
	if err := entry.processText(ctx, b); err != nil {
		return nil, err
	}
	if !entry.isRemote {
		if err := b.fileWatcher.Add(item.URI.Filename()); err != nil {
			return nil, err
		}
	}
	return entry, nil
}

// Ensure the given module file is loaded into the cache.
func (b *BufLsp) ensureLoaded(ctx context.Context, modFile bufmodule.ModuleFile, pin bufmoduleref.ModulePin) (*fileEntry, error) {
	uri := makeFileURI(modFile.ExternalPath())
	if stat, err := os.Stat(modFile.ExternalPath()); err != nil || stat.IsDir() {
		// Not a local file, create a temporary file
		if pin == nil {
			return nil, fmt.Errorf("no pin for %s", modFile.Path())
		}
		digest, err := bufcas.ParseDigest(pin.Digest())
		if err != nil {
			return nil, err
		}
		digestHex := hex.EncodeToString(digest.Value())
		tmpPath := path.Join(b.container.CacheDirPath(),
			"v2", "files", modFile.ModuleIdentity().IdentityString(), pin.Commit(), digest.Type().String(), digestHex, modFile.Path())
		uri = makeFileURI(tmpPath)
	}

	if entry, ok := b.fileCache[uri.Filename()]; ok {
		entry.refCount++
		return entry, nil
	}

	// Read the file data
	var fileData []byte
	buffer := make([]byte, 64*1024)
	for {
		size, err := modFile.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		fileData = append(fileData, buffer[:size]...)
	}

	return b.createFileEntry(ctx, protocol.TextDocumentItem{
		URI:  uri,
		Text: string(fileData),
	}, modFile.ExternalPath(), pin)
}

func annotToDiag(annot bufanalysis.FileAnnotation, severity protocol.DiagnosticSeverity) protocol.Diagnostic {
	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(annot.StartLine() - 1),
				Character: uint32(annot.StartColumn() - 1),
			},
			End: protocol.Position{
				Line:      uint32(annot.EndLine() - 1),
				Character: uint32(annot.EndColumn() - 1),
			},
		},
		Severity: severity,
		Message:  fmt.Sprintf("%s (%s)", annot.Message(), annot.Type()),
	}
}

func (b *BufLsp) refreshImage(ctx context.Context, module bufmodule.Module) error {
	fileInfos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		if entry, ok := b.fileCache[fileInfo.ExternalPath()]; ok {
			entry.bufDiags = nil
		}
	}
	diagsByFile := make(map[string][]protocol.Diagnostic)
	image, buildAnnots, err := b.imageBuilder.Build(ctx, module)
	if err != nil {
		return err
	}
	for _, annot := range buildAnnots {
		diagsByFile[annot.FileInfo().ExternalPath()] = append(diagsByFile[annot.FileInfo().ExternalPath()],
			annotToDiag(annot, protocol.DiagnosticSeverityError))
	}

	if image != nil {
		lintAnnots, err := b.lintHandler.Check(ctx, module.LintConfig(), image)
		if err != nil {
			return err
		}
		for _, annot := range lintAnnots {
			diagsByFile[annot.FileInfo().ExternalPath()] = append(diagsByFile[annot.FileInfo().ExternalPath()],
				annotToDiag(annot, protocol.DiagnosticSeverityWarning))
		}
	}
	for externalPath, diags := range diagsByFile {
		if entry, ok := b.fileCache[externalPath]; ok {
			entry.bufDiags = diags
			if err := b.updateDiags(ctx, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

// Update the diagnostics for the given file entry.
func (b *BufLsp) updateDiags(ctx context.Context, entry *fileEntry) error {
	if b.jconn == nil {
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
	return b.jconn.Notify(ctx, "textDocument/publishDiagnostics", diagParams)
}

// Restore the info for a temporary file from the cache.
//
// This happens if vscode opens a cached file directly, instead of through a call to Definition.
func (b *BufLsp) restoreCacheFile(ctx context.Context, path string, item protocol.TextDocumentItem) error {
	if strings.HasPrefix(path, depCacheDir) {
		path = strings.TrimPrefix(path, depCacheDir)
		// This is a temporary file, we need to recover the pin from the path
		parts := strings.Split(path, "/")
		if len(parts) < 6 {
			return fmt.Errorf("invalid temporary file path: %s", item.URI.Filename())
		}
		// Path is of the format:
		// <remote>/<owner>/<repository>/<commit>/<digest type>/<digest>/<path>
		pin, err := bufmoduleref.NewModulePin(parts[0], parts[1], parts[2], parts[3], parts[4]+":"+parts[5])
		if err != nil {
			return err
		}
		externalPath := strings.Join(parts[6:], "/")
		if _, err := b.createFileEntry(ctx, item, externalPath, pin); err != nil {
			return err
		}
	} else if strings.HasPrefix(path, wktCacheDir) {
		path = strings.TrimPrefix(path, wktCacheDir)
		// This is a wellknown type file
		if _, err := b.createFileEntry(ctx, item, path, nil); err != nil {
			return err
		}
	}
	return nil
}

func (b *BufLsp) loadWktFile(ctx context.Context, fileName string) (*fileEntry, error) {
	wktPath := path.Join(b.container.CacheDirPath(), "v2", "wkt", fileName)
	if wktEntry, ok := b.fileCache[wktPath]; ok {
		wktEntry.refCount++
		return wktEntry, nil
	}
	cachedName := "wkt/" + strings.TrimPrefix(fileName, wktSourceDir)
	wktData, err := wktFiles.ReadFile(cachedName)
	if err != nil {
		return nil, err
	}
	return b.createFileEntry(ctx, protocol.TextDocumentItem{
		URI:  makeFileURI(wktPath),
		Text: string(wktData),
	}, fileName, nil)
}

func (b *BufLsp) derefImports(entry *fileEntry) {
	for _, importEntry := range entry.imports {
		if importEntry.docURI != "" {
			if importFile, ok := b.fileCache[importEntry.docURI.Filename()]; ok {
				b.derefFileEntry(importFile)
			}
		}
	}
}

func (b *BufLsp) derefFileEntry(entry *fileEntry) {
	entry.refCount--
	if entry.refCount == 0 {
		b.derefImports(entry)
		if !entry.isRemote {
			if err := b.fileWatcher.Remove(entry.document.URI.Filename()); err != nil {
				b.logger.Sugar().Errorf("fileWatcher.Remove error: %s", err)
			}
		}
		delete(b.fileCache, entry.document.URI.Filename())
	}
}

// getModuleConfig returns the ModuleConfig (and include path) that defines the ModuleFile identified by
// the given external path.
func (b *BufLsp) getModuleConfig(
	ctx context.Context,
	externalPath string,
) (bufwire.ModuleConfig, string, error) {
	refParser := buffetch.NewRefParser(b.logger)
	sourceRef, err := refParser.GetSourceRef(ctx, externalPath)
	if err != nil {
		return nil, "", err
	}
	moduleConfigs, err := b.moduleConfigReader.GetModuleConfigSet(
		ctx,
		b.container,
		sourceRef,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		return nil, "", err
	}
	for _, moduleConfig := range moduleConfigs.ModuleConfigs() {
		fileInfos, err := moduleConfig.Module().TargetFileInfos(ctx)
		if err != nil {
			return nil, "", err
		}
		for _, fileInfo := range fileInfos {
			if fileInfo.ExternalPath() == externalPath {
				return moduleConfig, fileInfo.Path(), nil
			}
		}
	}
	return nil, "", fmt.Errorf("could not find %s in any module", externalPath)
}

func makeFileURI(path string) protocol.DocumentURI {
	return protocol.DocumentURI("file://" + path)
}
