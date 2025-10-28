// Copyright 2020-2025 Buf Technologies, Inc.
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
	"iter"
	"log/slog"
	"slices"
	"strings"
	"time"

	"buf.build/go/standard/xlog/xslog"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/protocompile/experimental/ast/predeclared"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/bufbuild/protocompile/experimental/incremental/queries"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"go.lsp.dev/protocol"
)

// file is a file that has been opened by the client.
//
// Mutating a file is thread-safe.
type file struct {
	lsp *lsp
	uri protocol.URI

	file *report.File
	// Version is an opaque version identifier given to us by the LSP client. This
	// is used in the protocol to disambiguate which version of a file e.g. publishing
	// diagnostics or symbols an operating refers to.
	version int32
	hasText bool // Whether this file has ever had text read into it.

	workspace  *workspace         // May be nil.
	objectInfo storage.ObjectInfo // Info in the context of the workspace.

	ir                   ir.File
	referenceableSymbols map[string]*symbol
	referenceSymbols     []*symbol
	symbols              []*symbol
	diagnostics          []protocol.Diagnostic
	cancelChecks         func()
}

// IsLocal returns whether this is a local file, i.e. a file that the editor
// is editing and not something from e.g. the BSR.
func (f *file) IsLocal() bool {
	if f.objectInfo == nil {
		return false
	}

	return f.objectInfo.LocalPath() == f.objectInfo.ExternalPath()
}

// IsWKT returns whether this file corresponds to a well-known type.
func (f *file) IsWKT() bool {
	_, ok := f.objectInfo.(wktObjectInfo)
	return ok
}

// Manager returns the file manager that owns this file.
func (f *file) Manager() *fileManager {
	return f.lsp.fileManager
}

// Reset clears all bookkeeping information on this file.
func (f *file) Reset(ctx context.Context) {
	f.lsp.logger.DebugContext(ctx, "resetting file", slog.String("uri", f.uri.Filename()))

	f.ir = ir.File{}
	f.diagnostics = nil
	f.symbols = nil
}

// Close marks a file as closed.
//
// This will not necessarily evict the file, since there may be more than one user
// for this file.
func (f *file) Close(ctx context.Context) {
	f.Manager().Close(ctx, f.uri)
	if f.workspace != nil {
		f.workspace.Release()
		f.workspace = nil
	}
}

// IsOpenInEditor returns whether this file was opened in the LSP client's
// editor.
//
// Some files may be opened as dependencies, so we want to avoid doing extra
// work like sending progress notifications.
func (f *file) IsOpenInEditor() bool {
	return f.version != -1 // See [file.ReadFromWorkspace].
}

// ReadFromWorkspace reads this file from the workspace if it has never had data loaded into it
// before.
//
// If it has been read from disk before, or has received updates from the LSP client, this function
// returns nil.
func (f *file) ReadFromWorkspace(ctx context.Context) (err error) {
	if f.hasText {
		return nil
	}

	fileName := f.uri.Filename()
	var reader io.ReadCloser
	switch info := f.objectInfo.(type) {
	case bufmodule.FileInfo:
		reader, err = info.Module().GetFile(ctx, info.Path())
	case wktObjectInfo:
		reader, err = f.lsp.wktBucket.Get(ctx, info.Path())
	default:
		return fmt.Errorf("unsupported objectInfo type %T", f.objectInfo)
	}
	if err != nil {
		return fmt.Errorf("read file %q from workspace", err)
	}
	defer reader.Close()

	var builder strings.Builder
	if _, err := io.Copy(&builder, reader); err != nil {
		return fmt.Errorf("could not read file %q from workspace", err)
	}
	text := builder.String()

	f.version = -1
	f.file = report.NewFile(fileName, text)
	f.hasText = true
	return nil
}

// Update updates the contents of this file with the given text received from
// the LSP client.
func (f *file) Update(ctx context.Context, version int32, text string) {
	f.Reset(ctx)
	f.CancelChecks(ctx)

	f.lsp.logger.InfoContext(ctx, "file updated", slog.String("uri", f.uri.Filename()), slog.Int("old_version", int(f.version)), slog.Int("new_version", int(version)))
	f.version = version
	f.file = report.NewFile(f.uri.Filename(), text)
	f.hasText = true
}

