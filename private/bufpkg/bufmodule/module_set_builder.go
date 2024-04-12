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
	"sync/atomic"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/multierr"
)

var (
	errBuildAlreadyCalled = syserror.New("ModuleSetBuilder.Build has already been called")
)

// ModuleSetBuilder builds ModuleSets.
//
// It is the effective primary entrypoint for this package.
//
// Modules are either targets or non-targets.
// A target Module is a module that we are directly targeting for operations.
// Targets the specific Modules within a Workspace that you are targeting.
//
// Modules are also either local or remote.
//
// A local Module is one which was built from sources from the "local context", such
// a Workspace containing Modules, or a ModuleNode in a CreateCommiteRequest. Local
// Modules are important for understanding what Modules to push, and what modules to
// check declared dependencies for unused dependencies.
//
// A remote Module is one which was not contained in the local context, such as
// dependencies specified in a buf.lock (with no correspoding Module in the Workspace),
// or a DepNode in a CreateCommitRequest with no corresponding ModuleNode. A module
// retrieved from a ModuleDataProvider via a ModuleKey is always remote.
type ModuleSetBuilder interface {
	// AddLocalModule adds a new local Module for the given Bucket.
	//
	// The Bucket used to construct the module will only be read for .proto files,
	// license file(s), and documentation file(s).
	//
	// The BucketID is required. If LocalModuleWithModuleFullName.* is used, the OpaqueID will
	// use this ModuleFullName, otherwise the OpaqueID will be the BucketID.
	//
	// The dependencies of the Module are unknown, since bufmodule does not parse configuration,
	// and therefore the dependencies of the Module are *not* automatically added to the ModuleSet.
	//
	// If you are using a v1 buf.yaml-backed Module, be sure to use LocalModuleWithBufYAMLObjectData and
	// LocalModuleWithBufLockObjectData!
	//
	// Returns the same ModuleSetBuilder.
	AddLocalModule(
		bucket storage.ReadBucket,
		bucketID string,
		isTarget bool,
		options ...LocalModuleOption,
	) ModuleSetBuilder
	// AddRemoteModule adds a new remote Module for the given ModuleKey.
	//
	// The ModuleDataProvider given to the ModuleSetBuilder at construction time will be used to
	// retrieve this Module.
	//
	// The resulting Module will not have a BucketID but will always have a ModuleFullName.
	//
	// The dependencies of the Module will are *not* automatically added to the ModuleSet.
	// It is the caller's responsibility to add transitive dependencies.
	//
	// Modules added with AddLocalModule always take precedence,
	// so if there are local bucket-based dependencies, these will be used.
	//
	// Remote modules are rarely targets. However, if we are reading a ModuleSet from a
	// ModuleProvider for example with a buf build buf.build/foo/bar call, then this

	// specific Module will be targeted, while its dependencies will not be.
	//
	// Returns the same ModuleSetBuilder.
	AddRemoteModule(
		moduleKey ModuleKey,
		isTarget bool,
		options ...RemoteModuleOption,
	) ModuleSetBuilder
	// Build builds the Modules into a ModuleSet.
	//
	// Any errors from Add* calls will be returned here as well.
	//
	// For future consideration, `Build` can take ...buildOption. A use case for this
	// would be for workspaces to have a unified/top-level README and/or LICENSE file.
	// The workspace at build time can pass `BuildWithREADME` and/or `BuildWithLicense` for
	// the module set. Then each module in the module set can refer to this through the module
	// set as needed.
	Build() (ModuleSet, error)

	isModuleSetBuilder()
}

// NewModuleSetBuilder returns a new ModuleSetBuilder.
func NewModuleSetBuilder(
	ctx context.Context,
	tracer tracing.Tracer,
	moduleDataProvider ModuleDataProvider,
	commitProvider CommitProvider,
) ModuleSetBuilder {
	return newModuleSetBuilder(ctx, tracer, moduleDataProvider, commitProvider)
}

