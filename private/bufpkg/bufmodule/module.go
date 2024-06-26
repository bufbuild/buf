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
	"context"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syncext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
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
	// An OpaqueID can be used as a human-readable identifier of the Module, suitable for printing
	// to a console. However, the OpaqueID may contain information on local directory structure, so
	// do not log or print it in contexts where such information may be sensitive.
	//
	// An OpaqueID's structure should not be relied upon, and is not a globally-unique identifier.
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
	// A BucketID's structure should not be relied upon, and is not a globally-unique identifier.
	// It's uniqueness property only applies to the lifetime of the Module, and only within
	// Modules commonly built from a ModuleSetBuilder.
	//
	// A BucketID may contain information on local directory structure, so do not log or print it
	// in contexts where such information may be sensitive.
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
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// May be empty, that is CommitID().IsNil() may be true.
	// Callers should not rely on this value being present.
	//
	// If ModuleFullName is nil, this will always be empty.
	CommitID() uuid.UUID

	// Digest returns the Module digest for the given DigestType.
	//
	// Note this is *not* a bufcas.Digest - this is a Digest. bufcas.Digests are a lower-level
	// type that just deal in terms of files and content. A Digest is a specific algorithm
	// applied to a set of files and dependencies.
	Digest(DigestType) (Digest, error)

	// ModuleDeps returns the dependencies for this specific Module.
	//
	// Includes transitive dependencies. Use ModuleDep.IsDirect() to determine if a dependency is direct
	// or transitive.
	//
	// This list is pruned - only Modules that this Module actually depends on (either directly or transitively)
	// via import statements within its .proto files will be returned.
	//
	// Dependencies with the same ModuleFullName will always have the same Commits and Digests.
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

	// V1Beta1OrV1BufYAMLObjectData returns the original source buf.yaml associated with this Module, if the
	// Module was backed with a v1beta1 or v1 buf.yaml.
	//
	// This may not be set, in the cases where a v1 Module was built with no buf.yaml (ie the defaults),
	// or with a v2 Module.
	//
	// This file content is just used for dependency calculations. It is not parsed.
	V1Beta1OrV1BufYAMLObjectData() (ObjectData, error)
	// V1Beta1OrV1BufLockObjectData returns the original source buf.lock associated with this Module, if the
	// Module was backed with a v1beta1 or v1 buf.lock.
	//
	// This may not be set, in the cases where a buf.lock was not present due to no dependencies, or
	// with a v2 Module.
	//
	// This file content is just used for dependency calculations. It is not parsed.
	V1Beta1OrV1BufLockObjectData() (ObjectData, error)

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
func ModuleToModuleKey(module Module, digestType DigestType) (ModuleKey, error) {
	return newModuleKey(
		module.ModuleFullName(),
		module.CommitID(),
		func() (Digest, error) {
			return module.Digest(digestType)
		},
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
	return newMultiProtoFileModuleReadBucket(modules, true), nil
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

// *** PRIVATE ***

// module

type module struct {
	ModuleReadBucket

	ctx                        context.Context
	getBucket                  func() (storage.ReadBucket, error)
	bucketID                   string
	moduleFullName             ModuleFullName
	commitID                   uuid.UUID
	isTarget                   bool
	isLocal                    bool
	getV1BufYAMLObjectData     func() (ObjectData, error)
	getV1BufLockObjectData     func() (ObjectData, error)
	getDeclaredDepModuleKeysB5 func() ([]ModuleKey, error)

	moduleSet ModuleSet

	digestTypeToGetDigest map[DigestType]func() (Digest, error)
	getModuleDeps         func() ([]ModuleDep, error)
}

// must set ModuleReadBucket after constructor via setModuleReadBucket
func newModule(
	ctx context.Context,
	// This function must already be filtered to include only module files and must be syncext.OnceValues wrapped!
	syncOnceValuesGetBucketWithStorageMatcherApplied func() (storage.ReadBucket, error),
	bucketID string,
	moduleFullName ModuleFullName,
	commitID uuid.UUID,
	isTarget bool,
	isLocal bool,
	getV1BufYAMLObjectData func() (ObjectData, error),
	getV1BufLockObjectData func() (ObjectData, error),
	getDeclaredDepModuleKeysB5 func() ([]ModuleKey, error),
	targetPaths []string,
	targetExcludePaths []string,
	protoFileTargetPath string,
	includePackageFiles bool,
) (*module, error) {
	// TODO FUTURE: get these validations into a common place
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
	if moduleFullName == nil && !commitID.IsNil() {
		return nil, syserror.New("moduleFullName not present and commitID present when constructing a remote Module")
	}

	normalizeAndValidateIfNotEmpty := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		return normalpath.NormalizeAndValidate(path)
	}
	targetPaths, err := slicesext.MapError(targetPaths, normalizeAndValidateIfNotEmpty)
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	targetExcludePaths, err = slicesext.MapError(targetExcludePaths, normalizeAndValidateIfNotEmpty)
	if err != nil {
		return nil, syserror.Wrap(err)
	}
	protoFileTargetPath, err = normalizeAndValidateIfNotEmpty(protoFileTargetPath)
	if err != nil {
		return nil, syserror.Wrap(err)
	}

	module := &module{
		ctx:                        ctx,
		getBucket:                  syncOnceValuesGetBucketWithStorageMatcherApplied,
		bucketID:                   bucketID,
		moduleFullName:             moduleFullName,
		commitID:                   commitID,
		isTarget:                   isTarget,
		isLocal:                    isLocal,
		getV1BufYAMLObjectData:     syncext.OnceValues(getV1BufYAMLObjectData),
		getV1BufLockObjectData:     syncext.OnceValues(getV1BufLockObjectData),
		getDeclaredDepModuleKeysB5: syncext.OnceValues(getDeclaredDepModuleKeysB5),
	}
	moduleReadBucket, err := newModuleReadBucketForModule(
		ctx,
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
	module.digestTypeToGetDigest = newSyncOnceValueDigestTypeToGetDigestFuncForModule(module)
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

func (m *module) CommitID() uuid.UUID {
	return m.commitID
}

func (m *module) Digest(digestType DigestType) (Digest, error) {
	getDigest, ok := m.digestTypeToGetDigest[digestType]
	if !ok {
		return nil, syserror.Newf("DigestType %v was not in module.digestTypeToGetDigest", digestType)
	}
	return getDigest()
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

func (m *module) V1Beta1OrV1BufYAMLObjectData() (ObjectData, error) {
	return m.getV1BufYAMLObjectData()
}

func (m *module) V1Beta1OrV1BufLockObjectData() (ObjectData, error) {
	return m.getV1BufLockObjectData()
}

func (m *module) ModuleSet() ModuleSet {
	return m.moduleSet
}

func (m *module) withIsTarget(isTarget bool) (Module, error) {
	// We don't just call newModule directly as we don't want to double syncext.OnceValues stuff.
	newModule := &module{
		ctx:                        m.ctx,
		getBucket:                  m.getBucket,
		bucketID:                   m.bucketID,
		moduleFullName:             m.moduleFullName,
		commitID:                   m.commitID,
		isTarget:                   isTarget,
		isLocal:                    m.isLocal,
		getV1BufYAMLObjectData:     m.getV1BufYAMLObjectData,
		getV1BufLockObjectData:     m.getV1BufLockObjectData,
		getDeclaredDepModuleKeysB5: m.getDeclaredDepModuleKeysB5,
	}
	moduleReadBucket, ok := m.ModuleReadBucket.(*moduleReadBucket)
	if !ok {
		return nil, syserror.Newf("expected ModuleReadBucket to be a *moduleReadBucket but was a %T", m.ModuleReadBucket)
	}
	newModule.ModuleReadBucket = moduleReadBucket.withModule(newModule)
	newModule.digestTypeToGetDigest = newSyncOnceValueDigestTypeToGetDigestFuncForModule(newModule)
	newModule.getModuleDeps = syncext.OnceValues(newGetModuleDepsFuncForModule(newModule))
	return newModule, nil
}

func (m *module) setModuleSet(moduleSet ModuleSet) {
	m.moduleSet = moduleSet
}

func (*module) isModule() {}

func newSyncOnceValueDigestTypeToGetDigestFuncForModule(module *module) map[DigestType]func() (Digest, error) {
	m := make(map[DigestType]func() (Digest, error))
	for digestType := range digestTypeToString {
		m[digestType] = syncext.OnceValues(newGetDigestFuncForModuleAndDigestType(module, digestType))
	}
	return m
}

func newGetDigestFuncForModuleAndDigestType(module *module, digestType DigestType) func() (Digest, error) {
	return func() (Digest, error) {
		bucket, err := module.getBucket()
		if err != nil {
			return nil, err
		}
		switch digestType {
		case DigestTypeB4:
			v1BufYAMLObjectData, err := module.getV1BufYAMLObjectData()
			if err != nil {
				return nil, err
			}
			v1BufLockObjectData, err := module.getV1BufLockObjectData()
			if err != nil {
				return nil, err
			}
			return getB4Digest(module.ctx, bucket, v1BufYAMLObjectData, v1BufLockObjectData)
		case DigestTypeB5:
			moduleDeps, err := module.ModuleDeps()
			if err != nil {
				return nil, err
			}
			// For remote modules to have consistent B5 digests, they must not change the digests of their
			// dependencies based on the local workspace. Use the pruned b5 module keys from
			// ModuleData.DeclaredDepModuleKeys to calculate the digest.
			if !module.isLocal {
				declaredDepModuleKeys, err := module.getDeclaredDepModuleKeysB5()
				if err != nil {
					return nil, err
				}
				moduleDepFullNames := make(map[string]struct{}, len(moduleDeps))
				for _, dep := range moduleDeps {
					fullName := dep.ModuleFullName()
					if fullName == nil {
						return nil, syserror.Newf("remote module dependencies should have full names")
					}
					moduleDepFullNames[fullName.String()] = struct{}{}
				}
				prunedDepModuleKeys := make([]ModuleKey, 0, len(declaredDepModuleKeys))
				for _, dep := range declaredDepModuleKeys {
					if _, ok := moduleDepFullNames[dep.ModuleFullName().String()]; ok {
						prunedDepModuleKeys = append(prunedDepModuleKeys, dep)
					}
				}
				return getB5DigestForBucketAndDepModuleKeys(module.ctx, bucket, prunedDepModuleKeys)
			}
			return getB5DigestForBucketAndModuleDeps(module.ctx, bucket, moduleDeps)
		default:
			return nil, syserror.Newf("unknown DigestType: %v", digestType)
		}
	}
}

func newGetModuleDepsFuncForModule(module *module) func() ([]ModuleDep, error) {
	return func() ([]ModuleDep, error) {
		return getModuleDeps(module.ctx, module)
	}
}
