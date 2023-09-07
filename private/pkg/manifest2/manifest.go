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

type DigestType int

func (d DigestType) String() string {
	s, ok := digestTypeToString[d]
	if !ok {
		return strconv.Itoa(int(d))
	}
	return s
}

func ParseDigestType(s string) (DigestType, error) {
	d, ok := stringToDigestType[s]
	if !ok {
		return 0, fmt.Errorf("unknown DigestType: %q", s)
	}
	return d, nil
}

type Digest interface {
	fmt.Stringer

	Type() DigestType
	Value() []byte
}

func NewDigest(digestType DigestType, value []byte) (Digest, error) {
	return nil, nil
}

func NewDigestForReader(digestType DigestType, reader io.Reader) (Digest, error) {
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

func DigestEqual(a Digest, b Digest) bool {
	return false
}

type Blob interface {
	Digest() Digest
	Content() []byte
}

// validates content matches digest
func NewBlob(digest Digest, content []byte) (Blob, error) {
	contentDigest, err := NewDigestForReader(digest.Type(), bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	if !DigestEqual(digest, contentDigest) {
		return nil, fmt.Errorf("Digest %v did not match Digest %v when creating a new Blob", digest, contentDigest)
	}
	return &blob{
		digest:  digest,
		content: content,
	}, nil
}

type BlobSet interface {
	GetBlob(digest Digest) (Blob, error)
	Blobs() []Blob
}

// Validates same digests have same content TODO isn't this already true via NewBlob validation?
func NewBlobSet(blobs []Blob) (BlobSet, error) {
	return nil, nil
}

func BlobEqual(a Blob, b Blob) bool {
	return false
}
