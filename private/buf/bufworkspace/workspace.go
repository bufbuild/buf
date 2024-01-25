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

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/gofrs/uuid/v5"
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
	// The ModuleRefs in this list will be unique by ModuleFullName. If there are two ModuleRefs
	// in the buf.yaml with the same ModuleFullName but different Refs, an error will be given
	// at workspace constructions. For example, with v1 buf.yaml, this is a union of the deps in
	// the buf.yaml files in the workspace. If different buf.yamls had different refs, an error
	// will be returned - we have no way to resolve what the user intended.
	//
	// Sorted.
	//
	// We use this to warn on unused dependencies in bufctl.
	ConfiguredDepModuleRefs() []bufmodule.ModuleRef

	isWorkspace()
}

// NewWorkspaceForBucket returns a new Workspace for the given Bucket.
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
func NewWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	bucket storage.ReadBucket,
	clientProvider bufapi.ClientProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	options ...WorkspaceBucketOption,
) (Workspace, error) {
	return newWorkspaceForBucket(ctx, logger, tracer, bucket, clientProvider, moduleDataProvider, commitProvider, options...)
}

// NewWorkspaceForModuleKey wraps the ModuleKey into a workspace, returning defaults
// for config values, and empty ConfiguredDepModuleRefs.
//
// This is useful for getting Workspaces for remote modules, but you still need
// associated configuration.
func NewWorkspaceForModuleKey(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	moduleKey bufmodule.ModuleKey,
	graphProvider bufmodule.GraphProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	options ...WorkspaceModuleKeyOption,
) (Workspace, error) {
	return newWorkspaceForModuleKey(ctx, logger, tracer, moduleKey, graphProvider, moduleDataProvider, commitProvider, options...)
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
	tracer tracing.Tracer,
	storageosProvider storageos.Provider,
	includeDirPaths []string,
	filePaths []string,
) (Workspace, error) {
	return newWorkspaceForProtoc(ctx, logger, tracer, storageosProvider, includeDirPaths, filePaths)
}

// *** PRIVATE ***

type workspace struct {
	bufmodule.ModuleSet

	logger                   *zap.Logger
	opaqueIDToLintConfig     map[string]bufconfig.LintConfig
	opaqueIDToBreakingConfig map[string]bufconfig.BreakingConfig
	configuredDepModuleRefs  []bufmodule.ModuleRef

	// createdFromBucket is a sanity check for updateableWorkspace to make sure that the
	// underlying workspace was really created from a bucket.
	createdFromBucket bool
	// If true, the workspace was created from v2 buf.yamls
	//
	// If false, the workspace was created from defaults, or v1beta1/v1 buf.yamls.
	//
	// updateableWorkspace uses this to determine what DigestType to use, and what version
	// of buf.lock to write.
	isV2 bool
	// updateableBufLockDirPath is the relative path within the bucket where a buf.lock can be written.
	//
	// If isV2 is true, this will be "." if no config overrides were used - buf.locks live at the root of the workspace.
	// If isV2 is false, this will be the path to the single, local, targeted Module within the workspace if no config
	// overrides were used. This is the only situation where we can do an update for a v1 buf.lock.
	// If isV2 is false and there is not a single, local, targeted Module, or a config override was used, this will be empty.
	//
	// The option withIgnoreAndDisallowV1BufWorkYAMLs is used by updateabeWorkspace to try
	// to satisfy the v1 condition.
	//
	// updateableWorkspace uses this to determine where to write to.
	updateableBufLockDirPath string
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
	tracer tracing.Tracer,
	moduleKey bufmodule.ModuleKey,
	graphProvider bufmodule.GraphProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
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
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, moduleDataProvider, commitProvider)
	// Add the input ModuleKey with path filters.
	moduleSetBuilder.AddRemoteModule(
		moduleKey,
		true,
		bufmodule.RemoteModuleWithTargetPaths(
			config.targetPaths,
			config.targetExcludePaths,
		),
	)
	graph, err := graphProvider.GetGraphForModuleKeys(ctx, []bufmodule.ModuleKey{moduleKey})
	if err != nil {
		return nil, err
	}
	if err := graph.WalkNodes(func(node bufmodule.ModuleKey, _ []bufmodule.ModuleKey, _ []bufmodule.ModuleKey) error {
		if node.CommitID() != moduleKey.CommitID() {
			// Add the dependency ModuleKey with no path filters.
			moduleSetBuilder.AddRemoteModule(node, false)
		}
		return nil
	}); err != nil {
		return nil, err
	}
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
		createdFromBucket:        false,
		isV2:                     false,
		updateableBufLockDirPath: "",
	}, nil
}

