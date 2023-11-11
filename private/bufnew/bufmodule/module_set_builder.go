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

	"github.com/bufbuild/buf/private/pkg/storage"
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
//
// Targets would represent modules in a local Workspace, or potentially just the specific
// Modules within a Workspace that you are targeting. This would be opposed to Modules
// solely from a buf.lock.
type ModuleSetBuilder interface {
	// AddModuleForBucket adds a new Module for the given Bucket.
	//
	// The Bucket used to construct the module will only be read for .proto files,
	// license file(s), and documentation file(s).
	//
	// The BucketID is required. If AddModuleForBucketWithModuleFullName is used, the OpaqueID will
	// use this ModuleFullName, otherwise the OpaqueID will be the BucketID.
	//
	// Returns the same ModuleSetBuilder.
	AddModuleForBucket(
		bucket storage.ReadBucket,
		bucketID string,
		isTargetModule bool,
		options ...AddModuleForBucketOption,
	) ModuleSetBuilder
	// AddModuleForModuleKey adds a new Module for the given ModuleKey.
	//
	// The ModuleDataProvider given to the ModuleSetBuilder at construction time will be used to
	// retrieve this Module.
	//
	// The resulting Module will not have a BucketID but will always have a ModuleFullName.
	//
	// The dependencies of the Module will are automatically added to the ModuleSet.
	// Note, however, that Modules added with AddModuleForBucket always take precedence,
	// so if there are local bucked-based dependencies, these will be used.
	//
	// Returns the same ModuleSetBuilder.
	AddModuleForModuleKey(
		moduleKey ModuleKey,
		isTargetModule bool,
		options ...AddModuleForModuleKeyOption,
	) ModuleSetBuilder
	// Build builds the Modules into a ModuleSet.
	//
	// Any errors from Add* calls will be returned here as well.
	Build() (ModuleSet, error)

	isModuleSetBuilder()
}

type AddModuleForBucketOption func(*addModuleForBucketOptions)

func AddModuleForBucketWithModuleFullName(moduleFullName ModuleFullName) AddModuleForBucketOption {
	return func(addModuleForBucketOptions *addModuleForBucketOptions) {
		addModuleForBucketOptions.moduleFullName = moduleFullName
	}
}

func AddModuleForBucketWithCommitID(commitID string) AddModuleForBucketOption {
	return func(addModuleForBucketOptions *addModuleForBucketOptions) {
		addModuleForBucketOptions.commitID = commitID
	}
}

func AddModuleForBucketWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) AddModuleForBucketOption {
	return func(addModuleForBucketOptions *addModuleForBucketOptions) {
		addModuleForBucketOptions.targetPaths = targetPaths
		addModuleForBucketOptions.targetExcludePaths = targetExcludePaths
	}
}

type AddModuleForModuleKeyOption func(*addModuleForModuleKeyOptions)

func AddModuleForModuleKeyWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) AddModuleForModuleKeyOption {
	return func(addModuleForModuleKeyOptions *addModuleForModuleKeyOptions) {
		addModuleForModuleKeyOptions.targetPaths = targetPaths
		addModuleForModuleKeyOptions.targetExcludePaths = targetExcludePaths
	}
}

// NewModuleSetBuilder returns a new ModuleSetBuilder.
func NewModuleSetBuilder(ctx context.Context, moduleDataProvider ModuleDataProvider) ModuleSetBuilder {
	return newModuleSetBuilder(ctx, moduleDataProvider)
}

/// *** PRIVATE ***

// moduleSetBuilder

type moduleSetBuilder struct {
	ctx                context.Context
	moduleDataProvider ModuleDataProvider

	cache *cache

	moduleSetModules []*moduleSetModule
	errs             []error
	buildCalled      bool
}

func newModuleSetBuilder(ctx context.Context, moduleDataProvider ModuleDataProvider) *moduleSetBuilder {
	cache := newCache()
	return &moduleSetBuilder{
		ctx:                ctx,
		moduleDataProvider: moduleDataProvider,
		cache:              cache,
	}
}

