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

package bufmodule

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

type targetingModule struct {
	Module
	targetPaths                    []string
	targetPathsAllowNotExistOnWalk bool
}

func newTargetingModule(
	delegate Module,
	targetPaths []string,
	targetPathsAllowNotExistOnWalk bool,
) (*targetingModule, error) {
	if len(targetPaths) == 0 {
		return nil, errors.New("targetingModule created without any target paths")
	}
	if err := validateModuleFilePaths(targetPaths); err != nil {
		return nil, err
	}
	return &targetingModule{
		Module:                         delegate,
		targetPaths:                    targetPaths,
		targetPathsAllowNotExistOnWalk: targetPathsAllowNotExistOnWalk,
	}, nil
}

func (m *targetingModule) TargetFileInfos(ctx context.Context) ([]bufcore.FileInfo, error) {
	var fileInfos []bufcore.FileInfo
	for _, targetPath := range m.targetPaths {
		objectInfo, err := m.getSourceReadBucket().Stat(ctx, targetPath)
		if err != nil {
			if m.targetPathsAllowNotExistOnWalk && storage.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		fileInfos = append(fileInfos, bufcore.NewFileInfoForObjectInfo(objectInfo, false))
	}
	if len(fileInfos) == 0 {
		return nil, ErrNoTargetFiles
	}
	sortFileInfos(fileInfos)
	return fileInfos, nil
}
