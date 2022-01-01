// Copyright 2020-2022 Buf Technologies, Inc.
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

import "github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"

type workspace struct {
	// bufmoduleref.ModuleIdentity -> bufmodule.Module
	namedModules map[string]Module
	allModules   []Module
}

func newWorkspace(
	namedModules map[string]Module,
	allModules []Module,
) *workspace {
	return &workspace{
		namedModules: namedModules,
		allModules:   allModules,
	}
}

func (w *workspace) GetModule(moduleIdentity bufmoduleref.ModuleIdentity) (Module, bool) {
	module, ok := w.namedModules[moduleIdentity.IdentityString()]
	return module, ok
}

func (w *workspace) GetModules() []Module {
	return w.allModules
}
