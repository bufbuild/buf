// Copyright 2020-2021 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

var _ ModuleFileSet = &moduleFileSet{}

type moduleFileSet struct {
	Module

	allReadBucket storage.ReadBucket
}

func newModuleFileSet(
	module Module,
	dependencies []Module,
) *moduleFileSet {
	readBuckets := []storage.ReadBucket{module.getSourceReadBucket()}
	for _, dependency := range dependencies {
		readBuckets = append(readBuckets, dependency.getSourceReadBucket())
	}
	return &moduleFileSet{
		Module:        module,
		allReadBucket: storage.MultiReadBucket(readBuckets...),
	}
}

func (m *moduleFileSet) AllFileInfos(ctx context.Context) ([]bufcore.FileInfo, error) {
	var fileInfos []bufcore.FileInfo
	if err := m.allReadBucket.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
		// super overkill but ok
		if err := validateModuleFilePathWithoutNormalization(objectInfo.Path()); err != nil {
			return err
		}
		isNotImport, err := storage.Exists(ctx, m.Module.getSourceReadBucket(), objectInfo.Path())
		if err != nil {
			return err
		}
		fileInfos = append(fileInfos, bufcore.NewFileInfoForObjectInfo(objectInfo, !isNotImport))
		return nil
	}); err != nil {
		return nil, err
	}
	bufcore.SortFileInfos(fileInfos)
	return fileInfos, nil
}

func (m *moduleFileSet) GetModuleFile(ctx context.Context, path string) (ModuleFile, error) {
	// super overkill but ok
	if err := validateModuleFilePath(path); err != nil {
		return nil, err
	}
	readObjectCloser, err := m.allReadBucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	isNotImport, err := storage.Exists(ctx, m.Module.getSourceReadBucket(), path)
	if err != nil {
		return nil, err
	}
	return newModuleFile(bufcore.NewFileInfoForObjectInfo(readObjectCloser, !isNotImport), readObjectCloser), nil
}

func (*moduleFileSet) isModuleFileSet() {}
