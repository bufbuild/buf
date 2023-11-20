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

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// Workspace is a buf workspace.
//
// It is a bufmodule.ModuleSet with associated configuration.
//
// See ModuleSet helper functions for many of your needs. Some examples:
//
//   - bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles
//   - bufmodule.ModuleSetToTargetModules
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

	// GetGenerateConfigs gets the generation configurations associated with the workspace.
	//
	// For v2 buf.yamls, these are read directly.
	// For v1 buf.yamls, these need to be separately read and provided to the Workspace
	// via WorkspaceWithGenerateConfig.
	GenerateConfigs() []bufconfig.GenerateConfig

	// ConfiguredDepModuleRefs returns the configured dependencies of the Workspace as ModuleRefs.
	//
	// These come from buf.yaml files.
	//
	// The ModuleRefs in this list will be unique by ModuleFullName. Resolution of ModuleRefs
	// is done at Workspace construction time. For example, with v1 buf.yaml, this is a union
	// of the buf.yaml files in the Workspace, resolving common ModuleFullNames to a single ModuleRef.
	//
	// Sorted by ModuleFullName.
	ConfiguredDepModuleRefs() []bufmodule.ModuleRef
	// LockedDepModuleKeys returns the locked dependencies of the Workspace as ModuleKeys.
	//
	// These come from buf.lock files.
	//
	// The ModuleKeys in this list will be unique by ModuleFullName. Resolution of ModuleKeys
	// is done at Workspace construction time. For example, with v1 buf.yaml, this is a union
	// of the buf.lock files in the Workspace, resolving common ModuleFullNames to a single ModuleKey.
	//
	// Sorted by ModuleFullName.
	LockedDepModuleKeys() []bufmodule.ModuleKey

	// GenerateConfigs returns the GenerateConfigs for the workspace, if they exist.
	//
	// v1 buf.yamls do not have GenerateConfigs. These need to be read from buf.gen.yaml files.
	//GenerateConfigs() []GenerateConfig

	isWorkspace()
}

// NewWorkspaceForBucket returns a new Workspace for the given Bucket.
//
// All parsing of configuration files is done behind the scenes here.
// This function can read a single v1 buf.yaml, a v1 buf.work.yaml, or a v2 buf.yaml.
//
// TODO: we might just want to pass a Bucket for the root OS directory, i.e. "/", along
// with a path, for any DirRef or ProtoFileRef. For git and archives, we'll pass a Bucket
// representing the root of the git repostory or archive, along with a path. Then, we let
// this function deal with all the finding of files. The one hiccup here is dealing with
// external paths - we want to have the same behavior as we've had with external paths.
// But removing all the SubDirPath, RootRelativePath (if possible), TerminalFileProvider
// stuff by massively simplifying this might be a win.
func NewWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceOption,
) (Workspace, error) {
	return newWorkspaceForBucket(ctx, bucket, moduleDataProvider, options...)
}

// WorkspaceOption is an option for a new Workspace.
type WorkspaceOption func(*workspaceOptions)

// This selects the specific directory within the Workspace bucket to target.
//
// Example: We have modules at foo/bar, foo/baz. "." will result in both
// modules being selected, so will "foo", but "foo/bar" will result in only
// the foo/bar module.
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

// WorkspaceWithGenerateConfig returns a new WorkspaceOption that adds the given GenerateConfig.
func WorkspaceWithGenerateConfig(generateConfig bufconfig.GenerateConfig) WorkspaceOption {
	return func(workspaceOptions *workspaceOptions) {
		workspaceOptions.generateConfigs = append(workspaceOptions.generateConfigs, generateConfig)
	}
}

// WorkspaceUnreferencedConfiguredDepModuleRefs returns those configured ModuleRefs that do not
// reference any Module within the workspace. These can be pruned from the buf.lock
// in both v1 and v2 buf.yamls.
//
// A ModuleRef is considered to reference a Module if it has the same ModuleFullName.
//
// TODO: This logic is likely broken, need to use ModuleSetRemoteDepsOfLocalModules. We may
// not even need to expose LockedDepModuleKeys.
func WorkspaceUnreferencedConfiguredDepModuleRefs(workspace Workspace) []bufmodule.ModuleRef {
	var unreferencedConfiguredDepModuleRefs []bufmodule.ModuleRef
	for _, configuredDepModuleRef := range workspace.ConfiguredDepModuleRefs() {
		// Workspaces are self-contained and have all dependencies, therefore
		// this check is all that is needed.
		if workspace.GetModuleForModuleFullName(configuredDepModuleRef.ModuleFullName()) == nil {
			unreferencedConfiguredDepModuleRefs = append(unreferencedConfiguredDepModuleRefs, configuredDepModuleRef)
		}
	}
	return unreferencedConfiguredDepModuleRefs
}

