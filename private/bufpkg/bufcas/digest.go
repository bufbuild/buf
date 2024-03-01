// Copyright 2020-2024 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/pkg/shake256"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// DigestTypeShake256 represents the shake256 digest type.
	//
	// This is both the default and the only currently-known value for DigestType.
	DigestTypeShake256 DigestType = iota + 1
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
// This reverses DigestType.String().
//
// Returns an error of type *ParseError if thie string could not be parsed.
func ParseDigestType(s string) (DigestType, error) {
	d, ok := stringToDigestType[s]
	if !ok {
		return 0, &ParseError{
			typeString: "digest type",
			input:      s,
			err:        fmt.Errorf("unknown type: %q", s),
		}
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

// NewDigest returns a new Digest for the value.
func NewDigest(value []byte, options ...DigestOption) (Digest, error) {
	digestOptions := newDigestOptions()
	for _, option := range options {
		option(digestOptions)
	}
	if digestOptions.digestType == 0 {
		digestOptions.digestType = DigestTypeShake256
	}
	switch digestOptions.digestType {
	case DigestTypeShake256:
		shake256Digest, err := shake256.NewDigest(value)
		if err != nil {
			return nil, err
		}
		return newDigest(DigestTypeShake256, shake256Digest.Value()), nil
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown DigestType: %v", digestOptions.digestType)
	}
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
		shake256Digest, err := shake256.NewDigestForContent(reader)
		if err != nil {
			return nil, err
		}
		return newDigest(DigestTypeShake256, shake256Digest.Value()), nil
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown DigestType: %v", digestOptions.digestType)
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
		// This should be considered a system error.
		return nil, errors.New("empty string passed to ParseDigest")
	}
	digestTypeString, hexValue, ok := strings.Cut(s, ":")
	if !ok {
		return nil, &ParseError{
			typeString: "digest",
			input:      s,
			err:        errors.New(`must in the form "digest_type:digest_hex_value"`),
		}
	}
	digestType, err := ParseDigestType(digestTypeString)
	if err != nil {
		return nil, &ParseError{
			typeString: "digest",
			input:      s,
			err:        err,
		}
	}
	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, &ParseError{
			typeString: "digest",
			input:      s,
			err:        errors.New(`could not parse hex: must in the form "digest_type:digest_hex_value"`),
		}
	}
	if err := validateDigestParameters(digestType, value); err != nil {
		return nil, &ParseError{
			typeString: "digest",
			input:      s,
			err:        err,
		}
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
	// Cache as we call String pretty often.
	// We could do this lazily but not worth it.
	stringValue string
}

// validation should occur outside of this function.
func newDigest(digestType DigestType, value []byte) *digest {
	return &digest{
		digestType:  digestType,
		value:       value,
		stringValue: digestType.String() + ":" + hex.EncodeToString(value),
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

func validateDigestParameters(digestType DigestType, value []byte) error {
	switch digestType {
	case DigestTypeShake256:
		_, err := shake256.NewDigest(value)
		if err != nil {
			return err
		}
	default:
		// This is really always a system error, but little harm in including it here, even
		// though it'll get converted into a ParseError in parse.
		return syserror.Newf(`unknown digest type: %q`, digestType.String())
	}
	return nil
}
