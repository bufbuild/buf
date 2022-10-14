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
	"context"
	"encoding"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
	"golang.org/x/crypto/sha3"
)

var errPartial error = errors.New("partial record")

const shake256Name = "shake256"

type Digest struct {
	dtype  string
	digest []byte
	hexstr string
}

func NewDigestFromBytes(dtype string, digest []byte) *Digest {
	return &Digest{
		dtype:  dtype,
		digest: digest,
		hexstr: hex.EncodeToString(digest),
	}
}

func NewDigestFromHex(dtype string, hexstr string) (*Digest, error) {
	digest, err := hex.DecodeString(hexstr)
	if err != nil {
		return nil, err
	}
	return NewDigestFromBytes(dtype, digest), nil
}

func NewDigestFromString(typedDigest string) (*Digest, error) {
	hashfunc, digestStr, found := strings.Cut(typedDigest, ":")
	if !found {
		return nil, errors.New("malformed digest string")
	}
	return NewDigestFromHex(hashfunc, digestStr)
}

func (d *Digest) String() string {
	return d.Type() + ":" + d.Hex()
}

func (d *Digest) Type() string {
	return d.dtype
}

func (d *Digest) Bytes() []byte {
	return d.digest
}

func (d *Digest) Hex() string {
	return d.hexstr
}

// ManifestError occurs when a manifest is malformed.
type ManifestError struct {
	lineno int
	msg    string
}

var _ error = (*ManifestError)(nil)

func newManifestError(lineno int, msg string) *ManifestError {
	return &ManifestError{
		lineno: lineno,
		msg:    msg,
	}
}

func (e *ManifestError) Error() string {
	return fmt.Sprintf("invalid manifest: %d: %s", e.lineno, e.msg)
}

// Manifest represents a list of paths and their digests.
type Manifest struct {
	paths   map[string]*Digest
	digests map[string]string
	hash    sha3.ShakeHash
}

var _ encoding.TextMarshaler = (*Manifest)(nil)
var _ encoding.TextUnmarshaler = (*Manifest)(nil)

// NewManifest creates an empty manifest.
func NewManifest() *Manifest {
	return &Manifest{
		paths:   make(map[string]*Digest),
		digests: make(map[string]string),
		hash:    sha3.NewShake256(),
	}
}

func splitManifest(data []byte, atEOF bool) (int, []byte, error) {
	// Return a line without LF.
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i], nil
	}

	// EOF occurred with a partial line.
	if atEOF && len(data) != 0 {
		return 0, nil, errPartial
	}

	return 0, nil, nil
}

// NewManifestFromReader builds a manifest an encoded manifest reader.
func NewManifestFromReader(manifest io.Reader) (*Manifest, error) {
	m := NewManifest()
	scanner := bufio.NewScanner(manifest)
	scanner.Split(splitManifest)
	lineno := 0
	for scanner.Scan() {
		lineno++
		hash, path, found := strings.Cut(scanner.Text(), "  ")
		if !found {
			return nil, newManifestError(lineno, "invalid entry")
		}
		digest, err := NewDigestFromString(hash)
		if err != nil {
			return nil, multierr.Append(
				newManifestError(lineno, "invalid hash"),
				err,
			)
		}
		if digest.Type() != shake256Name {
			return nil, newManifestError(lineno, "unknown hash")
		}
		if len(digest.Bytes()) != 64 {
			return nil, newManifestError(lineno, "short digest")
		}
		m.addDigest(path, digest)
	}
	err := scanner.Err()
	if err == errPartial {
		return nil, newManifestError(lineno, "partial record")
	}
	if err != nil {
		return nil, err
	}

	return m, nil
}

// NewManifestFromBucket creates a manifest from a bucket.
func NewManifestFromBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
) (*Manifest, error) {
	m := NewManifest()
	err := bucket.Walk(ctx, "", func(info storage.ObjectInfo) error {
		path := info.Path()
		obj, err := bucket.Get(ctx, path)
		if err != nil {
			return err
		}
		m.AddContent(path, obj)
		return obj.Close()
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manifest) addDigest(path string, digest *Digest) {
	m.paths[path] = digest
	m.digests[digest.String()] = path
}

// AddContent adds a manifest entry for path by its content.
func (m *Manifest) AddContent(path string, content io.Reader) {
	m.hash.Reset()
	// sha3.ShakeHash never errors, reading or writing. These checks are to
	// satisfy linting, possibly exploding if some breaking behavior change
	// happens.
	if _, err := io.Copy(m.hash, content); err != nil {
		panic(err)
	}
	digest := make([]byte, 64)
	if _, err := m.hash.Read(digest); err != nil {
		panic(err)
	}
	m.addDigest(path, NewDigestFromBytes(shake256Name, digest))
}

// GetPath returns the matching path for the given digest. The digest is
// expected to be a lower-case hex encoded value. path is the empty string and
// ok is false if no path is found.
func (m *Manifest) GetPath(digest *Digest) (path string, ok bool) {
	path, ok = m.digests[digest.String()]
	return path, ok
}

// GetDigest returns the matching digest for the given path. The path must be
// an exact match. The returned digest is a lower-case hex encoded value.
// digest is the empty string and ok is false if no digest is found.
func (m *Manifest) GetDigest(path string) (digest *Digest, ok bool) {
	digest, ok = m.paths[path]
	return digest, ok
}

// MarshalText encodes the manifest into its canonical form.
func (m *Manifest) MarshalText() ([]byte, error) {
	// order by paths
	paths := make([]string, 0, len(m.paths))
	for path := range m.paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var coded bytes.Buffer
	for _, path := range paths {
		fmt.Fprintf(&coded, "%s  %s\n", m.paths[path], path)
	}
	return coded.Bytes(), nil
}

// UnmarshalText decodes a manifest from member.
//
// Use NewManifestFromReader if you have an io.Reader and want to avoid memory
// copying.
func (m *Manifest) UnmarshalText(text []byte) error {
	newm, err := NewManifestFromReader(bytes.NewReader(text))
	if err != nil {
		return err
	}
	m.paths = newm.paths
	m.digests = newm.digests
	m.hash = newm.hash
	return nil
}
