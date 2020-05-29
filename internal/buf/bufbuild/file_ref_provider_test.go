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

package bufbuild

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufimage/bufimagetesting"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetFileRefs1(t *testing.T) {
	testGetFileRefs(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/b",
		},
		bufimagetesting.NewDirectFileRef(t, "a/1.proto", "proto", "testdata/1/proto/a/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/2.proto", "proto", "testdata/1/proto/a/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/3.proto", "proto", "testdata/1/proto/a/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/c/1.proto", "proto", "testdata/1/proto/a/c/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/c/2.proto", "proto", "testdata/1/proto/a/c/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/c/3.proto", "proto", "testdata/1/proto/a/c/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/1.proto", "proto", "testdata/1/proto/d/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/2.proto", "proto", "testdata/1/proto/d/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/3.proto", "proto", "testdata/1/proto/d/3.proto"),
	)
}

func TestGetFileRefs2(t *testing.T) {
	testGetFileRefs(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/b",
		},
		bufimagetesting.NewDirectFileRef(t, "a/1.proto", "proto", "testdata/1/proto/a/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/2.proto", "proto", "testdata/1/proto/a/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/3.proto", "proto", "testdata/1/proto/a/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/c/1.proto", "proto", "testdata/1/proto/a/c/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/c/2.proto", "proto", "testdata/1/proto/a/c/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/c/3.proto", "proto", "testdata/1/proto/a/c/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/1.proto", "proto", "testdata/1/proto/d/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/2.proto", "proto", "testdata/1/proto/d/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/3.proto", "proto", "testdata/1/proto/d/3.proto"),
	)
}

func TestGetFileRefs3(t *testing.T) {
	testGetFileRefs(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a",
		},
		bufimagetesting.NewDirectFileRef(t, "b/1.proto", "proto", "testdata/1/proto/b/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/2.proto", "proto", "testdata/1/proto/b/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/3.proto", "proto", "testdata/1/proto/b/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/1.proto", "proto", "testdata/1/proto/d/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/2.proto", "proto", "testdata/1/proto/d/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/3.proto", "proto", "testdata/1/proto/d/3.proto"),
	)
}

func TestGetFileRefs4(t *testing.T) {
	testGetFileRefs(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
		},
		bufimagetesting.NewDirectFileRef(t, "a/1.proto", "proto", "testdata/1/proto/a/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/2.proto", "proto", "testdata/1/proto/a/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/3.proto", "proto", "testdata/1/proto/a/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/1.proto", "proto", "testdata/1/proto/b/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/2.proto", "proto", "testdata/1/proto/b/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/3.proto", "proto", "testdata/1/proto/b/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/1.proto", "proto", "testdata/1/proto/d/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/2.proto", "proto", "testdata/1/proto/d/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "d/3.proto", "proto", "testdata/1/proto/d/3.proto"),
	)
}

func TestGetFileRefs5(t *testing.T) {
	testGetFileRefs(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
			"proto/d",
		},
		bufimagetesting.NewDirectFileRef(t, "a/1.proto", "proto", "testdata/1/proto/a/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/2.proto", "proto", "testdata/1/proto/a/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "a/3.proto", "proto", "testdata/1/proto/a/3.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/1.proto", "proto", "testdata/1/proto/b/1.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/2.proto", "proto", "testdata/1/proto/b/2.proto"),
		bufimagetesting.NewDirectFileRef(t, "b/3.proto", "proto", "testdata/1/proto/b/3.proto"),
	)
}

func TestGetAllFileRefsError1(t *testing.T) {
	testGetAllFileRefsError(
		t,
		"testdata/2",
		[]string{
			"a",
			"b",
		},
		[]string{},
	)
}

func TestGetAllFileRefsError2(t *testing.T) {
	t.Parallel()
	testGetAllFileRefsError(
		t,
		"testdata/1",
		[]string{
			"proto",
		},
		[]string{
			"proto/a/c",
			// error since not a directory
			"proto/d/1.proto",
		},
	)
}

func TestGetFileRefsForExternalFilePaths1(t *testing.T) {
	testGetFileRefsForExternalFilePathsError(
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

func testGetFileRefs(
	t *testing.T,
	relDir string,
	relRoots []string,
	relExcludes []string,
	expectedFileRefs ...bufimage.FileRef,
) {
	t.Parallel()
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(relDir)
	require.NoError(t, err)
	pathResolver := bufpath.NewDirPathResolver(relDir)

	fileRefs, err := NewFileRefProvider(zap.NewNop()).GetAllFileRefs(
		context.Background(),
		readWriteBucketCloser,
		pathResolver,
		relRoots,
		relExcludes,
	)
	assert.NoError(t, err)
	bufimagetesting.AssertFileRefsEqual(
		t,
		expectedFileRefs,
		fileRefs,
	)
	if len(expectedFileRefs) > 1 {
		expectedFileRefs = expectedFileRefs[:len(expectedFileRefs)-1]
		externalFilePaths := make([]string, len(expectedFileRefs))
		for i, expectedFileRef := range expectedFileRefs {
			externalFilePaths[i] = expectedFileRef.ExternalFilePath()
		}
		fileRefs, err := NewFileRefProvider(zap.NewNop()).GetFileRefsForExternalFilePaths(
			context.Background(),
			readWriteBucketCloser,
			pathResolver,
			relRoots,
			externalFilePaths,
		)
		assert.NoError(t, err)
		bufimagetesting.AssertFileRefsEqual(
			t,
			expectedFileRefs,
			fileRefs,
		)
	}
	assert.NoError(t, readWriteBucketCloser.Close())
}

func testGetAllFileRefsError(
	t *testing.T,
	relDir string,
	relRoots []string,
	relExcludes []string,
) {
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(relDir)
	require.NoError(t, err)

	pathResolver := bufpath.NewDirPathResolver(relDir)
	_, err = NewFileRefProvider(zap.NewNop()).GetAllFileRefs(
		context.Background(),
		readWriteBucketCloser,
		pathResolver,
		relRoots,
		relExcludes,
	)
	assert.Error(t, err)
	assert.NoError(t, readWriteBucketCloser.Close())
}

func testGetFileRefsForExternalFilePathsError(
	t *testing.T,
	relDir string,
	relRoots []string,
	externalFilePaths []string,
) {
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(relDir)
	require.NoError(t, err)
	pathResolver := bufpath.NewDirPathResolver(relDir)
	_, err = NewFileRefProvider(zap.NewNop()).GetFileRefsForExternalFilePaths(
		context.Background(),
		readWriteBucketCloser,
		pathResolver,
		relRoots,
		externalFilePaths,
	)
	assert.Error(t, err)
	assert.NoError(t, readWriteBucketCloser.Close())
}
