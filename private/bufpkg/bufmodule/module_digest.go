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

package bufmodule

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// ModuleDigestTypeB4 represents the b4 module digest type.
	//
	// This represents the pre-refactor shake256 digest type, and the string value of
	// this is "shake256" for backwards-compatibility reasons.
	ModuleDigestTypeB4 ModuleDigestType = iota + 1
	// ModuleDigestTypeB5 represents the b5 digest type.
	//
	// This is the newest digest type, and should generally be used. The string value
	// of this is "b5".
	ModuleDigestTypeB5
)

var (
	moduleDigestTypeToString = map[ModuleDigestType]string{
		ModuleDigestTypeB4: "shake256",
		ModuleDigestTypeB5: "b5",
	}
	stringToModuleDigestType = map[string]ModuleDigestType{
		"shake256": ModuleDigestTypeB4,
		"b5":       ModuleDigestTypeB5,
	}
)

// ModuleDigestType is a type of digest.
type ModuleDigestType int

// String prints the string representation of the ModuleDigestType.
func (d ModuleDigestType) String() string {
	s, ok := moduleDigestTypeToString[d]
	if !ok {
		return strconv.Itoa(int(d))
	}
	return s
}

// ParseModuleDigestType parses a ModuleDigestType from its string representation.
//
// This reverses ModuleDigestType.String().
//
// Returns an error of type *ParseError if thie string could not be parsed.
func ParseModuleDigestType(s string) (ModuleDigestType, error) {
	d, ok := stringToModuleDigestType[s]
	if !ok {
		return 0, &ParseError{
			typeString: "module digest type",
			input:      s,
			err:        fmt.Errorf("unknown type: %q", s),
		}
	}
	return d, nil
}

// ModuleDigest is a digest of some content.
//
// It consists of a ModuleDigestType and a digest value.
type ModuleDigest interface {
	// String() prints typeString:hexValue.
	fmt.Stringer

	// Type returns the type of digest.
	// Always a valid value.
	Type() ModuleDigestType
	// Value returns the digest value.
	//
	// Always non-empty.
	Value() []byte

	isModuleDigest()
}

// NewModuleDigest creates a new ModuleDigest.
func NewModuleDigest(moduleDigestType ModuleDigestType, bufcasDigest bufcas.Digest) (ModuleDigest, error) {
	switch moduleDigestType {
	case ModuleDigestTypeB4, ModuleDigestTypeB5:
		if bufcasDigest.Type() != bufcas.DigestTypeShake256 {
			return nil, syserror.Newf(
				"trying to create a %v ModuleDigest for a cas Digest of type %v",
				moduleDigestType,
				bufcasDigest.Type(),
			)
		}
		return newModuleDigest(moduleDigestType, bufcasDigest), nil
	default:
		// This is a system error.
		return nil, syserror.Newf("unknown ModuleDigestType: %v", moduleDigestType)
	}
}

// ParseModuleDigest parses a ModuleDigest from its string representation.
//
// A ModuleDigest string is of the form typeString:hexValue.
// The string is expected to be non-empty, If not, an error is treutned.
//
// This reverses ModuleDigest.String().
func ParseModuleDigest(s string) (ModuleDigest, error) {
	if s == "" {
		// This should be considered a system error.
		return nil, errors.New("empty string passed to ParseModuleDigest")
	}
	digestTypeString, hexValue, ok := strings.Cut(s, ":")
	if !ok {
		return nil, &ParseError{
			typeString: "module digest",
			input:      s,
			err:        errors.New(`must in the form "digest_type:digest_hex_value"`),
		}
	}
	moduleDigestType, err := ParseModuleDigestType(digestTypeString)
	if err != nil {
		return nil, &ParseError{
			typeString: "module digest",
			input:      s,
			err:        err,
		}
	}
	value, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, &ParseError{
			typeString: "module digest",
			input:      s,
			err:        errors.New(`could not parse hex: must in the form "digest_type:digest_hex_value"`),
		}
	}
	switch moduleDigestType {
	case ModuleDigestTypeB4, ModuleDigestTypeB5:
		bufcasDigest, err := bufcas.NewDigest(value)
		if err != nil {
			return nil, err
		}
		return NewModuleDigest(moduleDigestType, bufcasDigest)
	default:
		return nil, syserror.Newf("unknown ModuleDigestType: %v", moduleDigestType)
	}
}

