// Copyright 2020-2026 Buf Technologies, Inc.
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

package cas

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/pkg/shake256"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// DigestTypeShake256 represents the shake256 digest type.
	DigestTypeShake256 DigestType = iota + 1
	// DigestTypeSha256 represents the sha256 digest type.
	//
	// SHA-256 Digest string values are bare hex so that SHA-256 manifests use the
	// same digest spelling as standard checksum files.
	DigestTypeSha256

	sha256DigestLength = 32
)

var (
	// AllDigestTypes are all DigestTypes.
	AllDigestTypes = []DigestType{DigestTypeShake256, DigestTypeSha256}

	digestTypeToString = map[DigestType]string{
		DigestTypeShake256: "shake256",
		DigestTypeSha256:   "sha256",
	}
	stringToDigestType = map[string]DigestType{
		"shake256": DigestTypeShake256,
		"sha256":   DigestTypeSha256,
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
//
// Returns an error of type *ParseError if the string could not be parsed.
func ParseDigestType(s string) (DigestType, error) {
	d, ok := stringToDigestType[s]
	if !ok {
		return 0, newParseError(
			"digest type",
			s,
			fmt.Errorf("unknown type: %q", s),
		)
	}
	return d, nil
}

// Digest is a digest of some content.
//
// It consists of a DigestType and a digest value.
type Digest interface {
	// String prints typeString:hexValue for typed digests, or bare hex for SHA-256.
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

// NewDigest returns a new Digest for the already-computed digest value.
func NewDigest(digestType DigestType, value []byte) (Digest, error) {
	if err := validateDigestParameters(digestType, value); err != nil {
		return nil, err
	}
	return newDigest(digestType, bytes.Clone(value)), nil
}

// NewDigestForContent creates a new Digest based on the given content read from the Reader.
//
// A valid Digest is returned, even in the case of empty content.
//
// The Reader is read until io.EOF.
func NewDigestForContent(digestType DigestType, reader io.Reader) (Digest, error) {
	switch digestType {
	case DigestTypeShake256:
		shake256Digest, err := shake256.NewDigestForContent(reader)
		if err != nil {
			return nil, err
		}
		return newDigest(DigestTypeShake256, shake256Digest.Value()), nil
	case DigestTypeSha256:
		hash := sha256.New()
		if _, err := io.Copy(hash, reader); err != nil {
			return nil, err
		}
		return newDigest(DigestTypeSha256, hash.Sum(nil)[:]), nil
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

// ParseDigest parses a Digest from its string representation.
//
// A SHA-256 Digest string is of the form hexValue. All other Digest strings are
// of the form typeString:hexValue. The string is expected to be non-empty. If
// not, an error is returned.
//
// This reverses Digest.String().
//
// Returns an error of type *bufparse.ParseError if the string could not be parsed.
func ParseDigest(s string) (Digest, error) {
	if s == "" {
		// This should be considered a system error.
		return nil, errors.New("empty string passed to ParseDigest")
	}
	digestTypeString, hexValue, ok := strings.Cut(s, ":")
	if !ok {
		value, err := hex.DecodeString(s)
		if err != nil {
			return nil, newParseError(
				"digest",
				s,
				errors.New(`could not parse hex: must be in the form "digest_hex_value" or "digest_type:digest_hex_value"`),
			)
		}
		if err := validateDigestParameters(DigestTypeSha256, value); err != nil {
			return nil, newParseError(
				"digest",
				s,
				err,
			)
		}
		return newDigest(DigestTypeSha256, value), nil
	}
	digestType, err := ParseDigestType(digestTypeString)
	if err != nil {
		return nil, newParseError(
			"digest",
			s,
			err,
		)
	}
	if digestType == DigestTypeSha256 {
		return nil, newParseError(
			"digest",
			s,
			errors.New(`sha256 digests must be in the form "digest_hex_value"`),
		)
	}
	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, newParseError(
			"digest",
			s,
			errors.New(`could not parse hex: must in the form "digest_type:digest_hex_value"`),
		)
	}
	if err := validateDigestParameters(digestType, value); err != nil {
		return nil, newParseError(
			"digest",
			s,
			err,
		)
	}
	return newDigest(digestType, value), nil
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
}

// validation should occur outside of this function.
func newDigest(digestType DigestType, value []byte) *digest {
	return &digest{
		digestType: digestType,
		value:      value,
	}
}

func (d *digest) Type() DigestType {
	return d.digestType
}

func (d *digest) Value() []byte {
	return d.value
}

func (d *digest) String() string {
	valueString := hex.EncodeToString(d.value)
	if d.digestType == DigestTypeSha256 {
		return valueString
	}
	return d.digestType.String() + ":" + valueString
}

func (*digest) isDigest() {}

func validateDigestParameters(digestType DigestType, value []byte) error {
	switch digestType {
	case DigestTypeShake256:
		_, err := shake256.NewDigest(value)
		if err != nil {
			return err
		}
	case DigestTypeSha256:
		if len(value) != sha256DigestLength {
			return fmt.Errorf("invalid sha256 digest value: expected %d bytes, got %d", sha256DigestLength, len(value))
		}
	default:
		// This is really always a system error, but little harm in including it here, even
		// though it'll get converted into a bufparse.ParseError in parse.
		return syserror.Newf(`unknown digest type: %q`, digestType.String())
	}
	return nil
}
