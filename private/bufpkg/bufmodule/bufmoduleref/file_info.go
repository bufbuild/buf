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

package bufmoduleref

import (
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
)

var _ FileInfo = &fileInfo{}

type fileInfo struct {
	path                         string
	externalPath                 string
	isImport                     bool
	moduleIdentityOptionalCommit ModuleIdentityOptionalCommit
}

func newFileInfo(
	path string,
	externalPath string,
	isImport bool,
	moduleIdentity ModuleIdentity,
	commit string,
) (*fileInfo, error) {
	if err := protodescriptor.ValidateProtoPath("root relative file path", path); err != nil {
		return nil, err
	}
	if externalPath == "" {
		externalPath = path
	}
	if moduleIdentity != nil {
		moduleIdentityOptionalCommit, err := NewModuleIdentityOptionalCommit(
			moduleIdentity.Remote(),
			moduleIdentity.Owner(),
			moduleIdentity.Repository(),
			commit,
		)
		if err != nil {
			return nil, err
		}
		return newFileInfoNoValidate(
			path,
			externalPath,
			isImport,
			moduleIdentityOptionalCommit,
		), nil
	}
	// NEED TO DO EXPLICIT NIL, OTHERWISE WE GET THE NIL INTERFACE ISSUE
	return newFileInfoNoValidate(
		path,
		externalPath,
		isImport,
		nil,
	), nil
}

func newFileInfoNoValidate(
	path string,
	externalPath string,
	isImport bool,
	moduleIdentityOptionalCommit ModuleIdentityOptionalCommit,
) *fileInfo {
	return &fileInfo{
		path:                         path,
		externalPath:                 externalPath,
		isImport:                     isImport,
		moduleIdentityOptionalCommit: moduleIdentityOptionalCommit,
	}
}

func (f *fileInfo) Path() string {
	return f.path
}

func (f *fileInfo) ExternalPath() string {
	return f.externalPath
}

func (f *fileInfo) IsImport() bool {
	return f.isImport
}

func (f *fileInfo) ModuleIdentityOptionalCommit() ModuleIdentityOptionalCommit {
	return f.moduleIdentityOptionalCommit
}

func (f *fileInfo) WithIsImport(isImport bool) FileInfo {
	return newFileInfoNoValidate(
		f.path,
		f.externalPath,
		isImport,
		f.moduleIdentityOptionalCommit,
	)
}

func (*fileInfo) isFileInfo() {}
