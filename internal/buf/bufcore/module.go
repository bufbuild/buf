// Copyright 2020 Buf Technologies, Inc.
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

package bufcore

import (
	"context"
	"errors"
	"sort"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

var _ Module = &module{}

type module struct {
	sourceReadBucket               storage.ReadBucket
	importReadBucket               storage.ReadBucket
	allReadBucket                  storage.ReadBucket
	targetPaths                    []string
	targetPathsAllowNotExistOnWalk bool
}

func newModule(
	sourceReadBucket storage.ReadBucket,
	options ...ModuleOption,
) (*module, error) {
	module := &module{
		sourceReadBucket: storage.Map(
			sourceReadBucket,
			storage.MatchPathExt(".proto"),
		),
	}
	for _, option := range options {
		option(module)
	}
	if len(module.targetPaths) > 0 && module.targetPathsAllowNotExistOnWalk {
		return nil, errors.New("targetPathsAllowNotExistOnWalk set but targetPaths not specified")
	}
	for i, targetPath := range module.targetPaths {
		normalizedTargetPath, err := normalpath.NormalizeAndValidate(targetPath)
		if err != nil {
			return nil, err
		}
		module.targetPaths[i] = normalizedTargetPath
	}
	if module.importReadBucket != nil {
		module.importReadBucket = storage.Map(
			module.importReadBucket,
			storage.MatchPathExt(".proto"),
		)
		module.allReadBucket = storage.Multi(
			module.sourceReadBucket,
			module.importReadBucket,
		)
	} else {
		module.allReadBucket = sourceReadBucket
	}
	return module, nil
}

func (m *module) TargetFileInfos(ctx context.Context) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if len(m.targetPaths) > 0 {
		fileInfos = make([]FileInfo, 0, len(m.targetPaths))
		for _, targetPath := range m.targetPaths {
			objectInfo, err := m.sourceReadBucket.Stat(ctx, targetPath)
			if err != nil {
				if m.targetPathsAllowNotExistOnWalk && storage.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			fileInfos = append(fileInfos, newFileInfoForObjectInfo(objectInfo, false))
		}
	} else {
		if err := m.sourceReadBucket.Walk(
			ctx,
			"",
			func(objectInfo storage.ObjectInfo) error {
				fileInfos = append(fileInfos, newFileInfoForObjectInfo(objectInfo, false))
				return nil
			},
		); err != nil {
			return nil, err
		}
	}
	sortFileInfos(fileInfos)
	return fileInfos, nil
}

func (m *module) GetFile(ctx context.Context, path string) (ModuleFile, error) {
	readObjectCloser, err := m.allReadBucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	isImport := false
	if m.importReadBucket != nil {
		isImport, err = storage.Exists(ctx, m.importReadBucket, path)
		if err != nil {
			return nil, err
		}
	}
	return newModuleFile(
		newFileInfoForObjectInfo(
			readObjectCloser,
			isImport,
		),
		readObjectCloser,
	), nil
}

func (m *module) GetFileInfo(ctx context.Context, path string) (FileInfo, error) {
	objectInfo, err := m.allReadBucket.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	isImport := false
	if m.importReadBucket != nil {
		isImport, err = storage.Exists(ctx, m.importReadBucket, path)
		if err != nil {
			return nil, err
		}
	}
	return newFileInfoForObjectInfo(
		objectInfo,
		isImport,
	), nil
}

func (*module) isModule() {}

func sortFileInfos(fileInfos []FileInfo) {
	sort.Slice(
		fileInfos,
		func(i int, j int) bool {
			return fileInfos[i].Path() < fileInfos[j].Path()
		},
	)
}
