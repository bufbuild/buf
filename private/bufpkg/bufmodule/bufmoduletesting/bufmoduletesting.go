// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufmoduletesting

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

var (
	// 2023-01-01 at 12:00 UTC
	mockTime = time.Unix(1672574400, 0)
)

// ModuleData is the data needed to construct a Module in test.
//
// Exactly one of PathToData, Bucket, DirPath must be set.
//
// Name is the ModuleFullName string. When creating an OmniProvider, Name is required.
//
// CommitID is optional, but it must be unique across all ModuleDatas. If CommitID is not set,
// a mock commitID is created if Name is set.
//
// CreateTime is optional. If CreateTime is not set, a mock create Time is created. This create
// time is the same for all data without a Time.
//
// If ReadObjectDataFromBucket is true, buf.yamls and buf.locks will attempt to be read from
// PathToData, Bucket, or DirPath. Otherwise, BufYAMLObjectData and BufLockObjectData will be
// used. It is an error to both set ReadObjectDataFromBucket and set Buf.*ObjectData.
type ModuleData struct {
	Name                     string
	CommitID                 uuid.UUID
	CreateTime               time.Time
	DirPath                  string
	PathToData               map[string][]byte
	Bucket                   storage.ReadBucket
	NotTargeted              bool
	BufYAMLObjectData        bufmodule.ObjectData
	BufLockObjectData        bufmodule.ObjectData
	ReadObjectDataFromBucket bool
}

// OmniProvider is a ModuleKeyProvider, ModuleDataProvider, GraphProvider, CommitProvider, and ModuleSet for testing.
type OmniProvider interface {
	bufmodule.ModuleKeyProvider
	bufmodule.ModuleDataProvider
	bufmodule.GraphProvider
	bufmodule.CommitProvider
	bufmodule.ModuleSet
}

// NewOmniProvider returns a new OmniProvider.
//
// Note the ModuleDatas must be self-contained, that is they only import from each other.
func NewOmniProvider(
	moduleDatas ...ModuleData,
) (OmniProvider, error) {
	return newOmniProvider(moduleDatas)
}

// NewModuleSet returns a new ModuleSet.
//
// This can be used in cases where ModuleKeyProviders and ModuleDataProviders are not needed,
// and when ModuleFullNames do not matter.
//
// Note the ModuleDatas must be self-contained, that is they only import from each other.
func NewModuleSet(
	moduleDatas ...ModuleData,
) (bufmodule.ModuleSet, error) {
	return newModuleSet(moduleDatas, false, nil)
}

// NewModuleSetForDirPath returns a new ModuleSet for the directory path.
//
// This can be used in cases where ModuleKeyProviders and ModuleDataProviders are not needed,
// and when ModuleFullNames do not matter.
//
// Note that this Module cannot have any dependencies.
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
//
// Note that this Module cannot have any dependencies.
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
//
// Note that this Module cannot have any dependencies.
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
	commitIDToCreateTime map[uuid.UUID]time.Time
}

func newOmniProvider(
	moduleDatas []ModuleData,
) (*omniProvider, error) {
	commitIDToCreateTime := make(map[uuid.UUID]time.Time)
	moduleSet, err := newModuleSet(moduleDatas, true, commitIDToCreateTime)
	if err != nil {
		return nil, err
	}
	return &omniProvider{
		ModuleSet:            moduleSet,
		commitIDToCreateTime: commitIDToCreateTime,
	}, nil
}

func (o *omniProvider) GetModuleKeysForModuleRefs(
	ctx context.Context,
	moduleRefs []bufmodule.ModuleRef,
	digestType bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	moduleKeys := make([]bufmodule.ModuleKey, len(moduleRefs))
	for i, moduleRef := range moduleRefs {
		module := o.GetModuleForModuleFullName(moduleRef.ModuleFullName())
		if module == nil {
			return nil, &fs.PathError{Op: "read", Path: moduleRef.String(), Err: fs.ErrNotExist}
		}
		moduleKey, err := bufmodule.ModuleToModuleKey(module, digestType)
		if err != nil {
			return nil, err
		}
		moduleKeys[i] = moduleKey
	}
	return moduleKeys, nil
}

func (o *omniProvider) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, error) {
	if len(moduleKeys) == 0 {
		return nil, nil
	}
	if _, err := bufmodule.UniqueDigestTypeForModuleKeys(moduleKeys); err != nil {
		return nil, err
	}
	if _, err := bufmodule.ModuleFullNameStringToUniqueValue(moduleKeys); err != nil {
		return nil, err
	}
	return slicesext.MapError(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) (bufmodule.ModuleData, error) {
			return o.getModuleDataForModuleKey(ctx, moduleKey)
		},
	)
}

