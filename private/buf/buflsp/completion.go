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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/protocompile/ast"
	"go.lsp.dev/protocol"
)

type completionOptions map[string]protocol.CompletionItem

func (b *BufLsp) findRefCompletions(ref *symbolRef, options completionOptions) {
	prefix := ref.refName
	if len(prefix) > 0 {
		// Ignore the last part of the prefix, as it is being completed.
		prefix = prefix[:len(prefix)-1]
	}
	b.findRefPrefixCompletions(ref, prefix, options)
}

func (b *BufLsp) findRefPrefixCompletions(ref *symbolRef, prefix symbolRefName, options completionOptions) {
	b.findCompletions(ref.entry, prefix, ref.scope, options, ref.isField)
}

func (b *BufLsp) findCompletions(entry *fileEntry, name symbolRefName, scope symbolName, options completionOptions, isField bool) {
	candidates := findCandidates(name, scope)
	for _, candidate := range candidates {
		entry.findCompletions(candidate, options, isField)
		for _, importEntry := range entry.imports {
			b.findCompletionsFromImport(importEntry, candidate, options, isField)
		}
	}
}

func (b *BufLsp) findCompletionsFromImport(importEntry *importEntry, candidate symbolName, options completionOptions, isField bool) {
	if importEntry.docURI != "" {
		if importFile, ok := b.fileCache[importEntry.docURI.Filename()]; ok {
			importFile.findCompletions(candidate, options, isField)
			for _, subImportEntry := range importFile.imports {
				if subImportEntry.isPublic {
					b.findCompletionsFromImport(subImportEntry, candidate, options, isField)
				}
			}
		}
	}
}

func (b *BufLsp) findImportCompletionsAt(ctx context.Context, entry *fileEntry, pos ast.SourcePos) (completionOptions, bool, error) {
	importEntry := entry.findImportEntry(pos)
	if importEntry == nil {
		return nil, false, nil
	}

	// Extract the include prefix string.
	if !entry.containsPos(importEntry.node.Name, pos) {
		return nil, true, nil
	}
	colOffset := pos.Col - entry.fileNode.ItemInfo(importEntry.node.Name.Start().AsItem()).Start().Col - 1
	if colOffset < 0 {
		return nil, true, nil
	}
	prefix := importEntry.node.Name.AsString()[:colOffset]
	result, err := b.findImportCompletions(ctx, entry, prefix)
	return result, true, err
}

func (b *BufLsp) findImportCompletions(ctx context.Context, entry *fileEntry, prefix string) (completionOptions, error) {
	endPos := strings.LastIndex(prefix, "/")
	if endPos == -1 {
		prefix = ""
	} else {
		prefix = prefix[:endPos+1]
	}

	// Look for well known imports
	wktFiles, err := wktFiles.ReadDir("wkt")
	if err != nil {
		return nil, err
	}

	options := make(completionOptions)
	for _, wktFile := range wktFiles {
		path := wktSourceDir + strings.TrimPrefix(wktFile.Name(), "wkt/")
		if strings.HasPrefix(path, prefix) {
			item := makeIncludeCompletion(strings.TrimPrefix(path, prefix))
			options[item.Label] = item
		}
	}

	if entry.module != nil {
		if err := b.findModuleFileCompletions(ctx, entry.module, prefix, options); err != nil {
			return nil, err
		}
		for _, dep := range entry.module.DependencyModulePins() {
			depMod, err := b.moduleReader.GetModule(ctx, dep)
			if err != nil {
				return nil, err
			}
			if err := b.findModuleFileCompletions(ctx, depMod, prefix, options); err != nil {
				return nil, err
			}
		}
	}

	return options, nil
}

func (b *BufLsp) findModuleFileCompletions(ctx context.Context, module bufmodule.Module, prefix string, options completionOptions) error {
	infos, err := module.SourceFileInfos(ctx)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if strings.HasPrefix(info.Path(), prefix) {
			relPath := strings.TrimPrefix(info.Path(), prefix)
			item := makeIncludeCompletion(relPath)
			options[item.Label] = item
		}
	}
	return nil
}

func (b *BufLsp) findPrefixCompletions(ctx context.Context, entry *fileEntry, scope symbolName, prefixString string) completionOptions {
	options := make(completionOptions)
	inOption := strings.HasPrefix(prefixString, "[")

	if !inOption {
		prefixString = strings.TrimPrefix(prefixString, "(")
		refName := strings.Split(prefixString, ".")
		b.findRefCompletions(&symbolRef{
			entry:   entry,
			refName: refName,
			scope:   scope,
			isField: false,
		}, options)
		return options
	}

	prefixString = prefixString[1:]
	prefix := strings.Split(prefixString, ".")
	inExt := false
	switch len(prefix) {
	case 0:
	case 1:
		inExt = strings.HasPrefix(prefix[0], "(")
		prefix = nil
	default:
		prefix = prefix[:len(prefix)-1]
	}

	// Parse any extension parts
	var ext []string
	for i := 0; i < len(prefix); {
		part := prefix[i]
		switch {
		case strings.HasPrefix(part, "("):
			ext = []string{part[1:]}
			inExt = true
			prefix = prefix[i+1:]
			i = 0
		case strings.HasSuffix(part, ")"):
			ext = append(ext, part[:len(part)-1])
			inExt = false
			prefix = prefix[i+1:]
			i = 0
		case inExt:
			ext = append(ext, part)
			i++
		default:
			i++
		}
	}

	// Find the completions
	switch {
	case inExt:
		// Auto complete the extension name
		ref := &symbolRef{
			entry:   entry,
			refName: ext,
			scope:   entry.pkg,
			isField: true,
		}
		b.findRefPrefixCompletions(ref, ext, options)
	case len(ext) > 0:
		// Auto complete the extension field
		ref := &symbolRef{
			entry:   entry,
			refName: ext,
			scope:   entry.pkg,
			isField: true,
		}
		ref = b.resolveFieldType(ref)
		for _, part := range prefix {
			if ref == nil {
				break
			}
			ref.refName = append(ref.refName, part)
			ref = b.resolveFieldType(ref)
		}
		if ref != nil {
			ref.isField = true
			ref.scope = entry.pkg
			b.findRefPrefixCompletions(ref, ref.refName, options)
		}
	default:
		for _, wko := range wellKnownOptions {
			ref := b.resolveWellKnownExtendee(ctx, wko)
			for _, part := range prefix {
				ref.refName = append(ref.refName, part)
				ref.isField = true
				ref = b.resolveFieldType(ref)
			}
			ref.isField = true
			b.findRefPrefixCompletions(ref, ref.refName, options)
		}
	}
	return options
}

func makeIncludeCompletion(relPath string) protocol.CompletionItem {
	endPos := strings.Index(relPath, "/")
	if endPos == -1 {
		return protocol.CompletionItem{
			Label: relPath,
			Kind:  protocol.CompletionItemKindFile,
		}
	}

	label := relPath[:endPos]
	return protocol.CompletionItem{
		Label: label,
		Kind:  protocol.CompletionItemKindFolder,
	}
}
