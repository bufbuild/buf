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

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syncext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// Module presents a BSR module.
type Module interface {
	// ModuleReadBucket allows for reading of a Module's files.
	//
	// A Module consists of .proto files, documentation file(s), and license file(s). All of these
	// are accessible via the functions on ModuleReadBucket.
	//
	// This bucket is not self-contained - it requires the files from dependencies to be so.
	//
	// A ModuleReadBucket directly derived from a Module will always have at least one .proto file.
	// If this is not the case, WalkFileInfos will return an error when called.
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
	// However, this is always present for remote Modules.
	//
	// At least one of ModuleFullName and BucketID will always be present. Use OpaqueID
	// as an always-present identifier.
	ModuleFullName() ModuleFullName
	// CommitID returns the BSR ID of the Commit.
	//
	// A CommitID is always a dashless UUID.
	// The CommitID converted to using dashes is the ID of the Commit on the BSR.
	// May be empty. Callers should not rely on this value being present.
	//
	// If ModuleFullName is nil, this will always be empty.
	CommitID() string

	// ModuleDigest returns the Module digest.
	//
	// Note this is *not* a bufcas.Digest - this is a ModuleDigest. bufcas.Digests are a lower-level
	// type that just deal in terms of files and content. A ModuleDigest is a specific algorithm
	// applied to a set of files and dependencies.
	ModuleDigest() (ModuleDigest, error)

	// ModuleDeps returns the dependencies for this specific Module.
	//
	// Includes transitive dependencies. Use ModuleDep.IsDirect() to determine in a dependency is direct
	// or transitive.
	//
	// This list is pruned - only Modules that this Module actually depends on (either directly or transitively)
	// via import statements within its .proto files will be returned.
	//
	// Dependencies with the same ModuleFullName will always have the same Commits and ModuleDigests.
	//
	// Sorted by OpaqueID.
	ModuleDeps() ([]ModuleDep, error)

	// IsTarget returns true if the Module is a targeted Module.
	//
	// Modules are either targets or non-targets.
	// Modules directly returned from a ModuleProvider will always be marked as targets.
	// Modules created file ModuleSetBuilders may or may not be marked as targets.
	//
	// Files within a targeted Module can be targets or non-targets themselves (non-target = import).
	// FileInfos have a function FileInfo.IsTargetFile() to denote if they are targets.
	// Note that no Files from a Module will have IsTargetFile() set to true if
	// IsTarget() is false.
	//
	// If specific Files were not targeted but the Module was targeted, all Files in the Module
	// will have FileInfo.IsTargetFile() set to true, and this function will return all Files
	// that WalkFileInfos does.
	//
	// Note that a Module may be targeted but have none of its files targeted - this can occur
	// when path filtering occurs, but no paths given matched any paths in the Module, but
	// the Module itself was targeted.
	IsTarget() bool

	// IsLocal return true if the Module is a local Module.
	//
	// Modules are either local or remote.
	//
	// A local Module is one which was built from sources from the "local context", such
	// a Workspace containing Modules, or a ModuleNode in a CreateCommiteRequest. Local
	// Modules are important for understanding what Modules to push, and what modules to
	// check declared dependencies for unused dependencies.
	//
	// A remote Module is one which was not contained in the local context, such as
	// dependencies specified in a buf.lock (with no correspoding Module in the Workspace),
	// or a DepNode in a CreateCommitRequest with no corresponding ModuleNode.
	//
	// Remote Modules will always have ModuleFullNames.
	IsLocal() bool

	// ModuleSet returns the ModuleSet that this Module is contained within.
	//
	// Always present.
	ModuleSet() ModuleSet

	// Called in newModuleSet.
	setModuleSet(ModuleSet)

	// withIsTarget returns a copy of the Module with the specified target value.
	//
	// Do not expose publicly! This should only be called by ModuleSet.WithTargetOpaqueIDs.
	// Exposing this directly publicly can have unintended consequences - Modules have a
	// parent ModuleSet, which is self-contained, and a copy of a Module inside a ModuleSet
	// that itself has the same ModuleSet will break the expected pattern of the references.
	withIsTarget(isTarget bool) (Module, error)
	isModule()
}

