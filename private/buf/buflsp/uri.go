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
	"net/url"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// FilePathToURI converts a file path to a properly encoded URI.
func FilePathToURI(path string) protocol.URI {
	return normalizeURI(uri.File(path))
}

// normalizeURI encodes a URI to match VS Code's microsoft/vscode-uri behavior.
//
// Go's net/url follows RFC 3986 and permits '@' and ':' unencoded in path
// components (valid pchar); vscode-uri always encodes them. vscode-uri also
// lowercases Windows drive letters. When URIs differ, LSP operations like
// go-to-definition silently fail because the client and server URIs don't match.
func normalizeURI(u protocol.URI) protocol.URI {
	str := string(u)

	after, found := strings.CutPrefix(str, "file:///")
	if !found {
		// Non-file URIs: only encode @.
		return protocol.URI(strings.ReplaceAll(str, "@", "%40"))
	}

	segments := strings.Split(after, "/")
	for i, segment := range segments {
		// Decode first to avoid double-encoding already-normalized URIs.
		// PathUnescape only fails on malformed sequences (e.g. %2G); falling
		// back to the raw segment is the best we can do.
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			decoded = segment
		}
		// PathEscape encodes spaces as %20 (not +) and most special chars,
		// but permits '@' and ':' as RFC 3986 pchar. Encode those manually.
		encoded := url.PathEscape(decoded)
		encoded = strings.ReplaceAll(encoded, "@", "%40")
		encoded = strings.ReplaceAll(encoded, ":", "%3A")
		segments[i] = encoded
	}

	// vscode-uri lowercases Windows drive letters: C%3A → c%3A.
	// 'A'+32 == 'a' by ASCII identity; segments[0] is e.g. "C%3A" (4 bytes).
	if len(segments[0]) == 4 &&
		segments[0][0] >= 'A' && segments[0][0] <= 'Z' &&
		segments[0][1:] == "%3A" {
		segments[0] = string(segments[0][0]+32) + "%3A"
	}

	return protocol.URI("file:///" + strings.Join(segments, "/"))
}
