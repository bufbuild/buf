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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ ImageFile = &imageFile{}

type imageFile struct {
	fileDescriptorProto     *descriptorpb.FileDescriptorProto
	moduleFullName          bufmodule.ModuleFullName
	commitID                uuid.UUID
	externalPath            string
	localPath               string
	isImport                bool
	isSyntaxUnspecified     bool
	unusedDependencyIndexes []int32
}

func newImageFile(
	fileDescriptor protodescriptor.FileDescriptor,
	moduleFullName bufmodule.ModuleFullName,
	commitID uuid.UUID,
	externalPath string,
	localPath string,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) (*imageFile, error) {
	if err := protodescriptor.ValidateFileDescriptor(fileDescriptor); err != nil {
		return nil, err
	}
	return newImageFileNoValidate(
		fileDescriptor,
		moduleFullName,
		commitID,
		externalPath,
		localPath,
		isImport,
		isSyntaxUnspecified,
		unusedDependencyIndexes,
	), nil
}

func newImageFileNoValidate(
	fileDescriptor protodescriptor.FileDescriptor,
	moduleFullName bufmodule.ModuleFullName,
	commitID uuid.UUID,
	externalPath string,
	localPath string,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) *imageFile {
	// just to normalize in other places between empty and unset
	if len(unusedDependencyIndexes) == 0 {
		unusedDependencyIndexes = nil
	}
	return &imageFile{
		// protodescriptor.FileDescriptorProtoForFileDescriptor is a no-op if fileDescriptor
		// is already a *descriptorpb.FileDescriptorProto
		fileDescriptorProto:     protodescriptor.FileDescriptorProtoForFileDescriptor(fileDescriptor),
		moduleFullName:          moduleFullName,
		commitID:                commitID,
		externalPath:            externalPath,
		localPath:               localPath,
		isImport:                isImport,
		isSyntaxUnspecified:     isSyntaxUnspecified,
		unusedDependencyIndexes: unusedDependencyIndexes,
	}
}

func (f *imageFile) Path() string {
	return f.fileDescriptorProto.GetName()
}

func (f *imageFile) ExternalPath() string {
	if f.externalPath == "" {
		return f.Path()
	}
	return f.externalPath
}

func (f *imageFile) LocalPath() string {
	return f.localPath
}

func (f *imageFile) ModuleFullName() bufmodule.ModuleFullName {
	return f.moduleFullName
}

func (f *imageFile) CommitID() uuid.UUID {
	return f.commitID
}

func (f *imageFile) Imports() ([]string, error) {
	return slicesext.Copy(f.fileDescriptorProto.GetDependency()), nil
}

func (f *imageFile) IsImport() bool {
	return f.isImport
}

func (f *imageFile) FileDescriptorProto() *descriptorpb.FileDescriptorProto {
	return f.fileDescriptorProto
}

func (f *imageFile) IsSyntaxUnspecified() bool {
	return f.isSyntaxUnspecified
}

func (f *imageFile) UnusedDependencyIndexes() []int32 {
	return f.unusedDependencyIndexes
}

func (*imageFile) isImageFileInfo() {}
func (*imageFile) isImageFile()     {}
