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

// ModuleDep is the dependency of a Module.
//
// It's just a Module as well as whether or not the dependency is direct.
type ModuleDep interface {
	Module

	// IsDirect returns true if the module is a direct dependency.
	IsDirect() bool

	isModuleDep()
}

// *** PRIVATE ***

type moduleDep struct {
	Module

	isDirect bool
}

func newModuleDep(
	module Module,
	isDirect bool,
) *moduleDep {
	return &moduleDep{
		Module: module,
	}
}

func (m *moduleDep) IsDirect() bool {
	return m.isDirect
}

func (*moduleDep) isModuleDep() {}
