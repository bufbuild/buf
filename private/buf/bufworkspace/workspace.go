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

package bufworkspace

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracer"
	"go.uber.org/zap"
)

// Workspace is a buf workspace.
//
// It is a bufmodule.ModuleSet with associated configuration.
//
// See ModuleSet helper functions for many of your needs. Some examples:
//
//   - bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles
//   - bufmodule.ModuleSetToTargetModules
//   - bufmodule.ModuleSetRemoteDepsOfLocalModules - gives you exact deps to put in buf.lock
//
// To get a specific file from a Workspace:
//
//	moduleReadBucket := bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(workspace)
//	fileInfo, err := moduleReadBucket.GetFileInfo(ctx, path)
type Workspace interface {
	bufmodule.ModuleSet

	// GetLintConfigForOpaqueID gets the LintConfig for the OpaqueID, if the OpaqueID
	// represents a Module within the workspace.
	//
	// This will be the default value for Modules that didn't have an associated config,
	// such as Modules read from buf.lock files. These Modules will not be target Modules
	// in the workspace. This should result in items such as the linter or breaking change
	// detector ignoring these configs anyways.
	//
	// Returns nil if there is no Module with the given OpaqueID. However, as long
	// as the OpaqueID came from a Module contained within Modules(), this will always
	// return a non-nil value.
	//
	// Note that we originally designed exposing of Configs as:
	//
	//   type WorkspaceModule interface {
	//     bufmodule.Module
	//     LintConfig() LintConfig
	//   }
	//
	// However, this would mean that Workspace would not inherit ModuleSet, as we'd
	// want to create GetWorkspaceModule.* functions instead of GetModule.* functions,
	// and then provide a WorkpaceToModuleSet global function. This seems messier in
	// practice than having users call GetLintConfigForOpaqueID(module.OpaqueID())
	// in the situations where they need configuration.
	GetLintConfigForOpaqueID(opaqueID string) bufconfig.LintConfig

	// GetLintConfigForOpaqueID gets the LintConfig for the OpaqueID, if the OpaqueID
	// represents a Module within the workspace.
	//
	// This will be the default value for Modules that didn't have an associated config,
	// such as Modules read from buf.lock files. These Modules will not be target Modules
	// in the workspace. This should result in items such as the linter or breaking change
	// detector ignoring these configs anyways.
	GetBreakingConfigForOpaqueID(opaqueID string) bufconfig.BreakingConfig

	// ConfiguredDepModuleRefs returns the configured dependencies of the Workspace as ModuleRefs.
	//
	// These come from buf.yaml files.
	//
	// The ModuleRefs in this list may *not* be unique by ModuleFullName. When doing items
	// such as buf mod update, it is up to the caller to resolve conflicts. For example,
	// with v1 buf.yaml, this is a union of the deps in the buf.yaml files in the workspace.
	//
	// Sorted.
	// TODO: rename to AllConfiguredDepModuleRefs, to differentiate from BufYAMLFile?
	// TODO: use to warn on unused deps.
	ConfiguredDepModuleRefs() []bufmodule.ModuleRef

	isWorkspace()
}

// NewWorkspaceForBucket returns a new Workspace for the given Bucket.
//
// All parsing of configuration files is done behind the scenes here.
// This function can read a single v1 or v1beta1 buf.yaml, a v1 buf.work.yaml, or a v2 buf.yaml.
func NewWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceBucketOption,
) (Workspace, error) {
	return newWorkspaceForBucket(ctx, logger, bucket, moduleDataProvider, options...)
}

// NewWorkspaceForModuleKey wraps the ModuleKey into a workspace, returning defaults
// for config values, and empty ConfiguredDepModuleRefs.
//
// This is useful for getting Workspaces for remote modules, but you still need
// associated configuration.
func NewWorkspaceForModuleKey(
	ctx context.Context,
	logger *zap.Logger,
	moduleKey bufmodule.ModuleKey,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceModuleKeyOption,
) (Workspace, error) {
	return newWorkspaceForModuleKey(ctx, logger, moduleKey, moduleDataProvider, options...)
}