func (o *omniProvider) GetCommitsForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.Commit, error) {
	if len(moduleKeys) == 0 {
		return nil, nil
	}
	if _, err := bufmodule.UniqueDigestTypeForModuleKeys(moduleKeys); err != nil {
		return nil, err
	}
	commits := make([]bufmodule.Commit, len(moduleKeys))
	for i, moduleKey := range moduleKeys {
		createTime, ok := o.commitIDToCreateTime[moduleKey.CommitID()]
		if !ok {
			return nil, &fs.PathError{Op: "read", Path: moduleKey.String(), Err: fs.ErrNotExist}
		}
		commits[i] = bufmodule.NewCommit(
			moduleKey,
			func() (time.Time, error) {
				return createTime, nil
			},
		)
	}
	return commits, nil
}

func (o *omniProvider) GetCommitsForCommitKeys(
	ctx context.Context,
	commitKeys []bufmodule.CommitKey,
) ([]bufmodule.Commit, error) {
	if len(commitKeys) == 0 {
		return nil, nil
	}
	if _, err := bufmodule.UniqueDigestTypeForCommitKeys(commitKeys); err != nil {
		return nil, err
	}
	commits := make([]bufmodule.Commit, len(commitKeys))
	for i, commitKey := range commitKeys {
		module := o.GetModuleForCommitID(commitKey.CommitID())
		if module == nil {
			return nil, &fs.PathError{Op: "read", Path: uuidutil.ToDashless(commitKey.CommitID()), Err: fs.ErrNotExist}
		}
		createTime, ok := o.commitIDToCreateTime[commitKey.CommitID()]
		if !ok {
			return nil, &fs.PathError{Op: "read", Path: uuidutil.ToDashless(commitKey.CommitID()), Err: fs.ErrNotExist}
		}
		moduleKey, err := bufmodule.ModuleToModuleKey(module, commitKey.DigestType())
		if err != nil {
			return nil, err
		}
		commits[i] = bufmodule.NewCommit(
			moduleKey,
			func() (time.Time, error) {
				return createTime, nil
			},
		)
	}
	return commits, nil
}

func (o *omniProvider) GetGraphForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) (*dag.Graph[bufmodule.RegistryCommitID, bufmodule.ModuleKey], error) {
	graph := dag.NewGraph[bufmodule.RegistryCommitID, bufmodule.ModuleKey](bufmodule.ModuleKeyToRegistryCommitID)
	if len(moduleKeys) == 0 {
		return graph, nil
	}
	digestType, err := bufmodule.UniqueDigestTypeForModuleKeys(moduleKeys)
	if err != nil {
		return nil, err
	}
	modules := make([]bufmodule.Module, len(moduleKeys))
	for i, moduleKey := range moduleKeys {
		module := o.GetModuleForModuleFullName(moduleKey.ModuleFullName())
		if module == nil {
			return nil, &fs.PathError{Op: "read", Path: moduleKey.String(), Err: fs.ErrNotExist}
		}
		modules[i] = module
	}
	for _, module := range modules {
		if err := addModuleToGraphRec(module, graph, digestType); err != nil {
			return nil, err
		}
	}
	return graph, nil
}

func (o *omniProvider) getModuleDataForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	module := o.GetModuleForModuleFullName(moduleKey.ModuleFullName())
	if module == nil {
		return nil, &fs.PathError{Op: "read", Path: moduleKey.String(), Err: fs.ErrNotExist}
	}
	moduleDeps, err := module.ModuleDeps()
	if err != nil {
		return nil, err
	}
	digest, err := moduleKey.Digest()
	if err != nil {
		return nil, err
	}
	declaredDepModuleKeys, err := slicesext.MapError(
		moduleDeps,
		func(moduleDep bufmodule.ModuleDep) (bufmodule.ModuleKey, error) {
			return bufmodule.ModuleToModuleKey(moduleDep, digest.Type())
		},
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleData(
		ctx,
		moduleKey,
		func() (storage.ReadBucket, error) {
			return bufmodule.ModuleReadBucketToStorageReadBucket(module), nil
		},
		func() ([]bufmodule.ModuleKey, error) {
			return declaredDepModuleKeys, nil
		},
		func() (bufmodule.ObjectData, error) {
			return module.V1Beta1OrV1BufYAMLObjectData()
		},
		func() (bufmodule.ObjectData, error) {
			return module.V1Beta1OrV1BufLockObjectData()
		},
	), nil
}