// Refresh rebuilds all of a file's internal book-keeping.
func (f *file) Refresh(ctx context.Context) {
	var progress *progress
	if f.IsOpenInEditor() {
		// NOTE: Nil progress does nothing when methods are called. This helps
		// minimize RPC spam from the client when indexing lots of files.
		progress = newProgress(f.lsp)
	}
	progress.Begin(ctx, "Indexing")

	progress.Report(ctx, "Setting workspace", 1.0/4)
	f.RefreshWorkspace(ctx)

	progress.Report(ctx, "Parsing IR", 2.0/4)
	f.RefreshIR(ctx)

	progress.Report(ctx, "Indexing Symbols", 3.0/4)
	f.IndexSymbols(ctx)

	progress.Report(ctx, "Running Checks", 4.0/4)
	f.RunChecks(ctx)

	progress.Done(ctx)

	// NOTE: Diagnostics are published unconditionally. This is necessary even
	// if we have zero diagnostics, so that the client correctly ticks over from
	// n > 0 diagnostics to 0 diagnostics.
	f.PublishDiagnostics(ctx)
}

// RefreshWorkspace builds the workspace for the current file and sets the workspace it.
//
// The Buf workspace provides the sources for the compiler to work with.
func (f *file) RefreshWorkspace(ctx context.Context) {
	f.lsp.logger.Debug(
		"refresh workspace",
		slog.String("file", f.uri.Filename()),
		slog.Int("version", int(f.version)),
	)
	if f.workspace != nil {
		if err := f.workspace.Refresh(ctx); err != nil {
			f.lsp.logger.Error(
				"could not refresh workspace",
				slog.String("uri", string(f.uri)),
				xslog.ErrorAttr(err),
			)
		}
	} else {
		workspace, err := f.lsp.workspaceManager.LeaseWorkspace(ctx, f.uri)
		if err != nil {
			f.lsp.logger.Error(
				"could not lease workspace",
				slog.String("uri", string(f.uri)),
				xslog.ErrorAttr(err),
			)
			return
		}
		f.workspace = workspace
	}
}

// RefreshIR queries for the IR of the file and the IR of each import file.
// Diagnostics from the compiler are returned when applicable.
//
// This operation requires IndexImports.
func (f *file) RefreshIR(ctx context.Context) {
	f.lsp.logger.Info(
		"parsing IR for file",
		slog.String("uri", string(f.uri)),
		slog.Int("version", int(f.version)),
	)

	// Opener creates a cached view of all files in the workspace.
	pathToFiles := f.workspace.PathToFile()
	files := make([]*file, 0, len(pathToFiles))
	openerMap := make(map[string]string, len(pathToFiles))
	for path, file := range pathToFiles {
		openerMap[path] = file.file.Text()
		files = append(files, file)
	}
	opener := source.NewMap(openerMap)

	session := new(ir.Session)
	queries := xslices.Map(files, func(file *file) incremental.Query[ir.File] {
		return queries.IR{
			Opener:  opener,
			Path:    file.objectInfo.Path(),
			Session: session,
		}
	})
	results, diagnosticReport, err := incremental.Run(
		ctx,
		f.lsp.queryExecutor,
		queries...,
	)
	if err != nil {
		f.lsp.logger.Error(
			"failed to parse IR for file",
			slog.String("uri", string(f.uri)),
			slog.Int("version", int(f.version)),
			xslog.ErrorAttr(err),
		)
		return
	}
	for i, file := range files {
		file.ir = results[i].Value
		if f != file {
			// Update symbols for imports.
			file.IndexSymbols(ctx)
		}
	}
	// Only hold on to diagnostics where the primary span is for this path.
	fileDiagnostics := xslices.Filter(diagnosticReport.Diagnostics, func(d report.Diagnostic) bool {
		return d.Primary().Path() == f.objectInfo.Path()
	})
	diagnostics, err := xslices.MapError(
		fileDiagnostics,
		reportDiagnosticToProtocolDiagnostic,
	)
	if err != nil {
		f.lsp.logger.Error(
			"failed to parse report diagnostics",
			xslog.ErrorAttr(err),
		)
	}
	f.diagnostics = diagnostics
	f.lsp.logger.DebugContext(
		ctx, "ir diagnostic(s)",
		slog.String("uri", f.uri.Filename()),
		slog.Int("count", len(f.diagnostics)),
		slog.Any("diagnostics", f.diagnostics),
	)
}

