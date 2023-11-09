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
	"fmt"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

type TestProvider interface {
	bufmodule.ModuleInfoProvider
	bufmodule.ModuleProvider
}

func NewTestProviderForPathToData(
	ctx context.Context,
	moduleFullNameStringToPathToData map[string]map[string][]byte,
) (TestProvider, error) {
	moduleFullNameStringToBucket := make(map[string]storage.ReadBucket, len(moduleFullNameStringToPathToData))
	for moduleFullNameString, pathToData := range moduleFullNameStringToPathToData {
		bucket, err := storagemem.NewReadBucket(pathToData)
		if err != nil {
			return nil, err
		}
		moduleFullNameStringToBucket[moduleFullNameString] = bucket
	}
	return NewTestProviderForBuckets(ctx, moduleFullNameStringToBucket)
}

func NewTestProviderForBuckets(
	ctx context.Context,
	moduleFullNameStringToBucket map[string]storage.ReadBucket,
) (TestProvider, error) {
	testModuleDatas := make([]*testModuleData, 0, len(moduleFullNameStringToBucket))
	for moduleFullNameString, bucket := range moduleFullNameStringToBucket {
		testModuleDatas = append(
			testModuleDatas,
			&testModuleData{
				ModuleFullNameString: moduleFullNameString,
				Bucket:               bucket,
			},
		)
	}
	return newTestProvider(ctx, testModuleDatas)
}

// *** PRIVATE ***

type testModuleData struct {
	ModuleFullNameString string
	Bucket               storage.ReadBucket
}

type testProvider struct {
	moduleSet bufmodule.ModuleSet
}

func newTestProvider(ctx context.Context, testModuleDatas []*testModuleData) (*testProvider, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, nil)
	for i, testModuleData := range testModuleDatas {
		moduleFullName, err := bufmodule.ParseModuleFullName(testModuleData.ModuleFullNameString)
		if err != nil {
			return nil, err
		}
		moduleSetBuilder.AddModuleForBucket(
			// Not actually in the spirit of bucketID, this could be non-unique with other buckets in theory
			fmt.Sprintf("%d", i),
			testModuleData.Bucket,
			bufmodule.AddModuleForBucketWithModuleFullName(moduleFullName),
		)
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return &testProvider{
		moduleSet: moduleSet,
	}, nil
}

func (t *testProvider) GetModuleInfoForModuleRef(
	ctx context.Context,
	moduleRef bufmodule.ModuleRef,
) (bufmodule.ModuleInfo, error) {
	module := t.moduleSet.GetModuleForModuleFullName(moduleRef.ModuleFullName())
	if module == nil {
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
		module := t.moduleSet.GetModuleForModuleFullName(moduleFullName)
		if module == nil {
			return nil, fmt.Errorf("no test Module with name %q", moduleFullName.String())
		}
		return module, nil
	}
	digest, err := moduleInfo.Digest()
	if err != nil {
		return nil, err
	}
	module, err := t.moduleSet.GetModuleForDigest(digest)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, fmt.Errorf("no test Module with Digest %q", digest.String())
	}
	return module, nil
}
