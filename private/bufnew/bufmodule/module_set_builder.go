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

	"github.com/bufbuild/buf/private/pkg/storage"
)

var (
	errBuildAlreadyCalled = errors.New("ModuleSetBuilder.Build has already been called")
)

// ModuleSetBuilder builds ModuleSets.
//
// It is the effective primary entrypoint for this package.
type ModuleSetBuilder interface {
	// AddModuleForBucket adds a new Module for the given Bucket.
	//
	// This bucket will only be read for .proto files, license file(s), and documentation file(s).
	//
	// The BucketID is required.
	// If AddModuleForBucketWithModuleFullName is used, the OpaqueID will use this
	// ModuleFullName, otherwise it will be the BucketID.
	// Returns the same ModuleSetBuilder.
	AddModuleForBucket(bucketID string, bucket storage.ReadBucket, options ...AddModuleForBucketOption) ModuleSetBuilder
	// AddModuleForModuleInfo adds a new Module for the given ModuleInfo.
	//
	// The ModuleProvider given to the ModuleSetBuilder at construction time will be used to
	// retrieve this Module.
	//
	// The ModuleInfo must have ModuleFullName present.
	// The resulting Module will not have a BucketID but will always have a ModuleFullName.
	// Returns the same ModuleSetBuilder.
	AddModuleForModuleInfo(moduleInfo ModuleInfo) ModuleSetBuilder
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

// NewModuleSetBuilder returns a new ModuleSetBuilder.
func NewModuleSetBuilder(ctx context.Context, moduleProvider ModuleProvider) ModuleSetBuilder {
	return newModuleSetBuilder(ctx, moduleProvider)
}

/// *** PRIVATE ***

// moduleSetBuilder

type moduleSetBuilder struct {
	ctx            context.Context
	moduleProvider ModuleProvider

	bucketModules     []Module
	moduleInfoModules []Module
	cache             *cache

	errs        []error
	buildCalled bool
}

func newModuleSetBuilder(ctx context.Context, moduleProvider ModuleProvider) *moduleSetBuilder {
	cache := newCache()
	return &moduleSetBuilder{
		ctx:            ctx,
		moduleProvider: newLazyModuleProvider(moduleProvider, cache),
		cache:          cache,
	}
}

func (b *moduleSetBuilder) AddModuleForBucket(
	bucketID string,
	bucket storage.ReadBucket,
	options ...AddModuleForBucketOption,
) ModuleSetBuilder {
	if b.buildCalled {
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	addModuleForBucketOptions := newAddModuleForBucketOptions()
	for _, option := range options {
		option(addModuleForBucketOptions)
	}
	module, err := newModule(
		b.ctx,
		b.cache,
		bucketID,
		bucket,
		addModuleForBucketOptions.moduleFullName,
		addModuleForBucketOptions.commitID,
	)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	b.bucketModules = append(
		b.bucketModules,
		module,
	)
	return b
}

func (b *moduleSetBuilder) AddModuleForModuleInfo(moduleInfo ModuleInfo) ModuleSetBuilder {
	if b.buildCalled {
		b.errs = append(b.errs, errBuildAlreadyCalled)
		return b
	}
	if _, err := getAndValidateModuleFullName(moduleInfo); err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	if b.moduleProvider == nil {
		// We should perhaps have a ModuleSetBuilder without this method at all.
		// We do this in bufmoduletest.
		b.errs = append(b.errs, errors.New("cannot call AddModuleForModuleInfo with nil ModuleProvider"))
	}
	module, err := b.moduleProvider.GetModuleForModuleInfo(b.ctx, moduleInfo)
	if err != nil {
		b.errs = append(b.errs, err)
		return b
	}
	b.moduleInfoModules = append(b.moduleInfoModules, module)
	return b
}

func (b *moduleSetBuilder) Build() (ModuleSet, error) {
	if b.buildCalled {
		return nil, errBuildAlreadyCalled
	}
	b.buildCalled = true

	// prefer Bucket modules over ModuleInfo modules, i.e. local over remote.
	modules, err := getUniqueModulesByOpaqueIDWithEarlierPreferred(
		b.ctx,
		append(b.bucketModules, b.moduleInfoModules...),
	)
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
	if err := b.cache.setModuleSet(moduleSet); err != nil {
		return nil, err
	}
	return moduleSet, nil
}

func (*moduleSetBuilder) isModuleSetBuilder() {}

type addModuleForBucketOptions struct {
	moduleFullName ModuleFullName
	commitID       string
}

func newAddModuleForBucketOptions() *addModuleForBucketOptions {
	return &addModuleForBucketOptions{}
}
