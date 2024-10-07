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

package bufworkspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/google/uuid"
)

// WorkspaceProvider provides Workspaces and UpdateableWorkspaces.
type WorkspaceProvider interface {
	// GetWorkspaceForBucket returns a new Workspace for the given Bucket.
	//
	// If the underlying bucket has a v2 buf.yaml at the root, this builds a Workspace for this buf.yaml,
	// using TargetSubDirPath for targeting.
	//
	// If the underlying bucket has a buf.work.yaml at the root, this builds a Workspace with all the modules
	// specified in the buf.work.yaml, using TargetSubDirPath for targeting.
	//
	// Otherwise, this builds a Workspace with a single module at the TargetSubDirPath (which may be "."),
	// assuming v1 defaults.
	//
	// If a config override is specified, all buf.work.yamls are ignored. If the config override is v1,
	// this builds a single module at the TargetSubDirPath, if the config override is v2, the builds
	// at the root, using TargetSubDirPath for targeting.
	//
	// All parsing of configuration files is done behind the scenes here.
	GetWorkspaceForBucket(
		ctx context.Context,
		bucket storage.ReadBucket,
		bucketTargeting buftarget.BucketTargeting,
		options ...WorkspaceBucketOption,
	) (Workspace, error)

	// GetWorkspaceForModuleKey wraps the ModuleKey into a workspace, returning defaults
	// for config values, and empty ConfiguredDepModuleRefs.
	//
	// This is useful for getting Workspaces for remote modules, but you still need
	// associated configuration.
	GetWorkspaceForModuleKey(
		ctx context.Context,
		moduleKey bufmodule.ModuleKey,
		options ...WorkspaceModuleKeyOption,
	) (Workspace, error)
}

// NewWorkspaceProvider returns a new WorkspaceProvider.
func NewWorkspaceProvider(
	logger *slog.Logger,
	graphProvider bufmodule.GraphProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
) WorkspaceProvider {
	return newWorkspaceProvider(
		logger,
		graphProvider,
		moduleDataProvider,
		commitProvider,
	)
}

// *** PRIVATE ***

type workspaceProvider struct {
	logger             *slog.Logger
	graphProvider      bufmodule.GraphProvider
	moduleDataProvider bufmodule.ModuleDataProvider
	commitProvider     bufmodule.CommitProvider
}

func newWorkspaceProvider(
	logger *slog.Logger,
	graphProvider bufmodule.GraphProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
) *workspaceProvider {
	return &workspaceProvider{
		logger:             logger,
		graphProvider:      graphProvider,
		moduleDataProvider: moduleDataProvider,
		commitProvider:     commitProvider,
	}
}

