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

package bufimagetesting

import (
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// NewFileRef returns a new FileRef for testing.
func NewFileRef(
	t *testing.T,
	rootRelFilePath string,
	rootDirPath string,
	externalPathResolver bufpath.ExternalPathResolver,
) bufimage.FileRef {
	fileRef, err := bufimage.NewFileRef(
		rootRelFilePath,
		rootDirPath,
		externalPathResolver,
	)
	require.NoError(t, err)
	return fileRef
}

// NewDirectFileRef returns a new FileRef for testing.
func NewDirectFileRef(
	t *testing.T,
	rootRelFilePath string,
	rootDirPath string,
	externalFilePath string,
) bufimage.FileRef {
	fileRef, err := bufimage.NewDirectFileRef(
		rootRelFilePath,
		rootDirPath,
		externalFilePath,
	)
	require.NoError(t, err)
	return fileRef
}

// NewFile returns a new File for testing.
func NewFile(
	t *testing.T,
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	rootDirPath string,
	externalPathResolver bufpath.ExternalPathResolver,
	isImport bool,
) bufimage.File {
	file, err := bufimage.NewFile(
		fileDescriptorProto,
		rootDirPath,
		externalPathResolver,
		isImport,
	)
	require.NoError(t, err)
	return file
}

// NewDirectFile returns a new File for testing.
func NewDirectFile(
	t *testing.T,
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	rootDirPath string,
	externalFilePath string,
	isImport bool,
) bufimage.File {
	file, err := bufimage.NewDirectFile(
		fileDescriptorProto,
		rootDirPath,
		externalFilePath,
		isImport,
	)
	require.NoError(t, err)
	return file
}

// NewFileDescriptorProto returns a new FileDescriptorProto for testing.
func NewFileDescriptorProto(
	t *testing.T,
	rootRelFilePath string,
	importRootRelFilePaths ...string,
) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String(rootRelFilePath),
		Dependency: importRootRelFilePaths,
	}
}

// AssertFileRefsEqual asserts the expected FileRefs equal the acutal FileRefs.
func AssertFileRefsEqual(t *testing.T, expected []bufimage.FileRef, actual []bufimage.FileRef) {
	assert.Equal(t, expected, actual)
}

// AssertFilesEqual assets the expected ComparableFiles equal the acutal Files.
func AssertFilesEqual(t *testing.T, expected []bufimage.File, actual []bufimage.File) {
	expectedNormalizedFiles := normalizeFiles(t, expected)
	actualNormalizedFiles := normalizeFiles(t, actual)
	assert.Equal(t, expectedNormalizedFiles, actualNormalizedFiles)
}

func normalizeFiles(t *testing.T, files []bufimage.File) []bufimage.File {
	normalizedFiles := make([]bufimage.File, len(files))
	for i, file := range files {
		normalizedFiles[i] = NewDirectFile(
			t,
			NewFileDescriptorProto(
				t,
				file.Proto().GetName(),
				file.Proto().GetDependency()...,
			),
			file.RootDirPath(),
			file.ExternalFilePath(),
			file.IsImport(),
		)
	}
	return normalizedFiles
}
