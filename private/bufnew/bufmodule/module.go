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
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

// Module presents a BSR module.
type Module interface {
	// ModuleReadBucket allows for reading of a Module's files.
	//
	// A Module consists of .proto files, documentation file(s), and license file(s). All of these
	// are accessible via the functions on ModuleReadBucket.
	//
	// This bucket is not self-contained - it requires the files from dependencies to be so.
	ModuleReadBucket

	// OpaqueID returns an unstructured ID that can uniquely identify a Module relative
	// to other Modules it was built with from a ModuleSetBuilder.
	//
	// Always present, regardless of whether a Module was provided by a ModuleProvider,
	// or built with a ModuleSetBuilder.
	//
	// An OpaqueID can be used to denote expected uniqueness of content; if two Modules
	// have different IDs, they should be expected to be logically different Modules.
	//
	// This ID's structure should not be relied upon, and is not a globally-unique identifier.
	// It's uniqueness property only applies to the lifetime of the Module, and only within
	// Modules commonly built from a ModuleSetBuilder.
	//
	// If two Modules have the same ModuleFullName, they will have the same OpaqueID.
	//
	// While this should not be relied upion, this ID is currently equal to the ModuleFullName,
	// and if the ModuleFullName is not present, then the BucketID.
	OpaqueID() string
	// BucketID is an unstructured ID that represents the Bucket that this Module was constructed
	// with via ModuleSetProvider.
	//
	// A BucketID will be unique within a given ModuleSet.
	//
	// This ID's structure should not be relied upon, and is not a globally-unique identifier.
	// It's uniqueness property only applies to the lifetime of the Module, and only within
	// Modules commonly built from a ModuleSetBuilder.
	//
	// May be empty if a Module was not constructed with a Bucket via a ModuleSetProvider.
	BucketID() string
	// ModuleFullName returns the full name of the Module.
	//
	// May be nil. Callers should not rely on this value being present.
	//
	// At least one of ModuleFullName and BucketID will always be present. Use OpaqueID
	// as an always-present identifier.
	ModuleFullName() ModuleFullName
	// CommitID returns the BSR ID of the Commit.
	//
	// May be empty. Callers should not rely on this value being present. If
	// ModuleFullName is nil, this will always be empty.
	CommitID() string
	// Digest returns the Module digest.
	Digest() (bufcas.Digest, error)

	// ModuleDeps returns the dependencies for this specific Module.
	//
	// This list is pruned - only Modules that this Module actually depends on via import statements
	// within its .proto files will be returned.
	//
	// Dependencies with the same ModuleFullName will always have the same Commits and Digests.
	//
	// Sorted by OpaqueID.
	ModuleDeps() ([]ModuleDep, error)

	// IsTargetModule returns true if the Module is a targeted module.
	//
	// Modules are either targets or non-targets.
	// Modules directly returned from a ModuleProvider will always be marked as targets.
	// Modules created file ModuleSetBuilders may or may not be marked as targets.
	//
	// Files within a targeted Module can be targets or non-targets themselves
	// (non-target = import).
	// FileInfos have a function IsTargetFile() to denote if they are targets.
	// Note that no Files from a Module will have IsTargetFile() set to true if
	// Module.IsTargetModule() is false.
	//
	// If specific Files were not targeted but the Module was targeted, all Files in the Module
	// will have IsTargetFile() set to true, and this function will return all Files
	// that WalkFileInfos does.
	IsTargetModule() bool

	// ModuleSet returns the ModuleSet that this Module is contained within, if it was
	// constructed from a ModuleSet.
	//
	// May be nil. If the Module was solely retrieved from a ModuleProvider, this will be nil.
	ModuleSet() ModuleSet

	setModuleSet(ModuleSet)
	isModule()
}

// ModuleToModuleKey returns a new ModuleKey for the given Module.
//
// The given Module must have a ModuleFullName and CommitID, otherwise this will return error.
//
// Mostly used for testing.
func ModuleToModuleKey(module Module) (ModuleKey, error) {
	return newModuleKeyForLazyDigest(
		module.ModuleFullName(),
		module.CommitID(),
		module.Digest,
	)
}

// ModuleDirectModuleDeps is a convenience function that returns only the direct dependencies of the Module.
func ModuleDirectModuleDeps(module Module) ([]ModuleDep, error) {
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return nil, err
	}
	return slicesextended.Filter(
		moduleDeps,
		func(moduleDep ModuleDep) bool { return moduleDep.IsDirect() },
	), nil
}

// ModuleToModuleReadBucketWithOnlyProtoFilesIncludingAllDeps converts the Module into a new ModuleReadBucket
// that not only has the .proto files from the Module, but also has the .proto files from all the Module's dependencies.
//
// The input Module must be a targeted Module.
//
// All the files from the dependencies will be non-targets; only the files from the input Module will
// be targets.
//
// All the files in the resulting ModuleReadBucket will be .proto files.
//
// TODO: is this actually needed? We may want to work just in terms of ModuleSets.
func ModuleToModuleReadBucketWithOnlyProtoFilesIncludingAllDeps(module Module) (ModuleReadBucket, error) {
	return nil, errors.New("TODO")
}

// *** PRIVATE ***

// module

type module struct {
	ModuleReadBucket

	cache          *cache
	bucketID       string
	moduleFullName ModuleFullName
	commitID       string

	isTargetModule bool
	moduleSet      ModuleSet

	getBucket     func() (storage.ReadBucket, error)
	getDigest     func() (bufcas.Digest, error)
	getModuleDeps func() ([]ModuleDep, error)
}

// must set ModuleReadBucket after constructor via setModuleReadBucket
func newModule(
	ctx context.Context,
	cache *cache,
	getBucket func() (storage.ReadBucket, error),
	bucketID string,
	moduleFullName ModuleFullName,
	commitID string,
	isTargetModule bool,
	targetPaths []string,
	targetExcludePaths []string,
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
		isTargetModule: isTargetModule,
	}
	module.ModuleReadBucket = newModuleReadBucket(
		ctx,
		getBucket,
		module,
		targetPaths,
		targetExcludePaths,
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

func (m *module) IsTargetModule() bool {
	return m.isTargetModule
}

func (m *module) ModuleSet() ModuleSet {
	return m.moduleSet
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

func moduleReadBucketDigestB5(ctx context.Context, moduleReadBucket ModuleReadBucket) (bufcas.Digest, error) {
	var fileNodes []bufcas.FileNode
	if err := moduleReadBucket.WalkFileInfos(
		ctx,
		func(fileInfo FileInfo) (retErr error) {
			file, err := moduleReadBucket.GetFile(ctx, fileInfo.Path())
			if err != nil {
				return err
			}
			defer func() {
				retErr = multierr.Append(retErr, file.Close())
			}()
			// TODO: what about digest type?
			digest, err := bufcas.NewDigestForContent(file)
			if err != nil {
				return err
			}
			fileNode, err := bufcas.NewFileNode(fileInfo.Path(), digest)
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
	manifestBlob, err := bufcas.ManifestToBlob(manifest)
	if err != nil {
		return nil, err
	}
	return manifestBlob.Digest(), nil
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
