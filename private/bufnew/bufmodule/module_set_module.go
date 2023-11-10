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

package bufmodule

// moduleSetModule is a wrapper of Module that includes the properties of whether
// the Module was targeted, and whether or not it was built from a ModuleKey.
//
// We use these properties to construct the ModuleSet, including choosing
// priority order of locally-equivalent Modules added to the ModuleSet.
//
// We do not expose isTarget on Module, as this would make no sense in the
// context of a Module simply read from a ModuleProvider. Targeting only makes
// sense in the context of a ModuleSet and its construction, therefore we keep
// the exposing of targets vs non-targets (via TargetModules()) to just the
// ModuleSet.
//
// Note that we can't limit isTarget to Workspaces either - we use the Target
// information to denote priority order of logically-equivalent modules added
// to the ModuleSetBuilder, i.e. a Module added from a buf.lock vs a Module
// added from sources (we always prefer sources).
type moduleSetModule struct {
	Module

	target            bool
	createdFromBucket bool
}

func newModuleSetModule(
	module Module,
	isTarget bool,
	isCreatedFromBucket bool,
) *moduleSetModule {
	return &moduleSetModule{
		Module:            module,
		target:            isTarget,
		createdFromBucket: isCreatedFromBucket,
	}
}

func (m *moduleSetModule) isTarget() bool {
	return m.target
}

func (m *moduleSetModule) isCreatedFromBucket() bool {
	return m.createdFromBucket
}
