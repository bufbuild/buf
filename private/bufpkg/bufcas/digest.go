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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	storagev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/storage/v1beta1"
	"golang.org/x/crypto/sha3"
)

const (
	// DigestTypeShake256 represents the shake256 digest type.
	//
	// This is both the default and the only currently-known value for DigestType.
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
	digestTypeToProto = map[DigestType]storagev1beta1.Digest_Type{
		DigestTypeShake256: storagev1beta1.Digest_TYPE_SHAKE256,
	}
	protoToDigestType = map[storagev1beta1.Digest_Type]DigestType{
		storagev1beta1.Digest_TYPE_SHAKE256: DigestTypeShake256,
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
// This reverses DigestType.String().
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
	// Always a valid value.
	Type() DigestType
	// Value returns the digest value.
	//
	// Always non-empty.
	Value() []byte

	// Protect against creation of a Digest outside of this package, as we
	// do very careful validation.
	isDigest()
}

// NewDigestForContent creates a new Digest based on the given content read from the Reader.
//
// A valid Digest is returned, even in the case of empty content.
//
// The Reader is read until io.EOF.
func NewDigestForContent(reader io.Reader, options ...DigestOption) (Digest, error) {
	digestOptions := newDigestOptions()
	for _, option := range options {
		option(digestOptions)
	}
	if digestOptions.digestType == 0 {
		digestOptions.digestType = DigestTypeShake256
	}
	switch digestOptions.digestType {
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
		return newDigest(DigestTypeShake256, value)
	default:
		return nil, fmt.Errorf("unknown DigestType: %v", digestOptions.digestType)
	}
}

// DigestOption is an option for a new Digest.
type DigestOption func(*digestOptions)

// DigestWithDigestType returns a new DigestOption that specifies the DigestType to be used.
//
// The default is DigestTypeShake256.
func DigestWithDigestType(digestType DigestType) DigestOption {
	return func(digestOptions *digestOptions) {
		digestOptions.digestType = digestType
	}
}

// ParseDigest parses a Digest from its string representation.
//
// A Digest string is of the form typeString:hexValue.
// The string is expected to be non-empty, If not, an error is treutned.
//
// This reverses Digest.String().
func ParseDigest(s string) (Digest, error) {
	if s == "" {
		return nil, errors.New("empty string passed to ParseDigest")
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

// DigestToProto converts the given Digest to a proto Digest.
//
// TODO: validate the returned Digest.
func DigestToProto(digest Digest) (*storagev1beta1.Digest, error) {
	protoDigestType, ok := digestTypeToProto[digest.Type()]
	// Technically we have aleady done this validation but just to be safe.
	if !ok {
		return nil, fmt.Errorf("unknown DigestType: %v", digest.Type())
	}
	return &storagev1beta1.Digest{
		Type:  protoDigestType,
		Value: digest.Value(),
	}, nil
}

// ProtoToDigest converts the given proto Digest to a Digest.
//
// Validation is performed to ensure the DigestType is known, and the value
// is a valid digest value for the given DigestType.
// TODO: validate the input proto Digest.
func ProtoToDigest(protoDigest *storagev1beta1.Digest) (Digest, error) {
	digestType, ok := protoToDigestType[protoDigest.Type]
	if !ok {
		return nil, fmt.Errorf("unknown proto Digest.Type: %v", protoDigest.Type)
	}
	return newDigest(digestType, protoDigest.Value)
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

/// *** PRIVATE ***

type digest struct {
	digestType DigestType
	value      []byte
	// Cache as we call String pretty often.
	// We could do this lazily but not worth it.
	stringValue string
}

func newDigest(digestType DigestType, value []byte) (*digest, error) {
	switch digestType {
	case DigestTypeShake256:
		if len(value) != shake256Length {
			return nil, fmt.Errorf("invalid %s Digest value: expected %d bytes, got %d", digestType.String(), shake256Length, len(value))
		}
		return &digest{
			digestType:  digestType,
			value:       value,
			stringValue: digestType.String() + ":" + hex.EncodeToString(value),
		}, nil
	default:
		return nil, fmt.Errorf("unknown DigestType: %v", digestType)
	}
}

func (d *digest) Type() DigestType {
	return d.digestType
}

func (d *digest) Value() []byte {
	return d.value
}

func (d *digest) String() string {
	return d.stringValue
}

func (*digest) isDigest() {}

type digestOptions struct {
	digestType DigestType
}

func newDigestOptions() *digestOptions {
	return &digestOptions{}
}