// ModuleToModuleKey returns a new ModuleKey for the given Module.
//
// The given Module must have a ModuleFullName and CommitID, otherwise this will return error.
func ModuleToModuleKey(module Module) (ModuleKey, error) {
	return newModuleKey(
		module.ModuleFullName(),
		module.CommitID(),
		module.ModuleDigest,
	)
}

// ModuleToSelfContainedModuleReadBucketWithOnlyProtoFiles converts the Module to a
// ModuleReadBucket that contains all the .proto files of the Module and its dependencies.
//
// Targeting information will remain the same. Note that this means that the result ModuleReadBucket
// may have no target files! This can occur when path filtering was applied, but the path filters did
// not match any files in the Module, and none of the Module's files were targeted.
// It can also happen if the Module nor any of its dependencies were targeted.
//
// *** THIS IS PROBABLY NOT THE FUNCTION YOU ARE LOOKING FOR. *** You probably want
// ModuleSetToModuleReadBucketWithOnlyProtoFiles to convert a ModuleSet/Workspace to a
// ModuleReadBucket. This function is used for cases where we want to create an Image
// specifically for one Module, such as when we need to associate LintConfig and BreakingConfig
// on a per-Module basis for buf lint and buf breaking. See bufctl.Controller, which is likely
// the only place this should be used outside of testing.
func ModuleToSelfContainedModuleReadBucketWithOnlyProtoFiles(module Module) (ModuleReadBucket, error) {
	modules := []Module{module}
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return nil, err
	}
	for _, moduleDep := range moduleDeps {
		modules = append(modules, moduleDep)
	}
	return newMultiModuleReadBucket(
		slicesext.Map(
			modules,
			func(module Module) ModuleReadBucket {
				return ModuleReadBucketWithOnlyProtoFiles(module)
			},
		),
		true,
	), nil
}

// ModuleDirectModuleDeps is a convenience function that returns only the direct dependencies of the Module.
func ModuleDirectModuleDeps(module Module) ([]ModuleDep, error) {
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return nil, err
	}
	return slicesext.Filter(
		moduleDeps,
		func(moduleDep ModuleDep) bool { return moduleDep.IsDirect() },
	), nil
}

// ModuleRemoteModuleDeps is a convenience function that returns only the remote dependencies of the Module.
//
// This can be used for v1 buf.yamls to determine what needs to be in the buf.lock.
func ModuleRemoteModuleDeps(module Module) ([]ModuleDep, error) {
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return nil, err
	}
	return slicesext.Filter(
		moduleDeps,
		func(moduleDep ModuleDep) bool { return !moduleDep.IsLocal() },
	), nil
}

// *** PRIVATE ***

// module

type module struct {
	ModuleReadBucket

	ctx            context.Context
	logger         *zap.Logger
	getBucket      func() (storage.ReadBucket, error)
	bucketID       string
	moduleFullName ModuleFullName
	commitID       string
	isTarget       bool
	isLocal        bool

	moduleSet ModuleSet

	getModuleDigest func() (ModuleDigest, error)
	getModuleDeps   func() ([]ModuleDep, error)
}

