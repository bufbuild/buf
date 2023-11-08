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

	"github.com/bufbuild/buf/private/pkg/storage"
)

type ModuleBuilder interface {
	AddModuleForBucket(bucketID string, bucket storage.ReadBucket, options ...AddModuleForBucketOption) error
	AddModuleForModuleInfo(moduleInfo ModuleInfo) error
	Build() ([]Module, error)

	isModuleBuilder()
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

func NewModuleBuilder(ctx context.Context, moduleProvider ModuleProvider) ModuleBuilder {
	return newModuleBuilder(ctx, moduleProvider)
}

/// *** PRIVATE ***

// moduleBuilder

type moduleBuilder struct {
	ctx            context.Context
	moduleProvider ModuleProvider

	bucketModules     []Module
	moduleInfoModules []Module
	cache             *cache

	buildCalled bool
}

func newModuleBuilder(ctx context.Context, moduleProvider ModuleProvider) *moduleBuilder {
	cache := newCache()
	return &moduleBuilder{
		ctx:            ctx,
		moduleProvider: newLazyModuleProvider(moduleProvider, cache),
		cache:          cache,
	}
}

func (b *moduleBuilder) AddModuleForBucket(
	bucketID string,
	bucket storage.ReadBucket,
	options ...AddModuleForBucketOption,
) error {
	if b.buildCalled {
		return errors.New("Build already called")
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
		return err
	}
	b.bucketModules = append(
		b.bucketModules,
		module,
	)
	return nil
}

func (b *moduleBuilder) AddModuleForModuleInfo(moduleInfo ModuleInfo) error {
	if b.buildCalled {
		return errors.New("Build already called")
	}
	moduleFullName := moduleInfo.ModuleFullName()
	if moduleFullName == nil {
		return fmt.Errorf("ModuleInfo %v did not have ModuleFullName", moduleInfo)
	}
	if b.moduleProvider == nil {
		// We should perhaps have a ModuleBuilder without this method at all.
		// We do this in bufmoduletest.
		return errors.New("cannot call AddModuleForModuleInfo with nil ModuleProvider")
	}
	module, err := b.moduleProvider.GetModuleForModuleInfo(b.ctx, moduleInfo)
	if err != nil {
		return err
	}
	b.moduleInfoModules = append(b.moduleInfoModules, module)
	return nil
}

func (b *moduleBuilder) Build() ([]Module, error) {
	if b.buildCalled {
		return nil, errors.New("Build already called")
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
	if err := b.cache.SetModules(modules); err != nil {
		return nil, err
	}
	return modules, nil
}

func (*moduleBuilder) isModuleBuilder() {}

type addModuleForBucketOptions struct {
	moduleFullName ModuleFullName
	commitID       string
}

func newAddModuleForBucketOptions() *addModuleForBucketOptions {
	return &addModuleForBucketOptions{}
}
