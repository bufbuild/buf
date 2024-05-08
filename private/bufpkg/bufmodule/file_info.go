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

package bufmodule

import (
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syncext"
)

// FileInfo is the file info for a Module file.
//
// It comprises the typical storage.ObjectInfo, along with a pointer back to the Module.
// This allows callers to figure out i.e. the ModuleFullName, Commit, as well as any other
// data it may need.
type FileInfo interface {
	storage.ObjectInfo

	// Module returns the Module that contains this file.
	Module() Module
	// FileType returns the FileType of the file.
	//
	// This denotes if the File is a .proto file, documentation file, or license file.
	FileType() FileType
	// IsTargetFile returns true if the File is targeted.
	//
	// Files are either targets or imports.
	// If Module.IsTarget() is false, this will always be false.
	//
	// If specific Files were not targeted but Module.IsTarget() is true, all Files in
	// the Module will have IsTargetFile() set to true.
	IsTargetFile() bool

	// ProtoFileImports returns the file's declared .proto imports, if any.
	//
	// Always returns empty if this file is not a .proto file.
	ProtoFileImports() ([]string, error)

	// protoFilePackage returns the file's declared Protobuf package, any.
	//
	// Always returns empty if this file is not a .proto file.
	//
	// Not exposing this function publicly yet as we don't have a use case.
	protoFilePackage() (string, error)

	isFileInfo()
}

// FileInfoPaths is a convenience function that returns the paths of the FileInfos.
func FileInfoPaths(fileInfos []FileInfo) []string {
	return slicesext.Map(fileInfos, func(fileInfo FileInfo) string { return fileInfo.Path() })
}

// *** PRIVATE ***

type fileInfo struct {
	storage.ObjectInfo

	module              Module
	fileType            FileType
	isTargetFile        bool
	getProtoFileImports func() ([]string, error)
	getProtoFilePackage func() (string, error)
}

func newFileInfo(
	objectInfo storage.ObjectInfo,
	module Module,
	fileType FileType,
	isTargetFile bool,
	getProtoFileImports func() ([]string, error),
	getProtoFilePackage func() (string, error),
) *fileInfo {
	return &fileInfo{
		ObjectInfo:          objectInfo,
		module:              module,
		fileType:            fileType,
		isTargetFile:        isTargetFile,
		getProtoFileImports: syncext.OnceValues(getProtoFileImports),
		getProtoFilePackage: syncext.OnceValues(getProtoFilePackage),
	}
}

func (f *fileInfo) Module() Module {
	return f.module
}

func (f *fileInfo) FileType() FileType {
	return f.fileType
}

func (f *fileInfo) IsTargetFile() bool {
	return f.isTargetFile
}

func (f *fileInfo) ProtoFileImports() ([]string, error) {
	return f.getProtoFileImports()
}

func (f *fileInfo) protoFilePackage() (string, error) {
	return f.getProtoFilePackage()
}

func (*fileInfo) isFileInfo() {}
