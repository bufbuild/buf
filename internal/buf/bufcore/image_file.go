// Copyright 2020 Buf Technologies, Inc.
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
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ ImageFile = &imageFile{}

type imageFile struct {
	*fileInfo

	fileDescriptorProto *descriptorpb.FileDescriptorProto
}

func newImageFile(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	externalPath string,
	isImport bool,
) (*imageFile, error) {
	if err := validateFileDescriptorProto(fileDescriptorProto); err != nil {
		return nil, err
	}
	fileInfo, err := newFileInfo(
		fileDescriptorProto.GetName(),
		externalPath,
		isImport,
	)
	if err != nil {
		return nil, err
	}
	return &imageFile{
		fileInfo:            fileInfo,
		fileDescriptorProto: fileDescriptorProto,
	}, nil
}

func newImageFileNoValidate(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	externalPath string,
	isImport bool,
) *imageFile {
	fileInfo := newFileInfoNoValidate(
		fileDescriptorProto.GetName(),
		externalPath,
		isImport,
	)
	return &imageFile{
		fileInfo:            fileInfo,
		fileDescriptorProto: fileDescriptorProto,
	}
}

func (f *imageFile) Proto() *descriptorpb.FileDescriptorProto {
	return f.fileDescriptorProto
}

func (f *imageFile) ImportPaths() []string {
	return f.fileDescriptorProto.GetDependency()
}

func (*imageFile) isImageFile() {}
