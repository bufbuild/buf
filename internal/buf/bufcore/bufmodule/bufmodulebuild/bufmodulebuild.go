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
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/zap"
)

// ModuleFileSetBuilder builds ModuleFileSets from Modules.
type ModuleFileSetBuilder interface {
	Build(
		ctx context.Context,
		module bufmodule.Module,
	) (bufmodule.ModuleFileSet, error)
}

// NewModuleFileSetBuilder returns a new ModuleSetProvider.
func NewModuleFileSetBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
) ModuleFileSetBuilder {
	return newModuleFileSetBuilder(logger, moduleReader)
}

// ModuleBucketBuilder builds modules for buckets.
type ModuleBucketBuilder interface {
	// BuildForBucket builds a module for the given bucket.
	//
	// If paths is empty, all files are built.
	// Paths should be relative to the bucket, not the roots.
	BuildForBucket(
		ctx context.Context,
		readBucket storage.ReadBucket,
		config *Config,
		options ...BuildOption,
	) (bufmodule.Module, error)
}

// NewModuleBucketBuilder returns a new BucketBuilder.
func NewModuleBucketBuilder(logger *zap.Logger) ModuleBucketBuilder {
	return newModuleBucketBuilder(logger)
}

// ModuleIncludeBuilder builds modules for includes.
//
// This is used for protoc.
type ModuleIncludeBuilder interface {
	// BuildForIncludes builds a module for the given includes and file paths.
	BuildForIncludes(
		ctx context.Context,
		includeDirPaths []string,
		options ...BuildOption,
	) (bufmodule.Module, error)
}

// NewModuleIncludeBuilder returns a new ModuleIncludeBuilder.
//
// TODO: we should parse includeDirPaths for modules as well in theory
// would be nice to be able to do buf protoc -I path/to/dir -I buf.build/foo/bar/v1
func NewModuleIncludeBuilder(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) ModuleIncludeBuilder {
	return newModuleIncludeBuilder(logger, storageosProvider)
}

// BuildOption is an option for BuildForBucket.
type BuildOption func(*buildOptions)

// WithPaths returns a new BuildOption that specifies specific file or directory paths to build.
//
// These paths must exist.
// These paths must be relative to the bucket or include directory paths.
// These paths will be normalized.
// Multiple calls to this option and WithPathsAllowNotExist will override previous calls.
//
// This results in ModuleWithTargetPaths being used on the resulting build module.
// This is done within bufmodulebuild so we can resolve the paths relative to their roots.
func WithPaths(paths []string) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.paths = paths
	}
}

// WithPathsAllowNotExist returns a new BuildOption that specifies specific file or directory paths to build,
// but allows the specified paths to not exist.
//
// These paths must exist.
// These paths must be relative to the bucket or include directory paths.
// These paths will be normalized.
// Multiple calls to this option and WithPaths will override previous calls.
//
// This results in ModuleWithTargetPathsAllowNotExist being used on the resulting build module.
// This is done within bufmodulebuild so we can resolve the paths relative to their roots.
func WithPathsAllowNotExist(paths []string) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.paths = paths
		buildOptions.pathsAllowNotExist = true
	}
}

// Config is a configuration for build.
type Config struct {
	// RootToExcludes contains a map from root to the excludes for that root.
	//
	// Roots are the root directories within a bucket to search for Protobuf files.
	//
	// There will be no between the roots, ie foo/bar and foo are not allowed.
	// All Protobuf files must be unique relative to the roots, ie if foo and bar
	// are roots, then foo/baz.proto and bar/baz.proto are not allowed.
	//
	// All roots will be normalized and validated.
	//
	// Excludes are the directories within a bucket to exclude.
	//
	// There should be no overlap between the excludes, ie foo/bar and foo are not allowed.
	//
	// All excludes must reside within a root, but none will be equal to a root.
	// All excludes will be normalized and validated.
	// The excludes in this map will be relative to the root they map to!
	//
	// If RootToExcludes is empty, the default is "." with no excludes.
	RootToExcludes             map[string][]string
	DependencyModuleReferences []bufmodule.ModuleReference
}

// NewConfigV1Beta1 returns a new, validated Config for the ExternalConfig.
func NewConfigV1Beta1(externalConfig ExternalConfigV1Beta1, deps ...string) (*Config, error) {
	return newConfigV1Beta1(externalConfig, deps...)
}

// ExternalConfigV1Beta1 is an external config.
type ExternalConfigV1Beta1 struct {
	Roots    []string `json:"roots,omitempty" yaml:"roots,omitempty"`
	Excludes []string `json:"excludes,omitempty" yaml:"excludes,omitempty"`
}