// NewWorkspaceForProtoc is a specialized function that creates a new Workspace
// for given includes and file paths in the style of protoc.
//
// The returned Workspace will have a single targeted Module, with target files
// matching the filePaths.
//
// Technically this will work with len(filePaths) == 0 but we should probably make sure
// that is banned in protoc.
func NewWorkspaceForProtoc(
	ctx context.Context,
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	includeDirPaths []string,
	filePaths []string,
) (Workspace, error) {
	return newWorkspaceForProtoc(ctx, logger, storageosProvider, includeDirPaths, filePaths)
}

// *** PRIVATE ***

type workspace struct {
	bufmodule.ModuleSet

	logger                   *zap.Logger
	opaqueIDToLintConfig     map[string]bufconfig.LintConfig
	opaqueIDToBreakingConfig map[string]bufconfig.BreakingConfig
	configuredDepModuleRefs  []bufmodule.ModuleRef

	// Set if this workspace is a buf.yaml-v2-backed workspace.
	//
	// This may also be set if there was no buf.yaml in the future, depending on our defaults.
	// Do not depend on this actually having a v2 buf.yaml
	isV2BufYAMLWorkspace bool
	// The path where buf.lock files should be written.
	//
	// Only and always set if isV2BufYAMLWorkspace is set.
	bufLockDirPath string
}

func (w *workspace) GetLintConfigForOpaqueID(opaqueID string) bufconfig.LintConfig {
	return w.opaqueIDToLintConfig[opaqueID]
}

func (w *workspace) GetBreakingConfigForOpaqueID(opaqueID string) bufconfig.BreakingConfig {
	return w.opaqueIDToBreakingConfig[opaqueID]
}

func (w *workspace) ConfiguredDepModuleRefs() []bufmodule.ModuleRef {
	return slicesext.Copy(w.configuredDepModuleRefs)
}

func (*workspace) isWorkspace() {}

func newWorkspaceForModuleKey(
	ctx context.Context,
	logger *zap.Logger,
	moduleKey bufmodule.ModuleKey,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceModuleKeyOption,
) (*workspace, error) {
	config, err := newWorkspaceModuleKeyConfig(options)
	if err != nil {
		return nil, err
	}
	// By default, the assocated configuration for a Module gotten by ModuleKey is just
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
	targetModuleConfig := bufconfig.DefaultModuleConfig
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
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, moduleDataProvider)
	moduleSetBuilder.AddRemoteModule(
		moduleKey,
		true,
		bufmodule.RemoteModuleWithTargetPaths(
			config.targetPaths,
			config.targetExcludePaths,
		),
	)
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	opaqueIDToLintConfig := make(map[string]bufconfig.LintConfig)
	opaqueIDToBreakingConfig := make(map[string]bufconfig.BreakingConfig)
	for _, module := range moduleSet.Modules() {
		if bufmodule.ModuleFullNameEqual(module.ModuleFullName(), moduleKey.ModuleFullName()) {
			opaqueIDToLintConfig[module.OpaqueID()] = targetModuleConfig.LintConfig()
			opaqueIDToBreakingConfig[module.OpaqueID()] = targetModuleConfig.BreakingConfig()
		} else {
			opaqueIDToLintConfig[module.OpaqueID()] = bufconfig.DefaultLintConfig
			opaqueIDToBreakingConfig[module.OpaqueID()] = bufconfig.DefaultBreakingConfig
		}
	}
	return &workspace{
		ModuleSet:                moduleSet,
		logger:                   logger,
		opaqueIDToLintConfig:     opaqueIDToLintConfig,
		opaqueIDToBreakingConfig: opaqueIDToBreakingConfig,
		configuredDepModuleRefs:  nil,
	}, nil
}

