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

package buflsp

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bufbuild/protocompile/ast"
	"github.com/bufbuild/protocompile/experimental/seq"
	"go.lsp.dev/protocol"
)

// importWithComments represents an import along with its preceding and trailing comments
type importWithComments struct {
	path            string
	modifier        string
	leadingComments []string // lines of comments that precede this import
	trailingComment string   // comment after the semicolon (if any)
}

// computeOrganizeImportsEdits computes the text edits needed to organize imports:
//   - Remove unused imports
//   - Add missing imports (if unambiguous)
//   - Sort imports alphabetically
//   - Preserve comments
func (s *server) computeOrganizeImportsEdits(
	_ context.Context,
	file *file,
	parsed *ast.FileNode,
) ([]protocol.TextEdit, error) {
	// Collect all imports from the AST
	var allImports []*ast.ImportNode
	for _, decl := range parsed.Decls {
		if importNode, ok := decl.(*ast.ImportNode); ok {
			allImports = append(allImports, importNode)
		}
	}

	// Use the IR to determine which imports are used
	// If IR is nil (e.g., file has parse errors like duplicate imports),
	// we'll keep all imports and just deduplicate/sort them
	usedImportPaths := make(map[string]bool)
	hasIR := file.ir != nil
	if hasIR {
		for _, imp := range seq.All(file.ir.Imports()) {
			if imp.Used {
				usedImportPaths[imp.Path()] = true
			}
		}
	} else {
		// No IR available - keep all imports (can't determine usage)
		for _, imp := range allImports {
			usedImportPaths[imp.Name.AsString()] = true
		}
	}

	// Collect comments from ALL imports (used and unused)
	// We need to preserve all comments, even from unused imports
	var usedImports []importWithComments
	var orphanedComments []string // comments from unused imports

	// Get the original source text to preserve exact formatting
	sourceText := file.file.Text()

	for _, imp := range allImports {
		path := imp.Name.AsString()
		leading, trailing := s.extractComments(parsed, imp, sourceText)

		if usedImportPaths[path] {
			iwc := importWithComments{
				path:            path,
				leadingComments: leading,
				trailingComment: trailing,
			}
			// Track the modifier (public/weak)
			if imp.Modifier != nil {
				iwc.modifier = imp.Modifier.Val
			}
			usedImports = append(usedImports, iwc)
		} else {
			// Import is unused, but preserve its comments
			orphanedComments = append(orphanedComments, leading...)
			if trailing != "" {
				// For orphaned trailing comments, trim leading whitespace since they're
				// becoming regular comments (no longer trailing)
				orphanedComments = append(orphanedComments, strings.TrimLeft(trailing, " \t"))
			}
		}
	}

	// Find missing imports by walking the AST for type references
	// Only do this if we have IR (need it to determine which types are imported)
	var missingImportPaths []string
	if hasIR {
		missingImportPaths = s.findMissingImportPaths(file, parsed)
	}

	// Check if we need to make any changes
	hasDuplicates := importsHaveDuplicates(allImports)
	needsChanges := len(usedImports) != len(allImports) ||
		len(missingImportPaths) > 0 ||
		!importsAreSorted(allImports) ||
		hasDuplicates
	if !needsChanges {
		return nil, nil
	}

	// Create importWithComments for missing imports
	allImportsWithComments := make([]importWithComments, 0, len(usedImports)+len(missingImportPaths))
	allImportsWithComments = append(allImportsWithComments, usedImports...)
	for _, path := range missingImportPaths {
		allImportsWithComments = append(allImportsWithComments, importWithComments{path: path})
	}

	// Deduplicate while preserving comments
	// Group imports by path to handle duplicates (no need to sort first, we'll sort after dedup)
	importsByPath := make(map[string][]importWithComments)
	for _, imp := range allImportsWithComments {
		importsByPath[imp.path] = append(importsByPath[imp.path], imp)
	}

	// For each path, select the best representative
	uniqueImports := make([]importWithComments, 0, len(importsByPath))
	for path, imports := range importsByPath {
		if len(imports) == 1 {
			uniqueImports = append(uniqueImports, imports[0])
			continue
		}

		// Multiple imports with same path - need to merge intelligently
		// Find imports with trailing comments
		var withTrailing []importWithComments
		for _, imp := range imports {
			if imp.trailingComment != "" {
				withTrailing = append(withTrailing, imp)
			}
		}

		// Error if multiple duplicates have trailing comments (would lose a comment)
		if len(withTrailing) > 1 {
			return nil, fmt.Errorf("duplicate import %q has multiple trailing comments - cannot organize without losing comments", path)
		}

		// Prefer the import with a trailing comment
		var chosen importWithComments
		if len(withTrailing) == 1 {
			chosen = withTrailing[0]
		} else {
			chosen = imports[0]
		}

		// Merge all leading comments from duplicates
		var allLeadingComments []string
		for _, imp := range imports {
			allLeadingComments = append(allLeadingComments, imp.leadingComments...)
		}
		chosen.leadingComments = allLeadingComments

		uniqueImports = append(uniqueImports, chosen)
	}

	// Sort uniqueImports by path to maintain alphabetical order
	slices.SortFunc(uniqueImports, func(a, b importWithComments) int {
		return strings.Compare(a.path, b.path)
	})

	// Generate formatted import lines, preserving modifiers and comments
	var importLines strings.Builder

	// Add orphaned comments at the top (from unused imports)
	for _, comment := range orphanedComments {
		importLines.WriteString(comment)
		importLines.WriteString("\n")
	}

	for _, imp := range uniqueImports {
		// Add any leading comments
		for _, comment := range imp.leadingComments {
			importLines.WriteString(comment)
			importLines.WriteString("\n")
		}
		// Add the import line with trailing comment (if any)
		if imp.modifier != "" {
			fmt.Fprintf(&importLines, "import %s %q;", imp.modifier, imp.path)
		} else {
			fmt.Fprintf(&importLines, "import %q;", imp.path)
		}
		// Trailing comment already includes the whitespace before it
		importLines.WriteString(imp.trailingComment)
		importLines.WriteString("\n")
	}

	// Find the range to replace in the original file
	if len(allImports) == 0 {
		// No existing imports - insert after package declaration
		return s.insertImportsAfterPackage(parsed, importLines.String())
	}

	// Get the range of the existing import section, including leading comments
	// of the first import
	firstImport := allImports[0]
	lastImport := allImports[len(allImports)-1]

	startInfo := parsed.NodeInfo(firstImport)
	endInfo := parsed.NodeInfo(lastImport)

	startPos := startInfo.Start()
	endPos := endInfo.End()

	// Check if the first import has leading comments and adjust start position
	// Also check if there are orphaned comments
	if (len(usedImports) > 0 && len(usedImports[0].leadingComments) > 0) || len(orphanedComments) > 0 {
		// Adjust to start of first comment (if any)
		leadingComments := startInfo.LeadingComments()
		if leadingComments.Len() > 0 {
			startPos = leadingComments.Index(0).Start()
		}
	}

	// Convert to protocol positions (0-based line, 0-based character)
	startLine, err := toUint32(startPos.Line - 1)
	if err != nil {
		return nil, err
	}
	startCol, err := toUint32(startPos.Col - 1)
	if err != nil {
		return nil, err
	}
	endLine, err := toUint32(endPos.Line - 1)
	if err != nil {
		return nil, err
	}
	endCol, err := toUint32(endPos.Col - 1)
	if err != nil {
		return nil, err
	}

	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      startLine,
					Character: startCol,
				},
				End: protocol.Position{
					Line:      endLine,
					Character: endCol,
				},
			},
			NewText: strings.TrimSuffix(importLines.String(), "\n"),
		},
	}, nil
}

