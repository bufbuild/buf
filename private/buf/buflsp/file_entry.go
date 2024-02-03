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
	"io"
	"os"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/protocompile/ast"
	"go.lsp.dev/protocol"
)

type fileEntry struct {
	server   *server
	document *protocol.TextDocumentItem
	lines    []string
	refCount int

	resolver moduleSetResolver

	externalPath string
	isRemote     bool // If the file is part of a remote module.

	hasParseError bool // If parsing produced an error.
	fileNode      *ast.FileNode
	imports       []*importEntry
	pkg           symbolName
	parseDiags    []protocol.Diagnostic
	bufDiags      []protocol.Diagnostic
	docSymbols    []protocol.DocumentSymbol
	typeSymbols   []*symbolEntry
	fieldSymbols  []*symbolEntry

	// Hierarchy of symbols in the file.
	symbolScopes []*symbolScopeEntry
}

func newFileEntry(
	server *server,
	document *protocol.TextDocumentItem,
	resolver moduleSetResolver,
	externalPath string,
	isRemote bool,
) *fileEntry {
	result := &fileEntry{
		server:       server,
		document:     document,
		refCount:     1,
		resolver:     resolver,
		externalPath: externalPath,
		isRemote:     isRemote,
	}
	result.lines = strings.Split(result.document.Text, "\n")
	return result
}

func (f *fileEntry) getSourcePos(pos protocol.Position) ast.SourcePos {
	// TODO: Figure out the right conversion to Col and Offset. pos.Character is in utf-8, utf-16,
	// or utf-32 depending on the client capabilities. Also tabs might be counted as multiple 'columns'.
	return ast.SourcePos{
		Line:     int(pos.Line + 1),
		Col:      int(pos.Character + 1),
		Filename: f.externalPath,
	}
}

func (f *fileEntry) processText(ctx context.Context) error {
	f.tryParse()
	if err := f.resolveImports(ctx); err != nil {
		return err
	}
	return f.server.updateDiagnostics(ctx, f)
}

// Returns false if there was a diff.
func (f *fileEntry) updateText(ctx context.Context, text string) (bool, error) {
	f.document.Text = text
	f.lines = strings.Split(f.document.Text, "\n")
	matchDisk := false
	if fileReader, err := os.Open(normalpath.Unnormalize(f.externalPath)); err == nil {
		fileData, err := io.ReadAll(fileReader)
		if err != nil {
			return false, err
		}
		matchDisk = string(fileData) == f.document.Text
	}
	if !matchDisk {
		f.bufDiags = nil
	}
	return matchDisk, f.processText(ctx)
}

func (f *fileEntry) tryParse() {
	// Parse the file
	fileNode, diagnostics, err := parseFile("", strings.NewReader(f.document.Text))
	f.parseDiags = diagnostics
	f.fileNode = fileNode
	f.hasParseError = err != nil
	f.generateSymbols()
}

func (f *fileEntry) generateSymbols() {
	// Clear the symbol indexes.
	f.imports = nil
	f.typeSymbols = nil
	f.fieldSymbols = nil
	f.symbolScopes = nil

	gen := &docSymbolGen{
		fileEntry: f,
	}
	f.docSymbols = gen.getFileSymbols(f.fileNode)
	f.pkg = gen.pkg

	sort.Slice(f.typeSymbols, func(i, j int) bool {
		return compareSlices(f.typeSymbols[i].name(), f.typeSymbols[j].name()) < 0
	})
	sort.Slice(f.fieldSymbols, func(i, j int) bool {
		return compareSlices(f.fieldSymbols[i].name(), f.fieldSymbols[j].name()) < 0
	})
}

func (f *fileEntry) resolveImports(ctx context.Context) error {
	for _, importStatement := range f.imports {
		if importStatement.docURI != "" {
			continue
		}
		importEntry, err := f.server.resolveImport(ctx, f.resolver, importStatement.node.Name.AsString())
		if err != nil {
			return err
		} else if importEntry != nil {
			importStatement.docURI = importEntry.document.URI
		}
	}
	return nil
}

func (f *fileEntry) findImportEntry(pos ast.SourcePos) *importEntry {
	for _, importEntry := range f.imports {
		if f.containsPos(importEntry.node, pos) {
			return importEntry
		}
	}
	return nil
}

func (f *fileEntry) findSymbols(name symbolName, includeFields bool) []*symbolEntry {
	if !sliceHasPrefix(name, f.pkg) {
		return nil // All symbols are defined in the current package, so no symbols can match.
	}
	var result []*symbolEntry
	for _, symbol := range f.typeSymbols {
		if compareSlices(symbol.name(), name) == 0 {
			result = append(result, symbol)
		}
	}
	if includeFields {
		for _, symbol := range f.fieldSymbols {
			if compareSlices(symbol.name(), name) == 0 {
				result = append(result, symbol)
			}
		}
	}
	return result
}

