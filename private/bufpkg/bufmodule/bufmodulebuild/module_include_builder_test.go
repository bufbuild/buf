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

package bufmodulebuild

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIncludeGetFileInfos1(t *testing.T) {
	t.Parallel()
	testIncludeGetFileInfos(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		bufmoduletesting.NewFileInfo(t, "a/1.proto", "testdata/1/proto/a/1.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/2.proto", "testdata/1/proto/a/2.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/3.proto", "testdata/1/proto/a/3.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/c/1.proto", "testdata/1/proto/a/c/1.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/c/2.proto", "testdata/1/proto/a/c/2.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "a/c/3.proto", "testdata/1/proto/a/c/3.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/1.proto", "testdata/1/proto/b/1.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/2.proto", "testdata/1/proto/b/2.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "b/3.proto", "testdata/1/proto/b/3.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/1.proto", "testdata/1/proto/d/1.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/2.proto", "testdata/1/proto/d/2.proto", nil, ""),
		bufmoduletesting.NewFileInfo(t, "d/3.proto", "testdata/1/proto/d/3.proto", nil, ""),
	)
}

func TestIncludeGetFileInfos2(t *testing.T) {
	t.Parallel()
	testIncludeGetFileInfos(
		t,
		"testdata/3",
		[]string{
			".",
		},
	)
}

func TestIncludeGetFileInfosForExternalPathsError1(t *testing.T) {
	t.Parallel()
	testIncludeGetFileInfosForExternalPathsError(
		t,
		"testdata/2",
		[]string{
			"a",
			"b",
		},
		[]string{
			"testdata/2/a/1.proto",
			"testdata/2/a/2.proto",
			"testdata/2/a/3.proto",
			"testdata/2/b/1.proto",
			"testdata/2/b/4.proto",
		},
	)
}

func testIncludeGetFileInfos(
	t *testing.T,
	relDir string,
	relRoots []string,
	expectedFileInfos ...bufmoduleref.FileInfo,
) {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	for _, isAbs := range []bool{true, false} {
		isAbs := isAbs
		expectedFileInfos := expectedFileInfos
		if isAbs {
			expectedFileInfos = fileInfosToAbs(t, expectedFileInfos)
		}
		t.Run(fmt.Sprintf("abs=%v", isAbs), func(t *testing.T) {
			t.Parallel()
			includeDirPaths := testIncludeDirPaths(t, relDir, relRoots, isAbs)
			module, err := NewModuleIncludeBuilder(zap.NewNop(), storageosProvider).BuildForIncludes(
				context.Background(),
				includeDirPaths,
			)
			require.NoError(t, err)
			fileInfos, err := module.SourceFileInfos(context.Background())
			assert.NoError(t, err)
			assert.Equal(
				t,
				expectedFileInfos,
				fileInfos,
			)
			if len(expectedFileInfos) > 1 {
				expectedFileInfos = expectedFileInfos[:len(expectedFileInfos)-1]
				filePaths := make([]string, len(expectedFileInfos))
				for i := 0; i < len(expectedFileInfos); i++ {
					filePaths[i] = expectedFileInfos[i].ExternalPath()
				}
				module, err := NewModuleIncludeBuilder(zap.NewNop(), storageosProvider).BuildForIncludes(
					context.Background(),
					includeDirPaths,
					WithPaths(filePaths),
				)
				require.NoError(t, err)
				fileInfos, err := module.TargetFileInfos(context.Background())
				assert.NoError(t, err)
				assert.Equal(
					t,
					expectedFileInfos,
					fileInfos,
				)
			}
		})
	}
}

func testIncludeGetFileInfosForExternalPathsError(
	t *testing.T,
	relDir string,
	relRoots []string,
	externalPaths []string,
) {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	for _, isAbs := range []bool{true, false} {
		isAbs := isAbs
		t.Run(fmt.Sprintf("abs=%v", isAbs), func(t *testing.T) {
			t.Parallel()
			includeDirPaths := testIncludeDirPaths(t, relDir, relRoots, isAbs)
			_, err := NewModuleIncludeBuilder(zap.NewNop(), storageosProvider).BuildForIncludes(
				context.Background(),
				includeDirPaths,
				WithPaths(externalPaths),
			)
			assert.Error(t, err)
		})
	}
}

func testIncludeDirPaths(
	t *testing.T,
	relDir string,
	relRoots []string,
	isAbs bool,
) []string {
	includeDirPaths := make([]string, len(relRoots))
	for i, relRoot := range relRoots {
		includeDirPaths[i] = normalpath.Unnormalize(normalpath.Join(relDir, relRoot))
	}
	if isAbs {
		for i, includeDirPath := range includeDirPaths {
			absIncludeDirPath, err := filepath.Abs(includeDirPath)
			require.NoError(t, err)
			includeDirPaths[i] = absIncludeDirPath
		}
	}
	return includeDirPaths
}

// fileInfosToAbs converts the external paths to absolute.
func fileInfosToAbs(t *testing.T, fileInfos []bufmoduleref.FileInfo) []bufmoduleref.FileInfo {
	if fileInfos == nil {
		return nil
	}
	newFileInfos := make([]bufmoduleref.FileInfo, len(fileInfos))
	for i, fileInfo := range fileInfos {
		absExternalPath, err := normalpath.NormalizeAndAbsolute(fileInfo.ExternalPath())
		require.NoError(t, err)
		newFileInfo := bufmoduletesting.NewFileInfo(
			t,
			fileInfo.Path(),
			absExternalPath,
			nil,
			"",
		)
		newFileInfos[i] = newFileInfo
	}
	return newFileInfos
}
