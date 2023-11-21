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

package bufconfig

import (
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
)

// TODO
// Should match v1 buf.yaml
var DefaultModuleConfig ModuleConfig = newModuleConfig(
	"",
	nil,
	map[string][]string{
		".": []string{},
	},
	DefaultLintConfig,
	DefaultBreakingConfig,
)

// ModuleConfig is configuration for a specific Module.
//
// ModuleConfigs do not expose BucketID or OpaqueID, however DirPath is effectively BucketID,
// and ModuleFullName -> fallback to DirPath effectively is OpaqueID. Given that it is up to
// the user of this package to decide what to do with these fields, we do not name DirPath as
// BucketID, and we do not expose OpaqueID.
type ModuleConfig interface {
	// DirPath returns the path of the Module within the Workspace, if specified.
	//
	// For v1beta1 and v1 buf.yamls, this is always empty.
	//
	// In v2, this will be used as the BucketID within Workspaces. For v1, it is up
	// to the Workspace constructor to come up with a BucketID (likely the directory name
	// within buf.work.yaml).
	DirPath() string
	// ModuleFullName returns the ModuleFullName for the Module, if available.
	//
	// This may be nil.
	ModuleFullName() bufmodule.ModuleFullName
	// RootToExcludes contains a map from root to the excludes for that root.
	//
	// Roots are the root directories within a bucket to search for Protobuf files.
	//
	// There will be no between the roots, ie foo/bar and foo are not allowed.
	// All Protobuf files must be unique relative to the roots, ie if foo and bar
	// are roots, then foo/baz.proto and bar/baz.proto are not allowed.
	// All roots will be normalized and validated.
	//
	// Excludes are the directories within a bucket to exclude.
	// There should be no overlap between the excludes, ie foo/bar and foo are not allowed.
	// All excludes must reside within a root, but none will be equal to a root.
	// All excludes will be normalized and validated.
	//
	// *** The excludes in this map will be relative to the root they map to! ***
	// *** Note that root is relative to DirPath! ***
	// That is, the actual path to a root within a is DirPath()/root, and the
	// actual path to an exclude is DirPath()/root/exclude (in v1beta1 and v1, this
	// is just root and root/exclude).
	//
	// This will never return a nil value.
	// If RootToExcludes is empty in the buf.yaml, this will return "." -> []string{}.
	//
	// For v1beta1, this may contain multiple keys.
	// For v1 and v2, this will contain a single key ".", with potentially some excludes.
	RootToExcludes() map[string][]string
	// LintConfig returns the lint configuration.
	//
	// If this was not set, this will be set to the default lint configuration.
	LintConfig() LintConfig
	// BreakingConfig returns the breaking configuration.
	//
	// If this was not set, this will be set to the default breaking configuration.
	BreakingConfig() BreakingConfig

	// TODO: DependencyModuleReferences: how do these fit in? We likely add them here,
	// and do not have ModuleConfigs at the bufworkspace level.

	isModuleConfig()
}

// *** PRIVATE ***

type moduleConfig struct {
	dirPath        string
	moduleFullName bufmodule.ModuleFullName
	rootToExcludes map[string][]string
	lintConfig     LintConfig
	breakingConfig BreakingConfig
}

func newModuleConfig(
	dirPath string,
	moduleFullName bufmodule.ModuleFullName,
	rootToExcludes map[string][]string,
	lintConfig LintConfig,
	breakingConfig BreakingConfig,
) *moduleConfig {
	// TODO: validation (maybe outside of config construction)
	return &moduleConfig{
		dirPath:        dirPath,
		moduleFullName: moduleFullName,
		rootToExcludes: rootToExcludes,
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
	}
}

func (m *moduleConfig) DirPath() string {
	return m.dirPath
}

func (m *moduleConfig) ModuleFullName() bufmodule.ModuleFullName {
	return m.moduleFullName
}

func (m *moduleConfig) RootToExcludes() map[string][]string {
	c := make(map[string][]string)
	for key, value := range m.rootToExcludes {
		c[key] = slicesextended.Copy(value)
	}
	return c
}

func (m *moduleConfig) LintConfig() LintConfig {
	return m.lintConfig
}

func (m *moduleConfig) BreakingConfig() BreakingConfig {
	return m.breakingConfig
}

func (*moduleConfig) isModuleConfig() {}
