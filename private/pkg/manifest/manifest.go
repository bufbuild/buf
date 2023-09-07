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

// Package manifest defines generic content-addressable storage APIs which are
// used to store and verify Buf modules. The primary data types relevant to
// consumers of this package are [Manifest], [Blob], [BlobSet], and [Digest].
//
// [Manifest] can read and write manifest files. The canonical form of a
// manifest is produced when serialized with [Manifest.MarshalText].
// Interacting with a manifest is typically by path ([Manifest.Paths],
// [Manifest.DigestFor]) or by a [Digest] ([Manifest.PathsFor]). Note that it
// is possible for multiple paths in the manifest to have to same digest (and
// content), however a given path only has one digest.
//
// [Blob] represents file content and its digest. [NewMemoryBlob] provides an
// in-memory implementation of a Blob. A manifest, being a file, is also a blob.
// The [Manifest.Blob] function converts a manifest to a Blob.
//
// [BlobSet] collects related blobs together into a unique set (de-duplicating
// blobs with the same digest and content).
//
// Blobs are anonymous files and a manifest gives names to anonymous files.
// It's possible to view a manifest and its associated blobs as a file system.
// [NewFromBucket] creates a manifest and blob set from a storage bucket.
//
// Aside from the [Manifest.MarshalText] encoding, this package does not define
// the representation of its data types on the wire. See the
// private/bufpkg/bufmanifest package and
// buf/alpha/registry/v1alpha1/module.proto for details on how these data types
// are represented in Protobuf.
package manifest

import (
	"bufio"
	"bytes"
	"context"
	"encoding"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
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
	// needed for ordering
	paths         []string
	pathToDigest  map[string]Digest
	digestToPaths map[string][]string
}

var _ encoding.TextMarshaler = (*Manifest)(nil)
var _ encoding.TextUnmarshaler = (*Manifest)(nil)

// NewFromReader builds a manifest from an encoded manifest, like one produced
// by [Manifest.MarshalText].
func NewFromReader(manifest io.Reader) (*Manifest, error) {
	var m Manifest
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
		if errors.Is(err, errNoFinalNewline) {
			return nil, newError(lineno, "partial record")
		}
		return nil, err
	}
	return &m, nil
}

// NewFromBucket creates a manifest and blob set from the bucket's files. Blobs
// in the blob set use the [DigestTypeShake256] digest.
func NewFromBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
) (*Manifest, *BlobSet, error) {
	var m Manifest
	digester, err := NewDigester(DigestTypeShake256)
	if err != nil {
		return nil, nil, err
	}
	var blobs []Blob
	if walkErr := bucket.Walk(ctx, "", func(info storage.ObjectInfo) (retErr error) {
		path := info.Path()
		obj, err := bucket.Get(ctx, path)
		if err != nil {
			return err
		}
		defer func() { retErr = multierr.Append(retErr, obj.Close()) }()
		blob, err := NewMemoryBlobFromReaderWithDigester(obj, digester)
		if err != nil {
			return err
		}
		blobs = append(blobs, blob)
		return m.AddEntry(path, *blob.Digest())
	}); walkErr != nil {
		return nil, nil, walkErr
	}
	blobSet, err := NewBlobSet(ctx, blobs) // no need to pass validation options, we're building and digesting the blobs
	if err != nil {
		return nil, nil, err
	}
	return &m, blobSet, nil
}

// AddEntry adds an entry to the manifest with a path and its digest. It fails
// if the path already exists in the manifest with a different digest.
func (m *Manifest) AddEntry(path string, digest Digest) error {
	if path == "" {
		return errors.New("empty path")
	}
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
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
	// Already guaranteed that the path is not in the slice due to above check
	m.paths = append(m.paths, path)
	if m.pathToDigest == nil {
		m.pathToDigest = make(map[string]Digest)
	}
	m.pathToDigest[path] = digest
	key := digest.String()
	if m.digestToPaths == nil {
		m.digestToPaths = make(map[string][]string)
	}
	m.digestToPaths[key] = append(m.digestToPaths[key], path)
	return nil
}

// Paths returns all unique paths in the manifest by insertion order.
func (m *Manifest) Paths() []string {
	pathsCopy := make([]string, len(m.paths))
	copy(pathsCopy, m.paths)
	return pathsCopy
}

