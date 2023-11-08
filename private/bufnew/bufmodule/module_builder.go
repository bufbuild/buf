package bufmodule

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

type ModuleBuilder interface {
	AddModuleForBucket(storage.ReadBucket, ...AddModuleForBucketOption)
	AddModuleForModuleInfo(ModuleInfo)
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

	errs         []error
	alreadyBuilt bool
}

func newModuleBuilder(ctx context.Context, moduleProvider ModuleProvider) *moduleBuilder {
	return &moduleBuilder{
		ctx: ctx,
		// newLazyModuleProvider returns the parameter if the moduleProvider is already a *lazyModuleProvider.
		moduleProvider: newLazyModuleProvider(moduleProvider),
	}
}

func (b *moduleBuilder) AddModuleForBucket(bucket storage.ReadBucket, options ...AddModuleForBucketOption) {
	addModuleForBucketOptions := newAddModuleForBucketOptions()
	for _, option := range options {
		option(addModuleForBucketOptions)
	}
	module := newModule(
		b.ctx,
		addModuleForBucketOptions.moduleFullName,
		addModuleForBucketOptions.commitID,
	)
	module.setModuleReadBucket(
		newModuleReadBucket(
			b.ctx,
			bucket,
			module,
		),
	)
	b.bucketModules = append(
		b.bucketModules,
		module,
	)
}

func (b *moduleBuilder) AddModuleForModuleInfo(moduleInfo ModuleInfo) {
	module, err := b.moduleProvider.GetModuleForModuleInfo(b.ctx, moduleInfo)
	if err != nil {
		b.errs = append(b.errs, err)
		return
	}
	b.moduleInfoModules = append(b.moduleInfoModules, module)
}

func (b *moduleBuilder) Build(ctx context.Context) ([]Module, error) {
	if b.alreadyBuilt {
		return nil, errors.New("Build already called")
	}
	if b.errs != nil {
		return nil, multierr.Combine(b.errs...)
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