// NewModuleSetForRemoteModule is a convenience function that build a ModuleSet for for a single
// remote Module based on ModuleKey.
//
// The remote Module is targeted.
// All of the remote Module's transitive dependencies are automatically added as non-targets.
func NewModuleSetForRemoteModule(
	ctx context.Context,
	tracer tracing.Tracer,
	graphProvider GraphProvider,
	moduleDataProvider ModuleDataProvider,
	commitProvider CommitProvider,
	moduleKey ModuleKey,
	options ...RemoteModuleOption,
) (ModuleSet, error) {
	moduleSetBuilder := NewModuleSetBuilder(ctx, tracer, moduleDataProvider, commitProvider)
	moduleSetBuilder.AddRemoteModule(moduleKey, true, options...)
	graph, err := graphProvider.GetGraphForModuleKeys(ctx, []ModuleKey{moduleKey})
	if err != nil {
		return nil, err
	}
	if err := graph.WalkNodes(
		func(node ModuleKey, _ []ModuleKey, _ []ModuleKey) error {
			if node.CommitID() != moduleKey.CommitID() {
				// Add the dependency ModuleKey with no path filters.
				moduleSetBuilder.AddRemoteModule(node, false)
			}
			return nil
		},
	); err != nil {
		return nil, err
	}
	return moduleSetBuilder.Build()
}

// LocalModuleOption is an option for AddLocalModule.
type LocalModuleOption func(*localModuleOptions)

// LocalModuleWithModuleFullName returns a new LocalModuleOption that adds the given ModuleFullName to the result Module.
//
// Use LocalModuleWithModuleFullNameAndCommitID if you'd also like to add a CommitID.
func LocalModuleWithModuleFullName(moduleFullName ModuleFullName) LocalModuleOption {
	return func(localModuleOptions *localModuleOptions) {
		localModuleOptions.moduleFullName = moduleFullName
	}
}

// LocalModuleWithModuleFullName returns a new LocalModuleOption that adds the given ModuleFullName and CommitID
// to the result Module.
func LocalModuleWithModuleFullNameAndCommitID(moduleFullName ModuleFullName, commitID uuid.UUID) LocalModuleOption {
	return func(localModuleOptions *localModuleOptions) {
		localModuleOptions.moduleFullName = moduleFullName
		localModuleOptions.commitID = commitID
	}
}

// LocalModuleWithTargetPaths returns a new LocalModuleOption that specifically targets the given paths, and
// specifically excludes the given paths.
//
// Only valid for a targeted Module. If this option is given to a non-target Module, this will
// result in an error during Build().
func LocalModuleWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) LocalModuleOption {
	return func(localModuleOptions *localModuleOptions) {
		localModuleOptions.targetPaths = targetPaths
		localModuleOptions.targetExcludePaths = targetExcludePaths
	}
}

// LocalModuleWithProtoFileTargetPath returns a new LocalModuleOption that specifically targets
// a single .proto file, and optionally targets all other .proto files that are in the same package.
//
// If targetPath is empty, includePackageFiles is ignored.
// Exclusive with LocalModuleWithTargetPaths - only one of these can have a non-empty value.
//
// This is used for ProtoFileRefs only. Do not use this otherwise.
func LocalModuleWithProtoFileTargetPath(
	protoFileTargetPath string,
	includePackageFiles bool,
) LocalModuleOption {
	return func(localModuleOptions *localModuleOptions) {
		localModuleOptions.protoFileTargetPath = protoFileTargetPath
		localModuleOptions.includePackageFiles = includePackageFiles
	}
}

// LocalModuleWithV1Beta1OrV1BufYAMLObjectData returns a new LocalModuleOption that attaches the original
// source buf.yaml file associated with this module for v1 or v1beta1 buf.yaml-backed Modules.
//
// If a buf.yaml exists on disk, should be set for Modules backed with a v1beta1 or v1 buf.yaml. It is possible
// that a Module has no buf.yaml (if it was built from defaults), in which case this will not be set.
//
// For Modules backed with v2 buf.yamls, this should not be set.
//
// This file content is just used for dependency calculations. It is not parsed.
func LocalModuleWithV1Beta1OrV1BufYAMLObjectData(v1BufYAMLObjectData ObjectData) LocalModuleOption {
	return func(localModuleOptions *localModuleOptions) {
		localModuleOptions.v1BufYAMLObjectData = v1BufYAMLObjectData
	}
}