func (w *workspaceProvider) GetWorkspaceForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
	options ...WorkspaceModuleKeyOption,
) (Workspace, error) {
	defer slogext.DebugProfile(w.logger)()

	config, err := newWorkspaceModuleKeyConfig(options)
	if err != nil {
		return nil, err
	}
	// By default, the associated configuration for a Module gotten by ModuleKey is just
	// the default config. However, if we have a config override, we may have different
	// lint or breaking config. We will only apply this different config for the specific
	// module we are targeting, while the rest will retain the default config - generally,
	// you shouldn't be linting or doing breaking change detection for any module other
	// than the one your are targeting (which matches v1 behavior as well). In v1, we didn't
	// have a "workspace" for modules gotten by module reference, we just had the single
	// module we were building against, and whatever config override we had only applied
	// to that module. In v2, we have a ModuleSet, and we need lint and breaking config
	// for every modules in the ModuleSet, so we attach default lint and breaking config,
	// but given the presence of ignore_only, we don't want to apply configOverride to
	// non-target modules as the config override might have file paths, and we won't
	// lint or breaking change detect against non-target modules anyways.
	targetModuleConfig := bufconfig.DefaultModuleConfigV1
	// By default, there will be no plugin configs, however, similar to the lint and breaking
	// configs, there may be an override, in which case, we need to populate the plugin configs
	// from the override.
	var pluginConfigs []bufconfig.PluginConfig
	if config.configOverride != "" {
		bufYAMLFile, err := bufconfig.GetBufYAMLFileForOverride(config.configOverride)
		if err != nil {
			return nil, err
		}
		moduleConfigs := bufYAMLFile.ModuleConfigs()
		switch len(moduleConfigs) {
		case 0:
			return nil, syserror.New("had BufYAMLFile with 0 ModuleConfigs")
		case 1:
			// If we have a single ModuleConfig, we assume that regardless of whether or not
			// This ModuleConfig has a name, that this is what the user intends to associate
			// with the tqrget module. This also handles the v1 case - v1 buf.yamls will always
			// only have a single ModuleConfig, and it was expected pre-refactor that regardless
			// of if the ModuleConfig had a name associated with it or not, the lint and breaking
			// config that came from it would be associated.
			targetModuleConfig = moduleConfigs[0]
		default:
			// If we have more than one ModuleConfig, find the ModuleConfig that matches the
			// name from the ModuleKey. If none is found, just fall back to the default (ie do nothing here).
			for _, moduleConfig := range moduleConfigs {
				moduleFullName := moduleConfig.ModuleFullName()
				if moduleFullName == nil {
					continue
				}
				if bufmodule.ModuleFullNameEqual(moduleFullName, moduleKey.ModuleFullName()) {
					targetModuleConfig = moduleConfig
					// We know that the ModuleConfigs are unique by ModuleFullName.
					break
				}
			}
		}
		if bufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
			pluginConfigs = bufYAMLFile.PluginConfigs()
		}
	}

	moduleSet, err := bufmodule.NewModuleSetForRemoteModule(
		ctx,
		w.logger,
		w.graphProvider,
		w.moduleDataProvider,
		w.commitProvider,
		moduleKey,
		bufmodule.RemoteModuleWithTargetPaths(
			config.targetPaths,
			config.targetExcludePaths,
		),
	)
	if err != nil {
		return nil, err
	}

	opaqueIDToLintConfig := make(map[string]bufconfig.LintConfig)
	opaqueIDToBreakingConfig := make(map[string]bufconfig.BreakingConfig)
	for _, module := range moduleSet.Modules() {
		if bufmodule.ModuleFullNameEqual(module.ModuleFullName(), moduleKey.ModuleFullName()) {
			// Set the lint and breaking config for the single targeted Module.
			opaqueIDToLintConfig[module.OpaqueID()] = targetModuleConfig.LintConfig()
			opaqueIDToBreakingConfig[module.OpaqueID()] = targetModuleConfig.BreakingConfig()
		} else {
			// For all non-targets, set the default lint and breaking config.
			opaqueIDToLintConfig[module.OpaqueID()] = bufconfig.DefaultLintConfigV1
			opaqueIDToBreakingConfig[module.OpaqueID()] = bufconfig.DefaultBreakingConfigV1
		}
	}
	return newWorkspace(
		moduleSet,
		opaqueIDToLintConfig,
		opaqueIDToBreakingConfig,
		pluginConfigs,
		nil,
		false,
	), nil
}

func (w *workspaceProvider) GetWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
	options ...WorkspaceBucketOption,
) (Workspace, error) {
	defer slogext.DebugProfile(w.logger)()
	config, err := newWorkspaceBucketConfig(options)
	if err != nil {
		return nil, err
	}
	var overrideBufYAMLFile bufconfig.BufYAMLFile
	if config.configOverride != "" {
		overrideBufYAMLFile, err = bufconfig.GetBufYAMLFileForOverride(config.configOverride)
		if err != nil {
			return nil, err
		}
	}
	workspaceTargeting, err := newWorkspaceTargeting(
		ctx,
		w.logger,
		config,
		bucket,
		bucketTargeting,
		overrideBufYAMLFile,
		config.ignoreAndDisallowV1BufWorkYAMLs,
	)
	if err != nil {
		return nil, err
	}
	if workspaceTargeting.v2 != nil {
		return w.getWorkspaceForBucketBufYAMLV2(
			ctx,
			bucket,
			workspaceTargeting.v2,
		)
	}
	return w.getWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
		ctx,
		bucket,
		workspaceTargeting.v1,
	)
}

