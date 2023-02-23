// Copyright 2020-2023 Buf Technologies, Inc.
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

// A manifest is a file containing a list of paths and their hash digests,
// canonically ordered by path in increasing lexicographical order. Manifests
// are encoded as:
//
//	<digest type>:<digest>[SP][SP]<path>[LF]
//
// "shake256" is the only supported digest type. The digest is 64 bytes of hex
// encoded output of SHAKE256. See golang.org/x/crypto/sha3 and FIPS 202 for
// details on the SHAKE hash.
//
// [Manifest] can read and write manifest files. Canonical form is produced
// when serialized ([Manifest.MarshalText]). Non-canonical form is a valid
// manifest and will not produce errors when deserializing.
//
// Interacting with a manifest is typically by path ([Manifest.Paths],
// [Manifest.DigestFor]) or by a [Digest] ([Manifest.PathsFor]).
//
// [Blob] represents file content and its digest. [BlobSet] collects related
// blobs together into a set. [NewMemoryBlob] provides an in-memory
// implementation. A manifest, being a file, is also a blob ([Manifest.Blob]).
//
// Blobs are anonymous files and a manifest gives names to anonymous files.
// It's possible to view a manifest and its associated blobs as a file system.
// [NewBucket] creates a storage bucket from a manifest and blob set.
// [NewFromBucket] does the inverse: the creation of a manifest and blob set
// from a storage bucket.
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

func newError(lineno int, msg string) error {
	return fmt.Errorf("invalid manifest: %d: %s", lineno, msg)
}

func newErrorWrapped(lineno int, err error) error {
	return fmt.Errorf("invalid manifest: %d: %w", lineno, err)
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

// NewFromReader builds a manifest from an encoded manifest, like one produced
// by [Manifest.MarshalText].
func NewFromReader(manifest io.Reader) (*Manifest, error) {
	m := New()
	scanner := bufio.NewScanner(manifest)
	scanner.Split(splitManifest)
	lineno := 0
	for scanner.Scan() {
		lineno++
		encodedDigest, path, found := strings.Cut(scanner.Text(), "  ")
		if !found {
			return nil, newError(lineno, "invalid entry")
		}
		digest, err := NewDigestFromString(encodedDigest)
		if err != nil {
			return nil, newErrorWrapped(lineno, err)
		}
		if err := m.AddEntry(path, *digest); err != nil {
			return nil, newErrorWrapped(lineno, err)
		}
	}
	if err := scanner.Err(); err != nil {
		if err == errNoFinalNewline {
			return nil, newError(lineno, "partial record")
		}
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
	if !ok || len(paths) == 0 {
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
		if _, err := fmt.Fprintf(&coded, "%s  %s\n", &digest, path); err != nil {
			return nil, err
		}
	}
	return coded.Bytes(), nil
}

// UnmarshalText decodes a manifest from text.
//
// See [NewFromReader] if your manifest is available in an io.Reader.
func (m *Manifest) UnmarshalText(text []byte) error {
	newm, err := NewFromReader(bytes.NewReader(text))
	if err != nil {
		return err
	}
	m.pathToDigest = newm.pathToDigest
	m.digestToPaths = newm.digestToPaths
	return nil
}

// Blob returns the manifest as a blob.
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