// insertImportsAfterPackage creates an edit to insert imports after the package declaration
func (s *server) insertImportsAfterPackage(parsed *ast.FileNode, importText string) ([]protocol.TextEdit, error) {
	// Find the package declaration
	var pkgNode *ast.PackageNode
	for _, decl := range parsed.Decls {
		if pkg, ok := decl.(*ast.PackageNode); ok {
			pkgNode = pkg
			break
		}
	}

	if pkgNode == nil {
		// No package declaration - this shouldn't happen in valid proto files
		return nil, fmt.Errorf("no package declaration found")
	}

	// Get the end position of the package declaration
	endInfo := parsed.NodeInfo(pkgNode)
	endPos := endInfo.End()

	// Convert to protocol positions
	line, err := toUint32(endPos.Line - 1)
	if err != nil {
		return nil, err
	}
	col, err := toUint32(endPos.Col - 1)
	if err != nil {
		return nil, err
	}

	// Insert after the package line with a blank line in between
	return []protocol.TextEdit{
		{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      line,
					Character: col,
				},
				End: protocol.Position{
					Line:      line,
					Character: col,
				},
			},
			NewText: "\n\n" + strings.TrimSuffix(importText, "\n"),
		},
	}, nil
}

// findMissingImportPaths walks the AST to find type references that don't have imports,
// and returns import paths for unambiguous types.
func (s *server) findMissingImportPaths(file *file, parsed *ast.FileNode) []string {
	if file.workspace == nil {
		return nil
	}

	// Collect all type references from the AST using a visitor
	typeRefs := make(map[string]bool)
	visitor := &typeRefCollector{typeRefs: typeRefs}
	// Our visitor always returns nil, so this should never error
	if err := ast.Walk(parsed, visitor); err != nil {
		return nil
	}

	// Find types that are referenced but not imported
	var missingImportPaths []string
	seenPaths := make(map[string]bool)
	for typeName := range typeRefs {
		// Skip if already imported
		if isTypeImported(typeName, file) {
			continue
		}

		// Find which file(s) define this type
		definingFiles := s.findFilesDefiningType(file, typeName)

		// Only add import if exactly one file defines it (unambiguous)
		if len(definingFiles) == 1 && !seenPaths[definingFiles[0]] {
			missingImportPaths = append(missingImportPaths, definingFiles[0])
			seenPaths[definingFiles[0]] = true
		}
	}

	return missingImportPaths
}