func newModuleSet(
	moduleDatas []ModuleData,
	requireName bool,
	// may be nil
	commitIDToCreateTime map[uuid.UUID]time.Time,
) (bufmodule.ModuleSet, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(context.Background(), slogext.NopLogger, bufmodule.NopModuleDataProvider, bufmodule.NopCommitProvider)
	for i, moduleData := range moduleDatas {
		if err := addModuleDataToModuleSetBuilder(
			moduleSetBuilder,
			moduleData,
			requireName,
			commitIDToCreateTime,
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
	// may be nil
	commitIDToCreateTime map[uuid.UUID]time.Time,
	index int,
) error {
	if boolCount(
		moduleData.DirPath != "",
		moduleData.PathToData != nil,
		moduleData.Bucket != nil,
	) != 1 {
		return errors.New("exactly one of Bucket, PathToData, DirPath must be set on ModuleData")
	}
	if boolCount(
		moduleData.ReadObjectDataFromBucket,
		moduleData.BufYAMLObjectData != nil,
	) > 1 || boolCount(
		moduleData.ReadObjectDataFromBucket,
		moduleData.BufLockObjectData != nil,
	) > 1 {
		return errors.New("cannot set ReadObjectDataFromBucket alongside BufYAMLObjectData or BufLockObjectData")
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
		// Since it's possible to that there are multiple modules at the same DirPath, we append the
		// index to make sure the bucketID is unique. This does not need to have to same format as
		// bucketIDs of modules built in non-test code paths.
		bucketID = fmt.Sprintf("%s-%d", moduleData.DirPath, index)
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
	var localModuleOptions []bufmodule.LocalModuleOption
	if moduleData.Name != "" {
		moduleFullName, err := bufmodule.ParseModuleFullName(moduleData.Name)
		if err != nil {
			return err
		}
		commitID := moduleData.CommitID
		if commitID == uuid.Nil {
			commitID, err = uuidutil.New()
			if err != nil {
				return err
			}
		}
		if commitIDToCreateTime != nil {
			createTime := moduleData.CreateTime
			if createTime.IsZero() {
				createTime = mockTime
			}
			commitIDToCreateTime[commitID] = createTime
		}
		localModuleOptions = []bufmodule.LocalModuleOption{
			bufmodule.LocalModuleWithModuleFullNameAndCommitID(moduleFullName, commitID),
		}
	} else if requireName {
		return errors.New("ModuleData.Name was required in this context")
	}
	if moduleData.ReadObjectDataFromBucket {
		ctx := context.Background()
		bufYAMLObjectData, err := bufconfig.GetBufYAMLV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, ".")
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		bufLockObjectData, err := bufconfig.GetBufLockV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, ".")
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		localModuleOptions = append(
			localModuleOptions,
			bufmodule.LocalModuleWithV1Beta1OrV1BufYAMLObjectData(bufYAMLObjectData),
			bufmodule.LocalModuleWithV1Beta1OrV1BufLockObjectData(bufLockObjectData),
		)
	} else {
		if moduleData.BufYAMLObjectData != nil {
			localModuleOptions = append(
				localModuleOptions,
				bufmodule.LocalModuleWithV1Beta1OrV1BufYAMLObjectData(moduleData.BufYAMLObjectData),
			)
		}
		if moduleData.BufLockObjectData != nil {
			localModuleOptions = append(
				localModuleOptions,
				bufmodule.LocalModuleWithV1Beta1OrV1BufLockObjectData(moduleData.BufLockObjectData),
			)
		}
	}
	moduleSetBuilder.AddLocalModule(
		bucket,
		bucketID,
		!moduleData.NotTargeted,
		localModuleOptions...,
	)
	return nil
}

func addModuleToGraphRec(
	module bufmodule.Module,
	graph *dag.Graph[bufmodule.RegistryCommitID, bufmodule.ModuleKey],
	digestType bufmodule.DigestType,
) error {
	moduleKey, err := bufmodule.ModuleToModuleKey(module, digestType)
	if err != nil {
		return err
	}
	graph.AddNode(moduleKey)
	directModuleDeps, err := bufmodule.ModuleDirectModuleDeps(module)
	if err != nil {
		return err
	}
	for _, directModuleDep := range directModuleDeps {
		directDepModuleKey, err := bufmodule.ModuleToModuleKey(module, digestType)
		if err != nil {
			return err
		}
		graph.AddEdge(moduleKey, directDepModuleKey)
		if err := addModuleToGraphRec(directModuleDep, graph, digestType); err != nil {
			return err
		}
	}
	return nil
}

func boolCount(bools ...bool) int {
	return slicesext.Count(bools, func(value bool) bool { return value })
}