// IndexSymbols processes the IR of a file and generates symbols for each symbol in
// the document.
//
// This operation requires RefreshIR.
func (f *file) IndexSymbols(ctx context.Context) {
	defer xslog.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()
	// We cannot index symbols without the IR, so we keep the symbols as-is.
	if f.ir.IsZero() {
		return
	}

	// Throw away all the old symbols and rebuild symbols unconditionally. This is because if
	// this file depends on a file that has since been modified, we may need to update references.
	f.symbols = nil
	f.referenceSymbols = nil
	f.referenceableSymbols = make(map[string]*symbol)

	// Process all imports as symbols
	f.symbols = xslices.Map(seq.ToSlice(f.ir.Imports()), f.importToSymbol)

	resolved, unresolved := f.indexSymbols()
	f.symbols = append(f.symbols, resolved...)
	f.symbols = append(f.symbols, unresolved...)
	f.referenceSymbols = append(f.referenceSymbols, unresolved...)

	// Index all referenceable symbols
	for _, sym := range resolved {
		def, ok := sym.kind.(*referenceable)
		if !ok {
			continue
		}
		f.referenceableSymbols[def.ast.Name().Canonicalized()] = sym
	}

	// TODO: this could use a refactor, probably.

	// Resolve all unresolved symbols from this file
	for _, sym := range unresolved {
		ref, ok := sym.kind.(*reference)
		if !ok {
			// This shouldn't happen, logging a warning
			f.lsp.logger.Warn(
				"found unresolved non-reference symbol",
				slog.String("file", f.uri.Filename()),
				slog.Any("symbol", sym),
			)
			continue
		}
		file, ok := f.workspace.PathToFile()[ref.def.Span().Path()]
		if !ok {
			// Check current file
			if ref.def.Span().Path() != f.objectInfo.Path() {
				// This can happen if this references a predeclared type or if the file we are
				// checking has not indexed imports.
				continue
			}
			file = f
		}
		def, ok := file.referenceableSymbols[ref.def.Name().Canonicalized()]
		if !ok {
			// This could happen in the case where we are in the cache for example, and we do not
			// have access to a buildable workspace.
			continue
		}
		sym.def = def
		referenceable, ok := def.kind.(*referenceable)
		if !ok {
			// This shouldn't happen, logging a warning
			f.lsp.logger.Warn(
				"found non-referenceable symbol in index",
				slog.String("file", f.uri.Filename()),
				slog.Any("symbol", def),
			)
			continue
		}
		referenceable.references = append(referenceable.references, sym)
	}

	// Resolve all references outside of this file to symbols in this file
	for _, file := range f.workspace.PathToFile() {
		if f == file {
			continue // ignore self
		}
		for _, sym := range file.referenceSymbols {
			ref, ok := sym.kind.(*reference)
			if !ok {
				// This shouldn't happen, logging a warning
				f.lsp.logger.Warn(
					"found unresolved non-reference symbol",
					slog.String("file", f.uri.Filename()),
					slog.Any("symbol", sym),
				)
				continue
			}
			if ref.def.Span().Path() != f.objectInfo.Path() {
				continue
			}
			def, ok := f.referenceableSymbols[ref.def.Name().Canonicalized()]
			if !ok {
				// This shouldn't happen, if a symbol is pointing at this file, all definitions
				// should be resolved, logging a warning
				f.lsp.logger.Warn(
					"found reference to unknown symbol",
					slog.String("file", f.uri.Filename()),
					slog.Any("reference", sym),
				)
				continue
			}
			referenceable, ok := def.kind.(*referenceable)
			if !ok {
				// This shouldn't happen, logging a warning
				f.lsp.logger.Warn(
					"found non-referenceable symbol in index",
					slog.String("file", f.uri.Filename()),
					slog.Any("symbol", def),
				)
				continue
			}
			referenceable.references = append(referenceable.references, sym)
		}
	}

	// Finally, sort the symbols in position order, with shorter symbols sorting smaller.
	slices.SortFunc(f.symbols, func(s1, s2 *symbol) int {
		diff := s1.span.Start - s2.span.Start
		if diff == 0 {
			return s1.span.End - s2.span.End
		}
		return diff
	})

	f.lsp.logger.DebugContext(ctx, "symbol indexing complete", slog.String("uri", f.uri.Filename()))
}

