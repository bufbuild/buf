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

// moduleSetModule is a wrapper of Module that includes the properties we used
// to denote priority order of equivalent Modules added to the ModuleSetBuilder.
type moduleSetModule struct {
	Module

	createdFromBucket bool
}

func newModuleSetModule(
	module Module,
	isCreatedFromBucket bool,
) *moduleSetModule {
	return &moduleSetModule{
		Module:            module,
		createdFromBucket: isCreatedFromBucket,
	}
}

func (m *moduleSetModule) isCreatedFromBucket() bool {
	return m.createdFromBucket
}
