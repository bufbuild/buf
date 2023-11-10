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

package bufimage

import (
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
)

var _ ImageModuleDependency = &imageModuleDependency{}

type imageModuleDependency struct {
	moduleFullName bufmodule.ModuleFullName
	commitID       string
	isDirect       bool
}

func newImageModuleDependency(
	moduleFullName bufmodule.ModuleFullName,
	commitID string,
	isDirect bool,
) *imageModuleDependency {
	return &imageModuleDependency{
		moduleFullName: moduleFullName,
		commitID:       commitID,
		isDirect:       isDirect,
	}
}

func (i *imageModuleDependency) ModuleFullName() bufmodule.ModuleFullName {
	return i.moduleFullName
}

func (i *imageModuleDependency) CommitID() string {
	return i.commitID
}

func (i *imageModuleDependency) IsDirect() bool {
	return i.isDirect
}

func (i *imageModuleDependency) String() string {
	moduleFullNameString := i.moduleFullName.String()
	if i.commitID != "" {
		return moduleFullNameString + ":" + i.commitID
	}
	return moduleFullNameString
}

func (*imageModuleDependency) isImageModuleDependency() {}
