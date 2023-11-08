package bufmoduletest

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type TestModuleData struct {
	ModuleFullNameString string
	ModuleBucket         storage.ReadBucket
}

func NewTestModuleProvider(
	ctx context.Context,
	testModuleDatas ...*TestModuleData,
) (bufmodule.ModuleProvider, error) {
	return newTestModuleProvider(ctx, testModuleDatas)
}

// *** PRIVATE ***

type testModuleProvider struct {
	moduleFullNameStringToModule map[string]bufmodule.Module
	digestStringToModule         map[string]bufmodule.Module
}

func newTestModuleProvider(ctx, testModuleDatas []*TestModuleData) (*testModuleProvider, error) {
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
	return &testModuleProvider{
		moduleFullNameStringToModule: moduleFullNameStringToModule,
		digestStringToModule:         digestStringToModule,
	}, nil
}

func (t *testModuleProvider) GetModuleForModuleInfo(
	ctx context.Context,
	moduleInfo bufmodule.ModuleInfo,
) (bufmodule.Module, error) {
	moduleFullName := moduleInfo.ModuleFullName()
	if moduleFullName != nil {
		module, ok := t.moduleFullNameStringToModule[moduleFullName.String()]
		if !ok {
			return nil, fmt.Errorf("no test Module named %q", moduleFullName.String())
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