func (w *workspaceProvider) getWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
	ctx context.Context,
	bucket storage.ReadBucket,
	v1WorkspaceTargeting *v1Targeting,
) (*workspace, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, w.logger, w.moduleDataProvider, w.commitProvider)
	for _, moduleBucketAndTargeting := range v1WorkspaceTargeting.moduleBucketsAndTargeting {
		mappedModuleBucket := moduleBucketAndTargeting.bucket
		moduleTargeting := moduleBucketAndTargeting.moduleTargeting
		bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
			ctx,
			bucket, // Need to use the non-mapped bucket since the mapped bucket excludes the buf.lock
			moduleTargeting.moduleDirPath,
			bufconfig.BufLockFileWithDigestResolver(
				func(ctx context.Context, remote string, commitID uuid.UUID) (bufmodule.Digest, error) {
					commitKey, err := bufmodule.NewCommitKey(remote, commitID, bufmodule.DigestTypeB4)
					if err != nil {
						return nil, err
					}
					commits, err := w.commitProvider.GetCommitsForCommitKeys(ctx, []bufmodule.CommitKey{commitKey})
					if err != nil {
						return nil, err
					}
					return commits[0].ModuleKey().Digest()
				},
			),
		)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		} else {
			switch fileVersion := bufLockFile.FileVersion(); fileVersion {
			case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			case bufconfig.FileVersionV2:
				return nil, errors.New("got a v2 buf.lock file for a v1 buf.yaml - this is not allowed, run buf mod update to update your buf.lock file")
			default:
				return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
			}
			for _, depModuleKey := range bufLockFile.DepModuleKeys() {
				// DepModuleKeys from a BufLockFile is expected to have all transitive dependencies,
				// and we can rely on this property.
				moduleSetBuilder.AddRemoteModule(
					depModuleKey,
					false,
				)
			}
		}
		v1BufYAMLObjectData, err := bufconfig.GetBufYAMLV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, moduleTargeting.moduleDirPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		v1BufLockObjectData, err := bufconfig.GetBufLockV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, moduleTargeting.moduleDirPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		// Each moduleBucketAndTargeting represents a local module that we want to add to the moduleSet,
		// and we look up its moduleConfig by its bucketID, because that is guaranteed to be unique (technically,
		// moduleDirPath is also unique in v1/v1beta1, but just to be extra safe).
		moduleConfig, ok := v1WorkspaceTargeting.bucketIDToModuleConfig[moduleBucketAndTargeting.bucketID]
		if !ok {
			// This should not happen since moduleBucketAndTargeting is derived from the module
			// configs, however, we return this error as a safety check
			return nil, fmt.Errorf("no module config found for module at: %q", moduleTargeting.moduleDirPath)
		}
		moduleSetBuilder.AddLocalModule(
			mappedModuleBucket,
			moduleBucketAndTargeting.bucketID,
			moduleTargeting.isTargetModule,
			bufmodule.LocalModuleWithModuleFullName(moduleConfig.ModuleFullName()),
			bufmodule.LocalModuleWithTargetPaths(
				moduleTargeting.moduleTargetPaths,
				moduleTargeting.moduleTargetExcludePaths,
			),
			bufmodule.LocalModuleWithProtoFileTargetPath(
				moduleTargeting.moduleProtoFileTargetPath,
				moduleTargeting.includePackageFiles,
			),
			bufmodule.LocalModuleWithV1Beta1OrV1BufYAMLObjectData(v1BufYAMLObjectData),
			bufmodule.LocalModuleWithV1Beta1OrV1BufLockObjectData(v1BufLockObjectData),
			bufmodule.LocalModuleWithDescription(
				getLocalModuleDescription(
					// See comments on getLocalModuleDescription.
					moduleBucketAndTargeting.bucketID,
					moduleConfig,
				),
			),
		)
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return w.getWorkspaceForBucketModuleSet(
		moduleSet,
		v1WorkspaceTargeting.bucketIDToModuleConfig,
		nil,
		v1WorkspaceTargeting.allConfiguredDepModuleRefs,
		false,
	)
}

