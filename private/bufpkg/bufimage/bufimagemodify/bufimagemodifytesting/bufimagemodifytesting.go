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

package bufimagemodifytesting

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testRemote          = "modulerepo.internal"
	testRepositoryOwner = "testowner"
	testRepositoryName  = "testrepository"
)

// AssertFileOptionSourceCodeInfoEmpty asserts that a the source code info location
// for the given file path is not set for files in an image.
func AssertFileOptionSourceCodeInfoEmpty(
	t *testing.T,
	image bufimage.Image,
	fileOptionPath []int32,
	includeSourceInfo bool,
	assertOptions ...AssertSourceCodeInfoOption,
) {
	options := &assertSourceCodeInfoOptions{}
	for _, option := range assertOptions {
		option(options)
	}
	for _, imageFile := range image.Files() {
		descriptor := imageFile.Proto()

		if !includeSourceInfo {
			assert.Empty(t, descriptor.SourceCodeInfo)
			continue
		}

		if options.ignoreWKT && internal.IsWellKnownType(imageFile) {
			continue
		}

		var hasFileOption bool
		for _, location := range descriptor.SourceCodeInfo.Location {
			if len(location.Path) > 0 && internal.Int32SliceIsEqual(location.Path, fileOptionPath) {
				hasFileOption = true
				break
			}
		}
		assert.False(t, hasFileOption)
	}
}

// AssertSourceCodeInfoWithIgnoreWKT skips well known types.
func AssertSourceCodeInfoWithIgnoreWKT() AssertSourceCodeInfoOption {
	return func(options *assertSourceCodeInfoOptions) {
		options.ignoreWKT = true
	}
}

type AssertSourceCodeInfoOption func(*assertSourceCodeInfoOptions)

type assertSourceCodeInfoOptions struct {
	ignoreWKT bool
}

// AssertFileOptionSourceCodeInfoEmptyForFile asserts that the source code
// info for the given file option is not present for the given file, which must
// exist, and that all other files in the image has this path in source code info.
func AssertFileOptionSourceCodeInfoEmptyForFile(
	t *testing.T,
	imageFile bufimage.ImageFile,
	fileOptionPath []int32,
	includeSourceInfo bool,
) {
	descriptor := imageFile.Proto()

	if !includeSourceInfo {
		assert.Empty(t, descriptor.SourceCodeInfo)
		return
	}

	var hasFileOption bool
	for _, location := range descriptor.SourceCodeInfo.Location {
		if len(location.Path) > 0 && internal.Int32SliceIsEqual(location.Path, fileOptionPath) {
			hasFileOption = true
			break
		}
	}
	assert.False(t, hasFileOption)
}

// Asserts the source code info location for the given file option path is present.
func AssertFileOptionSourceCodeInfoNotEmpty(
	t *testing.T,
	image bufimage.Image,
	fileOptionPath []int32,
) {
	for _, imageFile := range image.Files() {
		descriptor := imageFile.Proto()

		var hasFileOption bool
		for _, location := range descriptor.SourceCodeInfo.Location {
			if len(location.Path) > 0 && internal.Int32SliceIsEqual(location.Path, fileOptionPath) {
				hasFileOption = true
				break
			}
		}
		assert.True(t, hasFileOption)
	}
}

// AssertFileOptionSourceCodeInfoNotEmptyForFile asserts that
// the source code info for the file option path is present for the file.
func AssertFileOptionSourceCodeInfoNotEmptyForFile(
	t *testing.T,
	imageFile bufimage.ImageFile,
	fileOptionPath []int32,
) {
	descriptor := imageFile.Proto()
	var hasFileOption bool
	for _, location := range descriptor.SourceCodeInfo.Location {
		if len(location.Path) > 0 && internal.Int32SliceIsEqual(location.Path, fileOptionPath) {
			hasFileOption = true
			break
		}
	}
	assert.True(t, hasFileOption)
}

// GetTestImage returns an image from a directory.
func GetTestImage(
	t *testing.T,
	dirPath string,
	includeSourceInfo bool,
) bufimage.Image {
	module := getTestModule(t, dirPath)
	var options []bufimagebuild.BuildOption
	if !includeSourceInfo {
		options = []bufimagebuild.BuildOption{bufimagebuild.WithExcludeSourceCodeInfo()}
	}
	image, annotations, err := bufimagebuild.NewBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleReader(),
	).Build(
		context.Background(),
		module,
		options...,
	)
	require.NoError(t, err)
	require.Empty(t, annotations)
	return image
}

// GetTestModuleIdentity returns a module identify for testing.
func GetTestModuleIdentity(t *testing.T) bufmoduleref.ModuleIdentity {
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)
	return moduleIdentity
}

func getTestModule(t *testing.T, dirPath string) bufmodule.Module {
	storageosProvider := storageos.NewProvider()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
	)
	require.NoError(t, err)
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(
		context.Background(),
		readWriteBucket,
		bufmodule.ModuleWithModuleIdentityAndCommit(moduleIdentity, bufmoduletesting.TestCommit),
	)
	require.NoError(t, err)
	return module
}