// Digests returns all unique digests in the manifest, in insertion order.
func (m *Manifest) Digests() []Digest {
	digests := make([]Digest, 0, len(m.digestToPaths))
	addedDigests := make(map[string]struct{}, len(m.digestToPaths))
	// Iterating over paths to guarantee ordering.
	for _, path := range m.paths {
		digest, ok := m.pathToDigest[path]
		if !ok {
			// This should be an error in the style of the rest of the codebase but
			// this was refactored and we didn't want to change the function signature.
			panic(fmt.Sprintf("path %q not present in pathToDigest", path))
		}
		if _, alreadyAdded := addedDigests[digest.String()]; alreadyAdded {
			continue
		}
		addedDigests[digest.String()] = struct{}{}
		digests = append(digests, digest)
	}
	return digests
}

// Range invokes a function for all the paths in the manifest, passing the path and its digest.
// Paths are invoked by insertion order.
// This func will stop iterating if an error is returned.
func (m *Manifest) Range(f func(path string, digest Digest) error) error {
	// Iterating over paths to guarantee ordering.
	for _, path := range m.paths {
		digest, ok := m.pathToDigest[path]
		if !ok {
			// This should be an error in the style of the rest of the codebase but
			// this was refactored and we didn't want to change the function signature.
			panic(fmt.Sprintf("path %q not present in pathToDigest", path))
		}
		if err := f(path, digest); err != nil {
			return err
		}
	}
	return nil
}

// PathsFor returns one or more matching path for a given digest. The digest is
// expected to be a lower-case hex encoded value. Returned paths are ordered by insertion time.
// Returns (nil, false) if no paths are found.
func (m *Manifest) PathsFor(digest string) ([]string, bool) {
	paths, ok := m.digestToPaths[digest]
	if !ok || len(paths) == 0 {
		return nil, false
	}
	return paths, true
}

// DigestFor returns the matching digest for the given path. The path must be an
// exact match. Returns (nil, false) if no digest is found.
func (m *Manifest) DigestFor(path string) (*Digest, bool) {
	digest, ok := m.pathToDigest[path]
	if !ok {
		return nil, false
	}
	return &digest, true
}

// MarshalText encodes the manifest into its canonical form, consisting of
// an ordered list of paths and their hash digests. Manifests are encoded as:
//
//	<digest_type>:<digest>[SP][SP]<path>[LF]
//
// The only supported digest_type is shake256. The digest is 64 bytes of hex
// encoded output of SHAKE256. See golang.org/x/crypto/sha3 and FIPS 202 for
// details on the SHAKE hash.
//
// An example encoded manifest for the acme/petapis module is:
//
//	shake256:cd22db48cf7c274bbffcb5494a854000cd21b074df7c6edabbd0102c4be8d7623e3931560fcda7acfab286ae1d4f506911daa31f223ee159f59ffce0c7acbbaa  buf.lock
//	shake256:3b353aa5aacd11015e8577f16e2c4e7a242ce773d8e3a16806795bb94f76e601b0db9bf42d5e1907fda63303e1fa1c65f1c175ecc025a3ef29c3456ad237ad84  buf.md
//	shake256:7c88a20cf931702d042a4ddee3fde5de84814544411f1c62dbf435b1b81a12a8866a070baabcf8b5a0d31675af361ccb2d93ddada4cdcc11bab7ea3d8d7c4667  buf.yaml
//	shake256:9db25155eafd19b36882cff129daac575baa67ee44d1cb1fd3894342b28c72b83eb21aa595b806e9cb5344759bc8308200c5af98e4329aa83014dde99afa903a  pet/v1/pet.proto
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

// UnmarshalText decodes a manifest from text. See [Manifest.MarshalText] for
// the expected encoding of a manifest.
//
// See [NewFromReader] if your manifest is available in an io.Reader.
func (m *Manifest) UnmarshalText(text []byte) error {
	newm, err := NewFromReader(bytes.NewReader(text))
	if err != nil {
		return err
	}
	m.paths = newm.paths
	m.pathToDigest = newm.pathToDigest
	m.digestToPaths = newm.digestToPaths
	return nil
}

// Blob returns the manifest as a blob. The Blob content is set to the
// canonical representation of a manifest ([Manifest.MarshalText]),
// and the digest is set to the SHAKE256 digest of the content.
func (m *Manifest) Blob() (Blob, error) {
	manifestText, err := m.MarshalText()
	if err != nil {
		return nil, err
	}
	return NewMemoryBlobFromReader(bytes.NewReader(manifestText))
}

// Empty returns true if the manifest has no entries.
func (m *Manifest) Empty() bool {
	return len(m.paths) == 0 && len(m.pathToDigest) == 0 && len(m.digestToPaths) == 0
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
