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
	"fmt"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// ModuleData presents raw Module data read by ModuleKey.
//
// It is not a fully-formed Module; only ModuleSetBuilders (and ModuleSets) can provide Modules.
//
// A ModuleData generally represents the data on a Module read from the BSR API or a cache.
type ModuleData interface {
	// ModuleKey contains the ModuleKey that was used to download this ModuleData.
	//
	// A ModuleKey from a ModuleData may not have a CommitID set.
	ModuleKey() ModuleKey
	// Bucket returns a Bucket of the Module's files.
	//
	// This bucket is not self-contained - it requires the files from dependencies to be so.
	Bucket() (storage.ReadBucket, error)
	// DeclaredDepModuleKeys returns the declared dependencies for this specific Module.
	DeclaredDepModuleKeys() ([]ModuleKey, error)

	isModuleData()
}

// NewModuleData returns a new ModuleData.
//
// getBucket and getDeclaredDepModuleKeys are meant to be lazily-loaded functions where possible.
func NewModuleData(
	moduleKey ModuleKey,
	getBucket func() (storage.ReadBucket, error),
	getDeclaredDepModuleKeys func() ([]ModuleKey, error),
	options ...ModuleDataOption,
) (ModuleData, error) {
	return newModuleData(
		moduleKey,
		getBucket,
		getDeclaredDepModuleKeys,
		options...,
	)
}

// ModuleDataOption is an option when constructing a ModuleData.
type ModuleDataOption func(*moduleData)

// ModuleDataWithActualDigest returns a new ModuleDataOption that specifies the actual
// Digest of the ModuleData as retrieved.
//
// If this is given, when Bucket() or DeclaredDepModuleKeys() is called, this Digest will
// be compared with the Digest from the ModuleKey, and if they are unequal, an error is returned.
//
// This is used for tamper-proofing.
func ModuleDataWithActualDigest(actualDigest bufcas.Digest) ModuleDataOption {
	return func(moduleData *moduleData) {
		moduleData.actualDigest = actualDigest
	}
}

// *** PRIVATE ***

// moduleData

type moduleData struct {
	moduleKey                ModuleKey
	getBucket                func() (storage.ReadBucket, error)
	getDeclaredDepModuleKeys func() ([]ModuleKey, error)
	actualDigest             bufcas.Digest
	// May be nil after construction.
	checkDigest func() error
}

func newModuleData(
	moduleKey ModuleKey,
	getBucket func() (storage.ReadBucket, error),
	getDeclaredDepModuleKeys func() ([]ModuleKey, error),
	options ...ModuleDataOption,
) (*moduleData, error) {
	moduleData := &moduleData{
		moduleKey:                moduleKey,
		getBucket:                sync.OnceValues(getBucket),
		getDeclaredDepModuleKeys: sync.OnceValues(getDeclaredDepModuleKeys),
	}
	for _, option := range options {
		option(moduleData)
	}
	if moduleData.actualDigest != nil {
		moduleData.checkDigest = sync.OnceValue(
			func() error {
				expectedDigest, err := moduleKey.Digest()
				if err != nil {
					return err
				}
				if !bufcas.DigestEqual(expectedDigest, moduleData.actualDigest) {
					moduleString := moduleKey.ModuleFullName().String()
					if commitID := moduleKey.CommitID(); commitID != "" {
						moduleString = moduleString + ":" + commitID
					}
					return fmt.Errorf(
						"expected Digest %q, got Digest %q, for Module %q",
						expectedDigest.String(),
						moduleData.actualDigest.String(),
						moduleString,
					)
				}
				return nil
			},
		)
	}
	return moduleData, nil
}

func (m *moduleData) ModuleKey() ModuleKey {
	return m.moduleKey
}

func (m *moduleData) Bucket() (storage.ReadBucket, error) {
	if m.checkDigest != nil {
		if err := m.checkDigest(); err != nil {
			return nil, err
		}
	}
	return m.getBucket()
}

func (m *moduleData) DeclaredDepModuleKeys() ([]ModuleKey, error) {
	// Do we need to tamper-proof when getting declared deps? Probably yes - this is
	// data that could be tampered with.
	//
	// Note that doing so kills some of our lazy-loading, as we call DeclaredDepModuleKeys
	// in ModuleSetBuilder right away. However, we still do the lazy-loading here, in the case
	// where ModuleData is loaded outside of a ModuleSetBuilder and users may defer calling this
	// function if it is not needed.
	if m.checkDigest != nil {
		if err := m.checkDigest(); err != nil {
			return nil, err
		}
	}
	return m.getDeclaredDepModuleKeys()
}

func (*moduleData) isModuleData() {}
