// Copyright 2020-2026 Buf Technologies, Inc.
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

package lspprotocol

// This file declares URI, DocumentUri, and its methods.
//
// For the LSP definition of these types, see
// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#uri

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// A DocumentURI is the URI of a client editor document.
//
// According to the LSP specification:
//
//	Care should be taken to handle encoding in URIs. For
//	example, some clients (such as VS Code) may encode colons
//	in drive letters while others do not. The URIs below are
//	both valid, but clients and servers should be consistent
//	with the form they use themselves to ensure the other party
//	doesn’t interpret them as distinct URIs. Clients and
//	servers should not assume that each other are encoding the
//	same way (for example a client encoding colons in drive
//	letters cannot assume server responses will have encoded
//	colons). The same applies to casing of drive letters - one
//	party should not assume the other party will return paths
//	with drive letters cased the same as it.
//
//	file:///c:/project/readme.md
//	file:///C%3A/project/readme.md
//
// This is done during JSON unmarshalling;
// see [DocumentURI.UnmarshalText] for details.
type DocumentURI string

// URI is an alias for DocumentURI. Both URI and DocumentURI refer to the same
// underlying type so callers can use either interchangeably without explicit conversion.
type URI = DocumentURI

// UnmarshalText implements decoding of DocumentUri values.
//
// In particular, it implements a systematic correction of various odd
// features of the definition of DocumentUri in the LSP spec that
// appear to be workarounds for bugs in VS Code. For example, it may
// URI-encode the URI itself, so that colon becomes %3A, and it may
// send file://foo.go URIs that have two slashes (not three) and no
// hostname.
//
// We use UnmarshalText, not UnmarshalJSON, because it is called even
// for non-addressable values such as keys and values of map[K]V,
// where there is no pointer of type *K or *V on which to call
// UnmarshalJSON. (See Go issue #28189 for more detail.)
//
// Non-empty DocumentUris are valid "file"-scheme URIs.
// The empty DocumentUri is valid.
func (uri *DocumentURI) UnmarshalText(data []byte) (err error) {
	*uri, err = ParseDocumentURI(string(data))
	return err
}

// Path returns the file path for the given URI.
//
// DocumentUri("").Path() returns the empty string.
//
// Path panics if called on a URI that is not a valid filename.
func (uri DocumentURI) Path() (string, error) {
	filename, err := filename(uri)
	if err != nil {
		// e.g. ParseRequestURI failed.
		//
		// This can only affect DocumentUris created by
		// direct string manipulation; all DocumentUris
		// received from the client pass through
		// ParseRequestURI, which ensures validity.
		return "", fmt.Errorf("invalid URI %q: %w", uri, err)
	}
	return filepath.FromSlash(filename), nil
}

func filename(uri DocumentURI) (string, error) {
	if uri == "" {
		return "", nil
	}

	// This conservative check for the common case
	// of a simple non-empty absolute POSIX filename
	// avoids the allocation of a net.URL.
	if strings.HasPrefix(string(uri), "file:///") {
		rest := string(uri)[len("file://"):] // leave one slash
		for i := range len(rest) {
			b := rest[i]
			// Reject these cases:
			if b < ' ' || b == 0x7f || // control character
				b == '%' || b == '+' || // URI escape
				b == ':' || // Windows drive letter
				b == '@' || b == '&' || b == '?' { // authority or query
				goto slow
			}
		}
		return rest, nil
	}
slow:

	u, err := url.ParseRequestURI(string(uri))
	if err != nil {
		return "", fmt.Errorf("parsing URI %q: %w", uri, err)
	}
	if u.Scheme != fileScheme {
		return "", fmt.Errorf("only file URIs are supported, got %q from %q", u.Scheme, uri)
	}
	// If the URI is a Windows URI, we trim the leading "/" and uppercase
	// the drive letter, which will never be case sensitive.
	if isWindowsDrivePath(u.Path) {
		u.Path = strings.ToUpper(string(u.Path[1])) + u.Path[2:]
	}

	return u.Path, nil
}

// ParseDocumentURI interprets a string as a DocumentUri, applying VS
// Code workarounds; see [DocumentURI.UnmarshalText] for details.
func ParseDocumentURI(s string) (DocumentURI, error) {
	if s == "" {
		return "", nil
	}

	if !strings.HasPrefix(s, "file://") {
		// Non-file URIs (e.g. https://) are valid link targets in LSP; accept
		// them as-is without further validation or encoding.
		return DocumentURI(s), nil
	}

	// VS Code sends URLs with only two slashes,
	// which are invalid. golang/go#39789.
	if !strings.HasPrefix(s, "file:///") {
		s = "file:///" + s[len("file://"):]
	}

	// Even though the input is a URI, it may not be in canonical form. VS Code
	// in particular over-escapes :, @, etc. Unescape and re-encode to canonicalize.
	path, err := url.PathUnescape(s[len("file://"):])
	if err != nil {
		return "", fmt.Errorf("unescaping URI path %q: %w", s, err)
	}

	// File URIs from Windows may have lowercase drive letters.
	// Since drive letters are guaranteed to be case insensitive,
	// we change them to uppercase to remain consistent.
	// For example, file:///c:/x/y/z becomes file:///C:/x/y/z.
	if isWindowsDrivePath(path) {
		path = path[:1] + strings.ToUpper(string(path[1])) + path[2:]
	}

	// Encode each path segment so that characters like '@' that are technically
	// valid in RFC 3986 paths but are encoded by VS Code (vscode-uri) are
	// consistently represented as percent-encoded sequences.
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		enc := url.PathEscape(seg)
		// url.PathEscape does not encode '@'; encode it explicitly so that URIs
		// round-trip through clients (like VS Code) that always encode it.
		enc = strings.ReplaceAll(enc, "@", "%40")
		segments[i] = enc
	}
	return DocumentURI("file://" + strings.Join(segments, "/")), nil
}

// URIFromPath returns DocumentUri for the supplied file path.
// Given "", it returns "".
func URIFromPath(path string) DocumentURI {
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if isWindowsDrivePath(path) {
		path = "/" + strings.ToUpper(string(path[0])) + path[1:]
	}
	path = filepath.ToSlash(path)
	filepath.Clean(path)
	u := url.URL{
		Scheme: fileScheme,
		Path:   path,
	}
	return DocumentURI(u.String())
}

// Filename returns the filesystem path for the URI, panicking if the URI is
// not a valid file URI.
func (uri DocumentURI) Filename() string {
	path, err := uri.Path()
	if err != nil {
		panic(fmt.Sprintf("DocumentURI.Filename: %v", err))
	}
	return path
}

const fileScheme = "file"

// isWindowsDrivePath returns true if the file path is of the form used by
// Windows. We check if the path begins with a drive letter, followed by a ":".
// For example: C:/x/y/z.
func isWindowsDrivePath(path string) bool {
	return filepath.VolumeName(path) != ""
}