// LocalModuleWithV1Beta1OrV1BufLockObjectData returns a new LocalModuleOption that attaches the original
// source buf.local file associated with this Module for v1 or v1beta1 buf.lock-backed Modules.
//
// If a buf.lock exists on disk, should be set for Modules backed with a v1beta1 or v1 buf.lock.
// Note that a buf.lock may not exist for a v1 Module, if there are no dependencies, and in this
// case, this is not set. However, if there is a buf.lock file that was generated, even if it
// had no dependencies, this is set.
//
// For Modules backed with v2 buf.locks, this should not be set.
//
// This file content is just used for dependency calculations. It is not parsed.
func LocalModuleWithV1Beta1OrV1BufLockObjectData(v1BufLockObjectData ObjectData) LocalModuleOption {
	return func(localModuleOptions *localModuleOptions) {
		localModuleOptions.v1BufLockObjectData = v1BufLockObjectData
	}
}

// RemoteModuleOption is an option for AddRemoteModule.
type RemoteModuleOption func(*remoteModuleOptions)

// RemoteModuleWithTargetPaths returns a new RemoteModuleOption that specifically targets the given paths, and
// specifically excludes the given paths.
//
// Only valid for a targeted Module. If this option is given to a non-target Module, this will
// result in an error during Build().
func RemoteModuleWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) RemoteModuleOption {
	return func(remoteModuleOptions *remoteModuleOptions) {
		remoteModuleOptions.targetPaths = targetPaths
		remoteModuleOptions.targetExcludePaths = targetExcludePaths
	}
}

/// *** PRIVATE ***

// moduleSetBuilder

type moduleSetBuilder struct {
	ctx                context.Context
	tracer             tracing.Tracer
	moduleDataProvider ModuleDataProvider
	commitProvider     CommitProvider

	addedModules []*addedModule
	errs         []error
	buildCalled  atomic.Bool
}

func newModuleSetBuilder(
	ctx context.Context,
	tracer tracing.Tracer,
	moduleDataProvider ModuleDataProvider,
	commitProvider CommitProvider,
) *moduleSetBuilder {
	return &moduleSetBuilder{
		ctx:                ctx,
		tracer:             tracer,
		moduleDataProvider: moduleDataProvider,
		commitProvider:     commitProvider,
	}
}

func (b *moduleSetBuilder) AddLocalModule(
	bucket storage.ReadBucket,
	bucketID string,
	isTarget bool,
	options ...LocalModuleOption,
) ModuleSetBuilder {
	if b.buildCalled.Load() {
		return b.addError(errBuildAlreadyCalled)
	}
	if bucketID == "" {
		return b.addError(syserror.New("bucketID is required when calling AddLocalModule"))
	}
	localModuleOptions := newLocalModuleOptions()
	for _, option := range options {
		option(localModuleOptions)
	}
	if localModuleOptions.moduleFullName == nil && !localModuleOptions.commitID.IsNil() {
		return b.addError(syserror.New("cannot set commitID without ModuleFullName when calling AddLocalModule"))
	}
	if !isTarget && (len(localModuleOptions.targetPaths) > 0 || len(localModuleOptions.targetExcludePaths) > 0) {
		return b.addError(syserror.Newf("cannot set TargetPaths for a non-target Module when calling AddLocalModule, bucketID=%q, targetPaths=%v, targetExcludePaths=%v", bucketID, localModuleOptions.targetPaths, localModuleOptions.targetExcludePaths))
	}
	if !isTarget && localModuleOptions.protoFileTargetPath != "" {
		return b.addError(syserror.Newf("cannot set ProtoFileTargetPath for a non-target Module when calling AddLocalModule, bucketID=%q, protoFileTargetPath=%q", bucketID, localModuleOptions.protoFileTargetPath))
	}
	if localModuleOptions.protoFileTargetPath != "" &&
		(len(localModuleOptions.targetPaths) > 0 || len(localModuleOptions.targetExcludePaths) > 0) {
		return b.addError(syserror.Newf("cannot set TargetPaths and ProtoFileTargetPath when calling AddLocalModule, bucketID=%q, protoFileTargetPath=%q, targetPaths=%v, targetExcludePaths=%v", bucketID, localModuleOptions.protoFileTargetPath, localModuleOptions.targetPaths, localModuleOptions.targetExcludePaths))
	}
	if localModuleOptions.protoFileTargetPath != "" &&
		normalpath.Ext(localModuleOptions.protoFileTargetPath) != ".proto" {
		return b.addError(syserror.Newf("proto file target %q is not a .proto file", localModuleOptions.protoFileTargetPath))
	}

	module, err := newModule(
		b.ctx,
		getSyncOnceValuesGetBucketWithStorageMatcherApplied(
			b.ctx,
			func() (storage.ReadBucket, error) {
				return bucket, nil
			},
		),
		bucketID,
		localModuleOptions.moduleFullName,
		localModuleOptions.commitID,
		isTarget,
		true,
		func() (ObjectData, error) { return localModuleOptions.v1BufYAMLObjectData, nil },
		func() (ObjectData, error) { return localModuleOptions.v1BufLockObjectData, nil },
		func() ([]ModuleKey, error) {
			// See comment in added_module.go when we construct remote Modules for why
			// we have this function in the first place.
			return nil, syserror.Newf("getDeclaredDepModuleKeysB5 should never be called for a local Module")
		},
		localModuleOptions.targetPaths,
		localModuleOptions.targetExcludePaths,
		localModuleOptions.protoFileTargetPath,
		localModuleOptions.includePackageFiles,
	)
	if err != nil {
		return b.addError(err)
	}
	b.addedModules = append(
		b.addedModules,
		newLocalAddedModule(
			module,
			isTarget,
		),
	)
	return b
}

