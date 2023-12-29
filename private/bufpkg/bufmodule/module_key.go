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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/syncext"
)

// ModuleKey provides identifying information for a Module.
//
// ModuleKeys are returned from ModuleKeyProvider, and represent a Module's complete identity.
// They also match to what we store in buf.lock files. ModuleKeys can be used to get Modules
// via a ModuleProvider.
type ModuleKey interface {
	// String returns "registry/owner/name:commitID".
	fmt.Stringer

	// ModuleFullName returns the full name of the Module.
	//
	// Always present.
	ModuleFullName() ModuleFullName
	// CommitID returns the ID of the Commit.
	//
	// A CommitID is always a dashless UUID.
	// The CommitID converted to using dashes is the ID of the Commit on the BSR.
	//
	// Always present.
	CommitID() string
	// Digest returns the Module digest.
	//
	// Note this is *not* a bufcas.Digest - this is a Digest. bufcas.Digests are a lower-level
	// type that just deal in terms of files and content. A Moduleigest is a specific algorithm
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
	commitID string,
	getDigest func() (Digest, error),
) (ModuleKey, error) {
	return newModuleKey(
		moduleFullName,
		commitID,
		getDigest,
	)
}

// OptionalModuleKey is a result from a ModuleKeyProvider.
//
// It returns whether or not the ModuleKey was found, and a non-nil
// ModuleKey if the ModuleKey was found.
type OptionalModuleKey interface {
	ModuleKey() ModuleKey
	Found() bool

	isOptionalModuleKey()
}

// NewOptionalModuleKey returns a new OptionalModuleKey.
//
// As opposed to most functions in this codebase, the input ModuleKey can be nil.
// If it is nil, then Found() will return false.
func NewOptionalModuleKey(moduleKey ModuleKey) OptionalModuleKey {
	return newOptionalModuleKey(moduleKey)
}

// *** PRIVATE ***

type moduleKey struct {
	moduleFullName ModuleFullName
	commitID       string

	getDigest func() (Digest, error)
}

func newModuleKey(
	moduleFullName ModuleFullName,
	commitID string,
	getDigest func() (Digest, error),
) (*moduleKey, error) {
	if moduleFullName == nil {
		return nil, errors.New("nil ModuleFullName when constructing ModuleKey")
	}
	if commitID == "" {
		return nil, errors.New("empty commitID when constructing ModuleKey")
	}
	if err := validateCommitID(commitID); err != nil {
		return nil, err
	}
	return &moduleKey{
		moduleFullName: moduleFullName,
		commitID:       commitID,
		getDigest:      syncext.OnceValues(getDigest),
	}, nil
}

func (m *moduleKey) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *moduleKey) CommitID() string {
	return m.commitID
}

func (m *moduleKey) Digest() (Digest, error) {
	return m.getDigest()
}

func (m *moduleKey) String() string {
	return m.moduleFullName.String() + ":" + m.commitID
}

func (*moduleKey) isModuleKey() {}

type optionalModuleKey struct {
	moduleKey ModuleKey
}

func newOptionalModuleKey(moduleKey ModuleKey) *optionalModuleKey {
	return &optionalModuleKey{
		moduleKey: moduleKey,
	}
}

func (o *optionalModuleKey) ModuleKey() ModuleKey {
	return o.moduleKey
}

func (o *optionalModuleKey) Found() bool {
	return o.moduleKey != nil
}

func (*optionalModuleKey) isOptionalModuleKey() {}