func newWorkspaceForProtoc(
	ctx context.Context,
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	includeDirPaths []string,
	filePaths []string,
) (*workspace, error) {
	absIncludeDirPaths, err := normalizeAndAbsolutePaths(includeDirPaths, "include directory")
	if err != nil {
		return nil, err
	}
	absFilePaths, err := normalizeAndAbsolutePaths(filePaths, "input file")
	if err != nil {
		return nil, err
	}
	var rootBuckets []storage.ReadBucket
	for _, includeDirPath := range includeDirPaths {
		rootBucket, err := storageosProvider.NewReadWriteBucket(
			includeDirPath,
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return nil, err
		}
		// need to do match extension here
		// https://github.com/bufbuild/buf/issues/113
		rootBuckets = append(rootBuckets, storage.MapReadBucket(rootBucket, storage.MatchPathExt(".proto")))
	}
	targetPaths, err := slicesext.MapError(
		absFilePaths,
		func(absFilePath string) (string, error) {
			return applyRootsToTargetPath(absIncludeDirPaths, absFilePath, normalpath.Absolute)
		},
	)
	if err != nil {
		return nil, err
	}

	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, bufmodule.NopModuleDataProvider)
	moduleSetBuilder.AddLocalModule(
		storage.MultiReadBucket(rootBuckets...),
		".",
		true,
		bufmodule.LocalModuleWithTargetPaths(
			targetPaths,
			nil,
		),
	)
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return &workspace{
		ModuleSet: moduleSet,
		logger:    logger,
		opaqueIDToLintConfig: map[string]bufconfig.LintConfig{
			".": bufconfig.DefaultLintConfig,
		},
		opaqueIDToBreakingConfig: map[string]bufconfig.BreakingConfig{
			".": bufconfig.DefaultBreakingConfig,
		},
		configuredDepModuleRefs: nil,
	}, nil
}

func newWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceBucketOption,
) (_ *workspace, retErr error) {
	ctx, span := tracer.Start(ctx, "bufbuild/buf", tracer.WithErr(&retErr))
	defer span.End()
	config, err := newWorkspaceBucketConfig(options)
	if err != nil {
		return nil, err
	}
	if config.configOverride != "" {
		overrideBufYAMLFile, err := bufconfig.GetBufYAMLFileForOverride(config.configOverride)
		if err != nil {
			return nil, err
		}
		logger.Debug(
			"creating new workspace with config override",
			zap.String("subDirPath", config.subDirPath),
		)
		switch fileVersion := overrideBufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			// We did not find any buf.work.yaml or buf.yaml, operate as if a
			// default v1 buf.yaml was at config.subDirPath.
			return newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
				ctx,
				logger,
				bucket,
				moduleDataProvider,
				config,
				[]string{config.subDirPath},
				overrideBufYAMLFile,
			)
		case bufconfig.FileVersionV2:
			return newWorkspaceForBucketBufYAMLV2(
				ctx,
				logger,
				bucket,
				moduleDataProvider,
				config,
				config.subDirPath,
				overrideBufYAMLFile,
			)
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}

	// Search for a workspace file that controls config.subDirPath. A workspace file is either
	// a buf.work.yaml file, or a v2 buf.yaml file, and the file controls config.subDirPath
	// either (1) we are directly targeting the workspace file, i.e curDirPath == config.subDirPath,
	// or (2) the workspace file refers to the config.subDirPath. If we find a controlling workspace
	// file, we use this to build our workspace. If we don't we assume that we're just building
	// a v1 buf.yaml with defaults at config.subDirPath.
	curDirPath := config.subDirPath
	// Loop recursively upwards to "." to check for buf.yamls and buf.work.yamls
	for {
		findControllingWorkspaceResult, err := bufconfig.FindControllingWorkspace(
			ctx,
			bucket,
			curDirPath,
			config.subDirPath,
		)
		if err != nil {
			return nil, err
		}
		if findControllingWorkspaceResult.Found() {
			// We have a v1 buf.work.yaml, per the documentation on bufconfig.FindControllingWorkspace.
			if bufWorkYAMLDirPaths := findControllingWorkspaceResult.BufWorkYAMLDirPaths(); len(bufWorkYAMLDirPaths) > 0 {
				logger.Debug(
					"creating new workspace based on v1 buf.work.yaml",
					zap.String("subDirPath", config.subDirPath),
					zap.String("bufWorkYAMLDirPath", curDirPath),
				)
				return newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
					ctx,
					logger,
					bucket,
					moduleDataProvider,
					config,
					bufWorkYAMLDirPaths,
					nil,
				)
			}
			logger.Debug(
				"creating new workspace based on v2 buf.yaml",
				zap.String("subDirPath", config.subDirPath),
				zap.String("bufYAMLDirPath", curDirPath),
			)
			// We have a v2 buf.yaml.
			return newWorkspaceForBucketBufYAMLV2(
				ctx,
				logger,
				bucket,
				moduleDataProvider,
				config,
				curDirPath,
				nil,
			)
		}
		// Break condition - we did not find any buf.work.yaml or buf.yaml.
		if curDirPath == "." {
			break
		}
		curDirPath = normalpath.Dir(curDirPath)
	}

	logger.Debug(
		"creating new workspace with no found buf.work.yaml or buf.yaml",
		zap.String("subDirPath", config.subDirPath),
	)
	// We did not find any buf.work.yaml or buf.yaml, operate as if a
	// default v1 buf.yaml was at config.subDirPath.
	return newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
		ctx,
		logger,
		bucket,
		moduleDataProvider,
		config,
		[]string{config.subDirPath},
		nil,
	)
}

