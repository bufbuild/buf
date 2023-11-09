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
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// Module presents a BSR module.
type Module interface {
	// ModuleInfo contains a Module's optional ModuleFullName, optional commit ID, and Digest.
	ModuleInfo

	// ModuleReadBucket allows for reading of a Module's files.
	//
	// A Module consists of .proto files, documentation file(s), and license file(s). All of these
	// are accessible via the functions on ModuleReadBucket.
	//
	// This bucket is not self-contained - it requires the files from dependencies to be so.
	//
	// This package currently exposes functionality to walk just the .proto files, and get the singular
	// documentation and license files, via WalkProtoFileInfos, GetDocFile, and GetLicenseFile.
	//
	// GetDocFile and GetLicenseFile may change in the future if other paths are accepted for
	// documentation or licenses, or if we allow multiple documentation or license files to
	// exist within a Module (currently, only one of each is allowed).
	ModuleReadBucket

	// ModuleDeps returns the dependency list for this specific module.
	//
	// This list is pruned - only Modules that this Module actually depends on via import statements
	// within its .proto files will be returned.
	//
	// Dependencies with the same ModuleFullName will always have the same commits and digests.
	//
	// The order of returned list of Modules will be stable between invocations, but should
	// not be considered to be sorted in any way.
	ModuleDeps() ([]ModuleDep, error)

	// ModuleSet returns the ModuleSet that this Module is contained within, if it was
	// constructed from a ModuleSet.
	//
	// If the Module was solely retrieved from a ModuleProvider, this will be nil.
	ModuleSet() ModuleSet
	// OpaqueID returns an unstructured ID that can uniquely identify a Module relative
	// to other Modules it was built with from a ModuleSetBuilder.
	//
	// An OpaqueID can be used to denote expected uniqueness of content; if two Modules
	// have different IDs, they should be expected to be logically different Modules.
	//
	// If two Modules have the same ModuleFullName, they will have the same OpaqueID. This is
	// useful when differentiating between a remotely-downloaded Module and a Module in a workspace.
	//
	// This ID's structure should not be relied upon, and is not a globally-unique identifier.
	// It's uniqueness property only applies to the lifetime of the Module, and only within
	// Modules commonly built from a ModuleSetBuilder.
	//
	// This ID is not stable between different invocations; the same Module built twice
	// in two separate ModuleSetBuilder invocations may have different IDs.
	//
	// This ID will never be empty.
	//
	// TODO: update comments
	// This is ModuleFullName -> fall back BucketID, and never empty.
	OpaqueID() string
	// May be empty.
	//
	// TODO: update comments
	// Should not be on ModuleInfo as this only relates to objects created from ModuleSetBuilder,
	// and ModuleInfos can also be constructed by ModuleInfoProviders.
	BucketID() string

	setModuleSet(ModuleSet)
	isModule()
}

// *** PRIVATE ***

// module

type module struct {
	ModuleReadBucket

	cache          *cache
	bucketID       string
	moduleFullName ModuleFullName
	commitID       string

	moduleSet ModuleSet

	getDigest     func() (bufcas.Digest, error)
	getModuleDeps func() ([]ModuleDep, error)
}

// must set ModuleReadBucket after constructor via setModuleReadBucket
func newModule(
	ctx context.Context,
	cache *cache,
	bucketID string,
	bucket storage.ReadBucket,
	moduleFullName ModuleFullName,
	commitID string,
) (*module, error) {
	if bucketID == "" && moduleFullName == nil {
		// This is a system error.
		return nil, errors.New("bucketID was empty and moduleFullName was nil when constructing a Module, one of these must be set")
	}
	module := &module{
		cache:          cache,
		bucketID:       bucketID,
		moduleFullName: moduleFullName,
		commitID:       commitID,
	}
	module.ModuleReadBucket = newModuleReadBucket(
		ctx,
		bucket,
		module,
	)
	module.getDigest = sync.OnceValues(
		func() (bufcas.Digest, error) {
			return moduleDigestB5(ctx, module)
		},
	)
	module.getModuleDeps = sync.OnceValues(
		func() ([]ModuleDep, error) {
			return getModuleDeps(ctx, module.cache, module)
		},
	)
	return module, nil
}

func (m *module) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *module) CommitID() string {
	return m.commitID
}

func (m *module) Digest() (bufcas.Digest, error) {
	return m.getDigest()
}

func (m *module) ModuleDeps() ([]ModuleDep, error) {
	return m.getModuleDeps()
}

func (m *module) ModuleSet() ModuleSet {
	return m.moduleSet
}

func (m *module) OpaqueID() string {
	// We know that one of bucketID and moduleFullName are present via construction.
	//
	// Prefer moduleFullName since modules with the same ModuleFullName should have the same OpaqueID.
	if m.moduleFullName != nil {
		return m.moduleFullName.String()
	}
	return m.bucketID
}

func (m *module) BucketID() string {
	return m.bucketID
}

func (m *module) setModuleSet(moduleSet ModuleSet) {
	m.moduleSet = moduleSet
}

func (*module) isModuleInfo() {}
func (*module) isModule()     {}

