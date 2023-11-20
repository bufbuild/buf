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
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

var (
	errBuildAlreadyCalled = errors.New("ModuleSetBuilder.Build has already been called")
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
func NewModuleSetBuilder(ctx context.Context, moduleDataProvider ModuleDataProvider) ModuleSetBuilder {
	return newModuleSetBuilder(ctx, moduleDataProvider)
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
	moduleDataProvider ModuleDataProvider

	modules     []Module
	errs        []error
	buildCalled atomic.Bool
}

func newModuleSetBuilder(ctx context.Context, moduleDataProvider ModuleDataProvider) *moduleSetBuilder {
	return &moduleSetBuilder{
		ctx:                ctx,
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
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	if bucketID == "" {
		b.errs = append(b.errs, errors.New("bucketID is required when calling AddLocalModule"))
		return b
	}
	localModuleOptions := newLocalModuleOptions()
	for _, option := range options {
		option(localModuleOptions)
	}
	if localModuleOptions.moduleFullName == nil && localModuleOptions.commitID != "" {
		b.errs = append(b.errs, errors.New("cannot set commitID without ModuleFullName when calling AddLocalModule"))
		return b
	}
	if !isTarget && (len(localModuleOptions.targetPaths) > 0 || len(localModuleOptions.targetExcludePaths) > 0) {
		b.errs = append(b.errs, errors.New("cannot set TargetPaths for a non-target Module when calling AddLocalModule"))
		return b
	}
	module, err := newModule(
		b.ctx,
		func() (storage.ReadBucket, error) {
			return bucket, nil
		},
		bucketID,
		localModuleOptions.moduleFullName,
		localModuleOptions.commitID,
		isTarget,
		true,
		localModuleOptions.targetPaths,
		localModuleOptions.targetExcludePaths,
	)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	b.modules = append(
		b.modules,
		module,
	)
	return b
}

func (b *moduleSetBuilder) AddRemoteModule(
	moduleKey ModuleKey,
	isTarget bool,
	options ...RemoteModuleOption,
) ModuleSetBuilder {
	if b.buildCalled.Load() {
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	remoteModuleOptions := newRemoteModuleOptions()
	for _, option := range options {
		option(remoteModuleOptions)
	}
	if !isTarget && (len(remoteModuleOptions.targetPaths) > 0 || len(remoteModuleOptions.targetExcludePaths) > 0) {
		b.errs = append(b.errs, errors.New("cannot set TargetPaths for a non-target Module when calling AddRemoteModule"))
		return b
	}
	// TODO: we could defer all this work to build, and coalesce ModuleKeys into a single call.
	moduleDatas, err := b.moduleDataProvider.GetModuleDatasForModuleKeys(b.ctx, moduleKey)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	if len(moduleDatas) != 1 {
		b.errs = append(b.errs, fmt.Errorf("expected 1 ModuleData, got %d", len(moduleDatas)))
		return b
	}
	moduleData := moduleDatas[0]
	if moduleData.ModuleKey().ModuleFullName() == nil {
		b.errs = append(b.errs, errors.New("got nil ModuleFullName for a ModuleKey returned from a ModuleDataProvider"))
		return b
	}
	if moduleData.ModuleKey().CommitID() == "" {
		b.errs = append(b.errs, fmt.Errorf("got empty CommitID for ModuleKey with ModuleFullName %q returned from a ModuleDataProvider", moduleData.ModuleKey().ModuleFullName().String()))
		return b
	}
	module, err := newModule(
		b.ctx,
		moduleData.Bucket,
		"",
		moduleData.ModuleKey().ModuleFullName(),
		moduleData.ModuleKey().CommitID(),
		isTarget,
		false,
		remoteModuleOptions.targetPaths,
		remoteModuleOptions.targetExcludePaths,
	)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	b.modules = append(
		b.modules,
		module,
	)
	declaredDepModuleKeys, err := moduleData.DeclaredDepModuleKeys()
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	for _, declaredDepModuleKey := range declaredDepModuleKeys {
		// Not a target Module.
		// If this Module is a target, this will be added by the caller.
		//
		// Do not filter on paths, i.e. no options - paths only apply to the module as added by the caller.
		//
		// We don't need to special-case these - they are lowest priority as they aren't targets and
		// are remote. If a caller adds one of these ModuleKeys as a target, or adds
		// an equivalent Module by as a local Module by Bucket, that add will take precedence.
		b.AddRemoteModule(declaredDepModuleKey, false)
	}
	return b
}

func (b *moduleSetBuilder) Build() (ModuleSet, error) {
	if !b.buildCalled.CompareAndSwap(false, true) {
		return nil, errBuildAlreadyCalled
	}
	if len(b.errs) > 0 {
		return nil, multierr.Combine(b.errs...)
	}
	if len(b.modules) == 0 {
		return nil, errors.New("no Modules added to ModuleSetBuilder")
	}
	if slicesextended.Count(b.modules, func(m Module) bool { return m.IsTarget() }) < 1 {
		return nil, errors.New("no Modules were targeted in ModuleSetBuilder")
	}
	modules, err := getUniqueModulesByOpaqueID(b.ctx, b.modules)
	if err != nil {
		return nil, err
	}
	moduleSet, err := newModuleSet(modules)
	if err != nil {
		return nil, err
	}
	for _, module := range modules {
		module.setModuleSet(moduleSet)
	}
	return moduleSet, nil
}

func (*moduleSetBuilder) isModuleSetBuilder() {}

// getUniqueSortedModulesByOpaqueID deduplicates and sorts the Module list.
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
func getUniqueModulesByOpaqueID(ctx context.Context, modules []Module) ([]Module, error) {
	// sort.SliceStable keeps equal elements in their original order, so this does
	// not affect the "earlier preferred" property.
	//
	// However, after this, we can really apply "earlier" preferred to denote "prefer targets over
	// non-targets, then prefer local over remote."
	sort.SliceStable(
		modules,
		func(i int, j int) bool {
			m1 := modules[i]
			m2 := modules[j]
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
	uniqueModules := make([]Module, 0, len(modules))
	for _, module := range modules {
		opaqueID := module.OpaqueID()
		if opaqueID == "" {
			return nil, errors.New("OpaqueID was empty which should never happen")
		}
		if _, ok := alreadySeenOpaqueIDs[opaqueID]; !ok {
			alreadySeenOpaqueIDs[opaqueID] = struct{}{}
			uniqueModules = append(uniqueModules, module)
		} else {
		}
	}
	sort.Slice(
		uniqueModules,
		func(i int, j int) bool {
			return uniqueModules[i].OpaqueID() < uniqueModules[j].OpaqueID()
		},
	)
	return uniqueModules, nil
}

type localModuleOptions struct {
	moduleFullName     ModuleFullName
	commitID           string
	targetPaths        []string
	targetExcludePaths []string
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
