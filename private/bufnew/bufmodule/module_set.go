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
)

// ModuleSet is a set of Modules constructed by a ModuleBuilder.
type ModuleSet interface {
	// Modules returns the Modules in the ModuleSet.
	//
	// These will be sorted by OpaqueID.
	Modules() []Module
	// GetModuleForModuleFullName gets the Module for the ModuleFullName, if it exists.
	//
	// Returns nil if there is no Module with the given ModuleFullName.
	GetModuleForModuleFullName(moduleFullName ModuleFullName) Module
	// GetModuleForOpaqueID gets the MOdule for the OpaqueID, if it exists.
	//
	// Returns nil if there is no Module with the given OpaqueID.
	GetModuleForOpaqueID(opaqueID string) Module
	// GetModuleForBucketID gets the MOdule for the BucketID, if it exists.
	//
	// Returns nil if there is no Module with the given BucketID.
	GetModuleForBucketID(bucketID string) Module
	// GetModuleForDigest gets the MOdule for the Digest, if it exists.
	//
	// Note that this function will result in Digest() being called on every Module in
	// the ModuleSet, which is potentially expensive.
	//
	// Returns nil if there is no Module with the given Digest.
	// Returns an error if there was an error when calling Digest() on a Module.
	GetModuleForDigest(digest bufcas.Digest) (Module, error)

	isModuleSet()
}

// *** PRIVATE ***

type moduleSet struct {
	modules                      []Module
	moduleFullNameStringToModule map[string]Module
	opaqueIDToModule             map[string]Module
	bucketIDToModule             map[string]Module
	getDigestStringToModule      func() (map[string]Module, error)
}

func newModuleSet(
	modules []Module,
) (*moduleSet, error) {
	moduleFullNameStringToModule := make(map[string]Module, len(modules))
	opaqueIDToModule := make(map[string]Module, len(modules))
	bucketIDToModule := make(map[string]Module, len(modules))
	for _, module := range modules {
		if moduleFullName := module.ModuleFullName(); moduleFullName != nil {
			moduleFullNameString := moduleFullName.String()
			if _, ok := moduleFullNameStringToModule[moduleFullNameString]; ok {
				// This should never happen.
				return nil, fmt.Errorf("duplicate ModuleFullName %q when constructing ModuleSet", moduleFullNameString)
			}
			moduleFullNameStringToModule[moduleFullNameString] = module
		}
		opaqueID := module.OpaqueID()
		if _, ok := opaqueIDToModule[opaqueID]; ok {
			// This should never happen.
			return nil, fmt.Errorf("duplicate OpaqueID %q when constructing ModuleSet", opaqueID)
		}
		opaqueIDToModule[opaqueID] = module
		bucketID := module.BucketID()
		if _, ok := bucketIDToModule[bucketID]; ok {
			// This should never happen.
			return nil, fmt.Errorf("duplicate BucketID %q when constructing ModuleSet", bucketID)
		}
		bucketIDToModule[bucketID] = module
	}
	return &moduleSet{
		modules:                      modules,
		moduleFullNameStringToModule: moduleFullNameStringToModule,
		opaqueIDToModule:             opaqueIDToModule,
		bucketIDToModule:             bucketIDToModule,
		getDigestStringToModule: sync.OnceValues(
			func() (map[string]Module, error) {
				digestStringToModule := make(map[string]Module, len(modules))
				for _, module := range modules {
					digest, err := module.Digest()
					if err != nil {
						return nil, err
					}
					digestString := digest.String()
					if _, ok := digestStringToModule[digestString]; ok {
						// Note that because we do this lazily, we're not getting built-in validation here
						// that a ModuleSet has unique Digests until we load them lazily. That's the best
						// we can do, and is likely OK.
						return nil, fmt.Errorf("duplicate Digest %q within ModuleSet", digestString)
					}
					digestStringToModule[digestString] = module
				}
				return digestStringToModule, nil
			},
		),
	}, nil
}

func (m *moduleSet) Modules() []Module {
	c := make([]Module, len(m.modules))
	copy(c, m.modules)
	return c
}

func (m *moduleSet) GetModuleForModuleFullName(moduleFullName ModuleFullName) Module {
	return m.moduleFullNameStringToModule[moduleFullName.String()]
}

func (m *moduleSet) GetModuleForOpaqueID(opaqueID string) Module {
	return m.opaqueIDToModule[opaqueID]
}

func (m *moduleSet) GetModuleForBucketID(bucketID string) Module {
	return m.bucketIDToModule[bucketID]
}

func (m *moduleSet) GetModuleForDigest(digest bufcas.Digest) (Module, error) {
	digestStringToModule, err := m.getDigestStringToModule()
	if err != nil {
		return nil, err
	}
	return digestStringToModule[digest.String()], nil
}

func (*moduleSet) isModuleSet() {}
