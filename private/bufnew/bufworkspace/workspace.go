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

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
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
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceOption,
) (Workspace, error) {
	return newWorkspaceForBucket(ctx, bucket, moduleDataProvider, options...)
}

// NewWorkspaceForModuleSet wraps the ModuleSet into a workspace, returning defaults
// for config values, and empty ConfiguredDepModuleRefs.
//
// This is useful for when ModuleSets are created from remotes, but you still need
// associated configuration.
func NewWorkspaceForModuleSet(moduleSet bufmodule.ModuleSet) (Workspace, error) {
	return newWorkspaceForModuleSet(moduleSet)
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
	storageosProvider storageos.Provider,
	includeDirPaths []string,
	filePaths []string,
) (Workspace, error) {
	return newWorkspaceForProtoc(
		ctx,
		storageosProvider,
		includeDirPaths,
		filePaths,
	)
}

// WorkspaceOption is an option for a new Workspace.
type WorkspaceOption func(*workspaceOptions)

// This selects the specific directory within the Workspace bucket to target.
//
// Example: We have modules at foo/bar, foo/baz. "." will result in both
// modules being selected, so will "foo", but "foo/bar" will result in only
// the foo/bar module.
//
// A subDirPath of "." is equivalent of not setting this option.
func WorkspaceWithTargetSubDirPath(subDirPath string) WorkspaceOption {
	return func(workspaceOptions *workspaceOptions) {
		workspaceOptions.subDirPath = subDirPath
	}
}

// Note these paths need to have the path/to/module stripped, and then each new path
// filtered to the specific module it applies to. If some modules do not have any
// target paths, but we specified WorkspaceWithTargetPaths, then those modules
// need to be built as non-targeted.
//
// Theese paths have to  be within the subDirPath, if it exists.
func WorkspaceWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) WorkspaceOption {
	return func(workspaceOptions *workspaceOptions) {
		workspaceOptions.targetPaths = targetPaths
		workspaceOptions.targetExcludePaths = targetExcludePaths
	}
}

// WorkspaceUnreferencedConfiguredDepModuleRefs returns those configured ModuleRefs that do not
// reference any Module within the workspace. These can be pruned from the buf.lock
// in both v1 and v2 buf.yamls.
//
// A ModuleRef is considered to reference a Module if it has the same ModuleFullName.
//
// TODO: This logic may be broken for pruning. Consider what happens when we add remotes we shouldnt to the ModuleSet.
func WorkspaceUnreferencedConfiguredDepModuleRefs(workspace Workspace) []bufmodule.ModuleRef {
	var resultDepModuleRefs []bufmodule.ModuleRef
	for _, configuredDepModuleRef := range workspace.ConfiguredDepModuleRefs() {
		module := workspace.GetModuleForModuleFullName(configuredDepModuleRef.ModuleFullName())
		// Workspaces are self-contained and have all dependencies, therefore
		// this check is all that is needed.
		if module == nil {
			resultDepModuleRefs = append(resultDepModuleRefs, configuredDepModuleRef)
		}
	}
	return resultDepModuleRefs
}

// WorkspaceUnreferencedOrLocalConfiguredDepModuleRefs returns those configured dependency ModuleRefs that
// do not reference any Module in the workspace, or reference local Modules within the Workspace.
// In theory, these can be pruned from v2 buf.yamls.
//
// Local modules are present in v1 buf.yaml, but they are not used by buf anymore. A note
// that this means that if we prune these, ***upgrading buf is a one-way door*** - if a buf.lock
// is pruned based on a newer version of buf, it will no longer be useable by
// old versions of buf, if we prune these. We should discuss what we want to do here - perhaps
// these should be pruned depending on v1 vs v2.
//
// A ModuleRef is considered to reference a Module if it has the same ModuleFullName.
//
// TODO: This logic may be broken for pruning. Consider what happens when we add remotes we shouldnt to the ModuleSet.
func WorkspaceUnreferencedOrLocalConfiguredDepModuleRefs(workspace Workspace) []bufmodule.ModuleRef {
	var resultDepModuleRefs []bufmodule.ModuleRef
	for _, configuredDepModuleRef := range workspace.ConfiguredDepModuleRefs() {
		module := workspace.GetModuleForModuleFullName(configuredDepModuleRef.ModuleFullName())
		// Workspaces are self-contained and have all dependencies, therefore
		// this check is all that is needed.
		if module == nil || module.IsLocal() {
			resultDepModuleRefs = append(resultDepModuleRefs, configuredDepModuleRef)
		}
	}
	return resultDepModuleRefs
}

// *** PRIVATE ***

