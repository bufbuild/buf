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
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

// ModuleData is the data needed to construct a Module in test.
//
// Exactly one of PathToData, Bucket, DirPath must be set.
//
// Name is the ModuleFullName string. When creating an OmniProvider, Name is required.
//
// CommitID is optional, and can be any string, but it must be unique across all ModuleDatas.
// If CommitID is not set, a mock commitID is created.
type ModuleData struct {
	Name        string
	CommitID    string
	DirPath     string
	PathToData  map[string][]byte
	Bucket      storage.ReadBucket
	NotTargeted bool
}

// OmniProvider is a ModuleKeyProvider, ModuleDataProvider, and ModuleSet for testing.
type OmniProvider interface {
	bufmodule.ModuleKeyProvider
	bufmodule.ModuleDataProvider
	bufmodule.ModuleSet
}

// NewOmniProvider returns a new OmniProvider.
func NewOmniProvider(
	moduleDatas ...ModuleData,
) (OmniProvider, error) {
	return newOmniProvider(moduleDatas)
}

// NewModuleSet returns a new ModuleSet.
//
// This can be used in cases where ModuleKeyProviders and ModuleDataProviders are not needed,
// and when ModuleFullNames do not matter.
func NewModuleSet(
	moduleDatas ...ModuleData,
) (bufmodule.ModuleSet, error) {
	return newModuleSet(moduleDatas, false)
}

// NewModuleSetForDirPath returns a new ModuleSet for the directory path.
//
// This can be used in cases where ModuleKeyProviders and ModuleDataProviders are not needed,
// and when ModuleFullNames do not matter.
func NewModuleSetForDirPath(
	dirPath string,
) (bufmodule.ModuleSet, error) {
	return NewModuleSet(
		ModuleData{
			DirPath: dirPath,
		},
	)
}

// NewModuleSetForPathToData returns a new ModuleSet for the path to data map.
//
// This can be used in cases where ModuleKeyProviders and ModuleDataProviders are not needed,
// and when ModuleFullNames do not matter.
func NewModuleSetForPathToData(
	pathToData map[string][]byte,
) (bufmodule.ModuleSet, error) {
	return NewModuleSet(
		ModuleData{
			PathToData: pathToData,
		},
	)
}

// NewModuleSetForBucket returns a new ModuleSet for the Bucket.
//
// This can be used in cases where ModuleKeyProviders and ModuleDataProviders are not needed,
// and when ModuleFullNames do not matter.
func NewModuleSetForBucket(
	bucket storage.ReadBucket,
) (bufmodule.ModuleSet, error) {
	return NewModuleSet(
		ModuleData{
			Bucket: bucket,
		},
	)
}

// *** PRIVATE ***

type omniProvider struct {
	bufmodule.ModuleSet
}

func newOmniProvider(
	moduleDatas []ModuleData,
) (*omniProvider, error) {
	moduleSet, err := newModuleSet(moduleDatas, true)
	if err != nil {
		return nil, err
	}
	return &omniProvider{
		ModuleSet: moduleSet,
	}, nil
}

func (o *omniProvider) GetModuleKeyForModuleRef(
	ctx context.Context,
	moduleRef bufmodule.ModuleRef,
) (bufmodule.ModuleKey, error) {
	module := o.GetModuleForModuleFullName(moduleRef.ModuleFullName())
	if module == nil {
		return nil, fmt.Errorf("no test ModuleKey with name %q", moduleRef.ModuleFullName().String())
	}
	return bufmodule.ModuleToModuleKey(module)
}

func (o *omniProvider) GetModuleDataForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	module := o.GetModuleForModuleFullName(moduleKey.ModuleFullName())
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

func newModuleSet(moduleDatas []ModuleData, requireName bool) (bufmodule.ModuleSet, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(context.Background(), bufmodule.NopModuleDataProvider)
	for i, moduleData := range moduleDatas {
		if err := addModuleDataToModuleSetBuilder(
			moduleSetBuilder,
			moduleData,
			requireName,
			i,
		); err != nil {
			return nil, err
		}
	}
	return moduleSetBuilder.Build()
}

func addModuleDataToModuleSetBuilder(
	moduleSetBuilder bufmodule.ModuleSetBuilder,
	moduleData ModuleData,
	requireName bool,
	index int,
) error {
	if boolCount(
		moduleData.DirPath != "",
		moduleData.PathToData != nil,
		moduleData.Bucket != nil,
	) != 1 {
		return errors.New("exactly one of Bucket, PathToData, DirPath must be set on ModuleData")
	}
	var bucket storage.ReadBucket
	var bucketID string
	var err error
	switch {
	case moduleData.DirPath != "":
		storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
		bucket, err = storageosProvider.NewReadWriteBucket(
			moduleData.DirPath,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return err
		}
		bucketID = moduleData.DirPath
	case moduleData.PathToData != nil:
		bucket, err = storagemem.NewReadBucket(moduleData.PathToData)
		if err != nil {
			return err
		}
		bucketID = fmt.Sprintf("omniProviderBucket-%d", index)
	case moduleData.Bucket != nil:
		bucket = moduleData.Bucket
		bucketID = fmt.Sprintf("omniProviderBucket-%d", index)
	default:
		// Should never get here.
		return errors.New("boolCount returned 1 but all ModuleData fields were nil")
	}
	var bucketOptions []bufmodule.BucketOption
	if moduleData.Name != "" {
		moduleFullName, err := bufmodule.ParseModuleFullName(moduleData.Name)
		if err != nil {
			return err
		}
		commitID := moduleData.CommitID
		if commitID == "" {
			// Not actually a realistic commitID, may need to change later if we validate Commit IDs.
			commitID = fmt.Sprintf("omniProviderCommit-%d", index)
		}
		bucketOptions = []bufmodule.BucketOption{
			bufmodule.BucketWithModuleFullName(moduleFullName),
			bufmodule.BucketWithCommitID(commitID),
		}
	} else if requireName {
		return errors.New("ModuleData.Name was required in this context")
	}
	moduleSetBuilder.AddModuleForBucket(
		bucket,
		bucketID,
		!moduleData.NotTargeted,
		bucketOptions...,
	)
	return nil
}

func boolCount(bools ...bool) int {
	count := 0
	for _, b := range bools {
		if b {
			count++
		}
	}
	return count
}
