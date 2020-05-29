// Copyright 2020 Buf Technologies Inc.
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
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ File = &file{}

type file struct {
	*fileRef

	fileDescriptorProto *descriptorpb.FileDescriptorProto
	isImport            bool
}

func newFile(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	rootDirPath string,
	externalPathResolver bufpath.ExternalPathResolver,
	isImport bool,
) (*file, error) {
	if err := validateFileDescriptorProto(fileDescriptorProto); err != nil {
		return nil, err
	}
	fileRef, err := newFileRef(
		fileDescriptorProto.GetName(),
		rootDirPath,
		externalPathResolver,
	)
	if err != nil {
		return nil, err
	}
	return &file{
		fileRef:             fileRef,
		fileDescriptorProto: fileDescriptorProto,
		isImport:            isImport,
	}, nil
}

func newDirectFile(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	rootDirPath string,
	externalFilePath string,
	isImport bool,
) (*file, error) {
	if err := validateFileDescriptorProto(fileDescriptorProto); err != nil {
		return nil, err
	}
	fileRef, err := newDirectFileRef(
		fileDescriptorProto.GetName(),
		rootDirPath,
		externalFilePath,
	)
	if err != nil {
		return nil, err
	}
	return &file{
		fileRef:             fileRef,
		fileDescriptorProto: fileDescriptorProto,
		isImport:            isImport,
	}, nil
}

func newFileNoValidate(
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	rootDirPath string,
	externalFilePath string,
	isImport bool,
) *file {
	fileRef := newFileRefNoValidate(
		fileDescriptorProto.GetName(),
		rootDirPath,
		externalFilePath,
	)
	return &file{
		fileRef:             fileRef,
		fileDescriptorProto: fileDescriptorProto,
		isImport:            isImport,
	}
}

func (f *file) ImportRootRelFilePaths() []string {
	return f.fileDescriptorProto.GetDependency()
}

func (f *file) Proto() *descriptorpb.FileDescriptorProto {
	return f.fileDescriptorProto
}

func (f *file) IsImport() bool {
	return f.isImport
}

func (*file) isFile() {}
