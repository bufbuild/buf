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
		isTarget bool,
		options ...BucketOption,
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
		isTarget bool,
		options ...ModuleKeyOption,
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

// BucketOption is an option for AddModuleForBucket.
type BucketOption func(*bucketOptions)

// BucketWithModuleFullName returns a new BucketOption that adds the given ModuleFullName to the result Module.
//
// Use BucketWithModuleFullNameAndCommitID if you'd also like to add a CommitID.
func BucketWithModuleFullName(moduleFullName ModuleFullName) BucketOption {
	return func(bucketOptions *bucketOptions) {
		bucketOptions.moduleFullName = moduleFullName
	}
}

// BucketWithModuleFullName returns a new BucketOption that adds the given ModuleFullName and CommitID
// to the result Module.
func BucketWithModuleFullNameAndCommitID(moduleFullName ModuleFullName, commitID string) BucketOption {
	return func(bucketOptions *bucketOptions) {
		bucketOptions.moduleFullName = moduleFullName
		bucketOptions.commitID = commitID
	}
}

// BucketWithTargetPaths returns a new BucketOption that specifically targets the given paths, and
// specifically excludes the given paths.
//
// Only valid for a targeted Module. If this option is given to a non-target Module, this will
// result in an error during Build().
func BucketWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) BucketOption {
	return func(bucketOptions *bucketOptions) {
		bucketOptions.targetPaths = targetPaths
		bucketOptions.targetExcludePaths = targetExcludePaths
	}
}

// ModuleKeyOption is an option for AddModuleForModuleKey.
type ModuleKeyOption func(*moduleKeyOptions)

// ModuleKeyWithTargetPaths returns a new ModuleKeyOption that specifically targets the given paths, and
// specifically excludes the given paths.
//
// Only valid for a targeted Module. If this option is given to a non-target Module, this will
// result in an error during Build().
func ModuleKeyWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) ModuleKeyOption {
	return func(moduleKeyOptions *moduleKeyOptions) {
		moduleKeyOptions.targetPaths = targetPaths
		moduleKeyOptions.targetExcludePaths = targetExcludePaths
	}
}

/// *** PRIVATE ***

// moduleSetBuilder

type moduleSetBuilder struct {
	ctx                context.Context
	moduleDataProvider ModuleDataProvider

	moduleSetModules []*moduleSetModule
	errs             []error
	buildCalled      atomic.Bool
}

func newModuleSetBuilder(ctx context.Context, moduleDataProvider ModuleDataProvider) *moduleSetBuilder {
	return &moduleSetBuilder{
		ctx:                ctx,
		moduleDataProvider: moduleDataProvider,
	}
}

func (b *moduleSetBuilder) AddModuleForBucket(
	bucket storage.ReadBucket,
	bucketID string,
	isTarget bool,
	options ...BucketOption,
) ModuleSetBuilder {
	if b.buildCalled.Load() {
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	if bucketID == "" {
		b.errs = append(b.errs, errors.New("bucketID is required when calling AddModuleForBucket"))
		return b
	}
	bucketOptions := newBucketOptions()
	for _, option := range options {
		option(bucketOptions)
	}
	if bucketOptions.moduleFullName == nil && bucketOptions.commitID != "" {
		b.errs = append(b.errs, errors.New("cannot set commitID without ModuleFullName when calling AddModuleForBucket"))
		return b
	}
	if !isTarget && (len(bucketOptions.targetPaths) > 0 || len(bucketOptions.targetExcludePaths) > 0) {
		b.errs = append(b.errs, errors.New("cannot set TargetPaths for a non-target Module when calling AddModuleForBucket"))
		return b
	}
	module, err := newModule(
		b.ctx,
		func() (storage.ReadBucket, error) {
			return bucket, nil
		},
		bucketID,
		bucketOptions.moduleFullName,
		bucketOptions.commitID,
		isTarget,
		bucketOptions.targetPaths,
		bucketOptions.targetExcludePaths,
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
	isTarget bool,
	options ...ModuleKeyOption,
) ModuleSetBuilder {
	if b.buildCalled.Load() {
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	moduleKeyOptions := newModuleKeyOptions()
	for _, option := range options {
		option(moduleKeyOptions)
	}
	if !isTarget && (len(moduleKeyOptions.targetPaths) > 0 || len(moduleKeyOptions.targetExcludePaths) > 0) {
		b.errs = append(b.errs, errors.New("cannot set TargetPaths for a non-target Module when calling AddModuleForModuleKey"))
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
	}
	moduleData := moduleDatas[0]
	module, err := newModule(
		b.ctx,
		moduleData.Bucket,
		"",
		moduleData.ModuleKey().ModuleFullName(),
		moduleData.ModuleKey().CommitID(),
		isTarget,
		moduleKeyOptions.targetPaths,
		moduleKeyOptions.targetExcludePaths,
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
		//
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
	if !b.buildCalled.CompareAndSwap(false, true) {
		return nil, errBuildAlreadyCalled
	}
	if len(b.errs) > 0 {
		return nil, multierr.Combine(b.errs...)
	}
	if len(b.moduleSetModules) == 0 {
		return nil, errors.New("no Modules added to ModuleSetBuilder")
	}
	if slicesextended.Count(b.moduleSetModules, func(m *moduleSetModule) bool { return m.IsTarget() }) < 1 {
		return nil, errors.New("no Modules were targeted in ModuleSetBuilder")
	}
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
			if m1.IsTarget() && !m2.IsTarget() {
				return true
			}
			if !m1.IsTarget() && m2.IsTarget() {
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

type bucketOptions struct {
	moduleFullName     ModuleFullName
	commitID           string
	targetPaths        []string
	targetExcludePaths []string
}

func newBucketOptions() *bucketOptions {
	return &bucketOptions{}
}

type moduleKeyOptions struct {
	targetPaths        []string
	targetExcludePaths []string
}

func newModuleKeyOptions() *moduleKeyOptions {
	return &moduleKeyOptions{}
}
