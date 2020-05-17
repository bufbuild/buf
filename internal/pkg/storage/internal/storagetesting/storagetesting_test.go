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

package storagetesting

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testProtoContent = `syntax = "proto3";

package foo;
`
	testTxtContent = `foo
`
)

func TestBasic1(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/b/2.txt":   "",
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/ab/2.txt":    "",
			"one/a/1.proto":   "",
			"one/a/1.txt":     testTxtContent,
			"one/a/bar.yaml":  "",
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
	)
}

func TestBasic2(t *testing.T) {
	testBasic(
		t,
		"testdata",
		".",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/b/2.txt":   "",
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/ab/2.txt":    "",
			"one/a/1.proto":   "",
			"one/a/1.txt":     testTxtContent,
			"one/a/bar.yaml":  "",
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
	)
}

func TestBasic3(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"./",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/b/2.txt":   "",
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/ab/2.txt":    "",
			"one/a/1.proto":   "",
			"one/a/bar.yaml":  "",
			"one/a/1.txt":     testTxtContent,
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
	)
}

func TestBasic4(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/ab/1.proto":  testProtoContent,
			"one/ab/2.proto":  testProtoContent,
			"one/a/1.proto":   "",
			"one/c/1.proto":   testProtoContent,
			"one/1.proto":     testProtoContent,
			"one/foo.yaml":    "",
		},
		normalpath.WithExt(".proto"),
		normalpath.WithExactPath("one/foo.yaml"),
	)
}

func TestBasic5(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"one/a",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/1.proto":   "",
		},
		normalpath.WithExt(".proto"),
		normalpath.WithExactPath("foo.yaml"),
	)
}

func TestBasic6(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"./one/a",
		map[string]string{
			"one/a/b/1.proto": testProtoContent,
			"one/a/b/2.proto": testProtoContent,
			"one/a/1.proto":   "",
		},
		normalpath.WithExt(".proto"),
		normalpath.WithExactPath("foo.yaml"),
	)
}

func TestBasic7(t *testing.T) {
	testBasic(
		t,
		"testdata",
		"",
		map[string]string{
			"a/b/1.proto": testProtoContent,
			"a/b/2.proto": testProtoContent,
			"ab/1.proto":  testProtoContent,
			"ab/2.proto":  testProtoContent,
			"a/1.proto":   "",
			"c/1.proto":   testProtoContent,
			"1.proto":     testProtoContent,
			"a/bar.yaml":  "",
		},
		normalpath.WithExt(".proto"),
		normalpath.WithExactPath("a/bar.yaml"),
		normalpath.WithStripComponents(1),
	)
}

func testBasic(
	t *testing.T,
	dirPath string,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...normalpath.TransformerOption,
) {
	t.Parallel()
	t.Run("static", func(t *testing.T) {
		testBasicStatic(
			t,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("mem", func(t *testing.T) {
		t.Parallel()
		testBasicMem(
			t,
			dirPath,
			false,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("os", func(t *testing.T) {
		t.Parallel()
		testBasicOS(
			t,
			dirPath,
			false,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("mem-tar", func(t *testing.T) {
		t.Parallel()
		testBasicMem(
			t,
			dirPath,
			true,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
	t.Run("os-tar", func(t *testing.T) {
		t.Parallel()
		testBasicOS(
			t,
			dirPath,
			true,
			walkPrefix,
			expectedPathToContent,
			transformerOptions...,
		)
	})
}

func testBasicStatic(
	t *testing.T,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...normalpath.TransformerOption,
) {
	pathToData := make(map[string][]byte)
	for path, content := range expectedPathToContent {
		pathToData[path] = []byte(content)
	}
	readBucket, err := storagemem.NewImmutableReadBucket(pathToData)
	require.NoError(t, err)
	assertExpectedPathToContent(t, readBucket, walkPrefix, expectedPathToContent)
}

func testBasicMem(
	t *testing.T,
	dirPath string,
	doAsTar bool,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...normalpath.TransformerOption,
) {
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	testBasicBucket(
		t,
		readWriteBucketCloser,
		dirPath,
		doAsTar,
		walkPrefix,
		expectedPathToContent,
		transformerOptions...,
	)
}

func testBasicOS(
	t *testing.T,
	dirPath string,
	doAsTar bool,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...normalpath.TransformerOption,
) {
	tempDirPath, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	require.NotEmpty(t, tempDirPath)
	defer func() {
		// won't work with requires but just temporary directory
		require.NoError(t, os.RemoveAll(tempDirPath))
	}()
	readWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(tempDirPath)
	require.NoError(t, err)
	testBasicBucket(
		t,
		readWriteBucketCloser,
		dirPath,
		doAsTar,
		walkPrefix,
		expectedPathToContent,
		transformerOptions...,
	)
}

func testBasicBucket(
	t *testing.T,
	readWriteBucketCloser storage.ReadWriteBucketCloser,
	dirPath string,
	doAsTar bool,
	walkPrefix string,
	expectedPathToContent map[string]string,
	transformerOptions ...normalpath.TransformerOption,
) {
	inputReadWriteBucketCloser, err := storageos.NewReadWriteBucketCloser(dirPath)
	require.NoError(t, err)
	if doAsTar {
		buffer := bytes.NewBuffer(nil)
		require.NoError(t, storageutil.Targz(
			context.Background(),
			buffer,
			inputReadWriteBucketCloser,
			"",
		))
		require.NoError(t, err)
		require.NoError(t, storageutil.Untargz(
			context.Background(),
			buffer,
			readWriteBucketCloser,
			transformerOptions...,
		))
	} else {
		_, err := storageutil.Copy(
			context.Background(),
			inputReadWriteBucketCloser,
			readWriteBucketCloser,
			"",
			transformerOptions...,
		)
		require.NoError(t, err)
	}
	require.NoError(t, inputReadWriteBucketCloser.Close())
	assertExpectedPathToContent(t, readWriteBucketCloser, walkPrefix, expectedPathToContent)
	assert.NoError(t, readWriteBucketCloser.Close())
}

func assertExpectedPathToContent(
	t *testing.T,
	readBucket storage.ReadBucket,
	walkPrefix string,
	expectedPathToContent map[string]string,
) {
	var paths []string
	require.NoError(t, readBucket.Walk(
		context.Background(),
		walkPrefix,
		func(path string) error {
			paths = append(paths, path)
			return nil
		},
	))
	require.Equal(t, len(paths), len(stringutil.SliceToUniqueSortedSlice(paths)))
	assert.Equal(t, len(expectedPathToContent), len(paths), paths)
	for _, path := range paths {
		expectedContent, ok := expectedPathToContent[path]
		assert.True(t, ok, path)
		expectedSize := len(expectedContent)
		objectInfo, err := readBucket.Stat(context.Background(), path)
		assert.NoError(t, err, path)
		// weird issue with int vs uint64
		if expectedSize == 0 {
			assert.Equal(t, 0, int(objectInfo.Size()), path)
		} else {
			assert.Equal(t, expectedSize, int(objectInfo.Size()), path)
		}
		readerCloser, err := readBucket.Get(context.Background(), path)
		assert.NoError(t, err, path)
		data, err := ioutil.ReadAll(readerCloser)
		assert.NoError(t, err, path)
		assert.NoError(t, readerCloser.Close())
		assert.Equal(t, expectedContent, string(data))
	}
}
