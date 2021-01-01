// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufcore

import (
	"github.com/bufbuild/buf/internal/buf/bufcore/internal/bufcorevalidate"
	"github.com/bufbuild/buf/internal/pkg/storage"
)

var _ FileInfo = &fileInfo{}

type fileInfo struct {
	path         string
	externalPath string
	isImport     bool
}

func newFileInfo(
	path string,
	externalPath string,
	isImport bool,
) (*fileInfo, error) {
	if err := bufcorevalidate.ValidateFileInfoPath(path); err != nil {
		return nil, err
	}
	if externalPath == "" {
		externalPath = path
	}
	return newFileInfoNoValidate(
		path,
		externalPath,
		isImport,
	), nil
}

func newFileInfoForObjectInfo(
	objectInfo storage.ObjectInfo,
	isImport bool,
) *fileInfo {
	return newFileInfoNoValidate(
		objectInfo.Path(),
		objectInfo.ExternalPath(),
		isImport,
	)
}

func newFileInfoNoValidate(
	path string,
	externalPath string,
	isImport bool,
) *fileInfo {
	return &fileInfo{
		path:         path,
		externalPath: externalPath,
		isImport:     isImport,
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

func (f *fileInfo) WithIsImport(isImport bool) FileInfo {
	return newFileInfoNoValidate(
		f.path,
		f.externalPath,
		isImport,
	)
}

func (*fileInfo) isFileInfo() {}
