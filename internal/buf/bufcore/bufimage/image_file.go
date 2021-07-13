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

package bufimage

import (
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ ImageFile = &imageFile{}

type imageFile struct {
	bufmodule.FileInfo

	fileDescriptorProto *descriptorpb.FileDescriptorProto

	isSyntaxUnspecified           bool
	storedUnusedDependencyIndexes []int32
}

func newImageFile(
	fileDescriptor protodescriptor.FileDescriptor,
	moduleIdentity bufmodule.ModuleIdentity,
	commit string,
	externalPath string,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) (*imageFile, error) {
	if err := protodescriptor.ValidateFileDescriptor(fileDescriptor); err != nil {
		return nil, err
	}
	coreFileInfo, err := bufcore.NewFileInfo(
		fileDescriptor.GetName(),
		externalPath,
		isImport,
	)
	if err != nil {
		return nil, err
	}
	// just to normalize in other places between empty and unset
	if len(unusedDependencyIndexes) == 0 {
		unusedDependencyIndexes = nil
	}
	return &imageFile{
		FileInfo: bufmodule.NewFileInfo(coreFileInfo, moduleIdentity, commit),
		// protodescriptor.FileDescriptorProtoForFileDescriptor is a no-op if fileDescriptor
		// is already a *descriptorpb.FileDescriptorProto
		fileDescriptorProto:           protodescriptor.FileDescriptorProtoForFileDescriptor(fileDescriptor),
		isSyntaxUnspecified:           isSyntaxUnspecified,
		storedUnusedDependencyIndexes: unusedDependencyIndexes,
	}, nil
}

func (f *imageFile) Proto() *descriptorpb.FileDescriptorProto {
	return f.fileDescriptorProto
}

func (f *imageFile) FileDescriptor() protodescriptor.FileDescriptor {
	return f.fileDescriptorProto
}

func (f *imageFile) IsSyntaxUnspecified() bool {
	return f.isSyntaxUnspecified
}

func (f *imageFile) UnusedDependencyIndexes() []int32 {
	return f.storedUnusedDependencyIndexes
}

func (f *imageFile) withIsImport(isImport bool) ImageFile {
	return &imageFile{
		FileInfo: bufmodule.NewFileInfo(
			f.FileInfo.WithIsImport(isImport),
			f.FileInfo.ModuleIdentity(),
			f.FileInfo.Commit(),
		),
		fileDescriptorProto: f.fileDescriptorProto,
		isSyntaxUnspecified: f.isSyntaxUnspecified,
	}
}

func (*imageFile) isImageFile() {}
