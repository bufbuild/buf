package bufmodule

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/storage"
)

type ModuleBuilder interface {
	AddModuleForBucket(storage.ReadBucket, ...AddModuleForBucketOption) error
	AddModuleForModuleInfo(ModuleInfo) error
	Build(context.Context) ([]Module, error)

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

func (b *moduleBuilder) AddModuleForBucket(bucket storage.ReadBucket, options ...AddModuleForBucketOption) error {
	addModuleForBucketOptions := newAddModuleForBucketOptions()
	for _, option := range options {
		option(addModuleForBucketOptions)
	}
	module := newModule(
		b.ctx,
		bucket,
		addModuleForBucketOptions.moduleFullName,
		addModuleForBucketOptions.commitID,
	)
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
	modules := append(b.bucketModules, b.moduleInfoModules...)

	for i, module := range modules {
		allOtherModules := modules[0:i]
		if i != len(modules)-1 {
			allOtherModules = append(allOtherModules, modules[i+1:len(modules)]...)
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
