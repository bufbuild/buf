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

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
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
	CommitID() string
	// Digest returns the Module digest.
	Digest() (bufcas.Digest, error)

	isModuleKey()
}

// NewModuleKey returns a new ModuleKey, assuming we know all the fields of a ModuleKey already.
//
// Used for reading buf.lock files.
func NewModuleKey(
	moduleFullName ModuleFullName,
	commitID string,
	digest bufcas.Digest,
) (*moduleKey, error) {
	return newModuleKeyForKnownDigest(
		moduleFullName,
		commitID,
		digest,
	)
}

// NewModuleKey returns a new ModuleKey for the given Module.
//
// The given Module must have a ModuleFullName and CommitID, otherwise this will return error.
//
// Mostly used for testing.
func NewModuleKeyForModule(module Module) (*moduleKey, error) {
	return newModuleKeyForLazyDigest(
		module.ModuleFullName(),
		module.CommitID(),
		module.Digest,
	)
}

// *** PRIVATE ***

type moduleKey struct {
	moduleFullName ModuleFullName
	commitID       string

	digest     bufcas.Digest
	digestFunc func() (bufcas.Digest, error)
}

func newModuleKeyForKnownDigest(
	moduleFullName ModuleFullName,
	commitID string,
	digest bufcas.Digest,
) (*moduleKey, error) {
	return newModuleKey(moduleFullName, commitID, digest, nil)
}

func newModuleKeyForLazyDigest(
	moduleFullName ModuleFullName,
	commitID string,
	digestFunc func() (bufcas.Digest, error),
) (*moduleKey, error) {
	return newModuleKey(moduleFullName, commitID, nil, digestFunc)
}

// Exactly one of digest and digestFunc should be set.
func newModuleKey(
	moduleFullName ModuleFullName,
	commitID string,
	digest bufcas.Digest,
	digestFunc func() (bufcas.Digest, error),
) (*moduleKey, error) {
	if moduleFullName == nil {
		return nil, errors.New("nil ModuleFullName when constructing ModuleKey")
	}
	if commitID == "" {
		return nil, errors.New("empty commitID when constructing ModuleKey")
	}
	if digest != nil && digestFunc != nil {
		return nil, errors.New("cannot set both digest and digestFunc when constructing ModuleKey")
	}
	if digest == nil && digestFunc == nil {
		return nil, errors.New("nil digest and digestFunc when constructing ModuleKey")
	}
	if digestFunc != nil {
		digestFunc = sync.OnceValues(digestFunc)
	}
	return &moduleKey{
		moduleFullName: moduleFullName,
		commitID:       commitID,
		digest:         digest,
		digestFunc:     digestFunc,
	}, nil
}

func (m *moduleKey) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *moduleKey) CommitID() string {
	return m.commitID
}

func (m *moduleKey) Digest() (bufcas.Digest, error) {
	if m.digest != nil {
		return m.digest, nil
	}
	return m.digestFunc()
}

func (*moduleKey) isModuleKey() {}
