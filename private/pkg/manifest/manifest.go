// Copyright 2020-2022 Buf Technologies, Inc.
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

// Manifests are a list of paths and their hash digests, canonically ordered by
// path in increasing lexographical order. Manifests are encoded as:
//
//	<hash type>:<digest>[SP][SP]<path>[LF]
//
// "shake256" is the only supported hash type. The digest is 64 bytes of hex
// encoded output of SHAKE256. See golang.org/x/crypto/sha3 and FIPS 202 for
// details on the SHAKE hash.
package manifest

import (
	"bufio"
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

var errNoFinalNewline = errors.New("partial record: missing newline")

// Error occurs when a manifest is malformed.
type Error struct {
	lineno  int
	msg     string
	wrapped error
}

var _ error = (*Error)(nil)

func newError(lineno int, msg string) *Error {
	return &Error{
		lineno: lineno,
		msg:    msg,
	}
}
func newErrorWrapped(lineno int, err error) *Error {
	return &Error{
		lineno:  lineno,
		msg:     err.Error(),
		wrapped: err,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("invalid manifest: %d: %s", e.lineno, e.msg)
}

func (e *Error) Unwrap() error {
	return e.wrapped
}

// Manifest represents a list of paths and their digests.
type Manifest struct {
	pathToDigest  map[string]Digest
	digestToPaths map[string][]string
}

var _ encoding.TextMarshaler = (*Manifest)(nil)
var _ encoding.TextUnmarshaler = (*Manifest)(nil)

// New creates an empty manifest.
func New() *Manifest {
	return &Manifest{
		pathToDigest:  make(map[string]Digest),
		digestToPaths: make(map[string][]string),
	}
}

// NewFromReader builds a manifest from an encoded manifest reader.
func NewFromReader(manifest io.Reader) (*Manifest, error) {
	m := New()
	scanner := bufio.NewScanner(manifest)
	scanner.Split(splitManifest)
	lineno := 0
	for scanner.Scan() {
		lineno++
		hash, path, found := strings.Cut(scanner.Text(), "  ")
		if !found {
			return nil, newError(lineno, "invalid entry")
		}
		digest, err := NewDigestFromString(hash)
		if err != nil {
			return nil, newErrorWrapped(lineno, err)
		}
		if err := m.AddEntry(path, *digest); err != nil {
			return nil, newErrorWrapped(lineno, err)
		}
	}
	err := scanner.Err()
	if err == errNoFinalNewline {
		return nil, newError(lineno, "partial record")
	}
	if err != nil {
		return nil, err
	}

	return m, nil
}

// AddEntry adds an entry to the manifest with a path and its digest. It fails
// if the path already exists in the manifest with a different digest.
func (m *Manifest) AddEntry(path string, digest Digest) error {
	if path == "" {
		return errors.New("empty path")
	}
	if digest.Type() == "" || digest.Hex() == "" {
		return errors.New("invalid digest")
	}
	if existingDigest, exists := m.pathToDigest[path]; exists {
		if existingDigest.Equal(digest) {
			return nil // same entry already in the manifest, nothing to do
		}
		return fmt.Errorf(
			"cannot add digest %q for path %q (already associated to digest %q)",
			digest.String(), path, existingDigest.String(),
		)
	}
	m.pathToDigest[path] = digest
	key := digest.String()
	m.digestToPaths[key] = append(m.digestToPaths[key], path)
	return nil
}

// Paths returns all paths in the manifest.
func (m *Manifest) Paths() []string {
	paths := make([]string, 0, len(m.pathToDigest))
	for path := range m.pathToDigest {
		paths = append(paths, path)
	}
	return paths
}

// PathsFor returns one or more matching path for a given digest. The digest is
// expected to be a lower-case hex encoded value. Returned paths are unordered.
// Paths is nil and ok is false if no paths are found.
func (m *Manifest) PathsFor(digest string) ([]string, bool) {
	paths, ok := m.digestToPaths[digest]
	if !ok {
		return nil, false
	}
	return paths, true
}

// DigestFor returns the matching digest for the given path. The path must be an
// exact match. Digest is nil and ok is false if no digest is found.
func (m *Manifest) DigestFor(path string) (*Digest, bool) {
	digest, ok := m.pathToDigest[path]
	if !ok {
		return nil, false
	}
	return &digest, true
}

// MarshalText encodes the manifest into its canonical form.
func (m *Manifest) MarshalText() ([]byte, error) {
	var coded bytes.Buffer
	paths := m.Paths()
	sort.Strings(paths)
	for _, path := range paths {
		digest := m.pathToDigest[path]
		fmt.Fprintf(&coded, "%s  %s\n", &digest, path)
	}
	return coded.Bytes(), nil
}

// UnmarshalText decodes a manifest from member.
//
// Use NewManifestFromReader if you have an io.Reader and want to avoid memory
// copying.
func (m *Manifest) UnmarshalText(text []byte) error {
	newm, err := NewFromReader(bytes.NewReader(text))
	if err != nil {
		return err
	}
	m.pathToDigest = newm.pathToDigest
	m.digestToPaths = newm.digestToPaths
	return nil
}

// Blob returns the manifest's blob.
func (m *Manifest) Blob() (Blob, error) {
	manifestText, err := m.MarshalText()
	if err != nil {
		return nil, err
	}
	return NewMemoryBlobFromReader(bytes.NewReader(manifestText))
}

func splitManifest(data []byte, atEOF bool) (int, []byte, error) {
	// Return a line without LF.
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i], nil
	}

	// EOF occurred with a partial line.
	if atEOF && len(data) != 0 {
		return 0, nil, errNoFinalNewline
	}

	return 0, nil, nil
}
