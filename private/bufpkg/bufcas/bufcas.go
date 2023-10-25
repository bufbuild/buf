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

package bufcas

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/pkg/storage"
	"golang.org/x/crypto/sha3"
)

const (
	// DigestTypeShake256 represents the shake256 digest type.
	DigestTypeShake256 DigestType = iota + 1

	shake256Length = 64
)

var (
	digestTypeToString = map[DigestType]string{
		DigestTypeShake256: "shake256",
	}
	stringToDigestType = map[string]DigestType{
		"shake256": DigestTypeShake256,
	}
)

// DigestType is a type of digest.
type DigestType int

// String prints the string representation of the DigestType.
func (d DigestType) String() string {
	s, ok := digestTypeToString[d]
	if !ok {
		return strconv.Itoa(int(d))
	}
	return s
}

// ParseDigestType parses a DigestType from its string representation.
//
// Reverses DigestType.String().
func ParseDigestType(s string) (DigestType, error) {
	d, ok := stringToDigestType[s]
	if !ok {
		return 0, fmt.Errorf("unknown DigestType: %q", s)
	}
	return d, nil
}

// Digest is a digest of some content.
//
// It consists of a DigestType and a digest value.
type Digest interface {
	// String() prints typeString:hexValue.
	fmt.Stringer

	// Type returns the type of digest.
	Type() DigestType
	// Value returns the digest value.
	Value() []byte

	// Protect against creation of a Digest outside of this package, as we
	// do very careful validation.
	isDigest()
}

// NewDigest creates a new Digest for the given DigestType and digest value.
//
// If the length of value is 0, a nil Digest is returned.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func NewDigest(digestType DigestType, value []byte) (Digest, error) {
	return newDigest(digestType, value)
}

// NewDigestForContent creates a new Digest based on the given content read from the Reader.
//
// If there is no content, a nil Digest is returned.
//
// The Reader is read until io.EOF.
// Validation is performed to ensure that the DigestType is known.
func NewDigestForContent(digestType DigestType, reader io.Reader) (Digest, error) {
	switch digestType {
	case DigestTypeShake256:
		shakeHash := sha3.NewShake256()
		shakeHash.Reset()
		n, err := io.Copy(shakeHash, reader)
		if err != nil {
			return nil, err
		}
		// No content, return a nil Digest.
		if n == 0 {
			return nil, nil
		}
		value := make([]byte, shake256Length)
		if _, err := shakeHash.Read(value); err != nil {
			// sha3.ShakeHash never errors or short reads. Something horribly wrong
			// happened if your computer ended up here.
			return nil, err
		}
		return newDigest(digestType, value)
	default:
		return nil, fmt.Errorf("unknown DigestType: %v", digestType)
	}
}

// NewDigestForString returns a new Digest for the given Digest string.
//
// If the string is empty, a nil Digest is returned.
//
// This reverses Digest.String().
// A Digest string is of the form typeString:hexValue.
func NewDigestForString(s string) (Digest, error) {
	if s == "" {
		return nil, nil
	}
	digestTypeString, hexValue, ok := strings.Cut(s, ":")
	if !ok {
		return nil, fmt.Errorf("invalid Digest string: %q", s)
	}
	digestType, err := ParseDigestType(digestTypeString)
	if err != nil {
		return nil, err
	}
	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, err
	}
	return newDigest(digestType, value)
}

// DigestEqual returns true if the given Digests are considered equal.
//
// If both Digests are nil, this returns true.
//
// This check both the DigestType and Digest value.
func DigestEqual(a Digest, b Digest) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if a.Type() != b.Type() {
		return false
	}
	return bytes.Equal(a.Value(), b.Value())
}

// Blob is content with its associated Digest.
type Blob interface {
	// Digest returns the Digest of the Blob.
	//
	// Always non-nil.
	//
	// NewDigestForContent(blob.Digest.Type(), bytes.NewReader(blob.Content()) should
	// always match this value.
	Digest() Digest
	// Content returns the content of the Blob.
	//
	// Always non-empty.
	Content() []byte

	// Protect against creation of a Blob outside of this package, as we
	// do very careful validation.
	isBlob()
}

