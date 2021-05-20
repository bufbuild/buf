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

	allReadBucket ReadBucket
}

func newModuleFileSet(
	module Module,
	dependencies []Module,
) *moduleFileSet {
	// TODO: We can remove the getModuleCommit method on the
	// Module type if we fetch FileInfos from the Module
	// and plumb in the ModuleCommit here.
	//
	// This approach assumes that all of the FileInfos returned
	// from SourceFileInfos will have their ModuleCommit
	// set to the same value. That can be enforced here.
	moduleReadBuckets := []ReadBucket{
		NewReadBucket(
			module.getSourceReadBucket(),
			module.getModuleCommit(),
		),
	}
	for _, dependency := range dependencies {
		moduleReadBuckets = append(
			moduleReadBuckets,
			NewReadBucket(
				dependency.getSourceReadBucket(),
				dependency.getModuleCommit(),
			),
		)
	}
	return &moduleFileSet{
		Module:        module,
		allReadBucket: newMultiReadBucket(moduleReadBuckets...),
	}
}

func (m *moduleFileSet) AllFileInfos(ctx context.Context) ([]FileInfo, error) {
	var fileInfos []FileInfo
	if err := m.allReadBucket.WalkModuleFiles(ctx, "", func(objectInfo ObjectInfo) error {
		if err := validateModuleFilePathWithoutNormalization(objectInfo.Path()); err != nil {
			return err
		}
		isNotImport, err := storage.Exists(ctx, m.Module.getSourceReadBucket(), objectInfo.Path())
		if err != nil {
			return err
		}
		coreFileInfo := bufcore.NewFileInfoForObjectInfo(objectInfo, !isNotImport)
		fileInfos = append(fileInfos, NewFileInfo(coreFileInfo, objectInfo.ModuleCommit()))
		return nil
	}); err != nil {
		return nil, err
	}
	sortFileInfos(fileInfos)
	return fileInfos, nil
}

func (m *moduleFileSet) GetModuleFile(ctx context.Context, path string) (ModuleFile, error) {
	if err := ValidateModuleFilePath(path); err != nil {
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
	objectInfo, err := m.allReadBucket.StatModuleFile(ctx, path)
	if err != nil {
		return nil, err
	}
	coreFileInfo := bufcore.NewFileInfoForObjectInfo(readObjectCloser, !isNotImport)
	return newModuleFile(NewFileInfo(coreFileInfo, objectInfo.ModuleCommit()), readObjectCloser), nil
}

func (*moduleFileSet) isModuleFileSet() {}