func newWorkspaceForBucketBufYAMLV2(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	config *workspaceBucketConfig,
	bufYAMLV2FileDirPath string,
	// This can be nil, this is only set if config.configOverride was set, which we
	// deal with outside of this function.
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) (*workspace, error) {
	var bufYAMLFile bufconfig.BufYAMLFile
	var err error
	if overrideBufYAMLFile != nil {
		bufYAMLFile = overrideBufYAMLFile
	} else {
		bufYAMLFile, err = bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, bufYAMLV2FileDirPath)
		if err != nil {
			// This should be apparent from above functions.
			return nil, syserror.Newf("error getting buf.yaml at %q: %w", bufYAMLV2FileDirPath, err)
		}
		if bufYAMLFile.FileVersion() != bufconfig.FileVersionV2 {
			return nil, syserror.Newf("expected v2 buf.yaml at %q but got %v", bufYAMLV2FileDirPath, bufYAMLFile.FileVersion())
		}
	}

	// config.subDirPath is the input subDirPath. We only want to target modules that are inside
	// this subDirPath. Example: bufWorkYAMLDirPath is "foo", subDirPath is "foo/bar",
	// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
	// and "foo/bar/bat".
	//
	// This is new behavior - before, we required that you input an exact match for the module directory path,
	// but now, we will take all the modules underneath this workspace.
	isTargetFunc := func(moduleDirPath string) bool {
		return normalpath.EqualsOrContainsPath(config.subDirPath, moduleDirPath, normalpath.Relative)
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, moduleDataProvider)
	bucketIDToModuleConfig := make(map[string]bufconfig.ModuleConfig)
	// We keep track of if any module was tentatively targeted, and then actually targeted via
	// the paths flags. We use this pre-building of the ModuleSet to see if the --path and
	// --exclude-path flags resulted in no targeted modules. This condition is represented
	// by hadIsTentativelyTargetModule == true && hadIsTargetModule = false
	//
	// If hadIsTentativelyTargetModule is false, this means that our input subDirPath was not
	// actually representative of any module that we detected in buf.work.yaml or v2 buf.yaml
	// directories, and this is a system error - this should be verified before we reach this function.
	var hadIsTentativelyTargetModule bool
	var hadIsTargetModule bool
	var moduleDirPaths []string
	for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
		moduleDirPath := moduleConfig.DirPath()
		moduleDirPaths = append(moduleDirPaths, moduleDirPath)
		bucketIDToModuleConfig[moduleDirPath] = moduleConfig
		// We figure out based on the paths if this is really a target module in moduleTargeting.
		isTentativelyTargetModule := isTargetFunc(moduleDirPath)
		if isTentativelyTargetModule {
			hadIsTentativelyTargetModule = true
		}
		mappedModuleBucket, moduleTargeting, err := getMappedModuleBucketAndModuleTargeting(
			ctx,
			bucket,
			config,
			moduleDirPath,
			moduleConfig,
			isTentativelyTargetModule,
		)
		if err != nil {
			return nil, err
		}
		if moduleTargeting.isTargetModule {
			hadIsTargetModule = true
		}
		moduleSetBuilder.AddLocalModule(
			mappedModuleBucket,
			moduleDirPath,
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
		)
	}
	bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, bucket, bufYAMLV2FileDirPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	} else {
		for _, depModuleKey := range bufLockFile.DepModuleKeys() {
			moduleSetBuilder.AddRemoteModule(
				depModuleKey,
				false,
			)
		}
	}
	if !hadIsTentativelyTargetModule {
		return nil, syserror.Newf("subDirPath %q did not result in any target modules from moduleDirPaths %v", config.subDirPath, moduleDirPaths)
	}
	if !hadIsTargetModule {
		// It would be nice to have a better error message than this in the long term.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return newWorkspaceForModuleSet(moduleSet, logger, bucketIDToModuleConfig, bufYAMLFile.ConfiguredDepModuleRefs())
}

func newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	config *workspaceBucketConfig,
	moduleDirPaths []string,
	// This can be nil, this is only set if config.configOverride was set, which we
	// deal with outside of this function.
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) (*workspace, error) {
	// config.subDirPath is the input subDirPath. We only want to target modules that are inside
	// this subDirPath. Example: bufWorkYAMLDirPath is "foo", subDirPath is "foo/bar",
	// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
	// and "foo/bar/bat".
	//
	// This is new behavior - before, we required that you input an exact match for the module directory path,
	// but now, we will take all the modules underneath this workspace.
	isTargetFunc := func(moduleDirPath string) bool {
		return normalpath.EqualsOrContainsPath(config.subDirPath, moduleDirPath, normalpath.Relative)
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, moduleDataProvider)
	bucketIDToModuleConfig := make(map[string]bufconfig.ModuleConfig)
	var allConfiguredDepModuleRefs []bufmodule.ModuleRef
	// We keep track of if any module was tentatively targeted, and then actually targeted via
	// the paths flags. We use this pre-building of the ModuleSet to see if the --path and
	// --exclude-path flags resulted in no targeted modules. This condition is represented
	// by hadIsTentativelyTargetModule == true && hadIsTargetModule = false
	//
	// If hadIsTentativelyTargetModule is false, this means that our input subDirPath was not
	// actually representative of any module that we detected in buf.work.yaml or v2 buf.yaml
	// directories, and this is a system error - this should be verified before we reach this function.
	var hadIsTentativelyTargetModule bool
	var hadIsTargetModule bool
	for _, moduleDirPath := range moduleDirPaths {
		moduleConfig, configuredDepModuleRefs, err := getModuleConfigAndConfiguredDepModuleRefsV1Beta1OrV1(
			ctx,
			bucket,
			moduleDirPath,
			overrideBufYAMLFile,
		)
		if err != nil {
			return nil, err
		}
		allConfiguredDepModuleRefs = append(allConfiguredDepModuleRefs, configuredDepModuleRefs...)
		bucketIDToModuleConfig[moduleDirPath] = moduleConfig
		bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, bucket, moduleDirPath)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		} else {
			for _, depModuleKey := range bufLockFile.DepModuleKeys() {
				moduleSetBuilder.AddRemoteModule(
					depModuleKey,
					false,
				)
			}
		}
		// We figure out based on the paths if this is really a target module in moduleTargeting.
		isTentativelyTargetModule := isTargetFunc(moduleDirPath)
		if isTentativelyTargetModule {
			hadIsTentativelyTargetModule = true
		}
		mappedModuleBucket, moduleTargeting, err := getMappedModuleBucketAndModuleTargeting(
			ctx,
			bucket,
			config,
			moduleDirPath,
			moduleConfig,
			isTentativelyTargetModule,
		)
		if err != nil {
			return nil, err
		}
		if moduleTargeting.isTargetModule {
			hadIsTargetModule = true
		}
		moduleSetBuilder.AddLocalModule(
			mappedModuleBucket,
			moduleDirPath,
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
		)
	}
	if !hadIsTentativelyTargetModule {
		return nil, syserror.Newf("subDirPath %q did not result in any target modules from moduleDirPaths %v", config.subDirPath, moduleDirPaths)
	}
	if !hadIsTargetModule {
		// It would be nice to have a better error message than this in the long term.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	return newWorkspaceForModuleSet(moduleSet, logger, bucketIDToModuleConfig, allConfiguredDepModuleRefs)
}