// indexSymbols takes the IR [ir.File] for each [file] and returns all the file symbols in
// two slices:
//   - The first slice contains definition symbols that are ready to go
//   - The second slice contains reference symbols need to be resolved
//
// For unresolved symbols, we need to track the definition we're attempting to resolve.
func (f *file) indexSymbols() ([]*symbol, []*symbol) {
	var resolved, unresolved []*symbol
	for i := range f.ir.Symbols().Len() {
		// We only index the symbols for this file.
		symbol := f.ir.Symbols().At(i)
		if symbol.File().Path() != f.objectInfo.Path() {
			continue
		}
		resolvedSyms, unresolvedSyms := f.irToSymbols(symbol)
		resolved = append(resolved, resolvedSyms...)
		unresolved = append(unresolved, unresolvedSyms...)
	}
	return resolved, unresolved
}

// irToSymbols takes the [ir.Symbol] and returns the corresponding symbols used by the LSP
// in two slices:
//   - The first slice contains resolved symbols that are ready to go
//   - The second slice contains symbols that resolution for their defs
func (f *file) irToSymbols(irSymbol ir.Symbol) ([]*symbol, []*symbol) {
	var resolved, unresolved []*symbol
	switch irSymbol.Kind() {
	case ir.SymbolKindMessage:
		msg := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsType().AST().AsMessage().Name.Span(),
			kind: &referenceable{
				ast: irSymbol.AsType().AST(),
			},
		}
		msg.def = msg
		resolved = append(resolved, msg)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsType().Options())...)
	case ir.SymbolKindEnum:
		enum := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsType().AST().AsEnum().Name.Span(),
			kind: &referenceable{
				ast: irSymbol.AsType().AST(),
			},
		}
		enum.def = enum
		resolved = append(resolved, enum)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsType().Options())...)
	case ir.SymbolKindEnumValue:
		name := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMember().AST().AsEnumValue().Name.Span(),
			kind: &static{
				ast: irSymbol.AsMember().AST(),
			},
		}
		name.def = name
		resolved = append(resolved, name)

		tag := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMember().AST().AsEnumValue().Tag.Span(),
			kind: &tag{},
		}
		tag.def = tag
		resolved = append(resolved, tag)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsMember().Options())...)
	case ir.SymbolKindField:
		typ := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMember().TypeAST().Span(),
		}
		kind, needsResolution := getKindForMember(irSymbol.AsMember())
		typ.kind = kind
		if needsResolution {
			unresolved = append(unresolved, typ)
		} else {
			resolved = append(resolved, typ)
		}

		field := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMember().AST().AsField().Name.Span(),
			kind: &referenceable{
				ast: irSymbol.AsMember().AST(),
			},
		}
		field.def = field
		resolved = append(resolved, field)

		tag := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMember().AST().AsField().Tag.Span(),
			kind: &tag{},
		}
		tag.def = tag
		resolved = append(resolved, tag)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsMember().Options())...)
	case ir.SymbolKindExtension:
		// TODO: we should figure out if we need to do any resolution here.
		ext := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMember().AST().AsExtend().Extendee.Span(),
			kind: &static{
				ast: irSymbol.AsMember().AST(),
			},
		}
		resolved = append(resolved, ext)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsMember().Options())...)
	case ir.SymbolKindOneof:
		oneof := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsOneof().AST().AsOneof().Name.Span(),
			kind: &referenceable{
				ast: irSymbol.AsOneof().AST(),
			},
		}
		oneof.def = oneof
		resolved = append(resolved, oneof)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsOneof().Options())...)
	case ir.SymbolKindService:
		service := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsService().AST().AsService().Name.Span(),
			kind: &static{
				ast: irSymbol.AsService().AST(),
			},
		}
		service.def = service
		resolved = append(resolved, service)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsService().Options())...)
	case ir.SymbolKindMethod:
		method := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMethod().AST().AsMethod().Name.Span(),
			kind: &static{
				ast: irSymbol.AsMethod().AST(),
			},
		}
		method.def = method
		resolved = append(resolved, method)

		input, _ := irSymbol.AsMethod().Input()
		inputSym := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMethod().AST().AsMethod().Signature.Inputs().Span(),
			kind: &reference{
				def: input.AST(), // Only messages can be method inputs and outputs
			},
		}
		unresolved = append(unresolved, inputSym)

		output, _ := irSymbol.AsMethod().Output()
		outputSym := &symbol{
			ir:   irSymbol,
			file: f,
			span: irSymbol.AsMethod().AST().AsMethod().Signature.Outputs().Span(),
			kind: &reference{
				def: output.AST(), // Only messages can be method inputs and outputs
			},
		}
		unresolved = append(unresolved, outputSym)
		unresolved = append(unresolved, f.messageToSymbols(irSymbol.AsMethod().Options())...)
	}
	return resolved, unresolved
}

