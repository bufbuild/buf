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

	// The type of digest.
	Type() DigestType
	// The digest value.
	Value() []byte
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
	// TODO
	return false
}

// Blob is content with its associated Digest.
type Blob interface {
	// The Digest of the Blob.
	//
	// NewDigestForContent(blob.Digest.Type(), bytes.NewReader(blob.Content()) should
	// always match this value.
	Digest() Digest
	// The content of the Blob.
	Content() []byte
}

// validates content matches digest
// NewBlob returns a new Blob for the given Digest and content.
//
// Validation is performed to ensure that the Digest matches the computed
// Digest of the content.
func NewBlob(digest Digest, content []byte) (Blob, error) {
	contentDigest, err := NewDigestForContent(digest.Type(), bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	if !DigestEqual(digest, contentDigest) {
		return nil, fmt.Errorf("Digest %v did not match Digest %v when creating a new Blob", digest, contentDigest)
	}
	return newBlob(digest, content), nil
}

// NewBlobForContent returns a new Blob with a Digest of the given DigestType,
// and the content as read from the Reader.
//
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

// BlobEqual returns true if the given Blobs are considered equal.
//
// This checks both the Digest and the content.
//
// Technically we do not need to compare the contents, as we know that the Digest
// is a valid Digest for the given content via valiation we did at construction time.
// However, in the absence of a performance-related reason, we do equality on the
// digests as a safety check. In the future, this could be removed.
func BlobEqual(a Blob, b Blob) bool {
	if !DigestEqual(a.Digest(), b.Digest()) {
		return false
	}
	aContent := a.Content()
	bContent := b.Content()
	for i := 0; i < len(aContent); i += 4096 {
		j := i + 4096
		if j > len(aContent) {
			j = len(aContent)
		}
		if !bytes.Equal(aContent[i:j], bContent[i:j]) {
			return false
		}
	}
	return true
}

type BlobSet interface {
	GetBlob(digest Digest) (Blob, error)
	Blobs() []Blob
}

// Validates same digests have same content TODO isn't this already true via NewBlob validation?
func NewBlobSet(blobs []Blob) (BlobSet, error) {
	return nil, nil
}
