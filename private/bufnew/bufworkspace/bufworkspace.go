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
	ConfiguredDepModuleRefs() []bufmodule.ModuleRef

	// GenerateConfigs returns the GenerateConfigs for the workspace, if they exist.
	//
	// v1 buf.yamls do not have GenerateConfigs. These need to be read from buf.gen.yaml files.
	//GenerateConfigs() []GenerateConfig

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