type workspace struct {
	bufmodule.ModuleSet

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

// *** PRIVATE ***

func newWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceOption,
) (*workspace, error) {
	workspaceOptions := newWorkspaceOptions()
	for _, option := range options {
		option(workspaceOptions)
	}
	if err := normalizeAndValidateWorkspaceOptions(workspaceOptions); err != nil {
		return nil, err
	}

	// Both of these functions validate that we're in v1beta1/v1 world. When we add v2, we will likely
	// need to significantly rework all of newWorkspaceForBucket.
	bufWorkYAMLExists, err := bufWorkYAMLExistsAtPrefix(ctx, bucket, workspaceOptions.subDirPath)
	if err != nil {
		return nil, err
	}
	bufYAMLExists, err := bufYAMLExistsAtPrefix(ctx, bucket, workspaceOptions.subDirPath)
	if err != nil {
		return nil, err
	}

	if bufWorkYAMLExists {
		if bufYAMLExists {
			// TODO: Does this match current behavior?
			// TODO: better error message, potentially take into account the location of the bucket via an option
			return nil, errors.New("both buf.yaml and buf.work.yaml discovered at input directory")
		}
		moduleDirPaths, err := getModuleDirPathsForConfirmedBufWorkYAMLDirPath(ctx, bucket, workspaceOptions.subDirPath)
		if err != nil {
			return nil, err
		}
		//fmt.Println("buf.work.yaml found at", workspaceOptions.subDirPath, "moduleDirPaths", moduleDirPaths)
		return newWorkspaceForBucketAndModuleDirPaths(
			ctx,
			bucket,
			moduleDataProvider,
			moduleDirPaths,
			workspaceOptions,
		)
	}

	// We did not find a buf.work.yaml at subDirPath, we will search for one.
	//
	// We skip this if we're already at "." before first iteration.
	if workspaceOptions.subDirPath != "." {
		curDirPath := normalpath.Dir(workspaceOptions.subDirPath)
		// We can't just do a normal for-loop, we want to run this condition even if curDirPath == ".", this is a do...while loop
		for {
			bufWorkYAMLExists, err := bufWorkYAMLExistsAtPrefix(ctx, bucket, curDirPath)
			if err != nil {
				return nil, err
			}
			if bufWorkYAMLExists {
				moduleDirPaths, err := getModuleDirPathsForConfirmedBufWorkYAMLDirPath(ctx, bucket, curDirPath)
				if err != nil {
					return nil, err
				}
				if len(moduleDirPaths) == 0 {
					// In this case, the enclosing buf.work.yaml does not list any module under subDirPath, and we will
					// operate as if there is no workspace.
					// TODO: do we instead want to error? I think we error right now, but we may not want to anymore.
					return newWorkspaceForBucketAndModuleDirPaths(
						ctx,
						bucket,
						moduleDataProvider,
						[]string{workspaceOptions.subDirPath},
						workspaceOptions,
					)
				}
				//fmt.Println("buf.work.yaml found at", curDirPath, "moduleDirPaths", moduleDirPaths)
				return newWorkspaceForBucketAndModuleDirPaths(
					ctx,
					bucket,
					moduleDataProvider,
					moduleDirPaths,
					workspaceOptions,
				)
			}
			if curDirPath == "." {
				break
			}
			curDirPath = normalpath.Dir(curDirPath)
		}
	}

	// No buf.work.yaml found, we operate as if the subDirPath is a single module with no enclosing workspace.
	//fmt.Println("no buf.work.yaml found")
	return newWorkspaceForBucketAndModuleDirPaths(
		ctx,
		bucket,
		moduleDataProvider,
		[]string{workspaceOptions.subDirPath},
		workspaceOptions,
	)
}

func newWorkspaceForModuleSet(moduleSet bufmodule.ModuleSet) (*workspace, error) {
	opaqueIDToLintConfig := make(map[string]bufconfig.LintConfig)
	opaqueIDToBreakingConfig := make(map[string]bufconfig.BreakingConfig)
	for _, module := range moduleSet.Modules() {
		opaqueIDToLintConfig[module.OpaqueID()] = bufconfig.DefaultLintConfig
		opaqueIDToBreakingConfig[module.OpaqueID()] = bufconfig.DefaultBreakingConfig
	}
	return &workspace{
		ModuleSet:                moduleSet,
		opaqueIDToLintConfig:     opaqueIDToLintConfig,
		opaqueIDToBreakingConfig: opaqueIDToBreakingConfig,
		configuredDepModuleRefs:  nil,
	}, nil
}