func (b *moduleSetBuilder) AddRemoteModule(
	moduleKey ModuleKey,
	isTarget bool,
	options ...RemoteModuleOption,
) ModuleSetBuilder {
	if b.buildCalled.Load() {
		return b.addError(errBuildAlreadyCalled)
	}
	remoteModuleOptions := newRemoteModuleOptions()
	for _, option := range options {
		option(remoteModuleOptions)
	}
	if !isTarget && (len(remoteModuleOptions.targetPaths) > 0 || len(remoteModuleOptions.targetExcludePaths) > 0) {
		return b.addError(syserror.New("cannot set TargetPaths for a non-target Module when calling AddRemoteModule"))
	}

	b.addedModules = append(
		b.addedModules,
		newRemoteAddedModule(
			moduleKey,
			remoteModuleOptions.targetPaths,
			remoteModuleOptions.targetExcludePaths,
			isTarget,
		),
	)
	return b
}

func (b *moduleSetBuilder) Build() (_ ModuleSet, retErr error) {
	ctx, span := b.tracer.Start(b.ctx, tracing.WithErr(&retErr))
	defer span.End()

	if !b.buildCalled.CompareAndSwap(false, true) {
		return nil, errBuildAlreadyCalled
	}
	if len(b.errs) > 0 {
		return nil, multierr.Combine(b.errs...)
	}
	if len(b.addedModules) == 0 {
		// Allow an empty ModuleSet.
		return newModuleSet(b.tracer, nil)
	}
	// If not empty, we need at least one target Module.
	if slicesext.Count(b.addedModules, func(m *addedModule) bool { return m.IsTarget() }) < 1 {
		return nil, syserror.New("no Modules were targeted in ModuleSetBuilder")
	}

	// Get the unique modules, preferring targets over non-targets, and local over remote.
	addedModules, err := getUniqueSortedAddedModulesByOpaqueID(ctx, b.commitProvider, b.addedModules)
	if err != nil {
		return nil, err
	}
	modules, err := slicesext.MapError(
		addedModules,
		func(addedModule *addedModule) (Module, error) {
			return addedModule.ToModule(ctx, b.moduleDataProvider, b.commitProvider)
		},
	)
	if err != nil {
		return nil, err
	}
	return newModuleSet(b.tracer, modules)
}

func (b *moduleSetBuilder) addError(err error) *moduleSetBuilder {
	b.errs = append(b.errs, err)
	return b
}

func (*moduleSetBuilder) isModuleSetBuilder() {}

type localModuleOptions struct {
	moduleFullName      ModuleFullName
	commitID            uuid.UUID
	targetPaths         []string
	targetExcludePaths  []string
	protoFileTargetPath string
	includePackageFiles bool
	v1BufYAMLObjectData ObjectData
	v1BufLockObjectData ObjectData
}

func newLocalModuleOptions() *localModuleOptions {
	return &localModuleOptions{}
}

type remoteModuleOptions struct {
	targetPaths        []string
	targetExcludePaths []string
}

func newRemoteModuleOptions() *remoteModuleOptions {
	return &remoteModuleOptions{}
}
