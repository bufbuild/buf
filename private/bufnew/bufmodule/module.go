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
	// are accessible via the functions on ModuleReadBucket. As such, the FileTypes() function will
	// return FileTypeProto, FileTypeDoc, FileTypeLicense.
	//
	// This bucket is not self-contained - it requires the files from dependencies to be so. As such,
	// IsProtoFilesSelfContained() returns false.
	//
	// This package currently exposes functionality to walk just the .proto files, and get the singular
	// documentation and license files, via WalkProtoFileInfos, GetDocFile, and GetLicenseFile.
	//
	// GetDocFile and GetLicenseFile may change in the future if other paths are accepted for
	// documentation or licenses, or if we allow multiple documentation or license files to
	// exist within a Module (currently, only one of each is allowed).
	ModuleReadBucket

	// DepModules returns the dependency list for this specific module.
	//
	// This list is pruned - only Modules that this Module actually depends on via import statements
	// within its .proto files will be returned.
	//
	// Dependencies with the same ModuleFullName will always have the same commits and digests.
	DepModules() ([]Module, error)

	opaqueID() string
	isModule()
}

// *** PRIVATE ***

// module

type module struct {
	ModuleReadBucket

	bucketID       string
	moduleFullName ModuleFullName
	commitID       string
	cache          *cache

	getDigest     func() (bufcas.Digest, error)
	getDepModules func() ([]Module, error)
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
	if bucketID == "" {
		return nil, errors.New("bucketID was empty when constructing a new bucket-based Module")
	}
	module := &module{
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
	module.getDepModules = sync.OnceValues(
		func() ([]Module, error) {
			return getActualDepModules(ctx, module.cache, module)
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

func (m *module) DepModules() ([]Module, error) {
	return m.getDepModules()
}

func (m *module) opaqueID() string {
	if m.moduleFullName != nil {
		return m.moduleFullName.String()
	}
	// We know bucketID is present via construction.
	return m.bucketID
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
	depModules, err := module.DepModules()
	if err != nil {
		return nil, err
	}
	digests := []bufcas.Digest{fileDigest}
	for _, depModule := range depModules {
		digest, err := depModule.Digest()
		if err != nil {
			return nil, err
		}
		digests = append(digests, digest)
	}

	// NewDigestForDigests deals with sorting.
	// TODO: what about digest type?
	return bufcas.NewDigestForDigests(digests)
}

// getActualDepModules gets the actual dependencies for the Module.
//
// TODO: go through imports, figure out which dep modules contain those imports, return just that list
// Make sure to memoize file -> imports mapping, and pass it around the ModuleBuilder.
func getActualDepModules(
	ctx context.Context,
	cache *cache,
	module Module,
) ([]Module, error) {
	depOpaqueIDToDepModule := make(map[string]Module)
	if err := getActualDepModulesRec(
		ctx,
		cache,
		module,
		make(map[string]struct{}),
		depOpaqueIDToDepModule,
	); err != nil {
		return nil, err
	}
	depModules := make([]Module, 0, len(depOpaqueIDToDepModule))
	for _, depModule := range depOpaqueIDToDepModule {
		depModules = append(depModules, depModule)
	}
	// Sorting by at least Opaque ID to get a consistent return order for a given call.
	// We should probably sort by digest TODO.
	sort.Slice(
		depModules,
		func(i int, j int) bool {
			return depModules[i].opaqueID() < depModules[j].opaqueID()
		},
	)
	return depModules, nil
}

func getActualDepModulesRec(
	ctx context.Context,
	cache *cache,
	module Module,
	// to detect circular imports
	visitedOpaqueIDs map[string]struct{},
	// already discovered deps
	depOpaqueIDToDepModule map[string]Module,
) error {
	opaqueID := module.opaqueID()
	if _, ok := visitedOpaqueIDs[opaqueID]; ok {
		// TODO: detect cycles, this is just making sure we don't recurse
		return nil
	}
	visitedOpaqueIDs[opaqueID] = struct{}{}
	// Just optimizing the number of recursive calls a bit/doing this BFS.
	var newDepModules []Module
	if err := ModuleReadBucketWithOnlyProtoFiles(module).WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) error {
			imports, err := cache.GetImportsForFilePath(ctx, fileInfo.Path())
			if err != nil {
				return err
			}
			for imp := range imports {
				potentialDepModule, err := cache.GetModuleForFilePath(ctx, imp)
				if err != nil {
					return err
				}
				potentialDepOpaqueID := potentialDepModule.opaqueID()
				// If this is in the same module, it's not a dep
				if potentialDepOpaqueID != opaqueID {
					// No longer just potential, now real dep.
					if _, ok := depOpaqueIDToDepModule[potentialDepOpaqueID]; !ok {
						depOpaqueIDToDepModule[potentialDepOpaqueID] = potentialDepModule
						newDepModules = append(newDepModules, potentialDepModule)
					}
				}
			}
			return nil
		},
	); err != nil {
		return err
	}
	for _, newDepModule := range newDepModules {
		if err := getActualDepModulesRec(
			ctx,
			cache,
			newDepModule,
			visitedOpaqueIDs,
			depOpaqueIDToDepModule,
		); err != nil {
			return err
		}
	}
	return nil
}
