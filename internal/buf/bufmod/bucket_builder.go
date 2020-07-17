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

package bufmod

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

type bucketBuilder struct {
	logger *zap.Logger
}

func newBucketBuilder(
	logger *zap.Logger,
) *bucketBuilder {
	return &bucketBuilder{
		logger: logger,
	}
}

func (b *bucketBuilder) BuildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *Config,
	options ...BuildOption,
) (bufcore.Module, error) {
	buildOptions := &buildOptions{}
	for _, option := range options {
		option(buildOptions)
	}
	return b.buildForBucket(
		ctx,
		readBucket,
		config,
		buildOptions.paths,
		buildOptions.pathsAllowNotExistOnWalk,
	)
}

func (b *bucketBuilder) buildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *Config,
	bucketRelPaths []string,
	bucketRelPathsAllowNotExistOnWalk bool,
) (bufcore.Module, error) {
	roots := make([]string, 0, len(config.RootToExcludes))
	var rootBuckets []storage.ReadBucket
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
			storage.Map(
				readBucket,
				mappers...,
			),
		)
	}
	moduleOptions, err := getModuleOptions(
		roots,
		bucketRelPaths,
		bucketRelPathsAllowNotExistOnWalk,
		normalpath.Relative,
	)
	if err != nil {
		return nil, err
	}
	return bufcore.NewModule(storage.Multi(rootBuckets...), moduleOptions...)
}