func (w *workspaceProvider) getWorkspaceForBucketBufYAMLV2(
	ctx context.Context,
	bucket storage.ReadBucket,
	v2Targeting *v2Targeting,
) (*workspace, error) {
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, w.logger, w.moduleDataProvider, w.commitProvider)
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
		ctx,
		bucket,
		// buf.lock files live next to the buf.yaml
		".",
		// We are not passing BufLockFileWithDigestResolver here because a buf.lock
		// v2 is expected to have digests
	)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	} else {
		switch fileVersion := bufLockFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			return nil, fmt.Errorf("got a %s buf.lock file for a v2 buf.yaml", bufLockFile.FileVersion().String())
		case bufconfig.FileVersionV2:
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
		for _, depModuleKey := range bufLockFile.DepModuleKeys() {
			// DepModuleKeys from a BufLockFile is expected to have all transitive dependencies,
			// and we can rely on this property.
			moduleSetBuilder.AddRemoteModule(
				depModuleKey,
				false,
			)
		}
	}
	// Only check for duplicate module description in v2, which would be an user error, i.e.
	// This is not a system error:
	// modules:
	//   - path: proto
	//     excludes:
	//       - proot/foo
	//   - path: proto
	//     excludes:
	//       - proot/foo
	// but duplicate module description in v1 is a system error, which the ModuleSetBuilder catches.
	seenModuleDescriptions := make(map[string]struct{})
	for _, moduleBucketAndTargeting := range v2Targeting.moduleBucketsAndTargeting {
		mappedModuleBucket := moduleBucketAndTargeting.bucket
		moduleTargeting := moduleBucketAndTargeting.moduleTargeting
		// Each moduleBucketAndTargeting represents a local module that we want to add to the moduleSet,
		// and we look up its moduleConfig by its bucketID, because that is guaranteed to be unique (moduleDirPaths
		// are not in a v2 workspace).
		moduleConfig, ok := v2Targeting.bucketIDToModuleConfig[moduleBucketAndTargeting.bucketID]
		if !ok {
			// This should not happen since moduleBucketAndTargeting is derived from the module
			// configs, however, we return this error as a safety check
			return nil, fmt.Errorf("no module config found for module at: %q", moduleTargeting.moduleDirPath)
		}
		moduleDescription := getLocalModuleDescription(
			// See comments on getLocalModuleDescription.
			moduleConfig.DirPath(),
			moduleConfig,
		)
		if _, ok := seenModuleDescriptions[moduleDescription]; ok {
			return nil, fmt.Errorf("multiple module configs found with the same description: %s", moduleDescription)
		}
		seenModuleDescriptions[moduleDescription] = struct{}{}
		moduleSetBuilder.AddLocalModule(
			mappedModuleBucket,
			moduleBucketAndTargeting.bucketID,
			moduleTargeting.isTargetModule,
			bufmodule.LocalModuleWithModuleFullName(moduleConfig.ModuleFullName()),
			bufmodule.LocalModuleWithTargetPaths(
				moduleTargeting.moduleTargetPaths,
				moduleTargeting.moduleTargetExcludePaths,
			),
			bufmodule.LocalModuleWithProtoFileTargetPath(
				moduleTargeting.moduleProtoFileTargetPath,
				moduleTargeting.includePackageFiles,
			),
			bufmodule.LocalModuleWithDescription(moduleDescription),
		)
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return w.getWorkspaceForBucketModuleSet(
		moduleSet,
		v2Targeting.bucketIDToModuleConfig,
		v2Targeting.bufYAMLFile.PluginConfigs(),
		v2Targeting.bufYAMLFile.ConfiguredDepModuleRefs(),
		true,
	)
}

