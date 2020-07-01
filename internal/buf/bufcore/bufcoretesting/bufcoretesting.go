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

package bufcoretesting

import (
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// NewFileInfo returns a new FileInfo for testing.
func NewFileInfo(
	t *testing.T,
	path string,
	externalPath string,
	isImport bool,
) bufcore.FileInfo {
	fileInfo, err := bufcore.NewFileInfo(
		path,
		externalPath,
		isImport,
	)
	require.NoError(t, err)
	return fileInfo
}

// NewImageFile returns a new ImageFile for testing.
func NewImageFile(
	t *testing.T,
	fileDescriptorProto *descriptorpb.FileDescriptorProto,
	externalPath string,
	isImport bool,
) bufcore.ImageFile {
	imageFile, err := bufcore.NewImageFile(
		fileDescriptorProto,
		externalPath,
		isImport,
	)
	require.NoError(t, err)
	return imageFile
}

// NewFileDescriptorProto returns a new FileDescriptorProto for testing.
func NewFileDescriptorProto(
	t *testing.T,
	path string,
	importPaths ...string,
) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String(path),
		Dependency: importPaths,
	}
}

// AssertFileInfosEqual asserts the expected FileInfos equal the actual FileInfos.
func AssertFileInfosEqual(t *testing.T, expected []bufcore.FileInfo, actual []bufcore.FileInfo) {
	assert.Equal(t, expected, actual)
}

// AssertImageFilesEqual asserts the expected ImageFiles equal the actual ImageFiles.
func AssertImageFilesEqual(t *testing.T, expected []bufcore.ImageFile, actual []bufcore.ImageFile) {
	expectedNormalizedImageFiles := normalizeImageFiles(t, expected)
	actualNormalizedImageFiles := normalizeImageFiles(t, actual)
	assert.Equal(t, expectedNormalizedImageFiles, actualNormalizedImageFiles)
}

func normalizeImageFiles(t *testing.T, imageFiles []bufcore.ImageFile) []bufcore.ImageFile {
	normalizedImageFiles := make([]bufcore.ImageFile, len(imageFiles))
	for i, imageFile := range imageFiles {
		normalizedImageFiles[i] = NewImageFile(
			t,
			NewFileDescriptorProto(
				t,
				imageFile.Proto().GetName(),
				imageFile.Proto().GetDependency()...,
			),
			imageFile.ExternalPath(),
			imageFile.IsImport(),
		)
	}
	return normalizedImageFiles
}