// NewBlobForContent returns a new Blob with a Digest of the given DigestType,
// and the content as read from the Reader.
//
// If the content is empty, returns nil.
//
// The reader is read until io.EOF.
// Validation is performed to ensure that the DigestType is known.
func NewBlobForContent(digestType DigestType, reader io.Reader) (Blob, error) {
	buffer := bytes.NewBuffer(nil)
	teeReader := io.TeeReader(reader, buffer)
	digest, err := NewDigestForContent(digestType, teeReader)
	if err != nil {
		return nil, err
	}
	// If the digest is nil, we have no content, return nil.
	if digest == nil {
		return nil, nil
	}
	return newBlob(digest, buffer.Bytes()), nil
}

// NewBlobForContentWithKnownDigest returns a new Blob for the given Digest and content
// as read from the Reader.
//
// If the content is empty, returns nil after verifying the known Digest is nil.
//
// The reader is read until io.EOF.
// Validation is performed to ensure that the Digest matches the computed Digest of the content.
func NewBlobForContentWithKnownDigest(knownDigest Digest, reader io.Reader) (Blob, error) {
	blob, err := NewBlobForContent(knownDigest.Type(), reader)
	if err != nil {
		return nil, err
	}
	if blob == nil {
		if knownDigest != nil {
			return nil, fmt.Errorf("Blob was empty but had a known non-empty Digest %v", knownDigest)
		}
		return nil, nil
	}
	if !DigestEqual(blob.Digest(), knownDigest) {
		return nil, fmt.Errorf("Digest %v did not match known Digest %v when creating a new Blob", blob.Digest(), knownDigest)
	}
	return blob, nil
}

// BlobEqual returns true if the given Blobs are considered equal.
//
// This checks both the Digest and the content.
//
// TODO: In the former version of this package, we compared content values as well.
// We should be able to remove this, and this is commented out for now. Technically we do not
// need to compare the contents, as we know that the Digest is a valid Digest for the
// given content via valiation we did at construction time.
func BlobEqual(a Blob, b Blob) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if a == nil {
		return true
	}
	if !DigestEqual(a.Digest(), b.Digest()) {
		return false
	}
	//aContent := a.Content()
	//bContent := b.Content()
	//for i := 0; i < len(aContent); i += 4096 {
	//j := i + 4096
	//if j > len(aContent) {
	//j = len(aContent)
	//}
	//if !bytes.Equal(aContent[i:j], bContent[i:j]) {
	//return false
	//}
	//}
	return true
}

// BlobSet is a set of deduplicated Blobs.
type BlobSet interface {
	// GetBlob gets the Blob for the given Digest, or nil if no such Blob exists.
	GetBlob(digest Digest) Blob
	// Blobs returns the Blobs associated with this BlobSet, ordered by
	// the sort value of the Digest.
	//
	// TODO: The former version of this package returns the Blobs in unspecified
	// order. We generally try to give a deterministic order in our codebase. There
	// are schools of arguments both ways, but we'd like to stay consistent.
	Blobs() []Blob

	// Protect against creation of a BlobSet outside of this package, as we
	// do very careful validation.
	isBlobSet()
}

// NewBlobSet returns a new BlobSet.
//
// TODO: in the former version of this package, we validated that Blob contents matched for Blobs
// with the same Digest via BlobEqual, however we no longer do this as BlobEqual no longer
// validates content matching. See the comment on BlobEqual for why.
// TODO: The former version of this package also validated that no Blobs were nil, but this
// is a basic expectation across our codebase. Given this and the previous TODO, NewBlobSet
// no longer needs to return an error.
func NewBlobSet(blobs []Blob) BlobSet {
	return newBlobSet(blobs)
}

// FileNode is a path and associated digest.
type FileNode interface {
	// String encodes the FileNode into its canonical form.
	//
	// If the digest is nil, this is simply:
	//
	//   path
	//
	// If the digest is not nil, this is:
	//
	//   digestString[SP][SP]path
	fmt.Stringer

	// Path returns the Path of the file.
	//
	// The path is normalized and non-empty.
	Path() string
	// Digest returns the Digest of the file.
	//
	// The Digest may be nil, in which case the file is empty.
	Digest() Digest

	// Protect against creation of a FileNode outside of this package, as we
	// do very careful validation.
	isFileNode()
}

