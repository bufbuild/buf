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

// This file implements useful code actions.

package buflsp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report/tags"
	"github.com/bufbuild/protocompile/experimental/seq"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/source/length"
	"github.com/bufbuild/protocompile/experimental/token"
	"github.com/bufbuild/protocompile/experimental/token/keyword"
	"go.lsp.dev/protocol"
)

type importInfo struct {
	path                   string
	importDecl             ast.DeclImport
	importWithCommentsSpan source.Span
}

// getOrganizeImportsCodeAction generates a code action for organizing imports.
// It uses IR diagnostics to find unresolved types, unused imports, and organizes all imports.
func (s *server) getOrganizeImportsCodeAction(ctx context.Context, file *file) *protocol.CodeAction {
	if file.workspace == nil || file.irReport == nil {
		return nil
	}

	s.logger.Debug("code action: checking IR diagnostics", "count", len(file.irReport.Diagnostics))

	// Find all unresolved type references and unused imports from the IR diagnostics
	unresolvedRefs := make(map[ir.FullName]bool) // full type name -> bool
	unusedImports := make(map[string]bool)       // import path -> bool

	for _, diag := range file.irReport.Diagnostics {
		if diag.Primary().Path() != file.file.Path() {
			continue
		}

		// Process UnknownSymbol diagnostics (missing imports)
		if diag.Tag() == tags.UnknownSymbol {
			// Get the exact symbol name as written in the source
			missingType := diag.Primary().Text()
			s.logger.Debug("code action: unknown symbol", "missingType", missingType, "message", diag.Message())

			if missingType != "" {
				// The text may have a leading dot for absolute paths, remove it
				typeName := strings.TrimPrefix(missingType, ".")
				unresolvedRefs[ir.FullName(typeName)] = true
				s.logger.Debug("code action: found unresolved type", "typeName", typeName)
			}
		}

		// Process UnusedImport diagnostics (imports to remove)
		if diag.Tag() == tags.UnusedImport {
			// The diagnostic text contains the import path
			unusedImportPath := diag.Primary().Text()
			s.logger.Debug("code action: unused import", "importPath", unusedImportPath, "message", diag.Message())
			if unusedImportPath != "" {
				// Remove quotes if present
				unusedImportPath = strings.Trim(unusedImportPath, "\"")
				unusedImports[unusedImportPath] = true
				s.logger.Debug("code action: found unused import", "importPath", unusedImportPath)
			}
		}
	}

	s.logger.Debug("code action: found unresolved references", "count", len(unresolvedRefs))
	s.logger.Debug("code action: found unused imports", "count", len(unusedImports))

	// Find imports needed for each unresolved type
	importsToAdd := make(map[string]bool) // import path -> bool
	for typeFullName := range unresolvedRefs {
		// Search for this type in all workspace files
		for _, workspaceFile := range file.workspace.PathToFile() {
			// Skip the current file
			if workspaceFile.file.Path() == file.file.Path() {
				continue
			}
			if symbolInFile(typeFullName, workspaceFile) {
				importPath := workspaceFile.objectInfo.Path()
				importsToAdd[importPath] = true
				s.logger.Debug("code action: found type in file",
					"typeFullName", typeFullName,
					"importPath", importPath)
				break
			}
		}
	}

	var (
		imports []importInfo
		edits   []protocol.TextEdit
	)
	for currentFileImport := range seq.Values(file.ir.Imports()) {
		importPath := currentFileImport.Path()
		importDecl := currentFileImport.Decl
		importWithCommentsSpan := captureImportSpan(currentFileImport.Decl)
		edits = append(edits, protocol.TextEdit{
			Range:   reportSpanToProtocolRange(importWithCommentsSpan),
			NewText: "", // delete
		})
		if unusedImports[importPath] {
			continue
		}
		imports = append(imports, importInfo{
			path:                   importPath,
			importDecl:             importDecl,
			importWithCommentsSpan: importWithCommentsSpan,
		})
	}
	// Add new imports.
	for importPath := range importsToAdd {
		imports = append(imports, importInfo{
			path: importPath,
		})
	}
	slices.SortFunc(imports, func(a, b importInfo) int {
		return strings.Compare(a.path, b.path)
	})

	// Build the new import text
	var importText strings.Builder
	if len(imports) > 0 {
		importText.WriteString("\n")
	}
	for _, info := range imports {
		if info.importWithCommentsSpan.IsZero() {
			fmt.Fprintf(&importText, "%s %q;\n", keyword.Import, info.path)
			continue
		}
		text := info.importWithCommentsSpan.Text()
		text = strings.TrimRightFunc(text, unicode.IsSpace)
		importText.WriteString(text + "\n")
	}

	// Find the insert position after the package or syntax declaration
	var insertLine int
	switch {
	case !file.ir.AST().Package().IsZero():
		insertLine = file.ir.AST().Package().Span().EndLoc().Line + 1
	case !file.ir.AST().Syntax().IsZero():
		insertLine = file.ir.AST().Syntax().Span().EndLoc().Line + 1
	default:
		insertLine = 1 // Default at top of file.
	}
	// Compare to the insert offset, at the newline (so increment 1)
	insertOffset := file.file.InverseLocation(insertLine, 0, length.Bytes).Offset
	if strings.HasPrefix(file.file.Text()[insertOffset+1:], importText.String()) {
		return nil // Matches, no changes needed.
	}

	edits = append(edits, protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(insertLine - 1)},
			End:   protocol.Position{Line: uint32(insertLine - 1)},
		},
		NewText: importText.String(),
	})
	return &protocol.CodeAction{
		Title: "Organize imports",
		Kind:  protocol.SourceOrganizeImports,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[protocol.DocumentURI][]protocol.TextEdit{
				file.uri: edits,
			},
		},
	}
}