// addTypeRef extracts type name from a node and adds to the set
func addTypeRef(node ast.Node, typeRefs map[string]bool) {
	switch t := node.(type) {
	case *ast.IdentNode:
		// Skip builtin types (scalar types)
		if !isBuiltinType(t.Val) {
			typeRefs[t.Val] = true
		}
	case *ast.CompoundIdentNode:
		// For compound idents (e.g., "example.v1.TypeA"), take the last component (the type name)
		if len(t.Components) > 0 {
			name := string(t.Components[len(t.Components)-1].AsIdentifier())
			if !isBuiltinType(name) {
				typeRefs[name] = true
			}
		}
	}
}

// isBuiltinType checks if a type name is a protobuf builtin type (scalar)
func isBuiltinType(typeName string) bool {
	_, ok := builtinDocs[typeName]
	return ok
}

// isTypeImported checks if a type is already imported
func isTypeImported(typeName string, file *file) bool {
	// Check if there's an import that provides this type
	if file.ir == nil {
		return false
	}

	for _, imp := range seq.All(file.ir.Imports()) {
		// Check if this import's file defines the type
		if imp.File != nil {
			for _, typ := range seq.All(imp.File.Types()) {
				if typ.Name() == typeName {
					return true
				}
			}
		}
	}
	return false
}

// findFilesDefiningType searches the workspace for files that define a given type
func (s *server) findFilesDefiningType(file *file, typeName string) []string {
	if file.workspace == nil {
		return nil
	}

	var files []string
	for path, f := range file.workspace.pathToFile {
		if f.ir == nil {
			continue
		}

		// Check if this file defines the type
		for _, typ := range seq.All(f.ir.Types()) {
			if typ.Name() == typeName {
				// Extract just the filename for the import
				files = append(files, filepath.Base(path))
				break
			}
		}
	}

	return files
}

// importsAreSorted checks if imports are in alphabetical order
func importsAreSorted(imports []*ast.ImportNode) bool {
	return slices.IsSortedFunc(imports, func(a, b *ast.ImportNode) int {
		return strings.Compare(a.Name.AsString(), b.Name.AsString())
	})
}

// importsHaveDuplicates checks if there are duplicate import paths
func importsHaveDuplicates(imports []*ast.ImportNode) bool {
	seen := make(map[string]bool)
	for _, imp := range imports {
		path := imp.Name.AsString()
		if seen[path] {
			return true
		}
		seen[path] = true
	}
	return false
}

// extractComments extracts both leading and trailing comments for an import node
func (s *server) extractComments(parsed *ast.FileNode, importNode *ast.ImportNode, sourceText string) (leading []string, trailing string) {
	nodeInfo := parsed.NodeInfo(importNode)

	// Extract leading comments
	leadingComments := nodeInfo.LeadingComments()
	for i := 0; i < leadingComments.Len(); i++ {
		comment := leadingComments.Index(i)
		leading = append(leading, comment.RawText())
	}

	// Extract trailing comment (if any), preserving exact whitespace
	trailingComments := nodeInfo.TrailingComments()
	if trailingComments.Len() > 0 {
		// Get the end position of the import statement (after semicolon)
		importEnd := nodeInfo.End()

		// Extract everything from after the semicolon to end of line (including whitespace)
		// Positions are 1-based, convert to 0-based for string indexing
		lines := strings.Split(sourceText, "\n")
		if importEnd.Line > 0 && importEnd.Line <= len(lines) {
			line := lines[importEnd.Line-1]
			// importEnd.Col is after the semicolon (1-based)
			// Extract from that position to end of line
			if importEnd.Col-1 < len(line) {
				trailing = line[importEnd.Col-1:]
			}
		}
	}

	return leading, trailing
}

// typeRefCollector is a visitor that collects type references from the AST
type typeRefCollector struct {
	ast.NoOpVisitor
	typeRefs map[string]bool
}

func (v *typeRefCollector) VisitFieldNode(node *ast.FieldNode) error {
	addTypeRef(node.FldType, v.typeRefs)
	return nil
}

func (v *typeRefCollector) VisitMapFieldNode(node *ast.MapFieldNode) error {
	addTypeRef(node.MapType.ValueType, v.typeRefs)
	return nil
}

func (v *typeRefCollector) VisitRPCNode(node *ast.RPCNode) error {
	if node.Input != nil {
		addTypeRef(node.Input.MessageType, v.typeRefs)
	}
	if node.Output != nil {
		addTypeRef(node.Output.MessageType, v.typeRefs)
	}
	return nil
}

// toUint32 safely converts an int to uint32, returning an error if out of range
func toUint32(n int) (uint32, error) {
	if n < 0 || n > math.MaxUint32 {
		return 0, fmt.Errorf("value %d out of range for uint32", n)
	}
	return uint32(n), nil
}