func (f *fileEntry) codeAt(pos ast.SourcePos) string {
	codeString := ""
	if pos.Line == 0 || pos.Line > len(f.lines) {
		return codeString
	}
	line := f.lines[pos.Line-1]
	for i := pos.Col - 2; i < len(line); i-- {
		char := rune(line[i])
		// If idnet char or '.' or '(' or ')'
		if strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.()[", char) {
			codeString = string(char) + codeString
		} else {
			break
		}
	}
	return codeString
}

func (f *fileEntry) tokenToStartPos(token ast.Token) ast.SourcePos {
	return f.fileNode.ItemInfo(token.AsItem()).Start()
}

func (f *fileEntry) tokenToEndPos(token ast.Token) ast.SourcePos {
	return f.fileNode.ItemInfo(token.AsItem()).End()
}

func (f *fileEntry) containsPos(node ast.Node, pos ast.SourcePos) bool {
	if node == nil {
		return false
	}
	return comparePos(f.tokenToStartPos(node.Start()), pos) <= 0 && comparePos(pos, f.tokenToEndPos(node.End())) <= 0
}

func (f *fileEntry) containsOrPastPos(node ast.Node, pos ast.SourcePos) bool {
	return comparePos(f.tokenToEndPos(node.End()), pos) >= 0
}

func (f *fileEntry) findSymbolScope(pos ast.SourcePos) *symbolScopeEntry {
	if f.fileNode == nil {
		return nil
	}
	var result *symbolScopeEntry
	cur := f.symbolScopes
	for len(cur) > 0 {
		found := false
		for _, codePos := range cur {
			if f.containsPos(codePos.symbol.node, pos) {
				result = codePos
				cur = codePos.children
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return result
}

func (f *fileEntry) findScope(pos ast.SourcePos) symbolName {
	scope := f.pkg
	sybmolScope := f.findSymbolScope(pos)
	if sybmolScope != nil {
		scope = sybmolScope.symbol.name()
	}
	return scope
}

// Returns an ordered list of candidate scopes for the given prefix.
// For example: scope = a.b.c, prefix = d.e
//
//	a.b.c.d.e
//	a.b.d.e
//	a.d.e
//	d.e
func findCandidates(prefix symbolRefName, scope symbolName) []symbolName {
	if prefix.isAbsolute() {
		return []symbolName{symbolName(prefix[1:])}
	}

	var candidates []symbolName
	for i := len(scope); i >= 0; i-- {
		newScope := symbolName{}
		newScope = append(newScope, scope[:i]...)
		newScope = append(newScope, prefix...)
		candidates = append(candidates, newScope)
	}
	return candidates
}

func (f *fileEntry) findCompletions(candidate []string, options map[string]protocol.CompletionItem, fieldsOnly bool) {
	if fieldsOnly {
		for _, symbol := range f.fieldSymbols {
			f.maybeAddCompletion(candidate, symbol, options, true)
		}
	} else {
		for _, symbol := range f.typeSymbols {
			f.maybeAddCompletion(candidate, symbol, options, false)
		}
	}
}

func (f *fileEntry) maybeAddCompletion(candidate []string, symbol *symbolEntry, options map[string]protocol.CompletionItem, prop bool) {
	if len(candidate) >= len(symbol.name()) {
		return
	}
	i := 0
	for ; i < len(candidate); i++ {
		if candidate[i] != symbol.name()[i] {
			return
		}
	}
	if len(symbol.name()) == i+1 {
		options[symbol.name()[i]] = getCompletionItem(symbol)
		return
	}

	if prop {
		// Don't dive into sub messages
		if symbol := f.findSymbols(symbol.name()[:i+1], false); len(symbol) > 0 {
			return
		}
	}

	if _, ok := options[symbol.name()[i]]; !ok {
		options[symbol.name()[i]] = protocol.CompletionItem{
			Label: symbol.name()[i],
			Kind:  protocol.CompletionItemKindModule,
		}
	}
}

func (f *fileEntry) tokenRange(token ast.Token) protocol.Range {
	info := f.fileNode.ItemInfo(token.AsItem())
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(info.Start().Line - 1),
			Character: uint32(info.Start().Col - 1),
		},
		End: protocol.Position{
			Line:      uint32(info.End().Line - 1),
			Character: uint32(info.End().Col - 1),
		},
	}
}

func (f *fileEntry) extentRange(start ast.Token, end ast.Token) protocol.Range {
	startInfo := f.fileNode.ItemInfo(start.AsItem())
	endInfo := f.fileNode.ItemInfo(end.AsItem())
	return protocol.Range{
		Start: protocol.Position{
			Line:      uint32(startInfo.Start().Line - 1),
			Character: uint32(startInfo.Start().Col - 1),
		},
		End: protocol.Position{
			Line:      uint32(endInfo.End().Line - 1),
			Character: uint32(endInfo.End().Col - 1),
		},
	}
}

func (f *fileEntry) nodeLocation(node ast.Node) protocol.Range {
	switch node := node.(type) {
	case *ast.MessageNode:
		return f.tokenRange(node.Name.Token())
	case *ast.EnumNode:
		return f.tokenRange(node.Name.Token())
	case *ast.EnumValueNode:
		return f.tokenRange(node.Name.Token())
	case *ast.MapFieldNode:
		return f.tokenRange(node.Name.Token())
	case *ast.FieldNode:
		return f.tokenRange(node.Name.Token())
	case *ast.ServiceNode:
		return f.tokenRange(node.Name.Token())
	case *ast.RPCNode:
		return f.tokenRange(node.Name.Token())
	default:
		return f.tokenRange(node.Start())
	}
}

func comparePos(lhs ast.SourcePos, rhs ast.SourcePos) int {
	switch {
	case lhs.Line < rhs.Line:
		return -1
	case lhs.Line > rhs.Line:
		return 1
	case lhs.Col < rhs.Col:
		return -1
	case lhs.Col > rhs.Col:
		return 1
	default:
		return 0
	}
}

func compareSlices(lhs []string, rhs []string) int {
	for i := 0; i < len(lhs) && i < len(rhs); i++ {
		if lhs[i] != rhs[i] {
			if lhs[i] < rhs[i] {
				return -1
			} else {
				return 1
			}
		}
	}
	return len(lhs) - len(rhs)
}

func sliceHasPrefix(slice []string, prefix []string) bool {
	if len(slice) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if slice[i] != prefix[i] {
			return false
		}
	}
	return true
}

