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

// Package storagetesting implements testing utilities and integration tests for storage.
package storagetesting

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/internal"
	"github.com/bufbuild/buf/internal/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// testProtoContent is the content of every .proto file in the testing directory.
	testProtoContent = `syntax = "proto3";

package foo;
`
	// testTxtContent is the content of every .txt file in the testing directory.
	testTxtContent = `foo
`
	// testYAMLContent is the content of every .yaml file in the testing directory.
	testYAMLContent = ``
)

// AssertNotExist asserts the path has the expected ObjectInfo.
func AssertNotExist(
	t *testing.T,
	readBucket storage.ReadBucket,
	path string,
) {
	_, err := readBucket.Stat(context.Background(), path)
	assert.Error(t, err)
	assert.True(t, storage.IsNotExist(err))
}

// AssertObjectInfo asserts the path has the expected ObjectInfo.
func AssertObjectInfo(
	t *testing.T,
	readBucket storage.ReadBucket,
	size uint32,
	path string,
	externalPath string,
) {
	objectInfo, err := readBucket.Stat(context.Background(), path)
	require.NoError(t, err)
	AssertObjectInfoEqual(
		t,
		internal.NewObjectInfo(
			size,
			path,
			externalPath,
		),
		objectInfo,
	)
}

// AssertObjectInfoEqual asserts the two ObjectInfos are equal.
func AssertObjectInfoEqual(
	t *testing.T,
	expected storage.ObjectInfo,
	actual storage.ObjectInfo,
) {
	assert.Equal(t, int(expected.Size()), int(actual.Size()))
	assert.Equal(t, expected.Path(), actual.Path())
	assert.Equal(t, expected.ExternalPath(), actual.ExternalPath())
}