// only use for workspaces created from buckets
func (w *workspaceProvider) getWorkspaceForBucketModuleSet(
	moduleSet bufmodule.ModuleSet,
	bucketIDToModuleConfig map[string]bufconfig.ModuleConfig,
	pluginConfigs []bufconfig.PluginConfig,
	// Expected to already be unique by ModuleFullName.
	configuredDepModuleRefs []bufmodule.ModuleRef,
	isV2 bool,
) (*workspace, error) {
	opaqueIDToLintConfig := make(map[string]bufconfig.LintConfig)
	opaqueIDToBreakingConfig := make(map[string]bufconfig.BreakingConfig)
	for _, module := range moduleSet.Modules() {
		if bucketID := module.BucketID(); bucketID != "" {
			moduleConfig, ok := bucketIDToModuleConfig[bucketID]
			if !ok {
				// This is a system error.
				return nil, syserror.Newf("could not get ModuleConfig for BucketID %q", bucketID)
			}
			opaqueIDToLintConfig[module.OpaqueID()] = moduleConfig.LintConfig()
			opaqueIDToBreakingConfig[module.OpaqueID()] = moduleConfig.BreakingConfig()
		} else {
			opaqueIDToLintConfig[module.OpaqueID()] = bufconfig.DefaultLintConfigV1
			opaqueIDToBreakingConfig[module.OpaqueID()] = bufconfig.DefaultBreakingConfigV1
		}
	}
	return newWorkspace(
		moduleSet,
		opaqueIDToLintConfig,
		opaqueIDToBreakingConfig,
		pluginConfigs,
		configuredDepModuleRefs,
		isV2,
	), nil
}

// This formats a module name based on its module config entry in the v2 buf.yaml:
// `path: foo, includes: ["foo/v1, "foo/v2"], excludes: "foo/v1/internal"`.
//
// For v1/v1beta1 modules, pathDescription should be bucketID.
// For v2 modules, pathDescription should be moduleConfig.DirPath().
//
// We edit bucketIDs in v2 to include an index since directories can be overlapping.
// We would want to use moduleConfig.DirPath() everywhere, but it is always "." in
// v1/v1beta1, and it's not a good description.
func getLocalModuleDescription(pathDescription string, moduleConfig bufconfig.ModuleConfig) string {
	description := fmt.Sprintf("path: %q", pathDescription)
	moduleDirPath := moduleConfig.DirPath()
	relIncludePaths := moduleConfig.RootToIncludes()["."]
	includePaths := slicesext.Map(relIncludePaths, func(relInclude string) string {
		return normalpath.Join(moduleDirPath, relInclude)
	})
	switch len(includePaths) {
	case 0:
	case 1:
		description = fmt.Sprintf("%s, includes: %q", description, includePaths[0])
	default:
		description = fmt.Sprintf("%s, includes: [%s]", description, stringutil.JoinSliceQuoted(includePaths, ", "))
	}
	relExcludePaths := moduleConfig.RootToExcludes()["."]
	excludePaths := slicesext.Map(relExcludePaths, func(relInclude string) string {
		return normalpath.Join(moduleDirPath, relInclude)
	})
	switch len(excludePaths) {
	case 0:
	case 1:
		description = fmt.Sprintf("%s, excludes: %q", description, excludePaths[0])
	default:
		description = fmt.Sprintf("%s, excludes: [%s]", description, stringutil.JoinSliceQuoted(excludePaths, ", "))
	}
	return description
}