func newWorkspaceForModuleSet(
	moduleSet bufmodule.ModuleSet,
	logger *zap.Logger,
	bucketIDToModuleConfig map[string]bufconfig.ModuleConfig,
	configuredDepModuleRefs []bufmodule.ModuleRef,
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
			opaqueIDToLintConfig[module.OpaqueID()] = bufconfig.DefaultLintConfig
			opaqueIDToBreakingConfig[module.OpaqueID()] = bufconfig.DefaultBreakingConfig
		}
	}
	return &workspace{
		ModuleSet:                moduleSet,
		logger:                   logger,
		opaqueIDToLintConfig:     opaqueIDToLintConfig,
		opaqueIDToBreakingConfig: opaqueIDToBreakingConfig,
		configuredDepModuleRefs:  configuredDepModuleRefs,
	}, nil
}

func getModuleConfigAndConfiguredDepModuleRefsV1Beta1OrV1(
	ctx context.Context,
	bucket storage.ReadBucket,
	moduleDirPath string,
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) (bufconfig.ModuleConfig, []bufmodule.ModuleRef, error) {
	var bufYAMLFile bufconfig.BufYAMLFile
	var err error
	if overrideBufYAMLFile != nil {
		bufYAMLFile = overrideBufYAMLFile
	} else {
		bufYAMLFile, err = bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, moduleDirPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// If we do not have a buf.yaml, we use the default config.
				// This is a v1 config.
				return bufconfig.DefaultModuleConfig, nil, nil
			}
			return nil, nil, err
		}
	}
	// Just a sanity check. This should have already been validated, but let's make sure.
	if bufYAMLFile.FileVersion() != bufconfig.FileVersionV1Beta1 && bufYAMLFile.FileVersion() != bufconfig.FileVersionV1 {
		return nil, nil, syserror.Newf("buf.yaml at %s did not have version v1beta1 or v1", moduleDirPath)
	}
	moduleConfigs := bufYAMLFile.ModuleConfigs()
	if len(moduleConfigs) != 1 {
		// This is a system error. This should never happen.
		return nil, nil, syserror.Newf("received %d ModuleConfigs from a v1beta1 or v1 BufYAMLFile", len(moduleConfigs))
	}
	return moduleConfigs[0], bufYAMLFile.ConfiguredDepModuleRefs(), nil
}

