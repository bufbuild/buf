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

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule/bufmoduletest"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	var options []bufimagebuild.BuildOption
	if !includeSourceInfo {
		options = []bufimagebuild.BuildOption{bufimagebuild.WithExcludeSourceCodeInfo()}
	}
	image, annotations, err := bufimagebuild.NewBuilder(
		zap.NewNop(),
	).Build(
		context.Background(),
		moduleSet,
		options...,
	)
	require.NoError(t, err)
	require.Empty(t, annotations)
	return image
}

type testDisable struct {
	path           string
	moduleFullName string
	fieldName      string
	fileOption     bufconfig.FileOption
	fieldOption    bufconfig.FieldOption
}

type testOverride struct {
	path           string
	moduleFullName string
	fieldName      string
	fileOption     bufconfig.FileOption
	fieldOption    bufconfig.FieldOption
	value          interface{}
}
