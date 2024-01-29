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
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
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

	// IsV2 signifies if this module was created from a v2 buf.yaml.
	//
	// THIS SHOULD ONLY BE USED IN EXTREMELY LIMITED SITUATIONS. The codebase should generally
	// handle v1 vs v2 transparently. Right now, this is only approved to be used in push
	// when we want to know whether we need to print out dashless CommitIDs. Any other usages
	// need to be evaluated.
	IsV2() bool

	isWorkspace()
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

func newWorkspace(
	moduleSet bufmodule.ModuleSet,
	logger *zap.Logger,
	opaqueIDToLintConfig map[string]bufconfig.LintConfig,
	opaqueIDToBreakingConfig map[string]bufconfig.BreakingConfig,
	configuredDepModuleRefs []bufmodule.ModuleRef,
	createdFromBucket bool,
	isV2 bool,
	updateableBufLockDirPath string,
) *workspace {
	return &workspace{
		ModuleSet:                moduleSet,
		logger:                   logger,
		opaqueIDToLintConfig:     opaqueIDToLintConfig,
		opaqueIDToBreakingConfig: opaqueIDToBreakingConfig,
		configuredDepModuleRefs:  configuredDepModuleRefs,
		createdFromBucket:        createdFromBucket,
		isV2:                     isV2,
		updateableBufLockDirPath: updateableBufLockDirPath,
	}
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

func (w *workspace) IsV2() bool {
	return w.isV2
}

func (*workspace) isWorkspace() {}
