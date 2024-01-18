// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufimage

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
)

var _ ImageModuleDependency = &imageModuleDependency{}

type imageModuleDependency struct {
	moduleIdentity bufmoduleref.ModuleIdentity
	commit         string
	isDirect       bool
}

func newImageModuleDependency(
	moduleIdentity bufmoduleref.ModuleIdentity,
	commit string,
	isDirect bool,
) *imageModuleDependency {
	return &imageModuleDependency{
		moduleIdentity: moduleIdentity,
		commit:         commit,
		isDirect:       isDirect,
	}
}

func (i *imageModuleDependency) ModuleIdentity() bufmoduleref.ModuleIdentity {
	return i.moduleIdentity
}

func (i *imageModuleDependency) Commit() string {
	return i.commit
}

func (i *imageModuleDependency) IsDirect() bool {
	return i.isDirect
}

func (i *imageModuleDependency) String() string {
	moduleIdentityString := i.moduleIdentity.IdentityString()
	if i.commit != "" {
		return moduleIdentityString + ":" + i.commit
	}
	return moduleIdentityString
}

func (*imageModuleDependency) isImageModuleDependency() {}
