// Copyright 2020-2025 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// NewImageFile returns a new ImageFile for testing.
//
// TODO FUTURE: moduleFullName and commit should be options.
func NewImageFile(
	t testing.TB,
	fileDescriptor protodescriptor.FileDescriptor,
	moduleFullName bufparse.FullName,
	commit uuid.UUID,
	externalPath string,
	localPath string,
	isImport bool,
	isSyntaxUnspecified bool,
	unusedDependencyIndexes []int32,
) bufimage.ImageFile {
	imageFile, err := bufimage.NewImageFile(
		fileDescriptor,
		moduleFullName,
		commit,
		externalPath,
		localPath,
		isImport,
		isSyntaxUnspecified,
		unusedDependencyIndexes,
	)
	require.NoError(t, err)
	return imageFile
}

// NewProtoImageFile returns a new *imagev1.ImageFile for testing.
//
// This is also a protodescriptor.FileDescriptor.
func NewProtoImageFile(
	t testing.TB,
	path string,
	importPaths ...string,
) *imagev1.ImageFile {
	return imagev1.ImageFile_builder{
		Name:       proto.String(path),
		Dependency: importPaths,
		BufExtension: imagev1.ImageFileExtension_builder{
			IsImport:            proto.Bool(false),
			IsSyntaxUnspecified: proto.Bool(false),
		}.Build(),
	}.Build()
}

// NewProtoImageFileIsImport returns a new *imagev1.ImageFile for testing that is an import.
//
// This is also a protodescriptor.FileDescriptor.
func NewProtoImageFileIsImport(
	t testing.TB,
	path string,
	importPaths ...string,
) *imagev1.ImageFile {
	return imagev1.ImageFile_builder{
		Name:       proto.String(path),
		Dependency: importPaths,
		BufExtension: imagev1.ImageFileExtension_builder{
			IsImport:            proto.Bool(true),
			IsSyntaxUnspecified: proto.Bool(false),
		}.Build(),
	}.Build()
}

// AssertImageFilesEqual asserts the expected ImageFiles equal the actual ImageFiles.
func AssertImageFilesEqual(t testing.TB, expected []bufimage.ImageFile, actual []bufimage.ImageFile) {
	expectedNormalizedImageFiles := normalizeImageFiles(t, expected)
	actualNormalizedImageFiles := normalizeImageFiles(t, actual)
	assert.Equal(t, expectedNormalizedImageFiles, actualNormalizedImageFiles)
}

func normalizeImageFiles(t testing.TB, imageFiles []bufimage.ImageFile) []bufimage.ImageFile {
	normalizedImageFiles := make([]bufimage.ImageFile, len(imageFiles))
	for i, imageFile := range imageFiles {
		normalizedImageFiles[i] = NewImageFile(
			t,
			NewProtoImageFile(
				t,
				imageFile.FileDescriptorProto().GetName(),
				imageFile.FileDescriptorProto().GetDependency()...,
			),
			imageFile.FullName(),
			imageFile.CommitID(),
			imageFile.ExternalPath(),
			imageFile.LocalPath(),
			imageFile.IsImport(),
			imageFile.IsSyntaxUnspecified(),
			imageFile.UnusedDependencyIndexes(),
		)
	}
	return normalizedImageFiles
}
