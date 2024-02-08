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

package bufmodulebuild

import (
	"context"
	"errors"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
)

type moduleBucketBuilder struct {
}

func newModuleBucketBuilder() *moduleBucketBuilder {
	return &moduleBucketBuilder{}
}

func (b *moduleBucketBuilder) BuildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *bufmoduleconfig.Config,
	options ...BuildOption,
) (*BuiltModule, error) {
	buildOptions := &buildOptions{}
	for _, option := range options {
		option(buildOptions)
	}
	return b.buildForBucket(
		ctx,
		readBucket,
		config,
		buildOptions,
	)
}

func (b *moduleBucketBuilder) buildForBucket(
	ctx context.Context,
	readBucket storage.ReadBucket,
	config *bufmoduleconfig.Config,
	buildOptions *buildOptions,
) (*BuiltModule, error) {
	// proxy plain files
	externalPaths := []string{
		buflock.ExternalConfigFilePath,
		bufmodule.LicenseFilePath,
	}
	externalPaths = append(externalPaths, bufconfig.AllConfigFilePaths...)
	rootBuckets := make([]storage.ReadBucket, 0, len(externalPaths)+1)
	for _, docPath := range bufmodule.AllDocumentationPaths {
		bucket, err := getFileReadBucket(ctx, readBucket, docPath)
		if err != nil {
			return nil, err
		}
		if bucket != nil {
			rootBuckets = append(rootBuckets, bucket)
			break
		}
	}
	for _, path := range externalPaths {
		bucket, err := getFileReadBucket(ctx, readBucket, path)
		if err != nil {
			return nil, err
		}
		if bucket != nil {
			rootBuckets = append(rootBuckets, bucket)
		}
	}

	// The below logic relies on all roots being represented in the Config.
	//
	// The config Godoc says that if no roots are specified, the default of a single root
	// if "." with no excludes should be assumed. We need to account for this in the case
	// where an empty configuration is passed to this function.
	//
	// To make sure that we do not edit the input Config, we make a proper copy of the map.
	rootToExcludes := make(map[string][]string, len(config.RootToExcludes))
	for root, excludes := range config.RootToExcludes {
		// You can't modify a slice, so this is OK
		rootToExcludes[root] = excludes
	}
	if len(rootToExcludes) == 0 {
		rootToExcludes["."] = []string{}
	}

	roots := make([]string, 0, len(rootToExcludes))
	for root, excludes := range rootToExcludes {
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
			buildOptions.moduleIdentity, // This may be nil
		),
		bufmodule.ModuleWithWorkspaceDirectory(
			buildOptions.workspaceDirectory,
		),
	)
	if err != nil {
		return nil, err
	}
	appliedModule, err := applyModulePaths(
		module,
		roots,
		buildOptions.paths,
		buildOptions.excludePaths,
		buildOptions.pathsAllowNotExist,
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
		if errors.Is(err, fs.ErrNotExist) {
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