// AssertPathToContent asserts the content.
func AssertPathToContent(
	t *testing.T,
	readBucket storage.ReadBucket,
	walkPrefix string,
	expectedPathToContent map[string]string,
) {
	var paths []string
	require.NoError(t, readBucket.Walk(
		context.Background(),
		walkPrefix,
		func(objectInfo storage.ObjectInfo) error {
			paths = append(paths, objectInfo.Path())
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
		readObjectCloser, err := readBucket.Get(context.Background(), path)
		assert.NoError(t, err, path)
		data, err := ioutil.ReadAll(readObjectCloser)
		assert.NoError(t, err, path)
		assert.NoError(t, readObjectCloser.Close())
		assert.Equal(t, expectedContent, string(data))
	}
}

// RunTestSuite runs the test suite.
//
// storagetestingDirPath is the path to this directory.
// newReadBucket takes a path to a directory.
func RunTestSuite(
	t *testing.T,
	storagetestingDirPath string,
	newReadBucket func(*testing.T, string) storage.ReadBucket,
	newWriteBucket func(*testing.T) storage.WriteBucket,
	writeBucketToReadBucket func(*testing.T, storage.WriteBucket) storage.ReadBucket,
) {
	oneDirPath := filepath.Join(storagetestingDirPath, "testdata", "one")
	twoDirPath := filepath.Join(storagetestingDirPath, "testdata", "two")
	threeDirPath := filepath.Join(storagetestingDirPath, "testdata", "three")
	diffDirPathA := filepath.Join(storagetestingDirPath, "testdata", "diff", "a")
	diffDirPathB := filepath.Join(storagetestingDirPath, "testdata", "diff", "b")

	for _, prefix := range []string{
		"",
		".",
		"./",
	} {
		prefix := prefix
		t.Run(fmt.Sprintf("root-%q", prefix), func(t *testing.T) {
			t.Parallel()
			readBucket := newReadBucket(t, oneDirPath)
			AssertPathToContent(
				t,
				readBucket,
				prefix,
				map[string]string{
					"root/a/b/1.proto": testProtoContent,
					"root/a/b/2.proto": testProtoContent,
					"root/a/b/2.txt":   testTxtContent,
					"root/ab/1.proto":  testProtoContent,
					"root/ab/2.proto":  testProtoContent,
					"root/ab/2.txt":    testTxtContent,
					"root/a/1.proto":   testProtoContent,
					"root/a/1.txt":     testTxtContent,
					"root/a/bar.yaml":  testYAMLContent,
					"root/c/1.proto":   testProtoContent,
					"root/1.proto":     testProtoContent,
					"root/foo.yaml":    testYAMLContent,
				},
			)
		})
	}

	t.Run("map-1", func(t *testing.T) {
		t.Parallel()
		readBucket := newReadBucket(t, oneDirPath)
		readBucket = storage.MapReadBucket(
			readBucket,
			storage.MapOnPrefix("root"),
		)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"a/b/1.proto": testProtoContent,
				"a/b/2.proto": testProtoContent,
				"a/b/2.txt":   testTxtContent,
				"ab/1.proto":  testProtoContent,
				"ab/2.proto":  testProtoContent,
				"ab/2.txt":    testTxtContent,
				"a/1.proto":   testProtoContent,
				"a/bar.yaml":  testYAMLContent,
				"a/1.txt":     testTxtContent,
				"c/1.proto":   testProtoContent,
				"1.proto":     testProtoContent,
				"foo.yaml":    testYAMLContent,
			},
		)
	})

	t.Run("map-2", func(t *testing.T) {
		t.Parallel()
		readBucket := newReadBucket(t, oneDirPath)
		readBucket = storage.MapReadBucket(
			readBucket,
			storage.MapOnPrefix("root/a"),
		)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"b/1.proto": testProtoContent,
				"b/2.proto": testProtoContent,
				"b/2.txt":   testTxtContent,
				"1.proto":   testProtoContent,
				"bar.yaml":  testYAMLContent,
				"1.txt":     testTxtContent,
			},
		)
		AssertObjectInfo(
			t,
			readBucket,
			33,
			"1.proto",
			filepath.Join(oneDirPath, "root", "a", "1.proto"),
		)
		readBucket = storage.MapReadBucket(
			readBucket,
			storage.MapOnPrefix("b"),
		)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"1.proto": testProtoContent,
				"2.proto": testProtoContent,
				"2.txt":   testTxtContent,
			},
		)
		AssertObjectInfo(
			t,
			readBucket,
			33,
			"1.proto",
			filepath.Join(oneDirPath, "root", "a", "b", "1.proto"),
		)
	})

	t.Run("map-3", func(t *testing.T) {
		t.Parallel()
		readBucket := newReadBucket(t, oneDirPath)
		readBucket = storage.MapReadBucket(
			readBucket,
			storage.MapOnPrefix("root/ab"),
		)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"1.proto": testProtoContent,
				"2.proto": testProtoContent,
				"2.txt":   testTxtContent,
			},
		)
		AssertObjectInfo(
			t,
			readBucket,
			33,
			"1.proto",
			filepath.Join(oneDirPath, "root", "ab", "1.proto"),
		)
		readBucket = storage.MapReadBucket(
			readBucket,
			storage.MatchOr(
				storage.MatchPathExt(".txt"),
				storage.MatchPathEqual("2.proto"),
			),
		)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"2.proto": testProtoContent,
				"2.txt":   testTxtContent,
			},
		)
		AssertObjectInfo(
			t,
			readBucket,
			33,
			"2.proto",
			filepath.Join(oneDirPath, "root", "ab", "2.proto"),
		)
	})

	t.Run("multi-all", func(t *testing.T) {
		t.Parallel()
		readBucket := newReadBucket(t, twoDirPath)
		readBucketMulti := storage.MultiReadBucket(
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("root1"),
			),
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("root2"),
			),
		)
		AssertPathToContent(
			t,
			readBucketMulti,
			"",
			map[string]string{
				// root1
				"a/b/1.proto": testProtoContent,
				"a/b/2.proto": testProtoContent,
				"a/b/2.txt":   testTxtContent,
				"ab/1.proto":  testProtoContent,
				"ab/2.proto":  testProtoContent,
				"ab/2.txt":    testTxtContent,
				"a/1.proto":   testProtoContent,
				"a/1.txt":     testTxtContent,
				"a/bar.yaml":  testYAMLContent,
				"c/1.proto":   testProtoContent,
				"1.proto":     testProtoContent,
				"foo.yaml":    testYAMLContent,
				// root2
				"a/b/3.proto": testProtoContent,
				"a/b/4.proto": testProtoContent,
				"a/b/4.txt":   testTxtContent,
				"ab/3.proto":  testProtoContent,
				"ab/4.proto":  testProtoContent,
				"ab/4.txt":    testTxtContent,
				"a/2.proto":   testProtoContent,
				"a/2.txt":     testTxtContent,
				"a/bat.yaml":  testYAMLContent,
				"c/3.proto":   testProtoContent,
				"2.proto":     testProtoContent,
				"baz.yaml":    testYAMLContent,
			},
		)
	})
	t.Run("multi-overlap", func(t *testing.T) {
		t.Parallel()
		readBucket := newReadBucket(t, twoDirPath)
		readBucketMulti := storage.MultiReadBucket(
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("root1"),
			),
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("rootoverlap"),
			),
		)
		_, err := readBucketMulti.Get(
			context.Background(),
			"a/b/1.proto",
		)
		assert.Error(t, err)
		assert.True(t, storage.IsExistsMultipleLocations(err))
		_, err = readBucketMulti.Stat(
			context.Background(),
			"a/b/1.proto",
		)
		assert.Error(t, err)
		assert.True(t, storage.IsExistsMultipleLocations(err))
		err = readBucketMulti.Walk(
			context.Background(),
			"",
			func(storage.ObjectInfo) error {
				return nil
			},
		)
		assert.Error(t, err)
		assert.True(t, storage.IsExistsMultipleLocations(err))
	})

	for _, testCase := range []struct {
		name                  string
		prefix                string
		stripComponentCount   uint32
		newReadBucketFunc     func(*testing.T) storage.ReadBucket
		mappers               []storage.Mapper
		expectedPathToContent map[string]string
	}{
		{
			name:              "proto-and-single-file",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathEqual("root/foo.yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"root/a/b/1.proto": testProtoContent,
				"root/a/b/2.proto": testProtoContent,
				"root/ab/1.proto":  testProtoContent,
				"root/ab/2.proto":  testProtoContent,
				"root/a/1.proto":   testProtoContent,
				"root/c/1.proto":   testProtoContent,
				"root/1.proto":     testProtoContent,
				"root/foo.yaml":    testYAMLContent,
			},
		},
		{
			name:              "proto-and-single-file-walk-prefix-root-a",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			prefix:            "root/a",
			mappers: []storage.Mapper{
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathEqual("foo.yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"root/a/b/1.proto": testProtoContent,
				"root/a/b/2.proto": testProtoContent,
				"root/a/1.proto":   testProtoContent,
			},
		},
		{
			name:              "proto-and-single-file-walk-prefix-root-a-2",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			prefix:            "./root/a",
			mappers: []storage.Mapper{
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathEqual("foo.yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"root/a/b/1.proto": testProtoContent,
				"root/a/b/2.proto": testProtoContent,
				"root/a/1.proto":   testProtoContent,
			},
		},
		{
			name:                "proto-and-single-file-strip-components",
			newReadBucketFunc:   func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			stripComponentCount: 1,
			mappers: []storage.Mapper{
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathEqual("a/bar.yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"a/b/1.proto": testProtoContent,
				"a/b/2.proto": testProtoContent,
				"ab/1.proto":  testProtoContent,
				"ab/2.proto":  testProtoContent,
				"a/1.proto":   testProtoContent,
				"c/1.proto":   testProtoContent,
				"1.proto":     testProtoContent,
				"a/bar.yaml":  testYAMLContent,
			},
		},
		{
			name:              "proto-and-single-file-map-prefix-root-a",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MapOnPrefix("root/a"),
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathEqual("bar.yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"b/1.proto": testProtoContent,
				"b/2.proto": testProtoContent,
				"1.proto":   testProtoContent,
				"bar.yaml":  testYAMLContent,
			},
		},
		{
			name:                "proto-and-single-file-map-prefix-a-strip-components",
			newReadBucketFunc:   func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			stripComponentCount: 1,
			mappers: []storage.Mapper{
				storage.MapOnPrefix("a"),
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathEqual("bar.yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"b/1.proto": testProtoContent,
				"b/2.proto": testProtoContent,
				"1.proto":   testProtoContent,
				"bar.yaml":  testYAMLContent,
			},
		},
		{
			name:              "all",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathExt(".txt"),
					storage.MatchPathExt(".yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"root/a/b/1.proto": testProtoContent,
				"root/a/b/2.proto": testProtoContent,
				"root/a/b/2.txt":   testTxtContent,
				"root/ab/1.proto":  testProtoContent,
				"root/ab/2.proto":  testProtoContent,
				"root/ab/2.txt":    testTxtContent,
				"root/a/1.proto":   testProtoContent,
				"root/a/1.txt":     testTxtContent,
				"root/a/bar.yaml":  testYAMLContent,
				"root/c/1.proto":   testProtoContent,
				"root/1.proto":     testProtoContent,
				"root/foo.yaml":    testYAMLContent,
			},
		},
		{
			name:              "proto-and-single-file-not-equal-or-contained-map-prefix",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MapOnPrefix("root"),
				storage.MatchNot(
					storage.MatchPathContained("a"),
				),
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathEqualOrContained("foo.yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				"ab/1.proto": testProtoContent,
				"ab/2.proto": testProtoContent,
				"c/1.proto":  testProtoContent,
				"1.proto":    testProtoContent,
				"foo.yaml":   testYAMLContent,
			},
		},
		{
			name:              "proto-and-single-file-not-equal-or-contained-map-prefix-and",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MapOnPrefix("root"),
				storage.MatchAnd(
					storage.MatchOr(
						storage.MatchPathExt(".proto"),
						storage.MatchPathEqualOrContained("foo.yaml"),
					),
					storage.MatchNot(
						storage.MatchPathContained("a"),
					),
				),
			},
			expectedPathToContent: map[string]string{
				"ab/1.proto": testProtoContent,
				"ab/2.proto": testProtoContent,
				"c/1.proto":  testProtoContent,
				"1.proto":    testProtoContent,
				"foo.yaml":   testYAMLContent,
			},
		},
		{
			name:              "proto-and-single-file-not-equal-or-contained-map-prefix-and-2",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MapOnPrefix("root"),
				storage.MatchAnd(
					storage.MatchOr(
						storage.MatchPathExt(".proto"),
						storage.MatchPathEqual("foo.yaml"),
					),
					storage.MatchNot(
						storage.MatchOr(
							storage.MatchPathEqualOrContained("a"),
							storage.MatchPathEqualOrContained("c"),
						),
					),
				),
			},
			expectedPathToContent: map[string]string{
				"ab/1.proto": testProtoContent,
				"ab/2.proto": testProtoContent,
				"1.proto":    testProtoContent,
				"foo.yaml":   testYAMLContent,
			},
		},
		{
			name:              "proto-and-single-file-not-equal-or-contained-map-prefix-and-3",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MapOnPrefix("root"),
				storage.MatchAnd(
					storage.MatchOr(
						storage.MatchPathExt(".proto"),
						storage.MatchPathEqual("foo.yaml"),
					),
					storage.MatchNot(
						storage.MatchPathEqualOrContained("a"),
					),
					storage.MatchNot(
						storage.MatchPathEqualOrContained("c"),
					),
				),
			},
			expectedPathToContent: map[string]string{
				"ab/1.proto": testProtoContent,
				"ab/2.proto": testProtoContent,
				"1.proto":    testProtoContent,
				"foo.yaml":   testYAMLContent,
			},
		},
		{
			name:              "proto-and-single-file-not-equal-or-contained-map-prefix-chained",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket { return newReadBucket(t, oneDirPath) },
			mappers: []storage.Mapper{
				storage.MapOnPrefix("root"),
				storage.MapOnPrefix("ab"),
				storage.MatchPathExt(".proto"),
			},
			expectedPathToContent: map[string]string{
				"1.proto": testProtoContent,
				"2.proto": testProtoContent,
			},
		},
		{
			name: "multi-all",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket := newReadBucket(t, twoDirPath)
				return storage.MultiReadBucket(
					storage.MapReadBucket(
						readBucket,
						storage.MapOnPrefix("root1"),
					),
					storage.MapReadBucket(
						readBucket,
						storage.MapOnPrefix("root2"),
					),
				)
			},
			mappers: []storage.Mapper{
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathExt(".txt"),
					storage.MatchPathExt(".yaml"),
				),
			},
			expectedPathToContent: map[string]string{
				// root1
				"a/b/1.proto": testProtoContent,
				"a/b/2.proto": testProtoContent,
				"a/b/2.txt":   testTxtContent,
				"ab/1.proto":  testProtoContent,
				"ab/2.proto":  testProtoContent,
				"ab/2.txt":    testTxtContent,
				"a/1.proto":   testProtoContent,
				"a/1.txt":     testTxtContent,
				"a/bar.yaml":  testYAMLContent,
				"c/1.proto":   testProtoContent,
				"1.proto":     testProtoContent,
				"foo.yaml":    testYAMLContent,
				// root2
				"a/b/3.proto": testProtoContent,
				"a/b/4.proto": testProtoContent,
				"a/b/4.txt":   testTxtContent,
				"ab/3.proto":  testProtoContent,
				"ab/4.proto":  testProtoContent,
				"ab/4.txt":    testTxtContent,
				"a/2.proto":   testProtoContent,
				"a/2.txt":     testTxtContent,
				"a/bat.yaml":  testYAMLContent,
				"c/3.proto":   testProtoContent,
				"2.proto":     testProtoContent,
				"baz.yaml":    testYAMLContent,
			},
		},
		{
			name: "multi-map-on-prefix",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket := newReadBucket(t, twoDirPath)
				return storage.MultiReadBucket(
					storage.MapReadBucket(
						readBucket,
						storage.MapOnPrefix("root1"),
					),
					storage.MapReadBucket(
						readBucket,
						storage.MapOnPrefix("root2"),
					),
				)
			},
			mappers: []storage.Mapper{
				storage.MapOnPrefix("a"),
			},
			expectedPathToContent: map[string]string{
				// root1
				"b/1.proto": testProtoContent,
				"b/2.proto": testProtoContent,
				"b/2.txt":   testTxtContent,
				"1.proto":   testProtoContent,
				"1.txt":     testTxtContent,
				"bar.yaml":  testYAMLContent,
				// root2
				"b/3.proto": testProtoContent,
				"b/4.proto": testProtoContent,
				"b/4.txt":   testTxtContent,
				"2.proto":   testProtoContent,
				"2.txt":     testTxtContent,
				"bat.yaml":  testYAMLContent,
			},
		},
		{
			name: "multi-map-on-prefix-filter",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket := newReadBucket(t, twoDirPath)
				return storage.MultiReadBucket(
					storage.MapReadBucket(
						readBucket,
						storage.MapOnPrefix("root1"),
					),
					storage.MapReadBucket(
						readBucket,
						storage.MapOnPrefix("root2"),
					),
				)
			},
			mappers: []storage.Mapper{
				storage.MapOnPrefix("a"),
				storage.MatchOr(
					storage.MatchPathExt(".proto"),
					storage.MatchPathExt(".txt"),
				),
			},
			expectedPathToContent: map[string]string{
				// root1
				"b/1.proto": testProtoContent,
				"b/2.proto": testProtoContent,
				"b/2.txt":   testTxtContent,
				"1.proto":   testProtoContent,
				"1.txt":     testTxtContent,
				// root2
				"b/3.proto": testProtoContent,
				"b/4.proto": testProtoContent,
				"b/4.txt":   testTxtContent,
				"2.proto":   testProtoContent,
				"2.txt":     testTxtContent,
			},
		},
	} {
		testCase := testCase
		if testCase.stripComponentCount == 0 {
			t.Run(fmt.Sprintf("copy-%s", testCase.name), func(t *testing.T) {
				t.Parallel()
				readBucket := testCase.newReadBucketFunc(t)
				readBucket = storage.MapReadBucket(readBucket, testCase.mappers...)
				writeBucket := newWriteBucket(t)
				_, err := storage.Copy(
					context.Background(),
					readBucket,
					writeBucket,
				)
				require.NoError(t, err)
				readBucket = writeBucketToReadBucket(t, writeBucket)
				AssertPathToContent(t, readBucket, testCase.prefix, testCase.expectedPathToContent)
			})
			t.Run(fmt.Sprintf("tar-mapper-read-%s", testCase.name), func(t *testing.T) {
				t.Parallel()
				readBucket := testCase.newReadBucketFunc(t)
				readBucket = storage.MapReadBucket(readBucket, testCase.mappers...)
				writeBucket := newWriteBucket(t)
				buffer := bytes.NewBuffer(nil)
				require.NoError(t, storagearchive.Tar(
					context.Background(),
					readBucket,
					buffer,
				))
				require.NoError(t, storagearchive.Untar(
					context.Background(),
					buffer,
					writeBucket,
					nil,
					testCase.stripComponentCount,
				))
				readBucket = writeBucketToReadBucket(t, writeBucket)
				AssertPathToContent(t, readBucket, testCase.prefix, testCase.expectedPathToContent)
			})
			t.Run(fmt.Sprintf("zip-mapper-read-%s", testCase.name), func(t *testing.T) {
				t.Parallel()
				readBucket := testCase.newReadBucketFunc(t)
				readBucket = storage.MapReadBucket(readBucket, testCase.mappers...)
				writeBucket := newWriteBucket(t)
				buffer := bytes.NewBuffer(nil)
				require.NoError(t, storagearchive.Zip(
					context.Background(),
					readBucket,
					buffer,
				))
				data := buffer.Bytes()
				require.NoError(t, storagearchive.Unzip(
					context.Background(),
					bytes.NewReader(data),
					int64(len(data)),
					writeBucket,
					nil,
					testCase.stripComponentCount,
				))
				readBucket = writeBucketToReadBucket(t, writeBucket)
				AssertPathToContent(t, readBucket, testCase.prefix, testCase.expectedPathToContent)
			})
		}
		t.Run(fmt.Sprintf("tar-mapper-write-%s", testCase.name), func(t *testing.T) {
			t.Parallel()
			readBucket := testCase.newReadBucketFunc(t)
			writeBucket := newWriteBucket(t)
			buffer := bytes.NewBuffer(nil)
			require.NoError(t, storagearchive.Tar(
				context.Background(),
				readBucket,
				buffer,
			))
			require.NoError(t, storagearchive.Untar(
				context.Background(),
				buffer,
				writeBucket,
				storage.MapChain(testCase.mappers...),
				testCase.stripComponentCount,
			))
			readBucket = writeBucketToReadBucket(t, writeBucket)
			AssertPathToContent(t, readBucket, testCase.prefix, testCase.expectedPathToContent)
		})
		t.Run(fmt.Sprintf("zip-mapper-write%s", testCase.name), func(t *testing.T) {
			t.Parallel()
			readBucket := testCase.newReadBucketFunc(t)
			writeBucket := newWriteBucket(t)
			buffer := bytes.NewBuffer(nil)
			require.NoError(t, storagearchive.Zip(
				context.Background(),
				readBucket,
				buffer,
			))
			data := buffer.Bytes()
			require.NoError(t, storagearchive.Unzip(
				context.Background(),
				bytes.NewReader(data),
				int64(len(data)),
				writeBucket,
				storage.MapChain(testCase.mappers...),
				testCase.stripComponentCount,
			))
			readBucket = writeBucketToReadBucket(t, writeBucket)
			AssertPathToContent(t, readBucket, testCase.prefix, testCase.expectedPathToContent)
		})
	}

	t.Run("diff", func(t *testing.T) {
		t.Parallel()
		readBucketA := newReadBucket(t, diffDirPathA)
		readBucketB := newReadBucket(t, diffDirPathB)
		diff, err := storage.Diff(
			context.Background(),
			readBucketA,
			readBucketB,
			"a-dir",
			"b-dir",
		)
		require.NoError(t, err)
		assert.Equal(
			t,
			`--- a-dir/1.txt
+++ b-dir/1.txt
@@ -1,2 +1,2 @@
-aaaa
 bbbb
+cccc
Only in a-dir: 2.txt
Only in b-dir: 3.txt
`,
			string(diff),
		)
	})

	t.Run("overlap-success", func(t *testing.T) {
		t.Parallel()
		readBucket := newReadBucket(t, threeDirPath)
		readBucket = storage.MapReadBucket(readBucket, storage.MatchPathExt(".proto"))
		readBucket = storage.MultiReadBucket(
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("a"),
			),
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("b"),
			),
		)
		allPaths, err := storage.AllPaths(
			context.Background(),
			readBucket,
			"",
		)
		require.NoError(t, err)
		assert.Equal(
			t,
			[]string{
				"one.proto",
				"two.proto",
			},
			allPaths,
		)
	})
	t.Run("overlap-error", func(t *testing.T) {
		t.Parallel()
		readBucket := newReadBucket(t, threeDirPath)
		readBucket = storage.MapReadBucket(
			storage.MultiReadBucket(
				storage.MapReadBucket(
					readBucket,
					storage.MapOnPrefix("a"),
				),
				storage.MapReadBucket(
					readBucket,
					storage.MapOnPrefix("b"),
				),
			),
			storage.MatchPathExt(".proto"),
		)
		_, err := storage.AllPaths(
			context.Background(),
			readBucket,
			"",
		)
		assert.True(t, storage.IsExistsMultipleLocations(err))
	})

	t.Run("map-write-bucket", func(t *testing.T) {
		t.Parallel()
		writeBucket := newWriteBucket(t)
		mapWriteBucket := storage.MapWriteBucket(
			writeBucket,
			storage.MapOnPrefix("a/b/c"),
		)
		writeObjectCloser, err := mapWriteBucket.Put(
			context.Background(),
			"hello",
			4,
		)
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte("abcd"))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())
		readBucket := writeBucketToReadBucket(t, writeBucket)
		data, err := storage.ReadPath(
			context.Background(),
			readBucket,
			"a/b/c/hello",
		)
		require.NoError(t, err)
		require.Equal(t, "abcd", string(data))
	})
}
