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
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/zap"
)

type builder struct {
	logger *zap.Logger
}

func newBuilder(
	logger *zap.Logger,
) *builder {
	return &builder{
		logger: logger,
	}
}

func (b *builder) BuildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *Config,
	options ...BuildForBucketOption,
) (bufcore.Module, error) {
	buildForBucketOptions := &buildForBucketOptions{}
	for _, option := range options {
		option(buildForBucketOptions)
	}
	return b.buildForBucket(
		ctx,
		readBucket,
		config,
		buildForBucketOptions.bucketRelPaths,
		buildForBucketOptions.bucketRelPathsAllowNotExistOnWalk,
	)
}
func (b *builder) buildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *Config,
	bucketRelPaths []string,
	bucketRelPathsAllowNotExistOnWalk bool,
) (bufcore.Module, error) {
	var rootBuckets []storage.ReadBucket
	for root, excludes := range config.RootToExcludes {
		mappers := []storage.Mapper{storage.MapOnPrefix(root)}
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
	var moduleOptions []bufcore.ModuleOption
	if len(bucketRelPaths) > 0 {
		targetPaths, err := bucketRelPathsToTargetPaths(config, bucketRelPaths)
		if err != nil {
			return nil, err
		}
		moduleOptions = append(moduleOptions, bufcore.ModuleWithTargetPaths(targetPaths...))
	}
	if bucketRelPathsAllowNotExistOnWalk {
		moduleOptions = append(moduleOptions, bufcore.ModuleWithTargetPathsAllowNotExistOnWalk())
	}
	return bufcore.NewModule(storage.Multi(rootBuckets...), moduleOptions...)
}

func bucketRelPathsToTargetPaths(config *Config, bucketRelPaths []string) ([]string, error) {
	roots := make([]string, 0, len(config.RootToExcludes))
	for root := range config.RootToExcludes {
		roots = append(roots, root)
	}
	if len(roots) == 1 && roots[0] == "." {
		return bucketRelPaths, nil
	}
	if len(roots) == 0 {
		// this should never happen
		return nil, errors.New("no roots on config")
	}

	targetPaths := make([]string, len(bucketRelPaths))
	for i, bucketRelPath := range bucketRelPaths {
		targetPath, err := bucketRelPathToTargetPath(roots, bucketRelPath)
		if err != nil {
			return nil, err
		}
		targetPaths[i] = targetPath
	}
	return targetPaths, nil
}

func bucketRelPathToTargetPath(roots []string, bucketRelPath string) (string, error) {
	var matchingRoots []string
	for _, root := range roots {
		if normalpath.ContainsPath(root, bucketRelPath) {
			matchingRoots = append(matchingRoots, root)
		}
	}
	switch len(matchingRoots) {
	case 0:
		// this is a user error and will likely happen often
		return "", fmt.Errorf("%s is not contained within any of %s", bucketRelPath, strings.Join(roots, ", "))
	case 1:
		targetPath, err := normalpath.Rel(matchingRoots[0], bucketRelPath)
		if err != nil {
			return "", err
		}
		// just in case
		return normalpath.NormalizeAndValidate(targetPath)
	default:
		// this should never happen
		return "", fmt.Errorf("%q is contained in multiple roots %v", bucketRelPath, roots)
	}
}

type buildForBucketOptions struct {
	bucketRelPaths                    []string
	bucketRelPathsAllowNotExistOnWalk bool
}