func getMappedModuleBucketAndModuleTargeting(
	ctx context.Context,
	bucket storage.ReadBucket,
	config *workspaceBucketConfig,
	moduleDirPath string,
	moduleConfig bufconfig.ModuleConfig,
	isTargetModule bool,
) (storage.ReadBucket, *moduleTargeting, error) {
	moduleBucket := storage.MapReadBucket(
		bucket,
		storage.MapOnPrefix(moduleDirPath),
	)
	rootToExcludes := moduleConfig.RootToExcludes()
	var rootBuckets []storage.ReadBucket
	for root, excludes := range rootToExcludes {
		// Roots only applies to .proto files.
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
				moduleBucket,
				mappers...,
			),
		)
	}
	rootBuckets = append(
		rootBuckets,
		bufmodule.GetDocStorageReadBucket(ctx, moduleBucket),
		bufmodule.GetLicenseStorageReadBucket(moduleBucket),
	)
	mappedModuleBucket := storage.MultiReadBucket(rootBuckets...)
	moduleTargeting, err := newModuleTargeting(
		moduleDirPath,
		slicesext.MapKeysToSlice(rootToExcludes),
		config,
		isTargetModule,
	)
	if err != nil {
		return nil, nil, err
	}
	return mappedModuleBucket, moduleTargeting, nil
}

// TODO: All the module_bucket_builder_test.go stuff needs to be copied over

type moduleTargeting struct {
	// Whether this module is really a target module.
	//
	// False if this was not specified as a target module by the caller.
	// Also false if there were config.targetPaths or config.protoFileTargetPath, but
	// these paths did not match anything in the module.
	isTargetModule bool
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleTargetPaths []string
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleTargetExcludePaths []string
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleProtoFileTargetPath string
	includePackageFiles       bool
}

func newModuleTargeting(
	moduleDirPath string,
	roots []string,
	config *workspaceBucketConfig,
	isTentativelyTargetModule bool,
) (*moduleTargeting, error) {
	if !isTentativelyTargetModule {
		// If this is not a target Module, we do not want to target anything, as targeting
		// paths for non-target Modules is an error.
		return &moduleTargeting{}, nil
	}
	// If we have no target paths, then we always match the value of isTargetModule.
	// Otherwise, we need to see that at least one path matches the moduleDirPath for us
	// to consider this module a target.
	isTargetModule := len(config.targetPaths) == 0 && config.protoFileTargetPath == ""
	var moduleTargetPaths []string
	var moduleTargetExcludePaths []string
	for _, targetPath := range config.targetPaths {
		if targetPath == moduleDirPath {
			// We're just going to be realists in our error messages here.
			// TODO: Do we error here currently? If so, this error remains. For extra credit in the future,
			// if we were really clever, we'd go back and just add this as a module path.
			return nil, fmt.Errorf("module %q was specified with --path - specify this module path directly as an input", targetPath)
		}
		if normalpath.ContainsPath(moduleDirPath, targetPath, normalpath.Relative) {
			isTargetModule = true
			moduleTargetPath, err := normalpath.Rel(moduleDirPath, targetPath)
			if err != nil {
				return nil, err
			}
			moduleTargetPaths = append(moduleTargetPaths, moduleTargetPath)
		}
	}
	for _, targetExcludePath := range config.targetExcludePaths {
		if targetExcludePath == moduleDirPath {
			// We're just going to be realists in our error messages here.
			// TODO: Do we error here currently? If so, this error remains. For extra credit in the future,
			// if we were really clever, we'd go back and just remove this as a module path if it was specified.
			// This really should be allowed - how else do you exclude from a workspace?
			return nil, fmt.Errorf("module %q was specified with --exclude-path - this flag cannot be used to specify module directories", targetExcludePath)
		}
		if normalpath.ContainsPath(moduleDirPath, targetExcludePath, normalpath.Relative) {
			moduleTargetExcludePath, err := normalpath.Rel(moduleDirPath, targetExcludePath)
			if err != nil {
				return nil, err
			}
			moduleTargetExcludePaths = append(moduleTargetExcludePaths, moduleTargetExcludePath)
		}
	}
	moduleTargetPaths, err := slicesext.MapError(
		moduleTargetPaths,
		func(moduleTargetPath string) (string, error) {
			return applyRootsToTargetPath(roots, moduleTargetPath, normalpath.Relative)
		},
	)
	if err != nil {
		return nil, err
	}
	moduleTargetExcludePaths, err = slicesext.MapError(
		moduleTargetExcludePaths,
		func(moduleTargetExcludePath string) (string, error) {
			return applyRootsToTargetPath(roots, moduleTargetExcludePath, normalpath.Relative)
		},
	)
	if err != nil {
		return nil, err
	}
	var moduleProtoFileTargetPath string
	var includePackageFiles bool
	if config.protoFileTargetPath != "" &&
		normalpath.ContainsPath(moduleDirPath, config.protoFileTargetPath, normalpath.Relative) {
		isTargetModule = true
		moduleProtoFileTargetPath, err = normalpath.Rel(moduleDirPath, config.protoFileTargetPath)
		if err != nil {
			return nil, err
		}
		moduleProtoFileTargetPath, err = applyRootsToTargetPath(roots, moduleProtoFileTargetPath, normalpath.Relative)
		if err != nil {
			return nil, err
		}
		includePackageFiles = config.includePackageFiles
	}
	return &moduleTargeting{
		isTargetModule:            isTargetModule,
		moduleTargetPaths:         moduleTargetPaths,
		moduleTargetExcludePaths:  moduleTargetExcludePaths,
		moduleProtoFileTargetPath: moduleProtoFileTargetPath,
		includePackageFiles:       includePackageFiles,
	}, nil
}