// ModuleDigestEqual returns true if the given ModuleDigests are considered equal.
//
// If both ModuleDigests are nil, this returns true.
//
// This checks both the ModuleDigestType and ModuleDigest value.
func ModuleDigestEqual(a ModuleDigest, b ModuleDigest) bool {
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

type moduleDigest struct {
	moduleDigestType ModuleDigestType
	bufcasDigest     bufcas.Digest
	// Cache as we call String pretty often.
	// We could do this lazily but not worth it.
	stringValue string
}

// validation should occur outside of this function.
func newModuleDigest(moduleDigestType ModuleDigestType, bufcasDigest bufcas.Digest) *moduleDigest {
	return &moduleDigest{
		moduleDigestType: moduleDigestType,
		bufcasDigest:     bufcasDigest,
		stringValue:      moduleDigestType.String() + ":" + hex.EncodeToString(bufcasDigest.Value()),
	}
}

func (d *moduleDigest) Type() ModuleDigestType {
	return d.moduleDigestType
}

func (d *moduleDigest) Value() []byte {
	return d.bufcasDigest.Value()
}

func (d *moduleDigest) String() string {
	return d.stringValue
}

func (*moduleDigest) isModuleDigest() {}

type hasModuleDigest interface {
	ModuleDigest() (ModuleDigest, error)
}

// getB5ModuleDigest computes a b5 ModuleDigest for the given set of module files and dependencies.
//
// A ModuleDigest is a composite digest of all Module Files, and all Module dependencies.
//
// All Files are added to a bufcas.Manifest, which is then turned into a bufcas.Digest.
// The file bufcas.Digest, along with all ModuleDigests of the dependencies, are then sorted,
// and then digested themselves as content.
//
// Note that the name of the Module and any of its dependencies has no effect on the ModuleDigest.
func getB5ModuleDigest[H hasModuleDigest, S ~[]H](
	ctx context.Context,
	bucketWithStorageMatcherApplied storage.ReadBucket,
	deps S,
) (ModuleDigest, error) {
	// First, compute the shake256 bufcas.Digest of the files. This will include a
	// sorted list of file names and their digests.
	filesDigest, err := getFilesDigestForB5ModuleDigest(ctx, bucketWithStorageMatcherApplied)
	if err != nil {
		return nil, err
	}
	// Next, we get the b5 digests of all the dependencies and sort their string representations.
	depModuleDigestStrings := make([]string, len(deps))
	for i, dep := range deps {
		depModuleDigest, err := dep.ModuleDigest()
		if err != nil {
			return nil, err
		}
		if depModuleDigest.Type() != ModuleDigestTypeB5 {
			// Even if the buf.lock file had a b4 digest, we should still end up retrieving the b5
			// digest from the BSR, we should never have a b5 digest here.
			return nil, syserror.Newf("trying to compute b5 Digest with dependency digest of type %v", depModuleDigest.Type())
		}
		depModuleDigestStrings[i] = depModuleDigest.String()
	}
	sort.Strings(depModuleDigestStrings)
	// Now, place the file digest first, then the sorted dependency digests afterwards.
	digestStrings := append([]string{filesDigest.String()}, depModuleDigestStrings...)
	// Join these strings together with newlines, and make a new shake256 digest.
	digestOfDigests, err := bufcas.NewDigestForContent(strings.NewReader(strings.Join(digestStrings, "\n")))
	if err != nil {
		return nil, err
	}
	// The resulting digest is a b5 digest.
	return NewModuleDigest(ModuleDigestTypeB5, digestOfDigests)
}

// The bucket should have already been filtered to just module fikes.
func getFilesDigestForB5ModuleDigest(
	ctx context.Context,
	bucketWithStorageMatcherApplied storage.ReadBucket,
) (bufcas.Digest, error) {
	var fileNodes []bufcas.FileNode
	if err := storage.WalkReadObjects(
		ctx,
		// This is extreme defensive programming. We've gone out of our way to make sure
		// that the bucket is already filtered, but it's just too important to mess up here.
		storage.MapReadBucket(bucketWithStorageMatcherApplied, getStorageMatcher(ctx, bucketWithStorageMatcherApplied)),
		"",
		func(readObject storage.ReadObject) error {
			digest, err := bufcas.NewDigestForContent(readObject)
			if err != nil {
				return err
			}
			fileNode, err := bufcas.NewFileNode(readObject.Path(), digest)
			if err != nil {
				return err
			}
			fileNodes = append(fileNodes, fileNode)
			return nil
		},
	); err != nil {
		return nil, err
	}
	manifest, err := bufcas.NewManifest(fileNodes)
	if err != nil {
		return nil, err
	}
	return bufcas.ManifestToDigest(manifest)
}
