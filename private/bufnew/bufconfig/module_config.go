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

import "github.com/bufbuild/buf/private/bufnew/bufmodule"

// TODO
var DefaultModuleConfig ModuleConfig = nil

// ModuleConfig is configuration for a specific Module.
//
// ModuleConfigs do not expose BucketID or OpaqueID, however RootPath is effectively BucketID,
// and ModuleFullName -> fallback to RootPath effectively is OpaqueID. Given that it is up to
// the user of this package to decide what to do with these fields, we do not name RootPath as
// BucketID, and we do not expose OpaqueID.
type ModuleConfig interface {
	// RootPath returns the root path of the Module, if set.
	//
	// For v1 buf.yamls, this is always empty.
	//
	// If not empty, this will be used as the BucketID within Workspaces. For v1, it is up
	// to the Workspace constructor to come up with a BucketID (likely the directory name
	// within buf.work.yaml).
	RootPath() string
	// ModuleFullName returns the ModuleFullName for the Module, if available.
	//
	// This may be nil.
	ModuleFullName() bufmodule.ModuleFullName
	// LintConfig returns the lint configuration.
	//
	// If this was not set, this will be set to the default lint configuration.
	LintConfig() LintConfig
	// BreakingConfig returns the breaking configuration.
	//
	// If this was not set, this will be set to the default breaking configuration.
	BreakingConfig() BreakingConfig

	// TODO: RootToExcludes
	// TODO: DependencyModuleReferences: how do these fit in? We likely add them here,
	// and do not have ModuleConfigs at the bufworkspace level.

	isModuleConfig()
}

// *** PRIVATE ***

type moduleConfig struct{}

func newModuleConfig() *moduleConfig {
	return &moduleConfig{}
}

func (m *moduleConfig) RootPath() string {
	panic("not implemented") // TODO: Implement
}

func (m *moduleConfig) ModuleFullName() bufmodule.ModuleFullName {
	panic("not implemented") // TODO: Implement
}

func (m *moduleConfig) LintConfig() LintConfig {
	panic("not implemented") // TODO: Implement
}

func (m *moduleConfig) BreakingConfig() BreakingConfig {
	panic("not implemented") // TODO: Implement
}

func (*moduleConfig) isModuleConfig() {}
