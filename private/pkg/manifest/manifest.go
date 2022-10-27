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

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/storage"
	"golang.org/x/crypto/sha3"
)

const (
	shake256Name   = "shake256"
	shake256Length = 64
)

var (
	errNoFinalNewline    = errors.New("partial record: missing newline")
	hashKindToDigestType = map[modulev1alpha1.HashKind]string{
		modulev1alpha1.HashKind_HASH_KIND_SHAKE256: shake256Name,
	}
)

// Digest represents a hash function's value.
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

func NewDigestFromBlobHash(hash *modulev1alpha1.Hash) (*Digest, error) {
	if hash == nil {
		return nil, fmt.Errorf("nil hash")
	}
	dType, ok := hashKindToDigestType[hash.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported hash kind: %s", hash.Kind.String())
	}
	return NewDigestFromBytes(dType, hash.Digest), nil
}

// String returns the hash in a manifest's string format: "<type>:<hex>"
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

// Valid checks if the current digest matches a passed content, and any error
// attempting to validate.
func (d *Digest) Valid(content io.Reader) (bool, error) {
	if d.Type() != shake256Name {
		return false, fmt.Errorf("unsupported hash: %s", d.Type())
	}
	digest, err := shake256DigestFrom(sha3.NewShake256(), content)
	if err != nil {
		return false, err
	}
	return d.Hex() == hex.EncodeToString(digest), nil
}

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

// Manifest represents a list of pathToDigest and their digests.
type Manifest struct {
	pathToDigest  map[string]Digest
	digestToPaths map[string][]string
	hash          sha3.ShakeHash
}

var _ encoding.TextMarshaler = (*Manifest)(nil)
var _ encoding.TextUnmarshaler = (*Manifest)(nil)

// New creates an empty manifest.
func New() *Manifest {
	return &Manifest{
		pathToDigest:  make(map[string]Digest),
		digestToPaths: make(map[string][]string),
		hash:          sha3.NewShake256(),
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
		if err := m.addDigest(path, digest); err != nil {
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

// NewFromBucket creates a manifest from a storage bucket.
func NewFromBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
) (*Manifest, error) {
	m := New()
	err := bucket.Walk(ctx, "", func(info storage.ObjectInfo) error {
		path := info.Path()
		obj, err := bucket.Get(ctx, path)
		if err != nil {
			return err
		}
		if err := m.AddContent(path, obj); err != nil {
			return err
		}
		return obj.Close()
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manifest) addDigest(path string, digest *Digest) error {
	if digest.Type() != shake256Name {
		return fmt.Errorf("unsupported hash: %s", digest.Type())
	}
	if n := len(digest.Bytes()); n != shake256Length {
		return fmt.Errorf("invalid digest: got %d bytes, expected %d bytes", n, shake256Length)
	}
	m.pathToDigest[path] = *digest
	key := digest.String()
	m.digestToPaths[key] = append(m.digestToPaths[key], path)
	return nil
}

// AddContent adds a manifest entry for path by its content. Returned errors
// are errors from reading content, except io.EOF is swallowed.
func (m *Manifest) AddContent(path string, content io.Reader) error {
	m.hash.Reset()
	digest, err := shake256DigestFrom(m.hash, content)
	if err != nil {
		return err
	}
	return m.addDigest(path, NewDigestFromBytes(shake256Name, digest))
}

// Paths returns a sorted list of all paths in the manifest.
func (m *Manifest) Paths() []string {
	paths := make([]string, 0, len(m.pathToDigest))
	for path := range m.pathToDigest {
		paths = append(paths, path)
	}
	sort.Strings(paths)
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
	sortedPaths := m.Paths()
	for _, path := range sortedPaths {
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
	m.hash = newm.hash
	return nil
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

func shake256DigestFrom(hash sha3.ShakeHash, content io.Reader) ([]byte, error) {
	if _, err := io.Copy(hash, content); err != nil {
		return nil, err
	}
	digest := make([]byte, shake256Length)
	if _, err := hash.Read(digest); err != nil {
		// sha3.ShakeHash never errors or short reads. Something horribly wrong
		// happened if your computer ended up here.
		return nil, err
	}
	return digest, nil
}
