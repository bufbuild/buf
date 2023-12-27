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
	"sync"
)

// ModuleKey provides identifying information for a Module.
//
// All fields of a ModuleKey are required. This is as opposed to a Module, where many common
// fields are optional.
//
// ModuleKeys are returned from ModuleKeyProvider, and represent a Module's complete identity.
// They also match to what we store in buf.lock files. ModuleKeys can be used to get Modules
// via a ModuleProvider.
type ModuleKey interface {
	// ModuleFullName returns the full name of the Module.
	ModuleFullName() ModuleFullName
	// CommitID returns the BSR ID of the Commit.
	//
	// This may be empty. However, note that there are certain situations (such as writing
	// v1beta1 or v1 buf.lock files) where this is required. It is up to the caller to verify
	// this is present in those situations.
	CommitID() string
	// ModuleDigest returns the Module digest.
	//
	// Note this is *not* a bufcas.Digest - this is a ModuleDigest. bufcas.Digests are a lower-level
	// type that just deal in terms of files and content. A Moduleigest is a specific algorithm
	// applied to a set of files and dependencies.
	ModuleDigest() (ModuleDigest, error)

	isModuleKey()
}

// NewModuleKey returns a new ModuleKey.
//
// Note that commit is optional.
//
// The ModuleDigest will be loaded lazily if needed. Note this means that NewModuleKey does
// *not* validate the digest. If you need to validate the digest, call ModuleDigest() and evaluate
// the returned error.
func NewModuleKey(
	moduleFullName ModuleFullName,
	commitID string,
	getModuleDigest func() (ModuleDigest, error),
) (ModuleKey, error) {
	return newModuleKey(
		moduleFullName,
		commitID,
		getModuleDigest,
	)
}

// *** PRIVATE ***

type moduleKey struct {
	moduleFullName ModuleFullName
	commitID       string

	getModuleDigest func() (ModuleDigest, error)
}

func newModuleKey(
	moduleFullName ModuleFullName,
	commitID string,
	getModuleDigest func() (ModuleDigest, error),
) (*moduleKey, error) {
	if moduleFullName == nil {
		return nil, errors.New("nil ModuleFullName when constructing ModuleKey")
	}
	return &moduleKey{
		moduleFullName:  moduleFullName,
		commitID:        commitID,
		getModuleDigest: sync.OnceValues(getModuleDigest),
	}, nil
}

func (m *moduleKey) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *moduleKey) CommitID() string {
	return m.commitID
}

func (m *moduleKey) ModuleDigest() (ModuleDigest, error) {
	return m.getModuleDigest()
}

func (*moduleKey) isModuleKey() {}