func newWorkspaceForProtoc(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
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

	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, bufmodule.NopModuleDataProvider, bufmodule.NopCommitProvider)
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
		configuredDepModuleRefs:  nil,
		createdFromBucket:        false,
		isV2:                     false,
		updateableBufLockDirPath: "",
	}, nil
}

func newWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	tracer tracing.Tracer,
	bucket storage.ReadBucket,
	clientProvider bufapi.ClientProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	options ...WorkspaceBucketOption,
) (_ *workspace, retErr error) {
	ctx, span := tracer.Start(ctx, tracing.WithErr(&retErr))
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
			zap.String("targetSubDirPath", config.targetSubDirPath),
		)
		switch fileVersion := overrideBufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			// Operate as if there was no buf.work.yaml, only a v1 buf.yaml at the specified
			// targetSubDirPath, specifying a single module.
			return newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
				ctx,
				logger,
				bucket,
				clientProvider,
				moduleDataProvider,
				commitProvider,
				config,
				[]string{config.targetSubDirPath},
				overrideBufYAMLFile,
			)
		case bufconfig.FileVersionV2:
			// Operate as if there was a v2 buf.yaml at the root of the bucket.
			return newWorkspaceForBucketBufYAMLV2(
				ctx,
				logger,
				storage.MapReadBucket(bucket, storage.MapOnPrefix(config.targetSubDirPath)),
				moduleDataProvider,
				commitProvider,
				config,
				overrideBufYAMLFile,
			)
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}

	findControllingWorkspaceResult, err := bufconfig.FindControllingWorkspace(
		ctx,
		bucket,
		".",
		config.targetSubDirPath,
	)
	if err != nil {
		return nil, err
	}
	if findControllingWorkspaceResult.Found() {
		// We have a v1 buf.work.yaml, per the documentation on bufconfig.FindControllingWorkspace.
		if bufWorkYAMLDirPaths := findControllingWorkspaceResult.BufWorkYAMLDirPaths(); len(bufWorkYAMLDirPaths) > 0 {
			if config.ignoreAndDisallowV1BufWorkYAMLs {
				// config.targetSubDirPath is normalized, so if it was empty, it will be ".".
				if config.targetSubDirPath == "." {
					// If config.targetSubDirPath is ".", this means we targeted a buf.work.yaml, not an individual module within the buf.work.yaml
					// This is disallowed.
					return nil, errors.New("workspaces defined with buf.work.yaml cannot be updated, only the individual modules within a workspace can be updated. Workspaces defined with a v2 buf.yaml can be updated, see the migration documentation for more details.")
				}
				// We targeted a specific module within the workspace. Based on the option we provided, we're going to ignore
				// the workspace entirely, and just act as if the buf.work.yaml did not exist.
				logger.Debug(
					"creating new workspace, ignoring v1 buf.work.yaml, just building on module at target",
					zap.String("targetSubDirPath", config.targetSubDirPath),
				)
				return newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
					ctx,
					logger,
					bucket,
					clientProvider,
					moduleDataProvider,
					commitProvider,
					config,
					[]string{config.targetSubDirPath},
					nil,
				)
			}
			logger.Debug(
				"creating new workspace based on v1 buf.work.yaml",
				zap.String("targetSubDirPath", config.targetSubDirPath),
			)
			return newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
				ctx,
				logger,
				bucket,
				clientProvider,
				moduleDataProvider,
				commitProvider,
				config,
				bufWorkYAMLDirPaths,
				nil,
			)
		}
		logger.Debug(
			"creating new workspace based on v2 buf.yaml",
			zap.String("targetSubDirPath", config.targetSubDirPath),
		)
		// We have a v2 buf.yaml.
		return newWorkspaceForBucketBufYAMLV2(
			ctx,
			logger,
			bucket,
			moduleDataProvider,
			commitProvider,
			config,
			nil,
		)
	}

	logger.Debug(
		"creating new workspace with no found buf.work.yaml or buf.yaml",
		zap.String("targetSubDirPath", config.targetSubDirPath),
	)
	// We did not find any buf.work.yaml or buf.yaml, operate as if a
	// default v1 buf.yaml was at config.targetSubDirPath.
	return newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
		ctx,
		logger,
		bucket,
		clientProvider,
		moduleDataProvider,
		commitProvider,
		config,
		[]string{config.targetSubDirPath},
		nil,
	)
}

