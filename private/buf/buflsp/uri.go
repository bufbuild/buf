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
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// normalizeURI ensures that URIs are properly percent-encoded for LSP compatibility.
//
// The go.lsp.dev/uri package (which uses Go's net/url) follows RFC 3986 strictly and
// allows '@' unencoded in path components. However, VS Code's LSP client uses the
// microsoft/vscode-uri package which encodes '@' as '%40' everywhere to avoid ambiguity
// with the authority component separator (user@host).
//
// Additionally, on Windows, the package also encodes ':' as '%3A' in drive letter paths
// (e.g., 'file:///d:/path' becomes 'file:///d%3A/path').
//
// When URIs don't match exactly, LSP operations like go-to-definition fail because
// the client's URI (with %40) doesn't match the server's URI (with @).
func normalizeURI(u protocol.URI) protocol.URI {
	normalized := strings.ReplaceAll(string(u), "@", "%40")

	if after, found := strings.CutPrefix(normalized, "file:///"); found {
		normalized = "file:///" + strings.ReplaceAll(after, ":", "%3A")
	}

	return protocol.URI(normalized)
}

// filePathToURI converts a file path to a properly encoded URI.
func filePathToURI(path string) protocol.URI {
	return normalizeURI(uri.File(path))
}
