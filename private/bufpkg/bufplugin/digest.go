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

package bufplugin

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// DigestTypeP1 represents the p4 plugin digest type.
	//
	// This is the newest digest type, and should generally be used. The string value
	// of this is "p1".
	DigestTypeP1 = iota + 1
)

var (
	// AllDigestTypes are all known DigestTypes.
	AllDigestTypes = []DigestType{
		DigestTypeP1,
	}
	digestTypeToString = map[DigestType]string{
		DigestTypeP1: "p1",
	}
	stringToDigestType = map[string]DigestType{
		"p1": DigestTypeP1,
	}
)

// DigestType is a type of digest.
type DigestType int

// ParseDigestType parses a DigestType from its string representation.
//
// This reverses DigestType.String().
//
// Returns an error of type *bufparse.ParseError if thie string could not be parsed.
func ParseDigestType(s string) (DigestType, error) {
	d, ok := stringToDigestType[s]
	if !ok {
		return 0, bufparse.NewParseError(
			"plugin digest type",
			s,
			fmt.Errorf("unknown type: %q", s),
		)
	}
	return d, nil
}

// String prints the string representation of the DigestType.
func (d DigestType) String() string {
	s, ok := digestTypeToString[d]
	if !ok {
		return strconv.Itoa(int(d))
	}
	return s
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

	isDigest()
}

// NewDigest creates a new Digest.
func NewDigest(digestType DigestType, bufcasDigest bufcas.Digest) (Digest, error) {
	switch digestType {
	case DigestTypeP1:
		if bufcasDigest.Type() != bufcas.DigestTypeShake256 {
			return nil, syserror.Newf(
				"trying to create a %v Digest for a cas Digest of type %v",
				digestType,
				bufcasDigest.Type(),
			)
		}
		return newDigest(digestType, bufcasDigest), nil
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

// DigestEqual returns true if the given Digests are considered equal.
//
// If both Digests are nil, this returns true.
//
// This checks both the DigestType and Digest value.
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
		return nil, bufparse.NewParseError(
			"plugin digest", s,
			errors.New(`must be in the form "digest_type:digest_hex_value"`),
		)
	}
	digestType, err := ParseDigestType(digestTypeString)
	if err != nil {
		return nil, bufparse.NewParseError(
			"plugin digest",
			digestTypeString,
			err,
		)
	}
	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, bufparse.NewParseError(
			"plugin digest",
			s,
			errors.New(`could not parse hex: must in the form "digest_type:digest_hex_value"`),
		)
	}
	switch digestType {
	case DigestTypeP1:
		bufcasDigest, err := bufcas.NewDigest(value)
		if err != nil {
			return nil, err
		}
		return NewDigest(digestType, bufcasDigest)
	default:
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

/// *** PRIVATE ***

type digest struct {
	digestType   DigestType
	bufcasDigest bufcas.Digest
	// Cache as we call String pretty often.
	// We could do this lazily but not worth it.
	stringValue string
}

// validation should occur outside of this function.
func newDigest(digestType DigestType, bufcasDigest bufcas.Digest) *digest {
	return &digest{
		digestType:   digestType,
		bufcasDigest: bufcasDigest,
		stringValue:  digestType.String() + ":" + hex.EncodeToString(bufcasDigest.Value()),
	}
}

func (d *digest) Type() DigestType {
	return d.digestType
}

func (d *digest) Value() []byte {
	return d.bufcasDigest.Value()
}

func (d *digest) String() string {
	return d.stringValue
}

func (*digest) isDigest() {}
