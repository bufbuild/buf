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

	// Parent returns the Module that this ModuleDep is a dependency of.
	//
	// Note this is not recursively - this points ot the top-level Module that dependencies
	// were created for. That is, if a -> b -> c, then a will have ModuleDeps b and c, both
	// of which have a as a parent.
	Parent() Module
	// IsDirect returns true if the Module is a direct dependency of this Module.
	IsDirect() bool

	isModuleDep()
}

// *** PRIVATE ***

type moduleDep struct {
	Module

	parent   Module
	isDirect bool
}

func newModuleDep(
	module Module,
	parent Module,
	isDirect bool,
) *moduleDep {
	return &moduleDep{
		Module:   module,
		parent:   parent,
		isDirect: isDirect,
	}
}

func (m *moduleDep) Parent() Module {
	return m.parent
}

func (m *moduleDep) IsDirect() bool {
	return m.isDirect
}

func (*moduleDep) isModuleDep() {}
