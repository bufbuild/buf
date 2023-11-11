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

	// GetModuleConfigForOpaqueID gets the ModuleConfig for the OpaqueID, if the OpaqueID
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
	//     Config() ModuleConfig
	//   }
	//
	// However, this would mean that Workspace would not inherit ModuleSet, as we'd
	// want to create GetWorkspaceModule.* functions instead of GetModule.* functions,
	// and then provide a WorkpaceToModuleSet global function. This seems messier in
	// practice than having users call GetModuleConfigForOpaqueID(module.OpaqueID())
	// in the situations where they need configuration.
	GetModuleConfigForOpaqueID(opaqueID string) ModuleConfig

	// ConfiguredDepModuleRefs returns the configured dependencies of the Workspace as ModuleRefs.
	//
	// These come from buf.yaml files.
	//
	// This may or may not include ModuleRefs that reference Modules within the ModuleSet.
	// ModuleSets include all dependencies, so in theory, all ModuleRefs should have a Module,
	// however we may have misconfigured ModuleRefs.
	//
	// The ModuleRefs in this list will be unique by ModuleFullName. Resolution of ModuleRefs
	// is done at Workspace construction time. For example, with v1 buf.yaml, this is a union
	// of the buf.yaml files in the Workspace, resolving common ModuleFullNames to a single ModuleRef.
	ConfiguredDepModuleRefs() []bufmodule.ModuleRef
	// LockedDepModuleKeys returns the locked dependencies of the Workspace as ModuleKeys.
	//
	// These come from buf.lock files.
	//
	// This may or may not include ModuleKeys that keyerence Modules within the ModuleSet.
	// ModuleSets include all dependencies, so in theory, all ModuleKeys should have a Module,
	// however we may have misconfigured ModuleKeys.
	//
	// The ModuleKeys in this list will be unique by ModuleFullName. Resolution of ModuleKeys
	// is done at Workspace construction time. For example, with v1 buf.yaml, this is a union
	// of the buf.lock files in the Workspace, resolving common ModuleFullNames to a single ModuleKey.
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
func NewWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
	options ...WorkspaceOption,
) (Workspace, error) {
	// TODO
	return nil, errors.New("TODO NewWorkspaceForBucket")
}

// WorkspaceOption is an option for a new Workspace.
type WorkspaceOption func(*workspaceOptions)

// This selects the specific modules within the Workspace bucket to target.
//
// Example: We have modules at foo/bar, foo/baz. "." will result in both
// modules being selected, so will "foo", but "foo/bar" will result in only
// the foo/bar module.
func WorkspaceWithTargetSubDirPaths(subDirPaths []string) WorkspaceOption {
	return func(workspaceOptions *workspaceOptions) {
		workspaceOptions.subDirPaths = subDirPaths
	}
}

// Note these paths need to have the path/to/module stripped, and then each new path
// filtered to the specific module it applies to. If some modules do not have any
// target paths, but we specified WorkspaceWithTargetPaths, then those modules
// need to be built as non-targeted.
func WorkspaceWithTargetPaths(
	targetPaths []string,
	targetExcludePaths []string,
) WorkspaceOption {
	return func(workspaceOptions *workspaceOptions) {
		workspaceOptions.targetPaths = targetPaths
		workspaceOptions.targetExcludePaths = targetExcludePaths
	}
}

// ModuleConfig is configuration for a specific Module.
//
// This only contains the information needed outside of Workspace construction.
type ModuleConfig interface {
	Lint() LintConfig
	Breaking() BreakingConfig

	isModuleConfig()
}

// CheckConfig is the common interface for the configuration shared by
// LintConfig and BreakingConfig.
type CheckConfig interface {
	UseIDs() []string
	ExceptIDs() string
	// Paths are specific to the Module.
	IgnorePaths() []string
	// Paths are specific to the Module.
	IgnoreIDToPaths() map[string][]string

	isCheckConfig()
}

// LintConfig is lint configuration for a specific Module.
type LintConfig interface {
	CheckConfig

	EnumZeroValueSuffix() string
	RPCAllowSameRequestResponse() bool
	RPCAllowGoogleProtobufEmptyRequests() bool
	RPCAllowGoogleProtobufEmptyResponses() bool
	ServiceSuffix() string
	AllowCommentIgnores() bool

	isLintConfig()
}

// BreakingConfig is breaking configuration for a specific Module.
type BreakingConfig interface {
	CheckConfig

	IgnoreUnstablePackages() bool

	isBreakingConfig()
}

//type GenerateConfig interface{}

type workspaceOptions struct {
	subDirPaths        []string
	targetPaths        []string
	targetExcludePaths []string
}

func newWorkspaceOptions() *workspaceOptions {
	return &workspaceOptions{}
}