// WorkspaceLocalConfiguredDepModuleRefs returns those configured dependency ModuleRefs that
// reference local Modules in the workspace. In theory, these can be pruned from v2 buf.yamls.
//
// These are present in v1 buf.yaml, but they are not used by buf anymore. A note
// that this means that if we prune these, upgrading buf is a one-way door - if a buf
// lock is pruned based on a newer version of buf, it will no longer be useable by
// old versions of buf, if we prune these. We should discuss what we want to do here - perhaps
// these should be pruned depending on v1 vs v2.
//
// A ModuleRef is considered to reference a Module if it has the same ModuleFullName.
//
// TODO: This logic is likely broken, need to use ModuleSetRemoteDepsOfLocalModules. We may
// not even need to expose LockedDepModuleKeys.
func WorkspaceLocalConfiguredDepModuleRefs(workspace Workspace) []bufmodule.ModuleRef {
	var localConfiguredDepModuleRefs []bufmodule.ModuleRef
	for _, configuredDepModuleRef := range workspace.ConfiguredDepModuleRefs() {
		module := workspace.GetModuleForModuleFullName(configuredDepModuleRef.ModuleFullName())
		if module == nil {
			continue
		}
		if module.IsLocal() {
			localConfiguredDepModuleRefs = append(localConfiguredDepModuleRefs, configuredDepModuleRef)
		}
	}
	return localConfiguredDepModuleRefs
}

// WorkspaceUnreferencedLockedDepModuleKeys returns those locked ModuleKeys that do not
// reference any Module within the workspace. These can be pruned from the buf.lock
// in both v1 and v2 buf.yamls.
//
// A ModuleKey is considered to reference a Module if it has the same ModuleFullName.
//
// TODO: This logic is likely broken, need to use ModuleSetRemoteDepsOfLocalModules. We may
// not even need to expose LockedDepModuleKeys.
func WorkspaceUnreferencedLockedDepModuleKeys(workspace Workspace) []bufmodule.ModuleKey {
	var unreferencedLockedDepModuleKeys []bufmodule.ModuleKey
	for _, lockedDepModuleKey := range workspace.LockedDepModuleKeys() {
		// Workspaces are self-contained and have all dependencies, therefore
		// this check is all that is needed.
		if workspace.GetModuleForModuleFullName(lockedDepModuleKey.ModuleFullName()) == nil {
			unreferencedLockedDepModuleKeys = append(unreferencedLockedDepModuleKeys, lockedDepModuleKey)
		}
	}
	return unreferencedLockedDepModuleKeys
}

// WorkspaceLocalLockedDepModuleKeys returns those locked dependency ModuleKeys that
// reference local Modules in the workspace. In theory, these can be pruned from the buf.lock
// in v2 buf.yamls.
//
// These are present in v1 buf.yaml buf.locks, but they are not used by buf anymore. A note
// that this means that if we prune these, upgrading buf is a one-way door - if a buf
// lock is pruned based on a newer version of buf, it will no longer be useable by
// old versions of buf, if we prune these. We should discuss what we want to do here - perhaps
// these should be pruned depending on v1 vs v2.
//
// A ModuleKey is considered to reference a Module if it has the same ModuleFullName.
//
// TODO: This logic is likely broken, need to use ModuleSetRemoteDepsOfLocalModules. We may
// not even need to expose LockedDepModuleKeys.
func WorkspaceLocalLockedDepModuleKeys(workspace Workspace) []bufmodule.ModuleKey {
	var localLockedDepModuleKeys []bufmodule.ModuleKey
	for _, lockedDepModuleKey := range workspace.LockedDepModuleKeys() {
		module := workspace.GetModuleForModuleFullName(lockedDepModuleKey.ModuleFullName())
		if module == nil {
			continue
		}
		if module.IsLocal() {
			localLockedDepModuleKeys = append(localLockedDepModuleKeys, lockedDepModuleKey)
		}
	}
	return localLockedDepModuleKeys
}

type workspaceOptions struct {
	subDirPath         string
	targetPaths        []string
	targetExcludePaths []string
	generateConfigs    []bufconfig.GenerateConfig
}

func newWorkspaceOptions() *workspaceOptions {
	return &workspaceOptions{}
}