// NewFileNode returns a new FileNode.
//
// The Digest may be nil.
//
// The path is validated to be normalized and non-empty.
func NewFileNode(path string, digest Digest) (FileNode, error) {
	return newFileNode(path, digest)
}

// NewFileNodeForString returns a new FileNode for the given FileNode string.
//
// This reverses FileNode.String().
func NewFileNodeForString(s string) (FileNode, error) {
	switch split := strings.Split(s, "  "); len(split) {
	case 1:
		return NewFileNode(split[0], nil)
	case 2:
		digest, err := NewDigestForString(split[0])
		if err != nil {
			return nil, err
		}
		return NewFileNode(split[1], digest)
	default:
		return nil, fmt.Errorf("unknown FileNode encoding: %q", s)
	}
}

// Maniest is a set FileNodes.
type Manifest interface {
	// fmt.Stringer encodes the Manifest into its canonical form, consisting of
	// an ordered list of paths and their hash digests. Sorted by path.
	//
	// See the documentation on FileNode for how FileNodes are encoded.
	//
	// An example encoded manifest:
	//
	//	shake256:cd22db48cf7c274bbffcb5494a854000cd21b074df7c6edabbd0102c4be8d7623e3931560fcda7acfab286ae1d4f506911daa31f223ee159f59ffce0c7acbbaa  buf.lock
	//	shake256:3b353aa5aacd11015e8577f16e2c4e7a242ce773d8e3a16806795bb94f76e601b0db9bf42d5e1907fda63303e1fa1c65f1c175ecc025a3ef29c3456ad237ad84  buf.md
	//	shake256:7c88a20cf931702d042a4ddee3fde5de84814544411f1c62dbf435b1b81a12a8866a070baabcf8b5a0d31675af361ccb2d93ddada4cdcc11bab7ea3d8d7c4667  buf.yaml
	//	shake256:9db25155eafd19b36882cff129daac575baa67ee44d1cb1fd3894342b28c72b83eb21aa595b806e9cb5344759bc8308200c5af98e4329aa83014dde99afa903a  pet/v1/pet.proto
	//  pet/v1/empty_file.proto
	fmt.Stringer

	// FileNodes returns the set of FileNodes that make up the Manifest.
	//
	// The paths of the given FileNodes are guaranteed to be unique.
	// The iteration order will be the sorted order of the paths.
	FileNodes() []FileNode

	// Protect against creation of a Manifest outside of this package, as we
	// do very careful validation.
	isManifest()
}

// NewManifest returns a new Manifest for the given path -> Digest map.
//
// FileNodes are deduplicated upon construction, however if two FileNodes
// with the same path have different Digests, an error is returned.
func NewManifest(fileNodes []FileNode) (Manifest, error) {
	return newManifest(fileNodes)
}

// NewManifestForString returns a new Manifest for the given Manifest string.
//
// This reverses Manifest.String().
func NewManifestForString(s string) (Manifest, error) {
	var fileNodes []FileNode
	if s[len(s)-1] != '\n' {
		return nil, errors.New("string for Manifest did not end with newline")
	}
	for i, line := range strings.Split(s, "\n") {
		fileNode, err := NewFileNodeForString(line)
		if err != nil {
			return nil, fmt.Errorf("invalid Manifest at line %d: %w", i, err)
		}
		fileNodes = append(fileNodes, fileNode)
	}
	return NewManifest(fileNodes)
}

// ManifestToBlob converts the string representation of the given Manifest into a Blob.
//
// The Manifest is assumed to be non-nil
func ManifestToBlob(manifest Manifest) (Blob, error) {
	return NewBlobForContent(DigestTypeShake256, strings.NewReader(manifest.String()))
}

// BlobToManifest converts the given Blob representing the string representaion of a Manifest into a Manifest.
//
// The Blob is assumed to be non-nil
func BlobToManifest(blob Blob) (Manifest, error) {
	return NewManifestForString(string(blob.Content()))
}

