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
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ ImageFile = &imageFile{}

type imageFile struct {
	bufmoduleref.FileInfo

	fileDescriptorProto     *descriptorpb.FileDescriptorProto
	isImport                bool
	isSyntaxUnspecified     bool
	unusedDependencyIndexes []int32
}

func newImageFile(
	fileDescriptor protodescriptor.FileDescriptor,
	moduleIdentity bufmoduleref.ModuleIdentity,
	commit string,
	externalPath string,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) (*imageFile, error) {
	if err := protodescriptor.ValidateFileDescriptor(fileDescriptor); err != nil {
		return nil, err
	}
	fileInfo, err := bufmoduleref.NewFileInfo(
		fileDescriptor.GetName(),
		externalPath,
		moduleIdentity,
		commit,
	)
	if err != nil {
		return nil, err
	}
	return newImageFileNoValidate(
		fileDescriptor,
		fileInfo,
		isImport,
		isSyntaxUnspecified,
		unusedDependencyIndexes,
	), nil
}

func newImageFileNoValidate(
	fileDescriptor protodescriptor.FileDescriptor,
	fileInfo bufmoduleref.FileInfo,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) *imageFile {
	// just to normalize in other places between empty and unset
	if len(unusedDependencyIndexes) == 0 {
		unusedDependencyIndexes = nil
	}
	return &imageFile{
		FileInfo: fileInfo,
		// protodescriptor.FileDescriptorProtoForFileDescriptor is a no-op if fileDescriptor
		// is already a *descriptorpb.FileDescriptorProto
		fileDescriptorProto:     protodescriptor.FileDescriptorProtoForFileDescriptor(fileDescriptor),
		isImport:                isImport,
		isSyntaxUnspecified:     isSyntaxUnspecified,
		unusedDependencyIndexes: unusedDependencyIndexes,
	}
}

func (f *imageFile) FileDescriptorProto() *descriptorpb.FileDescriptorProto {
	return f.fileDescriptorProto
}

func (f *imageFile) IsImport() bool {
	return f.isImport
}

func (f *imageFile) IsSyntaxUnspecified() bool {
	return f.isSyntaxUnspecified
}

func (f *imageFile) UnusedDependencyIndexes() []int32 {
	return f.unusedDependencyIndexes
}

func (*imageFile) isImageFile() {}