// moduleDigestB5 computes a b5 Digest for the given Module.
//
// A Module Digest is a composite Digest of all Module Files, and all Module dependencies.
//
// All Files are added to a bufcas.Manifest, which is then turned into a bufcas.Blob.
// The Digest of the Blob, along with all Digests of the dependencies, are then sorted,
// and then digested themselves as content.
//
// Note that the name of the Module and any of its dependencies has no effect on the Digest.
func moduleDigestB5(ctx context.Context, module Module) (bufcas.Digest, error) {
	fileDigest, err := moduleReadBucketDigestB5(ctx, module)
	if err != nil {
		return nil, err
	}
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return nil, err
	}
	digests := []bufcas.Digest{fileDigest}
	for _, moduleDep := range moduleDeps {
		digest, err := moduleDep.Digest()
		if err != nil {
			return nil, err
		}
		digests = append(digests, digest)
	}

	// NewDigestForDigests deals with sorting.
	// TODO: what about digest type?
	return bufcas.NewDigestForDigests(digests)
}

// getModuleDeps gets the actual dependencies for the Module.
func getModuleDeps(
	ctx context.Context,
	cache *cache,
	module Module,
) ([]ModuleDep, error) {
	depOpaqueIDToModuleDep := make(map[string]ModuleDep)
	if err := getModuleDepsRec(
		ctx,
		cache,
		module,
		make(map[string]struct{}),
		depOpaqueIDToModuleDep,
		true,
	); err != nil {
		return nil, err
	}
	moduleDeps := make([]ModuleDep, 0, len(depOpaqueIDToModuleDep))
	for _, moduleDep := range depOpaqueIDToModuleDep {
		moduleDeps = append(moduleDeps, moduleDep)
	}
	// Sorting by at least Opaque ID to get a consistent return order for a given call.
	sort.Slice(
		moduleDeps,
		func(i int, j int) bool {
			return moduleDeps[i].OpaqueID() < moduleDeps[j].OpaqueID()
		},
	)
	return moduleDeps, nil
}

func getModuleDepsRec(
	ctx context.Context,
	cache *cache,
	module Module,
	// to detect circular imports
	visitedOpaqueIDs map[string]struct{},
	// already discovered deps
	depOpaqueIDToModuleDep map[string]ModuleDep,
	isDirect bool,
) error {
	opaqueID := module.OpaqueID()
	if _, ok := visitedOpaqueIDs[opaqueID]; ok {
		// TODO: detect cycles, this is just making sure we don't recurse
		return nil
	}
	visitedOpaqueIDs[opaqueID] = struct{}{}
	// Doing this BFS so we add all the direct deps to the map first, then if we
	// see a dep later, it will still be a direct dep in the map, but will be ignored
	// on recursive calls.
	var newModuleDeps []ModuleDep
	if err := ModuleReadBucketWithOnlyProtoFiles(module).WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			imports, err := cache.GetImportsForFilePath(ctx, fileInfo.Path())
			if err != nil {
				return err
			}
			for imp := range imports {
				potentialModuleDep, err := cache.GetModuleForFilePath(ctx, imp)
				if err != nil {
					return err
				}
				potentialDepOpaqueID := potentialModuleDep.OpaqueID()
				// If this is in the same module, it's not a dep
				if potentialDepOpaqueID != opaqueID {
					// No longer just potential, now real dep.
					if _, ok := depOpaqueIDToModuleDep[potentialDepOpaqueID]; !ok {
						moduleDep := newModuleDep(potentialModuleDep, isDirect)
						depOpaqueIDToModuleDep[potentialDepOpaqueID] = moduleDep
						newModuleDeps = append(newModuleDeps, moduleDep)
					}
				}
			}
			return nil
		},
	); err != nil {
		return err
	}
	for _, newModuleDep := range newModuleDeps {
		if err := getModuleDepsRec(
			ctx,
			cache,
			newModuleDep,
			visitedOpaqueIDs,
			depOpaqueIDToModuleDep,
			// Always not direct on recursive calls
			false,
		); err != nil {
			return err
		}
	}
	return nil
}

// getUniqueModulesByOpaqueIDWithEarlierPreferred deduplicates the Module list with the earlier
// Modules in the slice being preferred.
//
// Callers should put modules built from local sources earlier than Modules built from remote sources.
//
// Duplication determined based opaqueID, that is if a Module has an equal
// opaqueID, it is considered a duplicate.
//
// We want to account for Modules with the same name but different digests, that is a dep in a workspace
// that has the same name as something in a buf.lock file, we prefer the local dep in the workspace.
//
// When returned, all modules have unique opaqueIDs and Digests.
//
// Note: Modules with the same ModuleFullName will automatically have the same commit and Digest after this,
// as there will be exactly one Module with a given ModuleFullName, given that an OpaqueID will be equal
// for Modules with equal ModuleFullNames.
func getUniqueModulesByOpaqueIDWithEarlierPreferred(ctx context.Context, modules []Module) ([]Module, error) {
	// Digest *cannot* be used here - it's a chicken or egg problem. Computing the digest requires the cache,
	// the cache requires the unique Modules, the unique Modules require this function. This is OK though -
	// we want to add all Modules that we *think* are unique to the cache. If there is a duplicate, it
	// will be detected via cache usage.
	alreadySeenOpaqueIDs := make(map[string]struct{})
	uniqueModules := make([]Module, 0, len(modules))
	for _, module := range modules {
		opaqueID := module.OpaqueID()
		if opaqueID == "" {
			return nil, errors.New("OpaqueID was empty which should never happen")
		}
		if _, ok := alreadySeenOpaqueIDs[opaqueID]; !ok {
			alreadySeenOpaqueIDs[opaqueID] = struct{}{}
			uniqueModules = append(uniqueModules, module)
		}
	}
	// To have stable order.
	sort.Slice(
		uniqueModules,
		func(i int, j int) bool {
			return uniqueModules[i].OpaqueID() < uniqueModules[j].OpaqueID()
		},
	)
	return uniqueModules, nil
}
