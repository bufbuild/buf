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
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
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

// WithProtoFileTargetPath returns a new WorkspaceBucketOption that specifically targets
// a single .proto file, and optionally targets all other .proto files that are in the same package.
//
// If targetPath is empty, includePackageFiles is ignored.
// Exclusive with WithTargetPaths - only one of these can have a non-empty value.
//
// This is used for ProtoFileRefs only. Do not use this otherwise.
func WithProtoFileTargetPath(
	protoFileTargetPath string,
	includePackageFiles bool,
) WorkspaceBucketOption {
	return &workspaceProtoFileTargetPathOption{
		protoFileTargetPath: protoFileTargetPath,
		includePackageFiles: includePackageFiles,
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
// This flag will only work if no buf.work.yaml is detected, and the buf.yaml is a v1beta1
// buf.yaml, v1 buf.yaml, or no buf.yaml. This flag will not work if a buf.work.yaml is
// detected, or a v2 buf.yaml is detected.

// If used with NewWorkspaceForModuleKey, this has no effect on the build,
// i.e. excludes are not respected, and the module name is ignored. This matches old behavior.
//
// This implements the deprected --config flag.
//
// See bufconfig.GetBufYAMLFileForPrefixOrOverride for more details.
//
// *** DO NOT USE THIS OUTSIDE OF THE CLI AND/OR IF YOU DON'T UNDERSTAND IT. ***
// *** DO NOT ADD THIS TO ANY NEW COMMANDS. ***
//
// Current comments that use this: build, breaking, lint, generate, format,
// export, ls-breaking-rules, ls-lint-rules.
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

type workspaceProtoFileTargetPathOption struct {
	protoFileTargetPath string
	includePackageFiles bool
}

func (p *workspaceProtoFileTargetPathOption) applyToWorkspaceBucketConfig(config *workspaceBucketConfig) {
	config.protoFileTargetPath = p.protoFileTargetPath
	config.includePackageFiles = p.includePackageFiles
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
	subDirPath          string
	targetPaths         []string
	targetExcludePaths  []string
	protoFileTargetPath string
	includePackageFiles bool
	configOverride      string
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
	if config.protoFileTargetPath != "" {
		config.protoFileTargetPath, err = normalpath.NormalizeAndValidate(config.protoFileTargetPath)
		if err != nil {
			return nil, err
		}
	}
	if len(config.targetPaths) > 0 || len(config.targetExcludePaths) > 0 {
		if config.protoFileTargetPath != "" {
			// This is just a system error. We messed up and called both exclusive options.
			return nil, syserror.New("cannot set targetPaths/targetExcludePaths with protoFileTargetPaths")
		}
		// These are actual user errors. This is us verifying --path and --exclude-path.
		// An argument could be made this should be at a higher level for user errors, and then
		// if it gets to this point, this should be a system error.
		//
		// We don't use --path, --exclude-path here because these paths have had ExternalPathToPath
		// applied to them. Which is another argument to do this at a higher level.
		for _, targetPath := range config.targetPaths {
			if targetPath == config.subDirPath {
				return nil, errors.New("given input is equal to a value of --path - this has no effect and is disallowed")
			}
			// We want this to be deterministic.  We don't have that many paths in almost all cases.
			// This being n^2 shouldn't be a huge issue unless someone has a diabolical wrapping shell script.
			// If this becomes a problem, there's optimizations we can do by turning targetExcludePaths into
			// a map but keeping the index in config.targetExcludePaths around to prioritize what error
			// message to print.
			for _, targetExcludePath := range config.targetExcludePaths {
				if targetPath == targetExcludePath {
					return nil, errors.New("cannot set the same path both --path and --exclude-path")
				}
				// This is new post-refactor. Before, we gave precedence to --path. While a change,
				// doing --path foo/bar --exclude-path foo seems like a bug rather than expected behavior to maintain.
				if normalpath.EqualsOrContainsPath(targetExcludePath, targetPath, normalpath.Relative) {
					return nil, fmt.Errorf("excluded path %q contains targeted path %q, which means all paths in %q will be excluded", targetExcludePath, targetPath, targetPath)
				}
			}
		}
		for _, targetExcludePath := range config.targetExcludePaths {
			if targetExcludePath == config.subDirPath {
				return nil, errors.New("given input is equal to a value of --exclude-path - this would exclude everything")
			}
		}
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
