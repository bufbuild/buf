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
	"sort"
	"sync/atomic"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/multierr"
	"go.uber.org/zap"
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
	// The dependencies of the Module will are automatically added to the ModuleSet.
	// Note, however, that Modules added with AddLocalModule always take precedence,
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
	Build() (ModuleSet, error)

	isModuleSetBuilder()
}

// NewModuleSetBuilder returns a new ModuleSetBuilder.
func NewModuleSetBuilder(
	ctx context.Context,
	logger *zap.Logger,
	moduleDataProvider ModuleDataProvider,
) ModuleSetBuilder {
	return newModuleSetBuilder(ctx, logger, moduleDataProvider)
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
func LocalModuleWithModuleFullNameAndCommitID(moduleFullName ModuleFullName, commitID string) LocalModuleOption {
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
	logger             *zap.Logger
	moduleDataProvider ModuleDataProvider

	addedModules []*addedModule
	errs         []error
	buildCalled  atomic.Bool
}

func newModuleSetBuilder(
	ctx context.Context,
	logger *zap.Logger,
	moduleDataProvider ModuleDataProvider,
) *moduleSetBuilder {
	return &moduleSetBuilder{
		ctx:                ctx,
		logger:             logger,
		moduleDataProvider: moduleDataProvider,
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
	if localModuleOptions.moduleFullName == nil && localModuleOptions.commitID != "" {
		return b.addError(syserror.New("cannot set commitID without ModuleFullName when calling AddLocalModule"))
	}
	if !isTarget && (len(localModuleOptions.targetPaths) > 0 || len(localModuleOptions.targetExcludePaths) > 0) {
		return b.addError(syserror.New("cannot set TargetPaths for a non-target Module when calling AddLocalModule"))
	}
	if !isTarget && localModuleOptions.protoFileTargetPath != "" {
		return b.addError(syserror.New("cannot set ProtoFileTargetPath for a non-target Module when calling AddLocalModule"))
	}
	if localModuleOptions.protoFileTargetPath != "" &&
		(len(localModuleOptions.targetPaths) > 0 || len(localModuleOptions.targetExcludePaths) > 0) {
		return b.addError(syserror.New("cannot set TargetPaths and ProtoFileTargetPath when calling AddLocalModule"))
	}
	if localModuleOptions.protoFileTargetPath != "" &&
		normalpath.Ext(localModuleOptions.protoFileTargetPath) != ".proto" {
		return b.addError(syserror.Newf("proto file target %q is not a .proto file", localModuleOptions.protoFileTargetPath))
	}
	// TODO: normalize and validate all paths
	module, err := newModule(
		b.ctx,
		b.logger,
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

func (b *moduleSetBuilder) Build() (ModuleSet, error) {
	if !b.buildCalled.CompareAndSwap(false, true) {
		return nil, errBuildAlreadyCalled
	}
	if len(b.errs) > 0 {
		return nil, multierr.Combine(b.errs...)
	}
	if len(b.addedModules) == 0 {
		return nil, syserror.New("no Modules added to ModuleSetBuilder")
	}
	if slicesext.Count(b.addedModules, func(m *addedModule) bool { return m.IsTarget() }) < 1 {
		return nil, syserror.New("no Modules were targeted in ModuleSetBuilder")
	}

	// Get the unique modules, preferring targets over non-targets, and local over remote.
	addedModules, err := getUniqueSortedAddedModulesByOpaqueID(b.ctx, b.addedModules)
	if err != nil {
		return nil, err
	}
	alreadySeenOpaqueIDs := make(map[string]struct{})
	for _, addedModule := range addedModules {
		if addedModule.IsLocal() {
			// Let getTransitiveModulesForRemoteModuleKey know that we've already seen
			// all the local Modules, so no need to try to fetch them.
			alreadySeenOpaqueIDs[addedModule.OpaqueID()] = struct{}{}
		}
	}
	modules := make([]Module, 0, len(addedModules))
	for _, addedModule := range addedModules {
		if addedModule.IsLocal() {
			// If the module was local, just add it - we're done.
			modules = append(modules, addedModule.localModule)
		} else {
			// If the module was remote, actually build the module and its dependencies IF
			// we have not already added those dependencies.
			transitiveModules, err := b.getTransitiveModulesForRemoteModuleKey(
				addedModule.remoteModuleKey,
				addedModule.remoteTargetPaths,
				addedModule.remoteTargetExcludePaths,
				addedModule.isTarget,
				alreadySeenOpaqueIDs,
			)
			if err != nil {
				return nil, err
			}
			// We know these modules are already not in the list, courtesy of alreadySeenOpaqueIDs.
			modules = append(modules, transitiveModules...)
		}
	}
	// We know modules is a unique slice, but the sorting may be messed up now courtesy
	// of our transitive module retrieval.
	sort.Slice(
		modules,
		func(i int, j int) bool {
			return modules[i].OpaqueID() < modules[j].OpaqueID()
		},
	)
	return newModuleSet(modules)
}

// getTransitiveRemoteModules gets the Module for the ModuleKey, plus any of its dependencies
// if those dependencies are not in the alreadySeenOpaqueIDs list.
//
// This function recursively calls itself with isTarget = false and no targetPaths or targetExcludePaths
// for dependencies of the remote Module. No recursive call is made for modules already in the alreadySeenOpaqueIDs.
func (b *moduleSetBuilder) getTransitiveModulesForRemoteModuleKey(
	remoteModuleKey ModuleKey,
	remoteTargetPaths []string,
	remoteTargetExcludePaths []string,
	isTarget bool,
	// This includes all the local Modules off the bat.
	alreadySeenOpaqueIDs map[string]struct{},
) ([]Module, error) {
	// We know that moduleKey.ModuleFullName().String() is the opaque ID for remote modules.
	opaqueID := remoteModuleKey.ModuleFullName().String()
	if _, ok := alreadySeenOpaqueIDs[opaqueID]; ok {
		// No need to process this or its dependencies. If we have already added this module
		// via a local module, we expect that we've added all its declared dependencies
		// via AddRemoteModule, and do not need to add any more dependencies. If we have
		// already added this module via a remote module, this function has already been called.
		return nil, nil
	}
	alreadySeenOpaqueIDs[opaqueID] = struct{}{}

	moduleDatas, err := GetModuleDatasForModuleKeys(
		b.ctx,
		b.moduleDataProvider,
		remoteModuleKey,
	)
	if err != nil {
		return nil, err
	}
	if len(moduleDatas) != 1 {
		return nil, syserror.Newf("expected 1 ModuleData, got %d", len(moduleDatas))
	}
	moduleData := moduleDatas[0]
	if moduleData.ModuleKey().ModuleFullName() == nil {
		return nil, syserror.New("got nil ModuleFullName for a ModuleKey returned from a ModuleDataProvider")
	}
	if remoteModuleKey.ModuleFullName().String() != moduleData.ModuleKey().ModuleFullName().String() {
		return nil, syserror.Newf(
			"mismatched ModuleFullName from ModuleDataProvider: input %q, output %q",
			remoteModuleKey.ModuleFullName().String(),
			moduleData.ModuleKey().ModuleFullName().String(),
		)
	}

	// TODO: normalize and validate all paths
	module, err := newModule(
		b.ctx,
		b.logger,
		// ModuleData.Bucket has sync.OnceValues and getStorageMatchers applied since it can
		// only be constructed via NewModuleData.
		//
		// TODO: This is a bit shady.
		moduleData.Bucket,
		"",
		moduleData.ModuleKey().ModuleFullName(),
		moduleData.ModuleKey().CommitID(),
		isTarget,
		false,
		remoteTargetPaths,
		remoteTargetExcludePaths,
		"",
		false,
	)
	if err != nil {
		return nil, err
	}

	// The return list of modules.
	allModules := []Module{module}
	declaredDepModuleKeys, err := moduleData.DeclaredDepModuleKeys()
	if err != nil {
		return nil, err
	}
	for _, declaredDepModuleKey := range declaredDepModuleKeys {
		// Not a target Module.
		// If this Module is a target, this will be added by the caller.
		//
		// Do not filter on paths, i.e. no options - paths only apply to the module as added by the caller.
		depModules, err := b.getTransitiveModulesForRemoteModuleKey(
			declaredDepModuleKey,
			nil,
			nil,
			false,
			alreadySeenOpaqueIDs,
		)
		if err != nil {
			return nil, err
		}
		allModules = append(allModules, depModules...)
	}
	return allModules, nil
}

func (b *moduleSetBuilder) addError(err error) *moduleSetBuilder {
	b.errs = append(b.errs, err)
	return b
}

func (*moduleSetBuilder) isModuleSetBuilder() {}

// getUniqueSortedModulesByOpaqueID deduplicates and sorts the addedModule list.
//
// Modules that are targets are preferred, followed by Modules that are local.
// Otherwise, Modules earlier in the slice are preferred.
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
func getUniqueSortedAddedModulesByOpaqueID(ctx context.Context, addedModules []*addedModule) ([]*addedModule, error) {
	// sort.SliceStable keeps equal elements in their original order, so this does
	// not affect the "earlier preferred" property.
	//
	// However, after this, we can really apply "earlier" preferred to denote "prefer targets over
	// non-targets, then prefer local over remote."
	sort.SliceStable(
		addedModules,
		func(i int, j int) bool {
			m1 := addedModules[i]
			m2 := addedModules[j]
			// If this ever comes up in the future: by preferring remote targets over local non-targets,
			// we are in a situation where we might have a local module, but we use the remote module
			// anyways, which leads to a BSR call we didn't want to make. See addedModule documentation.
			// We're making the bet that if we did add a remote target module, we had a good reason
			// to do so (i.e. we want that version of the module for some reason) so we're going
			// to prefer it.
			if m1.IsTarget() && !m2.IsTarget() {
				return true
			}
			if !m1.IsTarget() && m2.IsTarget() {
				return false
			}
			if m1.IsLocal() && !m2.IsLocal() {
				return true
			}
			// includes if !m1.IsLocal() && m2.IsLocal()
			return false
		},
	)
	// Digest *cannot* be used here - it's a chicken or egg problem. Computing the digest requires the cache,
	// the cache requires the unique Modules, the unique Modules require this function. This is OK though -
	// we want to add all Modules that we *think* are unique to the cache. If there is a duplicate, it
	// will be detected via cache usage.
	alreadySeenOpaqueIDs := make(map[string]struct{})
	uniqueAddedModules := make([]*addedModule, 0, len(addedModules))
	for _, addedModule := range addedModules {
		opaqueID := addedModule.OpaqueID()
		if opaqueID == "" {
			return nil, syserror.New("OpaqueID was empty which should never happen")
		}
		if _, ok := alreadySeenOpaqueIDs[opaqueID]; !ok {
			alreadySeenOpaqueIDs[opaqueID] = struct{}{}
			uniqueAddedModules = append(uniqueAddedModules, addedModule)
		}
	}
	sort.Slice(
		uniqueAddedModules,
		func(i int, j int) bool {
			return uniqueAddedModules[i].OpaqueID() < uniqueAddedModules[j].OpaqueID()
		},
	)
	return uniqueAddedModules, nil
}

type localModuleOptions struct {
	moduleFullName      ModuleFullName
	commitID            string
	targetPaths         []string
	targetExcludePaths  []string
	protoFileTargetPath string
	includePackageFiles bool
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

// addedModule represents a Module that was added.
// This is needed because when we add a remote Module, we make
// a call out to the API to get the ModuleData by ModuleKey. However, if we are in
// a situation where we have a v1 workspace with named modules, but those modules
// do not actually exist in the BSR, and only in the workspace, AND we have a buf.lock
// that represents those modules, we don't want to actually do the work to retrieve
// the Module from the BSR, as in the end, the local Module in the workspace will win
// out in getUniqueModulesByOpaqueID. Even if this weren't the case, we don't want to
// make unnecessary BSR calls. So, instead of making the call, we store the information
// that we will need in getUniqueModulesByOpaqueID, and once we've filtered out the
// modules we don't need, we actually create the remote Module. At this point, any modules
// that were both local (in the workspace) and remote (via a buf.lock) will have the
// buf.lock-added Modules filtered out, and no BSR call will be made.
type addedModule struct {
	localModule              Module
	remoteModuleKey          ModuleKey
	remoteTargetPaths        []string
	remoteTargetExcludePaths []string
	isTarget                 bool
}

func newLocalAddedModule(
	localModule Module,
	isTarget bool,
) *addedModule {
	return &addedModule{
		localModule: localModule,
		isTarget:    isTarget,
	}
}

func newRemoteAddedModule(
	remoteModuleKey ModuleKey,
	remoteTargetPaths []string,
	remoteTargetExcludePaths []string,
	isTarget bool,
) *addedModule {
	return &addedModule{
		remoteModuleKey:          remoteModuleKey,
		remoteTargetPaths:        remoteTargetPaths,
		remoteTargetExcludePaths: remoteTargetExcludePaths,
		isTarget:                 isTarget,
	}
}

func (a *addedModule) IsLocal() bool {
	return a.localModule != nil
}

func (a *addedModule) IsTarget() bool {
	return a.isTarget
}

func (a *addedModule) OpaqueID() string {
	if a.remoteModuleKey != nil {
		return a.remoteModuleKey.ModuleFullName().String()
	}
	return a.localModule.OpaqueID()
}
