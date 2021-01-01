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

package bufmodulebuild

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"go.uber.org/zap"
)

type moduleBucketBuilder struct {
	logger *zap.Logger
}

func newModuleBucketBuilder(
	logger *zap.Logger,
) *moduleBucketBuilder {
	return &moduleBucketBuilder{
		logger: logger,
	}
}

func (b *moduleBucketBuilder) BuildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *Config,
	options ...BuildOption,
) (bufmodule.Module, error) {
	buildOptions := &buildOptions{}
	for _, option := range options {
		option(buildOptions)
	}
	return b.buildForBucket(
		ctx,
		readBucket,
		config,
		buildOptions.paths,
		buildOptions.pathsAllowNotExist,
	)
}

func (b *moduleBucketBuilder) buildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *Config,
	bucketRelPaths []string,
	bucketRelPathsAllowNotExist bool,
) (bufmodule.Module, error) {
	roots := make([]string, 0, len(config.RootToExcludes))
	var rootBuckets []storage.ReadBucket
	lockFileReadBucket, err := getLockFileReadBucket(ctx, readBucket)
	if err != nil {
		return nil, err
	}
	if lockFileReadBucket != nil {
		rootBuckets = append(rootBuckets, lockFileReadBucket)
	}
	for root, excludes := range config.RootToExcludes {
		roots = append(roots, root)
		mappers := []storage.Mapper{
			// need to do match extension here
			// https://github.com/bufbuild/buf/issues/113
			storage.MatchPathExt(".proto"),
			storage.MapOnPrefix(root),
		}
		if len(excludes) != 0 {
			var notOrMatchers []storage.Matcher
			for _, exclude := range excludes {
				notOrMatchers = append(
					notOrMatchers,
					storage.MatchPathContained(exclude),
				)
			}
			mappers = append(
				mappers,
				storage.MatchNot(
					storage.MatchOr(
						notOrMatchers...,
					),
				),
			)
		}
		rootBuckets = append(
			rootBuckets,
			storage.MapReadBucket(
				readBucket,
				mappers...,
			),
		)
	}
	module, err := bufmodule.NewModuleForBucket(ctx, storage.MultiReadBucket(rootBuckets...))
	if err != nil {
		return nil, err
	}
	return applyModulePaths(
		module,
		roots,
		bucketRelPaths,
		bucketRelPathsAllowNotExist,
		normalpath.Relative,
	)
}

// may return nil
func getLockFileReadBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
) (storage.ReadBucket, error) {
	lockFileData, err := storage.ReadPath(ctx, readBucket, bufmodule.LockFilePath)
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(lockFileData) == 0 {
		return nil, nil
	}
	return storagemem.NewReadBucket(
		map[string][]byte{
			bufmodule.LockFilePath: lockFileData,
		},
	)
}
