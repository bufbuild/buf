// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufimagemodify

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testImportPathPrefix = "github.com/foo/bar/private/gen/proto/go"
	testRemote           = "buf.test"
	testRepositoryOwner  = "foo"
	testRepositoryName   = "bar"
)

func assertFileOptionSourceCodeInfoEmpty(t *testing.T, image bufimage.Image, fileOptionPath []int32, includeSourceInfo bool) {
	for _, imageFile := range image.Files() {
		descriptor := imageFile.FileDescriptorProto()

		if !includeSourceInfo {
			assert.Empty(t, descriptor.SourceCodeInfo)
			continue
		}

		var hasFileOption bool
		for _, location := range descriptor.SourceCodeInfo.Location {
			if len(location.Path) > 0 && int32SliceIsEqual(location.Path, fileOptionPath) {
				hasFileOption = true
				break
			}
		}
		assert.False(t, hasFileOption)
	}
}

func assertFileOptionSourceCodeInfoNotEmpty(t *testing.T, image bufimage.Image, fileOptionPath []int32) {
	for _, imageFile := range image.Files() {
		descriptor := imageFile.FileDescriptorProto()

		var hasFileOption bool
		for _, location := range descriptor.SourceCodeInfo.Location {
			if len(location.Path) > 0 && int32SliceIsEqual(location.Path, fileOptionPath) {
				hasFileOption = true
				break
			}
		}
		assert.True(t, hasFileOption)
	}
}

func testGetImage(t *testing.T, dirPath string, includeSourceInfo bool) bufimage.Image {
	moduleSet, err := bufmoduletest.NewModuleSet(
		bufmoduletest.ModuleData{
			Name:    testRemote + "/" + testRepositoryOwner + "/" + testRepositoryName,
			DirPath: dirPath,
		},
	)
	require.NoError(t, err)
	var options []bufimage.BuildImageOption
	if !includeSourceInfo {
		options = []bufimage.BuildImageOption{bufimage.WithExcludeSourceCodeInfo()}
	}
	image, annotations, err := bufimage.BuildImage(
		context.Background(),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		options...,
	)
	require.NoError(t, err)
	require.Empty(t, annotations)
	return image
}

func testGetImageFromDirs(
	t *testing.T,
	dirPathToModuleFullName map[string]string,
	includeSourceInfo bool,
) bufimage.Image {
	moduleDatas := make([]bufmoduletest.ModuleData, 0, len(dirPathToModuleFullName))
	for dirPath, moduleFullName := range dirPathToModuleFullName {
		moduleDatas = append(
			moduleDatas,
			bufmoduletest.ModuleData{
				Name:    moduleFullName,
				DirPath: dirPath,
			},
		)
	}
	moduleSet, err := bufmoduletest.NewModuleSet(moduleDatas...)
	require.NoError(t, err)
	var options []bufimage.BuildImageOption
	if !includeSourceInfo {
		options = []bufimage.BuildImageOption{bufimage.WithExcludeSourceCodeInfo()}
	}
	image, annotations, err := bufimage.BuildImage(
		context.Background(),
		bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet),
		options...,
	)
	require.NoError(t, err)
	require.Empty(t, annotations)
	return image
}
