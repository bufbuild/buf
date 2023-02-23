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

package manifest

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/sha3"
)

// DigestType is the type for digests in this package.
type DigestType string

const (
	DigestTypeShake256 DigestType = "shake256"

	shake256Length = 64
)

// Digest represents a hash function's value.
type Digest struct {
	dtype  DigestType
	digest []byte
	hexstr string
}

// NewDigestFromBytes builds a digest from a type and the digest bytes.
func NewDigestFromBytes(dtype DigestType, digest []byte) (*Digest, error) {
	if dtype == "" {
		return nil, errors.New("digest type cannot be empty")
	}
	if dtype != DigestTypeShake256 {
		return nil, fmt.Errorf("unsupported digest type: %q", dtype)
	}
	if len(digest) != shake256Length {
		return nil, fmt.Errorf(
			"invalid digest: got %d bytes, expected %d bytes for type %q",
			len(digest), shake256Length, dtype,
		)
	}
	return &Digest{
		dtype:  dtype,
		digest: digest,
		hexstr: hex.EncodeToString(digest),
	}, nil
}

// NewDigestFromHex builds a digest from a type and the hexadecimal string of
// the bytes. It returns an error if the received string is not a valid hex.
func NewDigestFromHex(dtype DigestType, hexstr string) (*Digest, error) {
	digest, err := hex.DecodeString(hexstr)
	if err != nil {
		return nil, err
	}
	return NewDigestFromBytes(dtype, digest)
}

// NewDigestFromString build a digest from a string representation of it.
func NewDigestFromString(typedDigest string) (*Digest, error) {
	dtype, hexstr, found := strings.Cut(typedDigest, ":")
	if !found {
		return nil, errors.New("malformed digest string")
	}
	return NewDigestFromHex(DigestType(dtype), hexstr)
}

// String returns the hash in a manifest's string format: "<type>:<hex>".
func (d *Digest) String() string {
	return string(d.dtype) + ":" + d.hexstr
}

// Type returns the digest type.
func (d *Digest) Type() DigestType {
	return d.dtype
}

// Bytes returns the digest bytes.
func (d *Digest) Bytes() []byte {
	return d.digest
}

// Hex returns the digest bytes in its hexadecimal string representation.
func (d *Digest) Hex() string {
	return d.hexstr
}

// Equal compares the digest type and bytes with other digest.
func (d *Digest) Equal(other Digest) bool {
	return d.dtype == other.dtype && bytes.Equal(d.digest, other.digest)
}

// Digester is something that can digest a content into a digest.
type Digester interface {
	Digest(content io.Reader) (*Digest, error)
}

type shake256Digester struct {
	hash sha3.ShakeHash
}

// NewDigester returns a digester of the requested type.
func NewDigester(dtype DigestType) (Digester, error) {
	if dtype != DigestTypeShake256 {
		return nil, fmt.Errorf("not supported digest type %q", dtype)
	}
	return &shake256Digester{hash: sha3.NewShake256()}, nil
}

func (d *shake256Digester) Digest(content io.Reader) (*Digest, error) {
	d.hash.Reset()
	if _, err := io.Copy(d.hash, content); err != nil {
		return nil, err
	}
	digest := make([]byte, shake256Length)
	if _, err := d.hash.Read(digest); err != nil {
		// sha3.ShakeHash never errors or short reads. Something horribly wrong
		// happened if your computer ended up here.
		return nil, err
	}
	return NewDigestFromBytes(DigestTypeShake256, digest)
}
