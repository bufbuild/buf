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

type Workspace interface {
	WorkspaceModules() []WorkspaceModule
	DeclaredDepModuleRefs() []bufmodule.ModuleRef
	//GenerateConfigs() []GenerateConfig

	isWorkspace()
}

type WorkspaceModule interface {
	bufmodule.Module

	// Will be default value for Modules that didn't have an associated config,
	// such as modules read from buf.lock files. These Modules shouldn't be
	// targeted, which will result in the linter/breaking change detector ignoring them.
	ModuleConfig() ModuleConfig
}

// Can read a single buf.yaml v1
// Can read a buf.work.yaml
// Can read a buf.yaml v2
func NewWorkspaceForBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
	options ...WorkspaceOption,
) (Workspace, error) {
	return nil, nil
}

type WorkspaceOption func(*workspaceOptions)

func WorkspaceWithTargetSubDirPaths(subDirPaths []string) WorkspaceOption {
	return nil
}

func WorkspaceWithTargetProtoPaths(paths []string, excludePaths []string) WorkspaceOption {
	return nil
}

func GetWorkspaceFileInfo(
	ctx context.Context,
	workspace Workspace,
	path string,
) (bufmodule.FileInfo, error) {
	return nil, errors.New("TODO")
}

func WorkspaceToModuleReadBucketWithOnlyProtoFiles(
	ctx context.Context,
	workspace Workspace,
) (bufmodule.ModuleReadBucket, error) {
	return nil, errors.New("TODO")
}

type workspaceOptions struct{}

type ModuleConfig interface {
	Version() ConfigVersion

	// Note: You could make the argument that you don't actually need this, however there
	// are situations where you just want to read a configuration on its own without
	// a corresponding Workspace.
	ModuleFullName() bufmodule.ModuleFullName

	//RootToExcludes() map[string][]string
	LintConfig() LintConfig
	BreakingConfig() BreakingConfig

	isModuleConfig()
}

type LintConfig interface {
	Version() ConfigVersion

	UseIDs() []string
	ExceptIDs() string
	IgnoreRootPaths() []string
	IgnoreIDToRootPaths() map[string][]string
	EnumZeroValueSuffix() string
	RPCAllowSameRequestResponse() bool
	RPCAllowGoogleProtobufEmptyRequests() bool
	RPCAllowGoogleProtobufEmptyResponses() bool
	ServiceSuffix() string
	AllowCommentIgnores() bool

	isLintConfig()
}

type BreakingConfig interface {
	Version() ConfigVersion

	UseIDs() []string
	ExceptIDs() string
	IgnoreRootPaths() []string
	IgnoreIDToRootPaths() map[string][]string
	IgnoreUnstablePackages() bool

	isBreakingConfig()
}

//type GenerateConfig interface{}