// getKindForMember takes a [ir.Member] and returns the symbol kind and whether or not the
// symbol is currently resolved.
func getKindForMember(member ir.Member) (kind, bool) {
	if member.TypeAST().AsPath().AsPredeclared() != predeclared.Unknown {
		return &builtin{
			predeclared: member.TypeAST().AsPath().AsPredeclared(),
		}, false
	}
	return &reference{
		def: member.Element().AST(),
	}, true
}

// importToSymbol takes an [ir.Import] and returns a symbol for it.
func (f *file) importToSymbol(imp ir.Import) *symbol {
	return &symbol{
		file: f,
		span: imp.Decl.Span(),
		kind: &imported{
			file: f.workspace.PathToFile()[imp.File.Path()],
		},
	}
}

func (f *file) messageToSymbols(msg ir.MessageValue) []*symbol {
	var symbols []*symbol
	for field := range msg.Fields() {
		if field.ValueAST().IsZero() {
			continue
		}
		for element := range seq.Values(field.Elements()) {
			span := element.Value().KeyASTs().At(element.ValueNodeIndex()).Span()
			elem := &symbol{
				// NOTE: no [ir.Symbol] for option elements
				file: f,
				span: span,
				kind: &reference{
					def: element.Field().AST(),
				},
				isOption: true,
			}
			symbols = append(symbols, elem)
			if !element.AsMessage().IsZero() {
				symbols = append(symbols, f.messageToSymbols(element.AsMessage())...)
			}
		}
	}
	return symbols
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
	// Check that cursor is before the end of the symbol. Range is half-open [Start, End).
	if comparePositions(symbol.Range().End, cursor) < 0 {
		return nil
	}

	return symbol
}

// CancelChecks cancels any currently running checks for this file.
func (f *file) CancelChecks(ctx context.Context) {
	if f.cancelChecks != nil {
		f.cancelChecks()
		f.cancelChecks = nil
	}
}

