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
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

// WorkspaceBucketOption is an option for a new Workspace created by a Bucket.
type WorkspaceBucketOption interface {
	applyToWorkspaceBucketConfig(*workspaceBucketConfig)
}

// This selects the specific directory within the Workspace bucket to target.
//
// Example: We have modules at foo/bar, foo/baz. "." will result in both
// modules being selected, so will "foo", but "foo/bar" will result in only
// the foo/bar module.
//
// A subDirPath of "." is equivalent of not setting this option.
func WithTargetSubDirPath(subDirPath string) WorkspaceBucketOption {
	return &workspaceTargetSubDirPathOption{
		subDirPath: subDirPath,
	}
}

// WorkspaceModuleKeyOption is an option for a new Workspace created by a ModuleKey.
type WorkspaceModuleKeyOption interface {
	applyToWorkspaceModuleKeyConfig(*workspaceModuleKeyConfig)
}

// WorkspaceOption is an option for a new Workspace created by either a Bucket or ModuleKey.
type WorkspaceOption interface {
	WorkspaceBucketOption
	WorkspaceModuleKeyOption
}

// Note these paths need to have the path/to/module stripped, and then each new path
// filtered to the specific module it applies to. If some modules do not have any
// target paths, but we specified WorkspaceWithTargetPaths, then those modules
// need to be built as non-targeted.
//
// Theese paths have to  be within the subDirPath, if it exists.
func WithTargetPaths(targetPaths []string, targetExcludePaths []string) WorkspaceOption {
	return &workspaceTargetPathsOption{
		targetPaths:        targetPaths,
		targetExcludePaths: targetExcludePaths,
	}
}

// WithConfigOverride applies the config override.
//
// This flag will only work if no buf.work.yaml is detected, and the buf.yaml is a v1beta1 buf.yaml, v1 buf.yaml, or no buf.yaml.
// This flag will not work if a buf.work.yaml is detected, or a v2 buf.yaml is detected.

// If used with NewWorkspaceForModuleKey, this has no effect on the build, i.e. excludes are not respected, and the module name
// is ignored. This matches old behavior.
//
// This implements the deprected --config flag.
//
// See bufconfig.GetBufYAMLFileForPrefixOrOverride for more details.
//
// *** DO NOT USE THIS OUTSIDE OF THE CLI AND/OR IF YOU DON'T UNDERSTAND IT. ***
// *** DO NOT ADD THIS TO ANY NEW COMMANDS. ***
//
// Current comments that use this: build, breaking, lint, generate, format, export, ls-breaking-rules, ls-lint-rules.
func WithConfigOverride(configOverride string) WorkspaceOption {
	return &workspaceConfigOverrideOption{
		configOverride: configOverride,
	}
}

type workspaceTargetSubDirPathOption struct {
	subDirPath string
}

func (s *workspaceTargetSubDirPathOption) applyToWorkspaceBucketConfig(config *workspaceBucketConfig) {
	config.subDirPath = s.subDirPath
}

type workspaceTargetPathsOption struct {
	targetPaths        []string
	targetExcludePaths []string
}

func (t *workspaceTargetPathsOption) applyToWorkspaceBucketConfig(config *workspaceBucketConfig) {
	config.targetPaths = t.targetPaths
	config.targetExcludePaths = t.targetExcludePaths
}

func (t *workspaceTargetPathsOption) applyToWorkspaceModuleKeyConfig(config *workspaceModuleKeyConfig) {
	config.targetPaths = t.targetPaths
	config.targetExcludePaths = t.targetExcludePaths
}

type workspaceConfigOverrideOption struct {
	configOverride string
}

func (c *workspaceConfigOverrideOption) applyToWorkspaceBucketConfig(config *workspaceBucketConfig) {
	config.configOverride = c.configOverride
}

func (c *workspaceConfigOverrideOption) applyToWorkspaceModuleKeyConfig(config *workspaceModuleKeyConfig) {
	config.configOverride = c.configOverride
}

type workspaceBucketConfig struct {
	subDirPath         string
	targetPaths        []string
	targetExcludePaths []string
	configOverride     string
}

func newWorkspaceBucketConfig(options []WorkspaceBucketOption) (*workspaceBucketConfig, error) {
	config := &workspaceBucketConfig{}
	for _, option := range options {
		option.applyToWorkspaceBucketConfig(config)
	}
	var err error
	config.subDirPath, err = normalpath.NormalizeAndValidate(config.subDirPath)
	if err != nil {
		return nil, err
	}
	config.targetPaths, err = slicesext.MapError(
		config.targetPaths,
		normalpath.NormalizeAndValidate,
	)
	if err != nil {
		return nil, err
	}
	config.targetExcludePaths, err = slicesext.MapError(
		config.targetExcludePaths,
		normalpath.NormalizeAndValidate,
	)
	if err != nil {
		return nil, err
	}
	return config, nil
}

type workspaceModuleKeyConfig struct {
	targetPaths        []string
	targetExcludePaths []string
	configOverride     string
}

func newWorkspaceModuleKeyConfig(options []WorkspaceModuleKeyOption) (*workspaceModuleKeyConfig, error) {
	config := &workspaceModuleKeyConfig{}
	for _, option := range options {
		option.applyToWorkspaceModuleKeyConfig(config)
	}
	var err error
	config.targetPaths, err = slicesext.MapError(
		config.targetPaths,
		normalpath.NormalizeAndValidate,
	)
	if err != nil {
		return nil, err
	}
	config.targetExcludePaths, err = slicesext.MapError(
		config.targetExcludePaths,
		normalpath.NormalizeAndValidate,
	)
	if err != nil {
		return nil, err
	}
	return config, nil
}