// must set ModuleReadBucket after constructor via setModuleReadBucket
func newModule(
	ctx context.Context,
	logger *zap.Logger,
	// This function must already be filtered to include only module files and must be syncext.OnceValues wrapped!
	syncOnceValuesGetBucketWithStorageMatcherApplied func() (storage.ReadBucket, error),
	bucketID string,
	moduleFullName ModuleFullName,
	commitID string,
	isTarget bool,
	isLocal bool,
	targetPaths []string,
	targetExcludePaths []string,
	protoFileTargetPath string,
	includePackageFiles bool,
) (*module, error) {
	// TODO: get these validations into a common place
	if protoFileTargetPath != "" && (len(targetPaths) > 0 || len(targetExcludePaths) > 0) {
		return nil, syserror.Newf("cannot set both protoFileTargetPath %q and either targetPaths %v or targetExcludePaths %v", protoFileTargetPath, targetPaths, targetExcludePaths)
	}
	if protoFileTargetPath != "" && normalpath.Ext(protoFileTargetPath) != ".proto" {
		return nil, syserror.Newf("protoFileTargetPath %q is not a .proto file", protoFileTargetPath)
	}
	if bucketID == "" && moduleFullName == nil {
		return nil, syserror.New("bucketID was empty and moduleFullName was nil when constructing a Module, one of these must be set")
	}
	if !isLocal && moduleFullName == nil {
		return nil, syserror.New("moduleFullName not present when constructing a remote Module")
	}
	if moduleFullName == nil && commitID != "" {
		return nil, syserror.New("moduleFullName not present and commitID present when constructing a remote Module")
	}
	if commitID != "" {
		if err := validateCommitID(commitID); err != nil {
			return nil, err
		}
	}
	module := &module{
		ctx:            ctx,
		logger:         logger,
		getBucket:      syncOnceValuesGetBucketWithStorageMatcherApplied,
		bucketID:       bucketID,
		moduleFullName: moduleFullName,
		commitID:       commitID,
		isTarget:       isTarget,
		isLocal:        isLocal,
	}
	moduleReadBucket, err := newModuleReadBucketForModule(
		ctx,
		logger,
		syncOnceValuesGetBucketWithStorageMatcherApplied,
		module,
		targetPaths,
		targetExcludePaths,
		protoFileTargetPath,
		includePackageFiles,
	)
	if err != nil {
		return nil, err
	}
	module.ModuleReadBucket = moduleReadBucket
	module.getModuleDigest = syncext.OnceValues(newGetModuleDigestFuncForModule(module))
	module.getModuleDeps = syncext.OnceValues(newGetModuleDepsFuncForModule(module))
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

func (m *module) ModuleDigest() (ModuleDigest, error) {
	return m.getModuleDigest()
}

func (m *module) ModuleDeps() ([]ModuleDep, error) {
	return m.getModuleDeps()
}

func (m *module) IsTarget() bool {
	return m.isTarget
}

func (m *module) IsLocal() bool {
	return m.isLocal
}

func (m *module) ModuleSet() ModuleSet {
	return m.moduleSet
}

func (m *module) withIsTarget(isTarget bool) (Module, error) {
	// We don't just call newModule directly as we don't want to double syncext.OnceValues stuff.
	newModule := &module{
		ctx:            m.ctx,
		logger:         m.logger,
		getBucket:      m.getBucket,
		bucketID:       m.bucketID,
		moduleFullName: m.moduleFullName,
		commitID:       m.commitID,
		isTarget:       isTarget,
		isLocal:        m.isLocal,
	}
	moduleReadBucket, ok := m.ModuleReadBucket.(*moduleReadBucket)
	if !ok {
		return nil, syserror.Newf("expected ModuleReadBucket to be a *moduleReadBucket but was a %T", m.ModuleReadBucket)
	}
	newModule.ModuleReadBucket = moduleReadBucket.withModule(newModule)
	newModule.getModuleDigest = syncext.OnceValues(newGetModuleDigestFuncForModule(newModule))
	newModule.getModuleDeps = syncext.OnceValues(newGetModuleDepsFuncForModule(newModule))
	return newModule, nil
}

func (m *module) setModuleSet(moduleSet ModuleSet) {
	m.moduleSet = moduleSet
}

func (*module) isModule() {}

func newGetModuleDigestFuncForModule(module *module) func() (ModuleDigest, error) {
	return func() (ModuleDigest, error) {
		bucket, err := module.getBucket()
		if err != nil {
			return nil, err
		}
		moduleDeps, err := module.ModuleDeps()
		if err != nil {
			return nil, err
		}
		return getB5ModuleDigest(module.ctx, bucket, moduleDeps)
	}
}

func newGetModuleDepsFuncForModule(module *module) func() ([]ModuleDep, error) {
	return func() ([]ModuleDep, error) {
		return getModuleDeps(module.ctx, module.logger, module)
	}
}
