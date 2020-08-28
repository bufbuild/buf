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

package bufmodulebuild

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/zap"
)

type moduleIncludeBuilder struct {
	logger *zap.Logger
}

func newModuleIncludeBuilder(
	logger *zap.Logger,
) *moduleIncludeBuilder {
	return &moduleIncludeBuilder{
		logger: logger,
	}
}

func (b *moduleIncludeBuilder) BuildForIncludes(
	ctx context.Context,
	includeDirPaths []string,
	options ...BuildOption,
) (bufmodule.Module, error) {
	buildOptions := &buildOptions{}
	for _, option := range options {
		option(buildOptions)
	}
	return b.buildForIncludes(
		ctx,
		includeDirPaths,
		buildOptions.paths,
		buildOptions.pathsAllowNotExist,
	)
}

func (b *moduleIncludeBuilder) buildForIncludes(
	ctx context.Context,
	includeDirPaths []string,
	filePaths []string,
	filePathsAllowNotExist bool,
) (bufmodule.Module, error) {
	if len(includeDirPaths) == 0 {
		includeDirPaths = []string{"."}
	}
	absIncludeDirPaths, err := normalizeAndCheckPaths(
		includeDirPaths,
		"include directory",
		normalpath.Absolute,
		true,
	)
	if err != nil {
		return nil, err
	}
	absFilePaths, err := normalizeAndCheckPaths(
		filePaths,
		"input file",
		normalpath.Absolute,
		false,
	)
	if err != nil {
		return nil, err
	}
	var rootBuckets []storage.ReadBucket
	for _, includeDirPath := range includeDirPaths {
		rootBucket, err := storageos.NewReadWriteBucket(includeDirPath)
		if err != nil {
			return nil, err
		}
		// need to do match extension here
		// https://github.com/bufbuild/buf/issues/113
		rootBuckets = append(rootBuckets, storage.MapReadBucket(rootBucket, storage.MatchPathExt(".proto")))
	}
	module, err := bufmodule.NewModuleForBucket(ctx, storage.MultiReadBucket(rootBuckets...))
	if err != nil {
		return nil, err
	}
	return applyModulePaths(
		module,
		absIncludeDirPaths,
		absFilePaths,
		filePathsAllowNotExist,
		normalpath.Absolute,
	)
}
