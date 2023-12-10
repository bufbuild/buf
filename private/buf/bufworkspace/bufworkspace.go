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

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
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
	//
	// We use this to warn on unused dependencies in bufctl.
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

// UpdateableWorkspace is a workspace that can be updated.
type UpdateableWorkspace interface {
	Workspace

	// PutBufLockFile updates the lock file that backs this Workspace.
	//
	// If a buf.lock does not exist, one will be created.
	//
	// This will fail for UpdateableWorkspaces not created from v2 buf.yamls.
	PutBufLockFile(ctx context.Context, bufLockFile bufconfig.BufLockFile) error

	isUpdateableWorkspace()
}

// NewUpdateableWorkspaceForBucket returns a new Workspace for the given Bucket.
//
// All parsing of configuration files is done behind the scenes here.
// This function can only read v2 buf.yamls.
func NewUpdateableWorkspaceForBucket(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadWriteBucket,
	moduleDataProvider bufmodule.ModuleDataProvider,
	options ...WorkspaceBucketOption,
) (UpdateableWorkspace, error) {
	return newUpdateableWorkspaceForBucket(ctx, logger, bucket, moduleDataProvider, options...)
}