func applyRootsToTargetPath(roots []string, path string, pathType normalpath.PathType) (string, error) {
	var matchingRoots []string
	for _, root := range roots {
		if normalpath.ContainsPath(root, path, pathType) {
			matchingRoots = append(matchingRoots, root)
		}
	}
	switch len(matchingRoots) {
	case 0:
		// this is a user error and will likely happen often
		return "", fmt.Errorf(
			"path %q is not contained within any of roots %s - note that specified paths "+
				"cannot be roots, but must be contained within roots",
			path,
			stringutil.SliceToHumanStringQuoted(roots),
		)
	case 1:
		targetPath, err := normalpath.Rel(matchingRoots[0], path)
		if err != nil {
			return "", err
		}
		// just in case
		return normalpath.NormalizeAndValidate(targetPath)
	default:
		// this should never happen
		return "", fmt.Errorf("%q is contained in multiple roots %s", path, stringutil.SliceToHumanStringQuoted(roots))
	}
}

// normalizeAndAbsolutePaths verifies that:
//
//   - No paths are empty.
//   - All paths are normalized.
//   - All paths are unique.
//   - No path contains another path.
//
// Normalizes, absolutes, and sorts the paths.
func normalizeAndAbsolutePaths(paths []string, name string) ([]string, error) {
	if len(paths) == 0 {
		return paths, nil
	}
	outputs := make([]string, len(paths))
	for i, path := range paths {
		if path == "" {
			return nil, fmt.Errorf("%s contained an empty path", name)
		}
		output, err := normalpath.NormalizeAndAbsolute(path)
		if err != nil {
			// user error
			return nil, err
		}
		outputs[i] = output
	}
	sort.Strings(outputs)
	for i := 0; i < len(outputs); i++ {
		for j := i + 1; j < len(outputs); j++ {
			output1 := outputs[i]
			output2 := outputs[j]

			if output1 == output2 {
				return nil, fmt.Errorf("duplicate %s %q", name, output1)
			}
			if normalpath.EqualsOrContainsPath(output2, output1, normalpath.Absolute) {
				return nil, fmt.Errorf("%s %q is within %s %q which is not allowed", name, output1, name, output2)
			}
			if normalpath.EqualsOrContainsPath(output1, output2, normalpath.Absolute) {
				return nil, fmt.Errorf("%s %q is within %s %q which is not allowed", name, output2, name, output1)
			}
		}
	}
	return outputs, nil
}
