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

package manifest2

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

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
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
func NewDigest(digestType DigestType, value []byte) (Digest, error) {
	switch digestType {
	case DigestTypeShake256:
		if len(value) != shake256Length {
			return nil, fmt.Errorf("invalid %s Digest value: expected %d bytes, got %d", digestType.String(), shake256Length, len(value))
		}
		return newDigest(digestType, value), nil
	default:
		return nil, fmt.Errorf("unknown DigestType: %v", digestType)
	}
}

// NewDigestForContent creates a new Digest based on the given content read from the Reader.
//
// The Reader is read until io.EOF.
// Validation is performed to ensure that the DigestType is known.
func NewDigestForContent(digestType DigestType, reader io.Reader) (Digest, error) {
	switch digestType {
	case DigestTypeShake256:
		shakeHash := sha3.NewShake256()
		shakeHash.Reset()
		if _, err := io.Copy(shakeHash, reader); err != nil {
			return nil, err
		}
		value := make([]byte, shake256Length)
		if _, err := shakeHash.Read(value); err != nil {
			// sha3.ShakeHash never errors or short reads. Something horribly wrong
			// happened if your computer ended up here.
			return nil, err
		}
		return NewDigest(digestType, value)
	default:
		return nil, fmt.Errorf("unknown DigestType: %v", digestType)
	}
}

// NewDigestForString returns a new Digest for the given Digest string.
//
// This reverses Digest.String().
// A Digest string is of the form typeString:hexValue.
func NewDigestForString(s string) (Digest, error) {
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
	return NewDigest(digestType, value)
}

// DigestEqual returns true if the given Digests are considered equal.
//
// This check both the DigestType and Digest value.
func DigestEqual(a Digest, b Digest) bool {
	if a.Type() != b.Type() {
		return false
	}
	return bytes.Equal(a.Value(), b.Value())
}

// Blob is content with its associated Digest.
type Blob interface {
	// Digest returns the Digest of the Blob.
	//
	// NewDigestForContent(blob.Digest.Type(), bytes.NewReader(blob.Content()) should
	// always match this value.
	Digest() Digest
	// Content returns the content of the Blob.
	Content() []byte

	// Protect against creation of a Blob outside of this package, as we
	// do very careful validation.
	isBlob()
}

// NewBlobForContent returns a new Blob with a Digest of the given DigestType,
// and the content as read from the Reader.
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
	return newBlob(digest, buffer.Bytes()), nil
}

// NewBlobForContentWithKnownDigest returns a new Blob for the given Digest and content
// as read from the Reader.
//
// The reader is read until io.EOF.
// Validation is performed to ensure that the Digest matches the computed Digest of the content.
func NewBlobForContentWithKnownDigest(knownDigest Digest, reader io.Reader) (Blob, error) {
	blob, err := NewBlobForContent(knownDigest.Type(), reader)
	if err != nil {
		return nil, err
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

// Validates same digests have same content TODO isn't this already true via NewBlob validation?
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