// FileSet is a pair of a Manifest and its associated BlobSet.
//
// This can be read and written from and to a storage.Bucket.
//
// The Manifest is guaranteed to exactly correlate with the Blobs in the BlobSet,
// that is the Digests of the FileNodes in the Manifest will exactly match the
// Digests in the Blobs. Note that some FileNodes may have empty Digests, in which
// case there is no corresponding Blob (as the content is empty).
type FileSet interface {
	// Manifest returns the associated Manifest.
	Manifest() Manifest
	// BlobSet returns the associated BlobSet.
	BlobSet() BlobSet

	// Protect against creation of a FileSet outside of this package, as we
	// do very careful validation.
	isFileSet()
}

// NewFileSet returns a new FileSet.
//
// Validation is done to ensure the Manifest exactly matches the BlobSet.
func NewFileSet(manifest Manifest, blobSet BlobSet) (FileSet, error) {
	manifestDigestStringMap := make(map[string]struct{})
	blobDigestStringMap := make(map[string]struct{})
	for _, fileNode := range manifest.FileNodes() {
		if digest := fileNode.Digest(); digest != nil {
			manifestDigestStringMap[digest.String()] = struct{}{}
		}
	}
	for _, blob := range blobSet.Blobs() {
		blobDigestStringMap[blob.Digest().String()] = struct{}{}
	}
	var onlyInManifest []string
	var onlyInBlobSet []string
	for manifestDigestString := range manifestDigestStringMap {
		if _, ok := blobDigestStringMap[manifestDigestString]; !ok {
			onlyInManifest = append(onlyInManifest, manifestDigestString)
		}
	}
	for blobDigestString := range blobDigestStringMap {
		if _, ok := manifestDigestStringMap[blobDigestString]; !ok {
			onlyInBlobSet = append(onlyInBlobSet, blobDigestString)
		}
	}
	if len(onlyInManifest) > 0 || len(onlyInBlobSet) > 0 {
		sort.Strings(onlyInManifest)
		sort.Strings(onlyInBlobSet)
		return nil, fmt.Errorf("mismatched Manifest and BlobSet at FileSet construction, digests only in Manifest: [%v], digests only in BlobSet: [%v]", onlyInManifest, onlyInBlobSet)
	}
	return newFileSet(manifest, blobSet), nil
}

// NewFileSetForBucket returns a new FileSet for the given ReadBucket.
func NewFileSetForBucket(ctx context.Context, bucket storage.ReadBucket) (FileSet, error) {
	var fileNodes []FileNode
	var blobs []Blob
	if err := storage.WalkReadObjects(
		ctx,
		bucket,
		"",
		func(readObject storage.ReadObject) error {
			blob, err := NewBlobForContent(DigestTypeShake256, readObject)
			if err != nil {
				return err
			}
			var digest Digest
			// Otherwise, we have an empty file.
			if blob != nil {
				digest = blob.Digest()
			}
			fileNode, err := NewFileNode(readObject.Path(), digest)
			if err != nil {
				return err
			}
			fileNodes = append(fileNodes, fileNode)
			blobs = append(blobs, blob)
			return nil
		},
	); err != nil {
		return nil, err
	}
	manifest, err := NewManifest(fileNodes)
	if err != nil {
		return nil, err
	}
	return newFileSet(
		manifest,
		newBlobSet(blobs),
	), nil
}

// PutFileSetToBucket writes the FileSet to the given WriteBucket.
func PutFileSetToBucket(
	ctx context.Context,
	fileSet FileSet,
	bucket storage.WriteBucket,
) error {
	for _, fileNode := range fileSet.Manifest().FileNodes() {
		var blob Blob
		if digest := fileNode.Digest(); digest != nil {
			blob = fileSet.BlobSet().GetBlob(digest)
			if blob == nil {
				// This should never happen given our validation.
				return fmt.Errorf("nil Blob with non-empty Digest %v in PutFileSetToBucket", digest)
			}
		}
		writeObjectCloser, err := bucket.Put(ctx, fileNode.Path(), storage.PutWithAtomic())
		if err != nil {
			return err
		}
		if blob != nil {
			if _, err := writeObjectCloser.Write(blob.Content()); err != nil {
				return err
			}
		}
		if err := writeObjectCloser.Close(); err != nil {
			return err
		}
	}
	return nil
}
