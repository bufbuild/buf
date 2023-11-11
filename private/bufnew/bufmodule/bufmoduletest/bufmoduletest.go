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

package bufmoduletest

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

// TestProvider is a ModuleKeyProvider and ModuleDataProvider for testing.
type TestProvider interface {
	bufmodule.ModuleKeyProvider
	bufmodule.ModuleDataProvider
}

// TestModuleData is the data needed to construct a Module in test.
type TestModuleData struct {
	// CommitID can be any string, but it must be unique across all TestModuleDatas.
	//
	// If not set, a mock commitID is created.
	CommitID string
	// Exactly one of PathToData or Bucket must be set.
	PathToData map[string][]byte
	// Exactly one of PathToData or Bucket must be set.
	Bucket storage.ReadBucket
}

func NewTestProvider(
	ctx context.Context,
	moduleFullNameStringToTestModuleData map[string]TestModuleData,
) (TestProvider, error) {
	return newTestProvider(ctx, moduleFullNameStringToTestModuleData)
}

// *** PRIVATE ***

type testProvider struct {
	moduleSet bufmodule.ModuleSet
}

func newTestProvider(
	ctx context.Context,
	moduleFullNameStringToTestModuleData map[string]TestModuleData,
) (*testProvider, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, bufmodule.NopModuleDataProvider)
	i := 0
	for moduleFullNameString, testModuleData := range moduleFullNameStringToTestModuleData {
		moduleFullName, err := bufmodule.ParseModuleFullName(moduleFullNameString)
		if err != nil {
			return nil, err
		}
		if testModuleData.Bucket == nil && len(testModuleData.PathToData) == 0 {
			return nil, errors.New("one of TestModuleData.Bucket or TestModuleData.PathToData must be set")
		}
		if testModuleData.Bucket != nil && len(testModuleData.PathToData) > 0 {
			return nil, errors.New("only one of TestModuleData.Bucket or TestModuleData.PathToData must be set")
		}
		bucket := testModuleData.Bucket
		if bucket == nil {
			bucket, err = storagemem.NewReadBucket(testModuleData.PathToData)
			if err != nil {
				return nil, err
			}
		}
		moduleSetBuilder.AddModuleForBucket(
			bucket,
			// Not actually in the spirit of bucketID, this could be non-unique with other buckets in theory.
			fmt.Sprintf("testProviderBucket-%s-%d", moduleFullNameString, i),
			false,
			bufmodule.AddModuleForBucketWithModuleFullName(moduleFullName),
			// Not actually a realistic commitID, may need to change later if we validate Commit IDs.
			bufmodule.AddModuleForBucketWithCommitID(fmt.Sprintf("testProviderCommit-%s-%d", moduleFullNameString, i)),
		)
		i++
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return &testProvider{
		moduleSet: moduleSet,
	}, nil
}

func (t *testProvider) GetModuleKeyForModuleRef(
	ctx context.Context,
	moduleRef bufmodule.ModuleRef,
) (bufmodule.ModuleKey, error) {
	module := t.moduleSet.GetModuleForModuleFullName(moduleRef.ModuleFullName())
	if module == nil {
		return nil, fmt.Errorf("no test ModuleKey with name %q", moduleRef.ModuleFullName().String())
	}
	return bufmodule.ModuleToModuleKey(module)
}

func (t *testProvider) GetModuleDataForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	module := t.moduleSet.GetModuleForModuleFullName(moduleKey.ModuleFullName())
	if module == nil {
		return nil, fmt.Errorf("no test ModuleData with name %q", moduleKey.ModuleFullName().String())
	}
	return bufmodule.NewModuleData(
		moduleKey,
		func() (storage.ReadBucket, error) {
			return bufmodule.ModuleReadBucketToStorageReadBucket(module), nil
		},
		func() ([]bufmodule.ModuleKey, error) {
			moduleDeps, err := module.ModuleDeps()
			if err != nil {
				return nil, err
			}
			return slicesextended.MapError(
				moduleDeps,
				func(moduleDep bufmodule.ModuleDep) (bufmodule.ModuleKey, error) {
					return bufmodule.ModuleToModuleKey(moduleDep)
				},
			)
		},
	)
}
