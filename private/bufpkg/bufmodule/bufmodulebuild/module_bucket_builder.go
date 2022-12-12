// Copyright 2020-2022 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

// BuiltModule ties a bufmodule.Module with the configuration and a bucket
// containing just the files required to build it.
type BuiltModule struct {
	Module bufmodule.Module
	Bucket storage.ReadBucket
}

type moduleBucketBuilder struct {
	opt buildOptions
}

func newModuleBucketBuilder(
	options ...BuildOption,
) *moduleBucketBuilder {
	opt := buildOptions{}
	for _, option := range options {
		option(&opt)
	}
	return &moduleBucketBuilder{opt: opt}
}

// BuildForBucket is an alternative constructor of NewModuleBucketBuilder
// followed by calling the BuildForBucket method.
func BuildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *bufmoduleconfig.Config,
	options ...BuildOption,
) (bufmodule.Module, error) {
	builder := newModuleBucketBuilder(options...)
	bm, err := builder.BuildForBucket(
		ctx,
		readBucket,
		config,
	)
	if err != nil {
		return nil, err
	}
	return bm.Module, nil
}

// BuildForBucket constructs a minimal bucket from the passed readBucket and
// builds a module from it.
//
// config's value is used even if the bucket contains configuration (buf.yaml).
// This means the module is built differently than described in storage, which
// may cause building to fail or succeed when it shouldn't. For your own
// sanity, you should pass a config value read from the provided bucket.
func (b *moduleBucketBuilder) BuildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *bufmoduleconfig.Config,
) (*BuiltModule, error) {
	// proxy plain files
	externalPaths := []string{
		buflock.ExternalConfigFilePath,
		bufmodule.DocumentationFilePath,
		bufmodule.LicenseFilePath,
	}
	externalPaths = append(externalPaths, bufconfig.AllConfigFilePaths...)
	rootBuckets := make([]storage.ReadBucket, 0, len(externalPaths))
	for _, path := range externalPaths {
		bucket, err := getFileReadBucket(ctx, readBucket, path)
		if err != nil {
			return nil, err
		}
		if bucket != nil {
			rootBuckets = append(rootBuckets, bucket)
		}
	}

	roots := make([]string, 0, len(config.RootToExcludes))
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
	bucket := storage.MultiReadBucket(rootBuckets...)
	module, err := bufmodule.NewModuleForBucket(
		ctx,
		bucket,
		bufmodule.ModuleWithModuleIdentity(
			b.opt.moduleIdentity, // This may be nil
		),
	)
	if err != nil {
		return nil, err
	}
	appliedModule, err := applyModulePaths(
		module,
		roots,
		b.opt.paths,
		b.opt.excludePaths,
		b.opt.pathsAllowNotExist,
		normalpath.Relative,
	)
	if err != nil {
		return nil, err
	}
	return &BuiltModule{
		Module: appliedModule,
		Bucket: bucket,
	}, nil
}

// may return nil.
func getFileReadBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	filePath string,
) (storage.ReadBucket, error) {
	fileData, err := storage.ReadPath(ctx, readBucket, filePath)
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(fileData) == 0 {
		return nil, nil
	}
	return storagemem.NewReadBucket(
		map[string][]byte{
			filePath: fileData,
		},
	)
}
