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
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/protocompile/experimental/ast"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/token"
	"go.lsp.dev/protocol"
)

const (
	// bsrTabTypeDocs is the BSR docs tab type.
	bsrTabTypeDocs = "docs"
	// bsrTabTypeFile is the BSR file (code) tab type.
	bsrTabTypeFile = "file"
)

// documentLink generates document links for imports and URLs in comments.
// For imports from BSR modules, this creates links to <remote>/<owner>/<module>/file/main:<file-path>.
// For WKT imports, this creates links to buf.build/protocolbuffers/wellknowntypes/file/main:<file-path>.
// For local imports without module names, it links to the local file.
// For https:// URLs found in comments, it creates clickable links to those URLs.
func (s *server) documentLink(file *file) []protocol.DocumentLink {
	var links []protocol.DocumentLink

	// Create links for import statements
	for _, symbol := range file.symbols {
		if imported, ok := symbol.kind.(*imported); ok && imported.file != nil {
			var targetURI protocol.DocumentURI

			// Check if this is a Well-Known Type (WKT)
			if imported.file.IsWKT() && imported.file.objectInfo != nil {
				bsrHost := bufconnect.DefaultRemote + "/protocolbuffers/wellknowntypes"
				filePath := imported.file.objectInfo.Path()
				url := "https://" + bsrHost + "/file/main:" + filePath
				targetURI = protocol.DocumentURI(url)
			} else if file.workspace != nil && imported.file.objectInfo != nil {
				// Try to get BSR module information for non-WKT imports
				module := file.workspace.GetModule(imported.file.uri)
				filePath := imported.file.objectInfo.Path()
				url := bsrURL(module, filePath, "", bsrTabTypeFile)
				if url != "" {
					targetURI = protocol.DocumentURI(url)
				}
			}

			// Fall back to local file link if no BSR module information
			if targetURI == "" {
				targetURI = imported.file.uri
			}

			links = append(links, protocol.DocumentLink{
				Range:  reportSpanToProtocolRange(symbol.span),
				Target: targetURI,
			})
		}
	}

	// Add links for URLs in comments
	if file.ir != nil {
		if astFile := file.ir.AST(); astFile != nil {
			links = append(links, s.findURLLinksInComments(astFile)...)
		}
	}

	return links
}

// bsrURL constructs a BSR URL for a module.
// Returns an empty string if the module has no FullName.
// tabType should be bsrTabTypeDocs or bsrTabTypeFile.
// For docs tab: https://remote/owner/name/docs/main:package#anchor
// For file tab: https://remote/owner/name/file/main:filePath
func bsrURL(module bufmodule.Module, pathOrPackage string, anchor string, tabType string) string {
	if module == nil {
		return ""
	}
	fullName := module.FullName()
	if fullName == nil {
		return ""
	}

	registry := fullName.Registry()
	owner := fullName.Owner()
	name := fullName.Name()

	// Default to buf.build if no remote or if it's the default remote
	if registry == "" {
		registry = bufconnect.DefaultRemote
	}

	url := "https://" + registry + "/" + owner + "/" + name + "/" + tabType + "/main:" + pathOrPackage
	if anchor != "" {
		url += "#" + anchor
	}
	return url
}

// findURLLinksInComments extracts document links for https:// URLs found in comments.
func (s *server) findURLLinksInComments(astFile *ast.File) []protocol.DocumentLink {
	var links []protocol.DocumentLink

	for tok := range astFile.Stream().All() {
		if tok.Kind() != token.Comment {
			continue
		}

		commentSpan := tok.Span()
		commentText := commentSpan.Text()

		// Find all https:// URLs in this comment using the regex
		matches := s.httpsURLRegex.FindAllStringIndex(commentText, -1)
		for _, match := range matches {
			urlStart := match[0]
			urlEnd := match[1]

			url := commentText[urlStart:urlEnd]
			urlSpan := source.Span{
				File:  commentSpan.File,
				Start: commentSpan.Start + urlStart,
				End:   commentSpan.Start + urlEnd,
			}

			links = append(links, protocol.DocumentLink{
				Range:  reportSpanToProtocolRange(urlSpan),
				Target: protocol.DocumentURI(url),
			})
		}
	}

	return links
}
