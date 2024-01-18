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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

// ModuleFileSetBuilder builds ModuleFileSets from Modules.
type ModuleFileSetBuilder interface {
	Build(
		ctx context.Context,
		module bufmodule.Module,
		options ...BuildModuleFileSetOption,
	) (bufmodule.ModuleFileSet, error)
}

// NewModuleFileSetBuilder returns a new ModuleSetProvider.
func NewModuleFileSetBuilder(
	logger *zap.Logger,
	moduleReader bufmodule.ModuleReader,
) ModuleFileSetBuilder {
	return newModuleFileSetBuilder(logger, moduleReader)
}

// BuildModuleFileSetOption is an option for Build.
type BuildModuleFileSetOption func(*buildModuleFileSetOptions)

// WithWorkspace returns a new BuildModuleFileSetOption that specifies a workspace.
func WithWorkspace(workspace bufmodule.Workspace) BuildModuleFileSetOption {
	return func(buildModuleFileSetOptions *buildModuleFileSetOptions) {
		buildModuleFileSetOptions.workspace = workspace
	}
}

// BuiltModule ties a bufmodule.Module with the configuration and a bucket
// containing just the files required to build it.
type BuiltModule struct {
	bufmodule.Module
	Bucket storage.ReadBucket
}

// ModuleBucketBuilder builds modules for buckets.
type ModuleBucketBuilder interface {
	// BuildForBucket constructs a minimal bucket from the passed readBucket and
	// builds a module from it.
	//
	// config's value is used even if the bucket contains configuration (buf.yaml).
	// This means the module is built differently than described in storage, which
	// may cause building to fail or succeed when it shouldn't. For your own
	// sanity, you should pass a config value read from the provided bucket.
	//
	// TODO: why do we pass a config here?! This parameter should be removed.
	BuildForBucket(
		ctx context.Context,
		readBucket storage.ReadBucket,
		config *bufmoduleconfig.Config,
		options ...BuildOption,
	) (*BuiltModule, error)
}

// NewModuleBucketBuilder returns a new BucketBuilder.
func NewModuleBucketBuilder() ModuleBucketBuilder {
	return newModuleBucketBuilder()
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
// would be nice to be able to do buf alpha protoc -I path/to/dir -I buf.build/foo/bar/v1
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
		buildOptions.paths = &paths
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
// This results in ModuleWithPathsAllowNotExist being used on the resulting build module.
// This is done within bufmodulebuild so we can resolve the paths relative to their roots.
func WithPathsAllowNotExist(paths []string) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.paths = &paths
		buildOptions.pathsAllowNotExist = true
	}
}

// WithModuleIdentity returns a new BuildOption that is used to construct a Module with a ModuleIdentity.
//
// TODO: this is never called
// TODO: we also have ModuleWithModuleIdentityAndCommit in bufmodule
// We need to disambiguate module building between bufmodule and bufmodulebuild
// bufimage and bufimagebuild work, but bufmodule and bufmodulebuild are a mess
func WithModuleIdentity(moduleIdentity bufmoduleref.ModuleIdentity) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.moduleIdentity = moduleIdentity
	}
}

// WithExcludePaths returns a new BuildOption that specifies files to be excluded from the build.
func WithExcludePaths(excludePaths []string) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.excludePaths = excludePaths
	}
}

// WithExcludePathsAllowNotExist returns a new BuildOption that specifies files to be excluded from the build,
// but allows the specified paths to not exist.
func WithExcludePathsAllowNotExist(excludePaths []string) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.excludePaths = excludePaths
		buildOptions.pathsAllowNotExist = true
	}
}

// WithWorkspaceDirectory returns a new BuildOption that specifies the workspace directory.
//
// See the comment on Module.WorkspaceDirectory() for more details.
func WithWorkspaceDirectory(workspaceDirectory string) BuildOption {
	return func(buildOptions *buildOptions) {
		buildOptions.workspaceDirectory = workspaceDirectory
	}
}