func newWorkspaceForBucketAndModuleDirPathsV1Beta1OrV1(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	clientProvider bufapi.ClientProvider,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	config *workspaceBucketConfig,
	moduleDirPaths []string,
	// This can be nil, this is only set if config.configOverride was set, which we
	// deal with outside of this function.
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) (*workspace, error) {
	// config.targetSubDirPath is the input subDirPath. We only want to target modules that are inside
	// this subDirPath. Example: bufWorkYAMLDirPath is "foo", subDirPath is "foo/bar",
	// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
	// and "foo/bar/bat".
	//
	// This is new behavior - before, we required that you input an exact match for the module directory path,
	// but now, we will take all the modules underneath this workspace.
	isTargetFunc := func(moduleDirPath string) bool {
		return normalpath.EqualsOrContainsPath(config.targetSubDirPath, moduleDirPath, normalpath.Relative)
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, moduleDataProvider, commitProvider)
	bucketIDToModuleConfig := make(map[string]bufconfig.ModuleConfig)
	// We use this to detect different refs across different files.
	moduleFullNameStringToConfiguredDepModuleRefString := make(map[string]string)
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
		for _, configuredDepModuleRef := range configuredDepModuleRefs {
			moduleFullNameString := configuredDepModuleRef.ModuleFullName().String()
			configuredDepModuleRefString := configuredDepModuleRef.String()
			existingConfiguredDepModuleRefString, ok := moduleFullNameStringToConfiguredDepModuleRefString[moduleFullNameString]
			if !ok {
				// We haven't encountered a ModuleRef with this ModuleFullName yet, add it.
				allConfiguredDepModuleRefs = append(allConfiguredDepModuleRefs, configuredDepModuleRef)
				moduleFullNameStringToConfiguredDepModuleRefString[moduleFullNameString] = configuredDepModuleRefString
			} else if configuredDepModuleRefString != existingConfiguredDepModuleRefString {
				// We encountered the same ModuleRef by ModuleFullName, but with a different Ref.
				return nil, fmt.Errorf("found different refs for the same module within buf.yaml deps in the workspace: %s %s", configuredDepModuleRefString, existingConfiguredDepModuleRefString)
			}
		}
		bucketIDToModuleConfig[moduleDirPath] = moduleConfig
		bufLockFile, err := bufconfig.GetBufLockFileForPrefix(
			ctx,
			bucket,
			// buf.lock files live at the module root
			moduleDirPath,
			bufconfig.BufLockFileWithDigestResolver(
				func(ctx context.Context, remote string, commitID uuid.UUID) (bufmodule.Digest, error) {
					return bufmoduleapi.DigestForCommitID(ctx, clientProvider, remote, commitID, bufmodule.DigestTypeB4)
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
			// TODO: re-enable once we fix tests
			//return nil, errors.New("got a v2 buf.lock file for a v1 buf.yaml")
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
		v1BufYAMLObjectData, err := bufconfig.GetBufYAMLV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, moduleDirPath)
		if err != nil {
			return nil, err
		}
		v1BufLockObjectData, err := bufconfig.GetBufLockV1Beta1OrV1ObjectDataForPrefix(ctx, bucket, moduleDirPath)
		if err != nil {
			return nil, err
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
			bufmodule.LocalModuleWithV1Beta1OrV1BufYAMLObjectData(v1BufYAMLObjectData),
			bufmodule.LocalModuleWithV1Beta1OrV1BufLockObjectData(v1BufLockObjectData),
		)
	}
	if !hadIsTentativelyTargetModule {
		return nil, syserror.Newf("subDirPath %q did not result in any target modules from moduleDirPaths %v", config.targetSubDirPath, moduleDirPaths)
	}
	if !hadIsTargetModule {
		// It would be nice to have a better error message than this in the long term.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	var updateableBufLockDirPath string
	if len(moduleDirPaths) == 1 && overrideBufYAMLFile == nil {
		// If we have a single moduleDirPath, we know at this point that this moduleDirPath is targeted as well, as otherwise
		// hadIsTargetModule would be false. hadIsTargetModule only flips to true if one or more moduleDirPaths has a target Module.
		// So, a single moduleDirPath after we have verified that hadIsTargetModule is true means that we have a single, local, target Module.
		//
		// Our other condition is that we didn't use config overrides, so we check that too.
		updateableBufLockDirPath = moduleDirPaths[0]
	}
	return newWorkspaceForBucketModuleSet(
		moduleSet,
		logger,
		bucketIDToModuleConfig,
		allConfiguredDepModuleRefs,
		false,
		updateableBufLockDirPath,
	)
}

func newWorkspaceForBucketBufYAMLV2(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	commitProvider bufmodule.CommitProvider,
	config *workspaceBucketConfig,
	// This can be nil, this is only set if config.configOverride was set, which we
	// deal with outside of this function.
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) (*workspace, error) {
	var bufYAMLFile bufconfig.BufYAMLFile
	var err error
	if overrideBufYAMLFile != nil {
		bufYAMLFile = overrideBufYAMLFile
		// We don't want to have ObjectData for a --config override.
		// TODO: What happened when you specified a --config pre-refactor with tamper-proofing? We might
		// have actually still used the buf.yaml for tamper-proofing, if so, we need to attempt to read it
		// regardless of whether override was specified.
	} else {
		bufYAMLFile, err = bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, ".")
		if err != nil {
			// This should be apparent from above functions.
			return nil, syserror.Newf("error getting v2 buf.yaml: %w", err)
		}
		if bufYAMLFile.FileVersion() != bufconfig.FileVersionV2 {
			return nil, syserror.Newf("expected v2 buf.yaml but got %v", bufYAMLFile.FileVersion())
		}
	}

	// config.targetSubDirPath is the input targetSubDirPath. We only want to target modules that are inside
	// this targetSubDirPath. Example: bufWorkYAMLDirPath is "foo", targetSubDirPath is "foo/bar",
	// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
	// and "foo/bar/bat".
	//
	// This is new behavior - before, we required that you input an exact match for the module directory path,
	// but now, we will take all the modules underneath this workspace.
	isTargetFunc := func(moduleDirPath string) bool {
		return normalpath.EqualsOrContainsPath(config.targetSubDirPath, moduleDirPath, normalpath.Relative)
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, logger, moduleDataProvider, commitProvider)

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
	if !hadIsTentativelyTargetModule {
		return nil, syserror.Newf("targetSubDirPath %q did not result in any target modules from moduleDirPaths %v", config.targetSubDirPath, moduleDirPaths)
	}
	if !hadIsTargetModule {
		// It would be nice to have a better error message than this in the long term.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}
	var updateableBufLockDirPath string
	if overrideBufYAMLFile == nil {
		// We have a v2 buf.yaml, and we have no config override. Therefore, we have a updateableBufLockDirPath.
		updateableBufLockDirPath = "."
	}
	// bufYAMLFile.ConfiguredDepModuleRefs() is unique by ModuleFullName.
	return newWorkspaceForBucketModuleSet(
		moduleSet,
		logger,
		bucketIDToModuleConfig,
		bufYAMLFile.ConfiguredDepModuleRefs(),
		true,
		updateableBufLockDirPath,
	)
}

// only use for workspaces created from buckets
func newWorkspaceForBucketModuleSet(
	moduleSet bufmodule.ModuleSet,
	logger *zap.Logger,
	bucketIDToModuleConfig map[string]bufconfig.ModuleConfig,
	// Expected to already be unique by ModuleFullName.
	configuredDepModuleRefs []bufmodule.ModuleRef,
	isV2 bool,
	updateableBufLockDirPath string,
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
		createdFromBucket:        true,
		isV2:                     isV2,
		updateableBufLockDirPath: updateableBufLockDirPath,
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
