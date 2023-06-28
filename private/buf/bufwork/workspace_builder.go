// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufwork

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type workspaceBuilder struct {
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder
	moduleCache         map[string]*cachedModule
}

func newWorkspaceBuilder(
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
) *workspaceBuilder {
	return &workspaceBuilder{
		moduleBucketBuilder: moduleBucketBuilder,
		moduleCache:         make(map[string]*cachedModule),
	}
}

// BuildWorkspace builds a bufmodule.Workspace for the given targetSubDirPath.
func (w *workspaceBuilder) BuildWorkspace(
	ctx context.Context,
	workspaceConfig *Config,
	readBucket storage.ReadBucket,
	relativeRootPath string,
	targetSubDirPath string,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (bufmodule.Workspace, error) {
	if workspaceConfig == nil {
		return nil, errors.New("received a nil workspace config")
	}
	// We know that if the file is actually buf.work for legacy reasons, this will be wrong,
	// but we accept that as this shouldn't happen often anymore and this is just
	// used for error messages.
	workspaceID := filepath.Join(normalpath.Unnormalize(relativeRootPath), ExternalConfigV1FilePath)
	namedModules := make(map[string]bufmodule.Module, len(workspaceConfig.Directories))
	allModules := make([]bufmodule.Module, 0, len(workspaceConfig.Directories))
	for _, directory := range workspaceConfig.Directories {
		if cachedModule, ok := w.moduleCache[directory]; ok {
			if directory == targetSubDirPath {
				continue
			}
			// We've already built this module, so we can use the cached-equivalent.
			if moduleIdentity := cachedModule.moduleConfig.ModuleIdentity; moduleIdentity != nil {
				if _, ok := namedModules[moduleIdentity.IdentityString()]; ok {
					return nil, fmt.Errorf(
						"module %q is provided by multiple workspace directories listed in %s",
						moduleIdentity.IdentityString(),
						workspaceID,
					)
				}
				namedModules[moduleIdentity.IdentityString()] = cachedModule.module
			}
			allModules = append(allModules, cachedModule.module)
			continue
		}
		readBucketForDirectory := storage.MapReadBucket(readBucket, storage.MapOnPrefix(directory))
		if err := validateWorkspaceDirectoryNonEmpty(ctx, readBucketForDirectory, directory, workspaceID); err != nil {
			return nil, err
		}
		if err := validateInputOverlap(directory, targetSubDirPath, workspaceID); err != nil {
			return nil, err
		}
		// Ignore the configOverride for anything that isn't the target path
		localConfigOverride := configOverride
		if directory != targetSubDirPath {
			localConfigOverride = ""
		}
		moduleConfig, err := bufconfig.ReadConfigOS(
			ctx,
			readBucketForDirectory,
			bufconfig.ReadConfigOSWithOverride(localConfigOverride),
		)
		if err != nil {
			return nil, fmt.Errorf(
				`failed to get module config for directory "%s" listed in %s: %w`,
				normalpath.Unnormalize(directory),
				workspaceID,
				err,
			)
		}
		externalToSubDirRelPaths, err := ExternalPathsToSubDirRelPaths(
			relativeRootPath,
			directory,
			externalDirOrFilePaths,
		)
		if err != nil {
			return nil, err
		}
		excludeToSubDirRelExcludePaths, err := ExternalPathsToSubDirRelPaths(
			relativeRootPath,
			directory,
			externalExcludeDirOrFilePaths,
		)
		if err != nil {
			return nil, err
		}
		subDirRelPaths := make([]string, 0, len(externalToSubDirRelPaths))
		for _, subDirRelPath := range externalToSubDirRelPaths {
			subDirRelPaths = append(subDirRelPaths, subDirRelPath)
		}
		subDirRelExcludePaths := make([]string, 0, len(excludeToSubDirRelExcludePaths))
		for _, subDirRelExcludePath := range excludeToSubDirRelExcludePaths {
			subDirRelExcludePaths = append(subDirRelExcludePaths, subDirRelExcludePath)
		}
		buildOptions, err := BuildOptionsForWorkspaceDirectory(
			ctx,
			workspaceConfig,
			moduleConfig,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			subDirRelPaths,
			subDirRelExcludePaths,
			externalDirOrFilePathsAllowNotExist,
		)
		if err != nil {
			return nil, err
		}
		module, err := bufmodulebuild.BuildForBucket(
			ctx,
			readBucketForDirectory,
			moduleConfig.Build,
			buildOptions...,
		)
		if err != nil {
			return nil, fmt.Errorf(
				`failed to initialize module for directory "%s" listed in %s: %w`,
				normalpath.Unnormalize(directory),
				workspaceID,
				err,
			)
		}
		w.moduleCache[directory] = newCachedModule(
			module,
			moduleConfig,
		)
		if directory == targetSubDirPath {
			// We don't want to include the module found at the targetSubDirPath
			// since it would otherwise be included twice. Note that we include
			// this check here so that the module is still built and cached upfront.
			continue
		}
		if moduleIdentity := moduleConfig.ModuleIdentity; moduleIdentity != nil {
			if _, ok := namedModules[moduleIdentity.IdentityString()]; ok {
				return nil, fmt.Errorf(
					"module %q is provided by multiple workspace directories listed in %s",
					moduleIdentity.IdentityString(),
					workspaceID,
				)
			}
			namedModules[moduleIdentity.IdentityString()] = module
		}
		allModules = append(allModules, module)
	}
	return bufmodule.NewWorkspace(
		ctx,
		namedModules,
		allModules,
	)
}

// GetModuleConfig returns the bufmodule.Module and *bufconfig.Config, associated with the given
// targetSubDirPath, if it exists.
func (w *workspaceBuilder) GetModuleConfig(targetSubDirPath string) (bufmodule.Module, *bufconfig.Config, bool) {
	cachedModule, ok := w.moduleCache[targetSubDirPath]
	if !ok {
		return nil, nil, false
	}
	return cachedModule.module, cachedModule.moduleConfig, true
}

func validateWorkspaceDirectoryNonEmpty(
	ctx context.Context,
	readBucket storage.ReadBucket,
	workspaceDirectory string,
	workspaceID string,
) error {
	isEmpty, err := storage.IsEmpty(
		ctx,
		storage.MapReadBucket(readBucket, storage.MatchPathExt(".proto")),
		"",
	)
	if err != nil {
		return err
	}
	if isEmpty {
		return fmt.Errorf(
			`directory "%s" listed in %s contains no .proto files`,
			normalpath.Unnormalize(workspaceDirectory),
			workspaceID,
		)
	}
	return nil
}

// validateInputOverlap returns a non-nil error if the given directories
// overlap in either direction. The last argument is only used for
// error reporting.
//
//	validateInputOverlap("foo", "bar", "buf.work.yaml")     -> OK
//	validateInputOverlap("foo/bar", "foo", "buf.work.yaml") -> NOT OK
//	validateInputOverlap("foo", "foo/bar", "buf.work.yaml") -> NOT OK
func validateInputOverlap(
	workspaceDirectory string,
	targetSubDirPath string,
	workspaceID string,
) error {
	if normalpath.ContainsPath(workspaceDirectory, targetSubDirPath, normalpath.Relative) {
		return fmt.Errorf(
			`failed to build input "%s" because it is contained by directory "%s" listed in %s`,
			normalpath.Unnormalize(targetSubDirPath),
			normalpath.Unnormalize(workspaceDirectory),
			workspaceID,
		)
	}

	if normalpath.ContainsPath(targetSubDirPath, workspaceDirectory, normalpath.Relative) {
		return fmt.Errorf(
			`failed to build input "%s" because it contains directory "%s" listed in %s`,
			normalpath.Unnormalize(targetSubDirPath),
			normalpath.Unnormalize(workspaceDirectory),
			workspaceID,
		)
	}
	return nil
}

// cachedModule encapsulates a module and its configuration.
type cachedModule struct {
	module       bufmodule.Module
	moduleConfig *bufconfig.Config
}

func newCachedModule(
	module bufmodule.Module,
	moduleConfig *bufconfig.Config,
) *cachedModule {
	return &cachedModule{
		module:       module,
		moduleConfig: moduleConfig,
	}
}
