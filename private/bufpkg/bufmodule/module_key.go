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
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syncext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
)

// ModuleKey provides identifying information for a Module.
//
// ModuleKeys are returned from ModuleKeyProvider, and represent a Module's complete identity.
// They also match to what we store in buf.lock files. ModuleKeys can be used to get Modules
// via a ModuleProvider.
type ModuleKey interface {
	// String returns "registry/owner/name:dashlessCommitID".
	fmt.Stringer

	// ModuleFullName returns the full name of the Module.
	//
	// Always present.
	ModuleFullName() ModuleFullName
	// CommitID returns the ID of the Commit.
	//
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// Always present, that is CommitID().IsNil() will always be false.
	CommitID() uuid.UUID
	// Digest returns the Module digest.
	//
	// Note this is *not* a bufcas.Digest - this is a Digest. bufcas.Digests are a lower-level
	// type that just deal in terms of files and content. A ModuleDigest is a specific algorithm
	// applied to a set of files and dependencies.
	Digest() (Digest, error)

	isModuleKey()
}

// NewModuleKey returns a new ModuleKey.
//
// The Digest will be loaded lazily if needed. Note this means that NewModuleKey does
// *not* validate the digest. If you need to validate the digest, call Digest() and evaluate
// the returned error.
func NewModuleKey(
	moduleFullName ModuleFullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (ModuleKey, error) {
	return newModuleKey(
		moduleFullName,
		commitID,
		getDigest,
	)
}

// UniqueDigestTypeForModuleKeys returns the single DigestType for the Digests on the ModuleKeys.
//
// If the ModuleKeys have different DigestTypes, an error is returned.
// If the ModuleKeys slice is empty, an error is returned.
func UniqueDigestTypeForModuleKeys(moduleKeys []ModuleKey) (DigestType, error) {
	if len(moduleKeys) == 0 {
		return 0, syserror.New("empty moduleKeys passed to UniqueDigestTypeForModuleKeys")
	}
	digests, err := slicesext.MapError(moduleKeys, ModuleKey.Digest)
	if err != nil {
		return 0, err
	}
	digestType := digests[0].Type()
	for _, digest := range digests[1:] {
		if digestType != digest.Type() {
			return 0, fmt.Errorf(
				"different digest types detected where the same digest type must be used: %v, %v\n%s",
				digestType,
				digest.Type(),
				strings.Join(slicesext.Map(moduleKeys, ModuleKey.String), "\n"),
			)
		}
	}
	return digestType, nil
}

// ModuleKeyToCommitKey converts a ModuleKey to a CommitKey.
//
// This is purely lossy - a ModuleKey has more information than a CommitKey, and a
// CommitKey does not have any information that a ModuleKey does not have.
func ModuleKeyToCommitKey(moduleKey ModuleKey) (CommitKey, error) {
	digest, err := moduleKey.Digest()
	if err != nil {
		return nil, err
	}
	return newCommitKey(moduleKey.ModuleFullName().Registry(), moduleKey.CommitID(), digest.Type())
}

// *** PRIVATE ***

type moduleKey struct {
	moduleFullName ModuleFullName
	commitID       uuid.UUID

	getDigest func() (Digest, error)
}

func newModuleKey(
	moduleFullName ModuleFullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (*moduleKey, error) {
	if moduleFullName == nil {
		return nil, errors.New("nil ModuleFullName when constructing ModuleKey")
	}
	if commitID.IsNil() {
		return nil, errors.New("empty commitID when constructing ModuleKey")
	}
	return newModuleKeyNoValidate(moduleFullName, commitID, getDigest), nil
}

func newModuleKeyNoValidate(
	moduleFullName ModuleFullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) *moduleKey {
	return &moduleKey{
		moduleFullName: moduleFullName,
		commitID:       commitID,
		getDigest:      syncext.OnceValues(getDigest),
	}
}

func (m *moduleKey) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *moduleKey) CommitID() uuid.UUID {
	return m.commitID
}

func (m *moduleKey) Digest() (Digest, error) {
	return m.getDigest()
}

func (m *moduleKey) String() string {
	return m.moduleFullName.String() + ":" + uuidutil.ToDashless(m.commitID)
}

func (*moduleKey) isModuleKey() {}