// symbolInFiles searches through the files exported types.
func symbolInFile(fullName ir.FullName, file *file) bool {
	for _, typ := range seq.All(file.ir.AllTypes()) {
		if typ.FullName() == fullName {
			return true
		}
	}
	for typ := range file.ir.AllMembers() {
		if typ.FullName() == fullName {
			return true
		}
	}
	return false
}

// captureImportSpan expands the import span declaration to include both
// leading and trailing comments and any trailing whitespace to the next
// declaration.
func captureImportSpan(decl ast.DeclImport) source.Span {
	span := decl.Span()
	if decl.IsZero() {
		return span
	}
	stream := decl.Context().Stream()

	// Capture up until the newline expanding upwards for all comments directly
	// above this declaration.
	tok, prev := stream.Around(decl.Span().Start)
	cursor := token.NewCursorAt(tok)
	for isTokenSpace(tok) {
		tok, prev = cursor.PrevSkippable(), tok
	}
	span.Start = tok.Span().Start
	for isTokenNewline(tok) {
		span.Start = prev.Span().Start // Capture the previous, up until this newline.
		tok, prev = cursor.PrevSkippable(), tok
		if tok.Kind() != token.Comment {
			break
		}
		// Consume all comments and space tokens
		for tok.Kind() == token.Comment || isTokenSpace(tok) {
			tok, prev = cursor.PrevSkippable(), tok
		}
	}

	// Extract trailing comment (same line after semicolon)
	tok, _ = stream.Around(decl.Span().End)
	cursor = token.NewCursorAt(tok)
	tok = cursor.NextSkippable()
	for isTokenSpace(tok) || tok.Kind() == token.Keyword && tok.Keyword() == keyword.Semi {
		tok = cursor.NextSkippable()
	}
	for isTokenSpace(tok) || tok.Kind() == token.Comment {
		tok = cursor.NextSkippable()
	}
	if tok.Kind() != token.Space {
		return span // unknown
	}
	for next := cursor.NextSkippable(); next.Kind() == token.Space; {
		tok = next // Capture anywhitespace
		next = cursor.NextSkippable()
	}
	span.End = tok.Span().End
	return span
}
