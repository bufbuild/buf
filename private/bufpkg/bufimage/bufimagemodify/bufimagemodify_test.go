// Copyright 2020-2022 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testImportPathPrefix = "github.com/foo/bar/private/gen/proto/go"
	testRemote           = "modulerepo.internal"
	testRepositoryOwner  = "testowner"
	testRepositoryName   = "testrepository"
)

func assertFileOptionSourceCodeInfoEmpty(t *testing.T, image bufimage.Image, fileOptionPath []int32, includeSourceInfo bool) {
	for _, imageFile := range image.Files() {
		descriptor := imageFile.Proto()

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
		descriptor := imageFile.Proto()

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
	moduleFileSet := testGetModuleFileSet(t, dirPath)
	var options []bufimagebuild.BuildOption
	if !includeSourceInfo {
		options = []bufimagebuild.BuildOption{bufimagebuild.WithExcludeSourceCodeInfo()}
	}
	image, annotations, err := bufimagebuild.NewBuilder(zap.NewNop()).Build(
		context.Background(),
		moduleFileSet,
		options...,
	)
	require.NoError(t, err)
	require.Empty(t, annotations)
	return image
}

func testGetModuleFileSet(t *testing.T, dirPath string) bufmodule.ModuleFileSet {
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
	moduleFileSet, err := bufmodulebuild.NewModuleFileSetBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleReader(),
	).Build(
		context.Background(),
		module,
	)
	require.NoError(t, err)
	return moduleFileSet
}
