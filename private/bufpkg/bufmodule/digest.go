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
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// DigestTypeB4 represents the b4 module digest type.
	//
	// This represents the pre-refactor shake256 digest type, and the string value of
	// this is "shake256" for backwards-compatibility reasons.
	DigestTypeB4 DigestType = iota + 1
	// DigestTypeB5 represents the b5 digest type.
	//
	// This is the newest digest type, and should generally be used. The string value
	// of this is "b5".
	DigestTypeB5
)

var (
	// AllDigestTypes are all known DigestTypes.
	AllDigestTypes = []DigestType{
		DigestTypeB4,
		DigestTypeB5,
	}
	digestTypeToString = map[DigestType]string{
		DigestTypeB4: "shake256",
		DigestTypeB5: "b5",
	}
	stringToDigestType = map[string]DigestType{
		"shake256": DigestTypeB4,
		"b5":       DigestTypeB5,
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
			typeString: "module digest type",
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

	isDigest()
}

// NewDigest creates a new Digest.
func NewDigest(digestType DigestType, bufcasDigest bufcas.Digest) (Digest, error) {
	switch digestType {
	case DigestTypeB4, DigestTypeB5:
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
			typeString: "module digest",
			input:      s,
			err:        errors.New(`must in the form "digest_type:digest_hex_value"`),
		}
	}
	digestType, err := ParseDigestType(digestTypeString)
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
	switch digestType {
	case DigestTypeB4, DigestTypeB5:
		bufcasDigest, err := bufcas.NewDigest(value)
		if err != nil {
			return nil, err
		}
		return NewDigest(digestType, bufcasDigest)
	default:
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

func getB4Digest(
	ctx context.Context,
	bucketWithStorageMatcherApplied storage.ReadBucket,
	v1BufYAMLObjectData ObjectData,
	v1BufLockObjectData ObjectData,
) (Digest, error) {
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
	for _, objectData := range []ObjectData{
		v1BufYAMLObjectData,
		v1BufLockObjectData,
	} {
		if objectData == nil {
			// We may not have object data for one of these files, this is valid.
			continue
		}
		digest, err := bufcas.NewDigestForContent(bytes.NewReader(objectData.Data()))
		if err != nil {
			return nil, err
		}
		fileNode, err := bufcas.NewFileNode(objectData.Name(), digest)
		if err != nil {
			return nil, err
		}
		fileNodes = append(fileNodes, fileNode)
	}
	manifest, err := bufcas.NewManifest(fileNodes)
	if err != nil {
		return nil, err
	}
	bufcasDigest, err := bufcas.ManifestToDigest(manifest)
	if err != nil {
		return nil, err
	}
	return NewDigest(DigestTypeB4, bufcasDigest)
}

func getB5DigestForBucketAndModuleDeps(
	ctx context.Context,
	bucketWithStorageMatcherApplied storage.ReadBucket,
	moduleDeps []ModuleDep,
) (Digest, error) {
	depDigests, err := slicesext.MapError(
		moduleDeps,
		func(moduleDep ModuleDep) (Digest, error) {
			return moduleDep.Digest(DigestTypeB5)
		},
	)
	if err != nil {
		return nil, err
	}
	return getB5DigestForBucketAndDepDigests(ctx, bucketWithStorageMatcherApplied, depDigests)
}

func getB5DigestForBucketAndDepModuleKeys(
	ctx context.Context,
	bucketWithStorageMatcherApplied storage.ReadBucket,
	depModuleKeys []ModuleKey,
) (Digest, error) {
	depDigests, err := slicesext.MapError(
		depModuleKeys,
		func(moduleKey ModuleKey) (Digest, error) {
			return moduleKey.Digest()
		},
	)
	if err != nil {
		return nil, err
	}
	return getB5DigestForBucketAndDepDigests(ctx, bucketWithStorageMatcherApplied, depDigests)
}

// getB5Digest computes a b5 Digest for the given set of module files and dependencies.
//
// A Digest is a composite digest of all Module Files, and all Module dependencies.
//
// All Files are added to a bufcas.Manifest, which is then turned into a bufcas.Digest.
// The file bufcas.Digest, along with all Digests of the dependencies, are then sorted,
// and then digested themselves as content.
//
// Note that the name of the Module and any of its dependencies has no effect on the Digest.
func getB5DigestForBucketAndDepDigests(
	ctx context.Context,
	bucketWithStorageMatcherApplied storage.ReadBucket,
	depDigests []Digest,
) (Digest, error) {
	// First, compute the shake256 bufcas.Digest of the files. This will include a
	// sorted list of file names and their digests.
	filesDigest, err := getFilesDigestForB5Digest(ctx, bucketWithStorageMatcherApplied)
	if err != nil {
		return nil, err
	}
	if filesDigest.Type() != bufcas.DigestTypeShake256 {
		return nil, syserror.Newf("trying to compute b5 Digest with files digest of type %v", filesDigest.Type())
	}
	// Next, we get the b5 digests of all the dependencies and sort their string representations.
	depDigestStrings, err := slicesext.MapError(
		depDigests,
		func(digest Digest) (string, error) {
			if digest.Type() != DigestTypeB5 {
				// Even if the buf.lock file had a b4 digest, we should still end up retrieving the b5
				// digest from the BSR, we should never have a b5 digest here.
				return "", syserror.Newf("trying to compute b5 Digest with dependency digest of type %v", digest.Type())
			}
			return digest.String(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	sort.Strings(depDigestStrings)
	// Now, place the file digest first, then the sorted dependency digests afterwards.
	digestStrings := append([]string{filesDigest.String()}, depDigestStrings...)
	// Join these strings together with newlines, and make a new shake256 digest.
	digestOfDigests, err := bufcas.NewDigestForContent(strings.NewReader(strings.Join(digestStrings, "\n")))
	if err != nil {
		return nil, err
	}
	// The resulting digest is a b5 digest.
	return NewDigest(DigestTypeB5, digestOfDigests)
}

// The bucket should have already been filtered to just module files.
func getFilesDigestForB5Digest(
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