func newWorkspaceForProtoc(
	ctx context.Context,
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

	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, bufmodule.NopModuleDataProvider)
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
		opaqueIDToLintConfig: map[string]bufconfig.LintConfig{
			".": bufconfig.DefaultLintConfig,
		},
		opaqueIDToBreakingConfig: map[string]bufconfig.BreakingConfig{
			".": bufconfig.DefaultBreakingConfig,
		},
		configuredDepModuleRefs: nil,
	}, nil
}

func newWorkspaceForBucketAndModuleDirPaths(
	ctx context.Context,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	moduleDirPaths []string,
	workspaceOptions *workspaceOptions,
) (*workspace, error) {
	// subDirPath is the input subDirPath. We only want to target modules that are inside
	// this subDirPath. Example: bufWorkYAMLDirPath is "foo", subDirPath is "foo/bar",
	// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
	// and "foo/bar/bat".
	//
	// This is new behavior - before, we required that you input an exact match for the module directory path,
	// but now, we will take all the modules underneath this workspace.
	//
	// We need to verify that at least one module is targeted.
	isTargetFunc := func(moduleDirPath string) bool {
		return normalpath.EqualsOrContainsPath(workspaceOptions.subDirPath, moduleDirPath, normalpath.Relative)
	}
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, moduleDataProvider)
	bucketIDToModuleConfig := make(map[string]bufconfig.ModuleConfig)
	var allConfiguredDepModuleRefs []bufmodule.ModuleRef
	for _, moduleDirPath := range moduleDirPaths {
		moduleConfig, configuredDepModuleRefs, err := getModuleConfigAndConfiguredDepModuleRefsForModuleDirPath(ctx, bucket, moduleDirPath)
		if err != nil {
			return nil, err
		}
		allConfiguredDepModuleRefs = append(allConfiguredDepModuleRefs, configuredDepModuleRefs...)
		bucketIDToModuleConfig[moduleDirPath] = moduleConfig
		moduleBucket := storage.MapReadBucket(
			bucket,
			storage.MapOnPrefix(moduleDirPath),
		)
		bufLockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, moduleBucket, ".")
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
		mappedModuleBucket, moduleTargeting, err := getMappedModuleBucketAndModuleTargeting(
			ctx,
			moduleBucket,
			moduleDirPath,
			moduleConfig,
			workspaceOptions,
		)
		moduleSetBuilder.AddLocalModule(
			mappedModuleBucket,
			moduleDirPath,
			isTargetFunc(moduleDirPath),
			bufmodule.LocalModuleWithModuleFullName(moduleConfig.ModuleFullName()),
			bufmodule.LocalModuleWithTargetPaths(
				moduleTargeting.moduleTargetPaths,
				moduleTargeting.moduleTargetExcludePaths,
			),
		)
	}
	moduleSet, err := moduleSetBuilder.Build()
	if err != nil {
		return nil, err
	}

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
		opaqueIDToLintConfig:     opaqueIDToLintConfig,
		opaqueIDToBreakingConfig: opaqueIDToBreakingConfig,
		configuredDepModuleRefs:  allConfiguredDepModuleRefs,
	}, nil
}

func getModuleDirPathsForConfirmedBufWorkYAMLDirPath(
	ctx context.Context,
	bucket storage.ReadBucket,
	// This may be a parent of subDirPath via search.
	bufWorkYAMLDirPath string,
) ([]string, error) {
	bufWorkYAMLFile, err := bufconfig.GetBufWorkYAMLFileForPrefix(ctx, bucket, bufWorkYAMLDirPath)
	if err != nil {
		return nil, err
	}
	// Just a sanity check. This should have already been validated, but let's make sure.
	if bufWorkYAMLFile.FileVersion() != bufconfig.FileVersionV1 {
		return nil, syserror.Newf("buf.work.yaml at %s did not have version v1", bufWorkYAMLDirPath)
	}
	moduleDirPaths := bufWorkYAMLFile.DirPaths()
	for i, moduleDirPath := range moduleDirPaths {
		// This is the full path relative to the root of the bucket.
		moduleDirPaths[i] = normalpath.Join(bufWorkYAMLDirPath, moduleDirPath)
	}
	return moduleDirPaths, nil
}