// RunChecks triggers the run of checks for this file. Diagnostics are published asynchronously.
func (f *file) RunChecks(ctx context.Context) {
	if f.IsWKT() || !f.IsOpenInEditor() {
		return // Must be open and not a WKT.
	}
	f.CancelChecks(ctx)

	path := f.objectInfo.Path()
	workspace := f.workspace.Workspace()
	module := f.workspace.GetModule(f.uri)
	checkClient := f.workspace.CheckClient()
	if workspace == nil || module == nil || checkClient == nil {
		f.lsp.logger.Debug("checks skipped", slog.String("uri", f.uri.Filename()))
		return
	}

	opener := make(fileOpener)
	for path, file := range f.workspace.PathToFile() {
		opener[path] = file.file.Text()
	}

	const checkTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), checkTimeout)
	f.cancelChecks = cancel

	go func() {
		image, diagnostics := buildImage(ctx, path, f.lsp.logger, opener)
		if image == nil {
			f.lsp.logger.DebugContext(ctx, "checks cancelled on image build", slog.String("uri", f.uri.Filename()))
			return
		}

		f.lsp.logger.DebugContext(ctx, "checks running lint", slog.String("uri", f.uri.Filename()), slog.String("module", module.OpaqueID()))
		var annotations []bufanalysis.FileAnnotation
		if err := checkClient.Lint(
			ctx,
			workspace.GetLintConfigForOpaqueID(module.OpaqueID()),
			image,
			bufcheck.WithPluginConfigs(workspace.PluginConfigs()...),
			bufcheck.WithPolicyConfigs(workspace.PolicyConfigs()...),
		); err != nil {
			var fileAnnotationSet bufanalysis.FileAnnotationSet
			if !errors.As(err, &fileAnnotationSet) {
				if errors.Is(err, context.Canceled) {
					f.lsp.logger.DebugContext(ctx, "checks cancelled", slog.String("uri", f.uri.Filename()), xslog.ErrorAttr(err))
				} else if errors.Is(err, context.DeadlineExceeded) {
					f.lsp.logger.WarnContext(ctx, "checks deadline exceeded", slog.String("uri", f.uri.Filename()), xslog.ErrorAttr(err))
				} else {
					f.lsp.logger.WarnContext(ctx, "checks failed", slog.String("uri", f.uri.Filename()), xslog.ErrorAttr(err))
				}
				return
			}
			if len(fileAnnotationSet.FileAnnotations()) == 0 {
				f.lsp.logger.DebugContext(ctx, "checks lint passed", slog.String("uri", f.uri.Filename()))
			} else {
				annotations = append(annotations, fileAnnotationSet.FileAnnotations()...)
			}
		}

		select {
		case <-ctx.Done():
			f.lsp.logger.DebugContext(ctx, "checks cancelled", slog.String("uri", f.uri.Filename()), xslog.ErrorAttr(ctx.Err()))
			return
		default:
		}

		f.lsp.lock.Lock()
		defer f.lsp.lock.Unlock()

		select {
		case <-ctx.Done():
			f.lsp.logger.DebugContext(ctx, "checks: cancelled after waiting for file lock", slog.String("uri", f.uri.Filename()), xslog.ErrorAttr(ctx.Err()))
			return // Context cancelled whilst waiting to publishing diagnostics.
		default:
		}

		// Update diagnostics and publish.
		if len(diagnostics) != 0 {
			// TODO: prefer diagnostics from the old compiler to the new compiler to remove duplicates from both.
			f.diagnostics = diagnostics
		}
		f.appendAnnotations("buf lint", annotations)
		f.PublishDiagnostics(ctx)
	}()
}

func (f *file) appendAnnotations(source string, annotations []bufanalysis.FileAnnotation) {
	for _, annotation := range annotations {
		// Convert 1-indexed byte-based line/column to byte offset.
		startLocation := f.file.InverseLocation(annotation.StartLine(), annotation.StartColumn(), positionalEncoding)
		endLocation := f.file.InverseLocation(annotation.EndLine(), annotation.EndColumn(), positionalEncoding)
		protocolRange := reportLocationsToProtocolRange(startLocation, endLocation)
		f.diagnostics = append(f.diagnostics, protocol.Diagnostic{
			Range:    protocolRange,
			Code:     annotation.Type(),
			Severity: protocol.DiagnosticSeverityWarning,
			Source:   source,
			Message:  annotation.Message(),
		})
	}
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

	defer xslog.DebugProfile(f.lsp.logger, slog.String("uri", string(f.uri)))()

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

// wktObjectInfo is a concrete type to help us identify WKTs among the importable files.
type wktObjectInfo struct {
	storage.ObjectInfo
}

// GetSymbols retrieves symbols for the file. If a query is passed, matches only symbols matching
// the case-insensitive substring match to the symbol.
//
// This operation requires [IndexSymbols].
func (f *file) GetSymbols(query string) iter.Seq[protocol.SymbolInformation] {
	return func(yield func(protocol.SymbolInformation) bool) {
		if f.ir.IsZero() {
			return
		}
		// Search through all symbols in this file.
		for _, sym := range f.symbols {
			if sym.ir.IsZero() {
				continue
			}
			// Only include definitions: static and referenceable symbols.
			// Skip references, imports, builtins, and tags
			_, isStatic := sym.kind.(*static)
			_, isReferenceable := sym.kind.(*referenceable)
			if !isStatic && !isReferenceable {
				continue
			}
			symbolInfo := sym.GetSymbolInformation()
			if symbolInfo.Name == "" {
				continue // Symbol information not supported for this symbol.
			}
			// Filter by query (case-insensitive substring match)
			if query != "" && !strings.Contains(strings.ToLower(symbolInfo.Name), query) {
				continue
			}
			if !yield(symbolInfo) {
				return
			}
		}
	}
}