type sigWriter struct {
	fileNode *ast.FileNode
	output   *strings.Builder
	err      error
}

func (f *fileEntry) genNodeSignature(node ast.Node) (string, error) {
	writer := &sigWriter{
		fileNode: f.fileNode,
		output:   &strings.Builder{},
	}
	writer.writeNode(node)
	if writer.err != nil {
		return "", writer.err
	}
	result := writer.output.String()
	leadingWhitespace := len(result) - len(strings.TrimLeft(result, " \t\n"))
	startPos := strings.LastIndex(result[:leadingWhitespace], "\n") + 1
	return deindent(result[startPos:]), nil
}

func (w *sigWriter) writeNode(node ast.Node) {
	switch node := node.(type) {
	case *ast.MessageNode:
		w.writeLeadingComments(node.Keyword)
		w.P("message " + node.Name.Val + " {")
	case *ast.EnumNode:
		w.writeLeadingComments(node.Keyword)
		w.P("enum " + node.Name.Val + " {")
	case *ast.FieldNode:
		w.writeFullNode(node)
	case *ast.MapFieldNode:
		w.writeFullNode(node)
	}
}

func (w *sigWriter) writeFullNode(node ast.Node) {
	err := ast.Walk(node, &ast.SimpleVisitor{
		DoVisitTerminalNode: func(token ast.TerminalNode) error {
			info := w.fileNode.NodeInfo(token)
			w.writeComments(info.LeadingComments())
			w.WriteString(info.LeadingWhitespace())
			w.WriteString(info.RawText())
			w.writeComments(info.TrailingComments())
			return nil
		},
	})
	if err != nil && w.err == nil {
		w.err = err
	}
}

func (w *sigWriter) writeComments(comments ast.Comments) {
	for i := 0; i < comments.Len(); i++ {
		comment := comments.Index(i)
		w.WriteString(comment.LeadingWhitespace())
		w.WriteString(comment.RawText())
	}
}

func (w *sigWriter) writeLeadingComments(node ast.TerminalNode) {
	info := w.fileNode.NodeInfo(node)
	for i := 0; i < info.LeadingComments().Len(); i++ {
		w.P(info.LeadingComments().Index(i).RawText())
	}
}

func (w *sigWriter) P(str string) {
	if w.err != nil {
		return
	}
	if _, err := w.output.WriteString(str); err != nil {
		w.err = err
		return
	}
	if _, err := w.output.WriteString("\n"); err != nil {
		w.err = err
	}
}

func (w *sigWriter) WriteString(str string) {
	if w.err != nil {
		return
	}
	if _, err := w.output.WriteString(str); err != nil {
		w.err = err
	}
}

func deindent(str string) string {
	lines := strings.Split(str, "\n")
	minIndent := findIndent(lines[0])
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		indent := findIndent(line)
		if indent < minIndent {
			minIndent = indent
		}
	}
	for i := 0; i < len(lines); i++ {
		lines[i] = lines[i][minIndent:]
	}
	return strings.Join(lines, "\n")
}

func findIndent(str string) int {
	return len(str) - len(strings.TrimLeft(str, " \t"))
}