// This helper function kind of sucks. When we go to v2, we'll just want to pass back the BufYAMLFile
// and let above functions deal with it, but for now we get some validation that this is just v1.
func getModuleConfigAndConfiguredDepModuleRefsForModuleDirPath(
	ctx context.Context,
	bucket storage.ReadBucket,
	moduleDirPath string,
) (bufconfig.ModuleConfig, []bufmodule.ModuleRef, error) {
	bufYAMLFile, err := bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, moduleDirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If we do not have a buf.yaml, we use the default config.
			// This is a v1 config.
			return bufconfig.DefaultModuleConfig, nil, nil
		}
		return nil, nil, err
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
	moduleBucket storage.ReadBucket,
	moduleDirPath string,
	moduleConfig bufconfig.ModuleConfig,
	workspaceOptions *workspaceOptions,
) (storage.ReadBucket, *moduleTargeting, error) {
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
		workspaceOptions,
	)
	if err != nil {
		return nil, nil, err
	}
	return mappedModuleBucket, moduleTargeting, nil
}

func bufWorkYAMLExistsAtPrefix(ctx context.Context, bucket storage.ReadBucket, prefix string) (bool, error) {
	fileVersion, err := bufconfig.GetBufWorkYAMLFileVersionForPrefix(ctx, bucket, prefix)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	// Just a sanity check. This should have already been validated, but let's make sure.
	if fileVersion != bufconfig.FileVersionV1 {
		return false, syserror.Newf("buf.work.yaml at %s did not have version v1", prefix)
	}
	return true, nil
}

func bufYAMLExistsAtPrefix(ctx context.Context, bucket storage.ReadBucket, prefix string) (bool, error) {
	fileVersion, err := bufconfig.GetBufYAMLFileVersionForPrefix(ctx, bucket, prefix)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	// Just a sanity check. This should have already been validated, but let's make sure.
	if fileVersion != bufconfig.FileVersionV1Beta1 && fileVersion != bufconfig.FileVersionV1 {
		return false, syserror.Newf("buf.yaml at %s did not have version v1beta1 or v1", prefix)
	}
	return true, nil
}

type workspaceOptions struct {
	subDirPath         string
	targetPaths        []string
	targetExcludePaths []string
}

func newWorkspaceOptions() *workspaceOptions {
	return &workspaceOptions{}
}

// this is so we can rely on all paths in workspaceOptions being normalized and validated everywhere
func normalizeAndValidateWorkspaceOptions(workspaceOptions *workspaceOptions) error {
	var err error
	workspaceOptions.subDirPath, err = normalpath.NormalizeAndValidate(workspaceOptions.subDirPath)
	if err != nil {
		return err
	}
	for i, targetPath := range workspaceOptions.targetPaths {
		targetPath, err = normalpath.NormalizeAndValidate(targetPath)
		if err != nil {
			return err
		}
		workspaceOptions.targetPaths[i] = targetPath
	}
	for i, targetExcludePath := range workspaceOptions.targetExcludePaths {
		targetExcludePath, err = normalpath.NormalizeAndValidate(targetExcludePath)
		if err != nil {
			return err
		}
		workspaceOptions.targetExcludePaths[i] = targetExcludePath
	}
	return nil
}

// TODO: All the module_bucket_builder_test.go stuff needs to be copied over

type moduleTargeting struct {
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleTargetPaths []string
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleTargetExcludePaths []string
}

func newModuleTargeting(
	moduleDirPath string,
	roots []string,
	workspaceOptions *workspaceOptions,
) (*moduleTargeting, error) {
	var moduleTargetPaths []string
	var moduleTargetExcludePaths []string

	for _, targetPath := range workspaceOptions.targetPaths {
		if targetPath == moduleDirPath {
			// We're just going to be realists in our error messages here.
			// TODO: Do we error here currently? If so, this error remains. For extra credit in the future,
			// if we were really clever, we'd go back and just add this as a module path.
			return nil, fmt.Errorf("%q was specified with --path but is also the path to a module - specify this module path directly as an input", targetPath)
		}
		if normalpath.ContainsPath(moduleDirPath, targetPath, normalpath.Relative) {
			moduleTargetPath, err := normalpath.Rel(moduleDirPath, targetPath)
			if err != nil {
				return nil, err
			}
			moduleTargetPaths = append(moduleTargetPaths, moduleTargetPath)
		}
	}
	for _, targetExcludePath := range workspaceOptions.targetExcludePaths {
		if targetExcludePath == moduleDirPath {
			// We're just going to be realists in our error messages here.
			// TODO: Do we error here currently? If so, this error remains. For extra credit in the future,
			// if we were really clever, we'd go back and just remove this as a module path if it was specified.
			return nil, fmt.Errorf("%q was specified with --exclude-path but is also the path to a module - specify this module path directly as an input", targetExcludePath)
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

	return &moduleTargeting{
		moduleTargetPaths:        moduleTargetPaths,
		moduleTargetExcludePaths: moduleTargetExcludePaths,
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