func (b *moduleSetBuilder) AddModuleForBucket(
	bucket storage.ReadBucket,
	bucketID string,
	isTargetModule bool,
	options ...AddModuleForBucketOption,
) ModuleSetBuilder {
	if b.buildCalled {
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	if bucketID == "" {
		b.errs = append(b.errs, errors.New("bucketID is required when calling AddModuleForBucket"))
		return b
	}
	addModuleForBucketOptions := newAddModuleForBucketOptions()
	for _, option := range options {
		option(addModuleForBucketOptions)
	}
	module, err := newModule(
		b.ctx,
		b.cache,
		func() (storage.ReadBucket, error) {
			return bucket, nil
		},
		bucketID,
		addModuleForBucketOptions.moduleFullName,
		addModuleForBucketOptions.commitID,
		isTargetModule,
		addModuleForBucketOptions.targetPaths,
		addModuleForBucketOptions.targetExcludePaths,
	)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	b.moduleSetModules = append(
		b.moduleSetModules,
		newModuleSetModule(
			module,
			true,
		),
	)
	return b
}

func (b *moduleSetBuilder) AddModuleForModuleKey(
	moduleKey ModuleKey,
	isTargetModule bool,
	options ...AddModuleForModuleKeyOption,
) ModuleSetBuilder {
	if b.buildCalled {
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	if b.moduleDataProvider == nil {
		// We should perhaps have a ModuleSetBuilder without this method at all.
		// We do this in bufmoduletest.
		b.errs = append(b.errs, errors.New("cannot call AddModuleForModuleKey with nil ModuleDataProvider"))
	}
	addModuleForModuleKeyOptions := newAddModuleForModuleKeyOptions()
	for _, option := range options {
		option(addModuleForModuleKeyOptions)
	}
	moduleData, err := b.moduleDataProvider.GetModuleDataForModuleKey(b.ctx, moduleKey)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	module, err := newModule(
		b.ctx,
		b.cache,
		moduleData.Bucket,
		"",
		moduleData.ModuleKey().ModuleFullName(),
		moduleData.ModuleKey().CommitID(),
		isTargetModule,
		addModuleForModuleKeyOptions.targetPaths,
		addModuleForModuleKeyOptions.targetExcludePaths,
	)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	b.moduleSetModules = append(
		b.moduleSetModules,
		newModuleSetModule(
			module,
			false,
		),
	)
	declaredDepModuleKeys, err := moduleData.DeclaredDepModuleKeys()
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	for _, declaredDepModuleKey := range declaredDepModuleKeys {
		// Not a target module.
		// Do not filter on paths, i.e. no options - paths only apply to the module as added by the caller.
		//
		// We don't need to special-case these - they are lowest priority as they aren't targets and
		// are added by ModuleKey. If a caller adds one of these ModuleKeys as a target, or adds
		// an equivalent Module by Bucket, that add will take precedence.
		b.AddModuleForModuleKey(declaredDepModuleKey, false)
	}
	return b
}

func (b *moduleSetBuilder) Build() (ModuleSet, error) {
	if b.buildCalled {
		return nil, errBuildAlreadyCalled
	}
	b.buildCalled = true

	moduleSetModules, err := getUniqueModulesByOpaqueID(b.ctx, b.moduleSetModules)
	if err != nil {
		return nil, err
	}
	moduleSet, err := newModuleSet(moduleSetModules)
	if err != nil {
		return nil, err
	}
	for _, moduleSetModule := range moduleSetModules {
		moduleSetModule.setModuleSet(moduleSet)
	}
	if err := b.cache.setModuleSet(moduleSet); err != nil {
		return nil, err
	}
	return moduleSet, nil
}

func (*moduleSetBuilder) isModuleSetBuilder() {}

// getUniqueSortedModulesByOpaqueID deduplicates and sorts the Module list.
//
// Modules that are targets are preferred, followed by Modules built from Buckets.
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
func getUniqueModulesByOpaqueID(ctx context.Context, moduleSetModules []*moduleSetModule) ([]*moduleSetModule, error) {
	// sort.SliceStable keeps equal elements in their original order, so this does
	// not affect the "earlier preferred" property.
	//
	// However, after this, we can really apply "earlier" preferred to denote "prefer targets over
	// non-targets, then prefer buckets over ModuleKeys."
	sort.SliceStable(
		moduleSetModules,
		func(i int, j int) bool {
			m1 := moduleSetModules[i]
			m2 := moduleSetModules[j]
			if m1.IsTargetModule() && !m2.IsTargetModule() {
				return true
			}
			if !m1.IsTargetModule() && m2.IsTargetModule() {
				return false
			}
			if m1.isCreatedFromBucket() && !m2.isCreatedFromBucket() {
				return true
			}
			// includes if !m1.isCreatedFromBucket() && m2.isCreatedFromBucket()
			return false
		},
	)
	// Digest *cannot* be used here - it's a chicken or egg problem. Computing the digest requires the cache,
	// the cache requires the unique Modules, the unique Modules require this function. This is OK though -
	// we want to add all Modules that we *think* are unique to the cache. If there is a duplicate, it
	// will be detected via cache usage.
	alreadySeenOpaqueIDs := make(map[string]struct{})
	uniqueModuleSetModules := make([]*moduleSetModule, 0, len(moduleSetModules))
	for _, moduleSetModule := range moduleSetModules {
		opaqueID := moduleSetModule.OpaqueID()
		if opaqueID == "" {
			return nil, errors.New("OpaqueID was empty which should never happen")
		}
		if _, ok := alreadySeenOpaqueIDs[opaqueID]; !ok {
			alreadySeenOpaqueIDs[opaqueID] = struct{}{}
			uniqueModuleSetModules = append(uniqueModuleSetModules, moduleSetModule)
		}
	}
	sort.Slice(
		uniqueModuleSetModules,
		func(i int, j int) bool {
			return uniqueModuleSetModules[i].OpaqueID() < uniqueModuleSetModules[j].OpaqueID()
		},
	)
	return uniqueModuleSetModules, nil
}

type addModuleForBucketOptions struct {
	moduleFullName     ModuleFullName
	commitID           string
	targetPaths        []string
	targetExcludePaths []string
}

func newAddModuleForBucketOptions() *addModuleForBucketOptions {
	return &addModuleForBucketOptions{}
}

type addModuleForModuleKeyOptions struct {
	targetPaths        []string
	targetExcludePaths []string
}

func newAddModuleForModuleKeyOptions() *addModuleForModuleKeyOptions {
	return &addModuleForModuleKeyOptions{}
}
