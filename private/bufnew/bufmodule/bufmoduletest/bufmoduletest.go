package bufmoduletest

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type TestProvider interface {
	ModuleInfoProvider
	ModuleProvider
}

type TestModuleData struct {
	ModuleFullNameString string
	ModuleBucket         storage.ReadBucket
}

func NewTestProvider(
	ctx context.Context,
	testModuleDatas ...*TestModuleData,
) (testProvider, error) {
	return newTestProvider(ctx, testModuleDatas)
}

// *** PRIVATE ***

type testProvider struct {
	moduleFullNameStringToModule map[string]bufmodule.Module
	digestStringToModule         map[string]bufmodule.Module
}

func newTestProvider(ctx, testModuleDatas []*TestModuleData) (*testProvider, error) {
	moduleBuilder := bufmodule.NewModuleBuilder(ctx, nil)
	for i, testModuleData := range testModuleDatas {
		moduleFullName, err := bufmodule.ParseModuleFullName(testModuleData.ModuleFullNameString)
		if err != nil {
			return nil, err
		}
		if err := moduleBuilder.AddModuleForBucket(
			// Not actually in the spirit of bucketID, this could be non-unique with other buckets in theory
			fmt.Sprintf("%d", i),
			testModuleData.ModuleBucket,
			bufmodule.AddModuleForBucketWithModuleFullName(moduleFullName),
		); err != nil {
			return nil, err
		}
	}
	modules, err := moduleBuilder.Build()
	if err != nil {
		return nil, err
	}
	moduleFullNameStringToModule := make(map[string]bufmodule.Module, len(modules))
	digestStringToModule := make(map[string]bufmodule.Module, len(modules))
	for _, module := range modules {
		moduleFullNameString := module.ModuleFullName().String()
		if _, ok := moduleFullNameStringToModule[moduleFullNameString]; ok {
			return nil, fmt.Errorf("duplicate test ModuleFullName: %q", moduleFullNameString)
		}
		moduleFullNameStringToModule[moduleFullNameString] = module
		digest, err := module.Digest()
		if err != nil {
			return nil, err
		}
		digestString := digest.String()
		if _, ok := digestStringToModule[digest.String()]; ok {
			return nil, fmt.Errorf("duplicate test Digest: %q", digestString)
		}
		digestStringToModule[digestString] = module
	}
	return &testProvider{
		moduleFullNameStringToModule: moduleFullNameStringToModule,
		digestStringToModule:         digestStringToModule,
	}, nil
}

func (t *testProvider) GetModuleInfoForModuleRef(ctx context.Context, moduleRef ModuleRef) (ModuleInfo, error) {
	module, ok := t.moduleFullNameStringToModule[moduleRef.ModuleFullName().String()]
	if !ok {
		return nil, fmt.Errorf("no test ModuleInfo with name %q", moduleRef.ModuleFullName().String())
	}
	return module, nil
}

func (t *testProvider) GetModuleForModuleInfo(
	ctx context.Context,
	moduleInfo bufmodule.ModuleInfo,
) (bufmodule.Module, error) {
	moduleFullName := moduleInfo.ModuleFullName()
	if moduleFullName != nil {
		module, ok := t.moduleFullNameStringToModule[moduleFullName.String()]
		if !ok {
			return nil, fmt.Errorf("no test Module with name %q", moduleFullName.String())
		}
		return module, nil
	}
	digest, err := moduleInfo.Digest()
	if err != nil {
		return nil, err
	}
	digestString := digest.String()
	module, ok := t.digestStringToModule[digestString]
	if !ok {
		return nil, fmt.Errorf("no test Module with Digest %q", digestString)
	}
	return module, nil
}
