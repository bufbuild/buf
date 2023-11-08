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
	Build(ctx context.Context) ([]Module, error)

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

	alreadyBuilt bool
}

func newModuleBuilder(ctx context.Context, moduleProvider ModuleProvider) *moduleBuilder {
	return &moduleBuilder{
		ctx: ctx,
		// newLazyModuleProvider returns the parameter if the moduleProvider is already a *lazyModuleProvider.
		moduleProvider: newLazyModuleProvider(moduleProvider),
	}
}

func (b *moduleBuilder) AddModuleForBucket(
	bucketID string,
	bucket storage.ReadBucket,
	options ...AddModuleForBucketOption,
) error {
	addModuleForBucketOptions := newAddModuleForBucketOptions()
	for _, option := range options {
		option(addModuleForBucketOptions)
	}
	module, err := newModule(
		b.ctx,
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
	moduleFullName := moduleInfo.ModuleFullName()
	if moduleFullName == nil {
		return fmt.Errorf("ModuleInfo %v did not have ModuleFullName", moduleInfo)
	}
	module, err := b.moduleProvider.GetModuleForModuleInfo(b.ctx, moduleInfo)
	if err != nil {
		return err
	}
	b.moduleInfoModules = append(b.moduleInfoModules, module)
	return nil
}

func (b *moduleBuilder) Build(ctx context.Context) ([]Module, error) {
	if b.alreadyBuilt {
		return nil, errors.New("Build already called")
	}

	// prefer Bucket modules over ModuleInfo modules, i.e. local over remote.
	modules, err := getUniqueModulesWithEarlierPreferred(ctx, append(b.bucketModules, b.moduleInfoModules...))
	if err != nil {
		return nil, err
	}

	for i, module := range modules {
		allOtherModules := modules[0:i]
		if i != len(modules)-1 {
			allOtherModules = append(allOtherModules, modules[i+1:]...)
		}
		module.addPotentialDepModules(allOtherModules...)
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

// uniqueModulesWithEarlierPreferred deduplicates the Module list with the earlier modules being preferred.
//
// Callers should put modules built from local sources earlier than Modules built from remote sources.
//
// Duplication determined based opaqueID and on Digest, that is if a Module has an equal
// opaqueID, or an equal Digest, it is considered a duplicate.
//
// We want to account for Modules with the same name but different digests, that is a dep in a workspace
// that has the same name as something in a buf.lock file, we prefer the local dep in the workspace.
//
// When returned, all modules have unique opaqueIDs and Digests.
func getUniqueModulesWithEarlierPreferred(ctx context.Context, modules []Module) ([]Module, error) {
	alreadySeenOpaqueIDs := make(map[string]struct{})
	alreadySeenDigestStrings := make(map[string]struct{})
	uniqueModules := make([]Module, 0, len(modules))
	for _, module := range modules {
		opaqueID := module.opaqueID()
		if opaqueID == "" {
			return nil, errors.New("opaqueID was empty which should never happen")
		}
		digest, err := module.Digest()
		if err != nil {
			return nil, err
		}
		digestString := digest.String()

		_, alreadySeenModuleByID := alreadySeenOpaqueIDs[opaqueID]
		_, alreadySeenModulebyDigest := alreadySeenDigestStrings[digestString]

		alreadySeenOpaqueIDs[opaqueID] = struct{}{}
		alreadySeenDigestStrings[digestString] = struct{}{}

		if !alreadySeenModuleByID && !alreadySeenModulebyDigest {
			uniqueModules = append(uniqueModules, module)
		}
	}
	return nil, errors.New("TODO")
}
