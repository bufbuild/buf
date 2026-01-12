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

	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report/tags"
	"go.lsp.dev/protocol"
)

// getOrganizeImportsCodeAction generates a code action for organizing imports.
// It uses IR diagnostics to find unresolved types and adds imports for them.
func (s *server) getOrganizeImportsCodeAction(ctx context.Context, file *file) *protocol.CodeAction {
	if file.workspace == nil || file.irReport == nil {
		return nil
	}

	s.logger.Debug("code action: checking IR diagnostics", "count", len(file.irReport.Diagnostics))

	// Find all unresolved type references from the IR diagnostics
	unresolvedRefs := make(map[ir.FullName]bool) // full type name -> bool

	for _, diag := range file.irReport.Diagnostics {
		// Note: file.objectInfo.Path() returns the relative proto import path (e.g., "import_test.proto")
		// while diag.Primary().Path() returns the absolute source path. We need to compare against
		// the file's source path instead.
		if diag.Primary().Path() != file.file.Path() {
			continue
		}

		// Only process UnknownSymbol diagnostics (missing imports)
		if diag.Tag() != tags.UnknownSymbol {
			continue
		}

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

	s.logger.Debug("code action: found unresolved references", "count", len(unresolvedRefs))
	if len(unresolvedRefs) == 0 {
		return nil
	}

	// Find imports needed for each unresolved type
	importsToAdd := make(map[string]bool) // import path -> bool

	for fullTypeName := range unresolvedRefs {
		// Search for this type in all workspace files
		for _, workspaceFile := range file.workspace.PathToFile() {
			// Skip the current file
			if workspaceFile.file.Path() == file.file.Path() {
				continue
			}
			if !workspaceFile.ir.FindSymbol(fullTypeName).IsZero() {
				importPath := workspaceFile.objectInfo.Path()
				importsToAdd[importPath] = true
				s.logger.Debug("code action: found type in file",
					"fullTypeName", fullTypeName,
					"importPath", importPath)
				break
			}
		}
	}

	if len(importsToAdd) == 0 {
		return nil
	}

	// Find where to insert the imports and get existing imports
	insertLine, currentImports := findImportInsertLine(file)

	// Filter out imports that already exist
	var newImportsToAdd []string
	for path := range importsToAdd {
		if _, exists := currentImports[path]; !exists {
			newImportsToAdd = append(newImportsToAdd, path)
		}
	}

	if len(newImportsToAdd) == 0 {
		return nil
	}

	// Sort for deterministic output
	slices.Sort(newImportsToAdd)

	var importText strings.Builder
	for _, path := range newImportsToAdd {
		importText.WriteString(fmt.Sprintf("import %q;\n", path))
	}

	// Create the text edit
	edit := protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{Line: insertLine, Character: 0},
			End:   protocol.Position{Line: insertLine, Character: 0},
		},
		NewText: importText.String(),
	}

	// Create the workspace edit
	changes := make(map[protocol.DocumentURI][]protocol.TextEdit)
	changes[file.uri] = []protocol.TextEdit{edit}

	workspaceEdit := protocol.WorkspaceEdit{
		Changes: changes,
	}

	// Create the code action
	return &protocol.CodeAction{
		Title: "Organize imports",
		Kind:  protocol.QuickFix,
		Edit:  &workspaceEdit,
	}
}
