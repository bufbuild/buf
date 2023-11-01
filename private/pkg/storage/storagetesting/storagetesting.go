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

// Package storagetesting implements testing utilities and integration tests for storage.
package storagetesting

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/tmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
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
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

// AssertObjectInfo asserts the path has the expected ObjectInfo.
func AssertObjectInfo(
	t *testing.T,
	readBucket storage.ReadBucket,
	path string,
	externalPath string,
) {
	objectInfo, err := readBucket.Stat(context.Background(), path)
	require.NoError(t, err)
	AssertObjectInfoEqual(
		t,
		storageutil.NewObjectInfo(
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
		_, err := readBucket.Stat(context.Background(), path)
		require.NoError(t, err, path)
		readObjectCloser, err := readBucket.Get(context.Background(), path)
		require.NoError(t, err, path)
		data, err := io.ReadAll(readObjectCloser)
		assert.NoError(t, err, path)
		assert.NoError(t, readObjectCloser.Close())
		assert.Equal(t, expectedContent, string(data))
	}
}

// AssertPaths asserts the paths.
func AssertPaths(
	t *testing.T,
	readBucket storage.ReadBucket,
	walkPrefix string,
	expectedPaths ...string,
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
	sort.Strings(paths)
	assert.Equal(t, stringutil.SliceToUniqueSortedSlice(expectedPaths), paths)
}

// GetExternalPathFunc can be used to get the external path of
// a path given the root path.
type GetExternalPathFunc func(*testing.T, string, string) string

// RunTestSuite runs the test suite.
//
// storagetestingDirPath is the path to this directory.
// newReadBucket takes a path to a directory.
func RunTestSuite(
	t *testing.T,
	storagetestingDirPath string,
	newReadBucket func(*testing.T, string, storageos.Provider) (storage.ReadBucket, GetExternalPathFunc),
	newWriteBucket func(*testing.T, storageos.Provider) storage.WriteBucket,
	writeBucketToReadBucket func(*testing.T, storage.WriteBucket) storage.ReadBucket,
) {
	oneDirPath := filepath.Join(storagetestingDirPath, "testdata", "one")
	twoDirPath := filepath.Join(storagetestingDirPath, "testdata", "two")
	threeDirPath := filepath.Join(storagetestingDirPath, "testdata", "three")
	fourDirPath := filepath.Join(storagetestingDirPath, "testdata", "four")
	fiveDirPath := filepath.Join(storagetestingDirPath, "testdata", "five")
	symlinkSuccessDirPath := filepath.Join(storagetestingDirPath, "testdata", "symlink_success")
	symlinkLoopDirPath := filepath.Join(storagetestingDirPath, "testdata", "symlink_loop")
	defaultProvider := storageos.NewProvider()
	runner := command.NewRunner()

	for _, prefix := range []string{
		"",
		".",
		"./",
	} {
		prefix := prefix
		t.Run(fmt.Sprintf("root-%q", prefix), func(t *testing.T) {
			t.Parallel()
			readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
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
		readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
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
		readBucket, getExternalPathFunc := newReadBucket(t, oneDirPath, defaultProvider)
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
			"1.proto",
			getExternalPathFunc(t, oneDirPath, filepath.Join("root", "a", "1.proto")),
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
			"1.proto",
			getExternalPathFunc(t, oneDirPath, filepath.Join("root", "a", "b", "1.proto")),
		)
	})

	t.Run("map-3", func(t *testing.T) {
		t.Parallel()
		readBucket, getExternalPathFunc := newReadBucket(t, oneDirPath, defaultProvider)
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
			"1.proto",
			getExternalPathFunc(t, oneDirPath, filepath.Join("root", "ab", "1.proto")),
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
			"2.proto",
			getExternalPathFunc(t, oneDirPath, filepath.Join("root", "ab", "2.proto")),
		)
	})

	t.Run("multi-all", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, twoDirPath, defaultProvider)
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

	t.Run("multi-overlapping-files-error", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, twoDirPath, defaultProvider)
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

	// this is testing that if we have i.e. protoc -I root/a -I root
	// that even if this is an error in our world, this is not a problem
	// in terms of storage buckets
	t.Run("multi-overlapping-dirs-success", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, fourDirPath, defaultProvider)
		readBucketMulti := storage.MultiReadBucket(
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("root/a"),
			),
			storage.MapReadBucket(
				readBucket,
				storage.MapOnPrefix("root"),
			),
		)
		AssertPathToContent(
			t,
			readBucketMulti,
			"",
			map[string]string{
				"a/b/1.proto": testProtoContent,
				"a/b/2.proto": testProtoContent,
				"a/3.proto":   testProtoContent,
				"b/1.proto":   testProtoContent,
				"b/2.proto":   testProtoContent,
				"3.proto":     testProtoContent,
			},
		)
	})

	// this is testing that two roots can have a file with the same
	// name, but one could be a directory and the other could be a
	// regular file.
	t.Run("multi-dir-file-collision", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, fiveDirPath, defaultProvider)
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
				"foo": testProtoContent,
				// root2
				"foo/bar.proto": testProtoContent,
			},
		)
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
			name: "proto-and-single-file",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-walk-prefix-root-a",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
			prefix: "root/a",
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
			name: "proto-and-single-file-walk-prefix-root-a-2",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
			prefix: "./root/a",
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
			name: "proto-and-single-file-strip-components",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-map-prefix-root-a",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-map-prefix-a-strip-components",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "all",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-not-equal-or-contained-map-prefix",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-not-equal-or-contained-map-prefix-and",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-not-equal-or-contained-map-prefix-and-2",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-not-equal-or-contained-map-prefix-and-3",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
			name: "proto-and-single-file-not-equal-or-contained-map-prefix-chained",
			newReadBucketFunc: func(t *testing.T) storage.ReadBucket {
				readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
				return readBucket
			},
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
				readBucket, _ := newReadBucket(t, twoDirPath, defaultProvider)
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
				readBucket, _ := newReadBucket(t, twoDirPath, defaultProvider)
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
				readBucket, _ := newReadBucket(t, twoDirPath, defaultProvider)
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
				writeBucket := newWriteBucket(t, defaultProvider)
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
				writeBucket := newWriteBucket(t, defaultProvider)
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
				writeBucket := newWriteBucket(t, defaultProvider)
				buffer := bytes.NewBuffer(nil)
				require.NoError(t, storagearchive.Zip(
					context.Background(),
					readBucket,
					buffer,
					true,
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
			writeBucket := newWriteBucket(t, defaultProvider)
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
			writeBucket := newWriteBucket(t, defaultProvider)
			buffer := bytes.NewBuffer(nil)
			require.NoError(t, storagearchive.Zip(
				context.Background(),
				readBucket,
				buffer,
				true,
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
		diffDirPathA := filepath.Join(storagetestingDirPath, "testdata", "diff", "a")
		diffDirPathB := filepath.Join(storagetestingDirPath, "testdata", "diff", "b")
		readBucketA, getExternalPathFuncA := newReadBucket(t, diffDirPathA, defaultProvider)
		readBucketB, getExternalPathFuncB := newReadBucket(t, diffDirPathB, defaultProvider)
		readBucketA = storage.MapReadBucket(readBucketA, storage.MapOnPrefix("prefix"))
		readBucketB = storage.MapReadBucket(readBucketB, storage.MapOnPrefix("prefix"))
		externalPathPrefixA := getExternalPathFuncA(t, diffDirPathA, "prefix") + string(os.PathSeparator)
		externalPathPrefixB := getExternalPathFuncB(t, diffDirPathB, "prefix") + string(os.PathSeparator)
		a1TxtPath := filepath.ToSlash(externalPathPrefixA + "1.txt")
		b1TxtPath := filepath.ToSlash(externalPathPrefixB + "1.txt")
		a2TxtPath := filepath.ToSlash(externalPathPrefixA + "2.txt")
		b2TxtPath := filepath.ToSlash(externalPathPrefixB + "2.txt")

		diff, err := storage.DiffBytes(
			context.Background(),
			runner,
			readBucketA,
			readBucketB,
			storage.DiffWithSuppressTimestamps(),
			storage.DiffWithExternalPaths(),
			storage.DiffWithExternalPathPrefixes(
				externalPathPrefixA,
				externalPathPrefixB,
			),
		)

		// This isn't great, but it tests the exact behavior of the diff
		// functionality. Headers are always "ToSlash" paths and `\n`. The
		// contents of the diff are platform dependent.
		diff1 := `@@ -1,2 +1,2 @@
-aaaa
 bbbb
+cccc
`
		diff2 := `@@ -1 +0,0 @@
-dddd
`
		expectDiff := fmt.Sprintf(
			`diff -u %s %s
--- %s
+++ %s
%sdiff -u %s %s
--- %s
+++ %s
%s`,
			a1TxtPath,
			b1TxtPath,
			a1TxtPath,
			b1TxtPath,
			diff1,
			a2TxtPath,
			b2TxtPath,
			a2TxtPath,
			b2TxtPath,
			diff2,
		)

		require.NoError(t, err)
		assert.Equal(
			t,
			expectDiff,
			string(diff),
		)
	})

	t.Run("overlap-success", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, threeDirPath, defaultProvider)
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
		readBucket, _ := newReadBucket(t, threeDirPath, defaultProvider)
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
		writeBucket := newWriteBucket(t, defaultProvider)
		mapWriteBucket := storage.MapWriteBucket(
			writeBucket,
			storage.MapOnPrefix("a/b/c"),
		)
		writeObjectCloser, err := mapWriteBucket.Put(
			context.Background(),
			"hello",
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

	t.Run("absolute-path-read-error", func(t *testing.T) {
		t.Parallel()

		absolutePath := "/absolute/path"
		if runtime.GOOS == "windows" {
			absolutePath = "C:\\Fake\\Absolute\\Path"
		}
		expectErr := fmt.Sprintf("%s: expected to be relative", absolutePath)

		readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
		_, err := readBucket.Get(context.Background(), absolutePath)
		require.EqualError(t, err, expectErr, "should be using storageutil.ValidatePath on Get")
		_, err = readBucket.Stat(context.Background(), absolutePath)
		require.EqualError(t, err, expectErr, "should be using storageutil.ValidatePath on Stat")
		err = readBucket.Walk(context.Background(), absolutePath, nil)
		require.EqualError(t, err, expectErr, "should be using storageutil.ValidatePrefix on Walk")
	})

	t.Run("absolute-path-write-error", func(t *testing.T) {
		t.Parallel()

		absolutePath := "/absolute/path"
		if runtime.GOOS == "windows" {
			absolutePath = "C:\\Fake\\Absolute\\Path"
		}
		expectErr := fmt.Sprintf("%s: expected to be relative", absolutePath)

		writeBucket := newWriteBucket(t, defaultProvider)
		_, err := writeBucket.Put(context.Background(), absolutePath)
		require.EqualError(t, err, expectErr, "should be using normalize.NormalizeAndValidate on Put")
		err = writeBucket.Delete(context.Background(), absolutePath)
		require.EqualError(t, err, expectErr, "should be using normalize.NormalizeAndValidate on Delete")
	})

	t.Run("root-path-error", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
		_, err := readBucket.Get(context.Background(), ".")
		require.EqualError(t, err, "cannot use root", "should be using storageutil.ValidatePath on Get")
		_, err = readBucket.Stat(context.Background(), ".")
		require.EqualError(t, err, "cannot use root", "should be using storageutil.ValidatePath on Stat")
	})

	t.Run("write-bucket-put-delete", func(t *testing.T) {
		t.Parallel()
		writeBucket := newWriteBucket(t, defaultProvider)
		err := writeBucket.Delete(context.Background(), "hello")
		require.True(t, errors.Is(err, fs.ErrNotExist))
		writeObjectCloser, err := writeBucket.Put(
			context.Background(),
			"hello",
		)
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte("abcd"))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())
		err = writeBucket.Delete(context.Background(), "hello")
		require.NoError(t, err)
		err = writeBucket.Delete(context.Background(), "hello")
		require.True(t, errors.Is(err, fs.ErrNotExist))
		writeObjectCloser, err = writeBucket.Put(
			context.Background(),
			"hello",
		)
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte("abcd"))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())
		err = writeBucket.Delete(context.Background(), "hello")
		require.NoError(t, err)
		err = writeBucket.Delete(context.Background(), "hello")
		require.True(t, errors.Is(err, fs.ErrNotExist))
	})

	t.Run("write-bucket-put-delete-all", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		writeBucket := newWriteBucket(t, defaultProvider)
		// this test starts with this data in the bucket, and then
		// deletes it over time in different ways
		pathToData := map[string]string{
			"a.txt":      testTxtContent,
			"b/d.txt":    testTxtContent,
			"b/d/e.txt":  testTxtContent,
			"b/d/f.txt":  testTxtContent,
			"c.d/e.txt":  testTxtContent,
			"c.de/f.txt": testTxtContent,
			"g.txt":      testTxtContent,
		}
		for path, data := range pathToData {
			err := storage.PutPath(ctx, writeBucket, path, []byte(data))
			require.NoError(t, err)
		}
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err := writeBucket.DeleteAll(ctx, "h")
		require.NoError(t, err)
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "b/d/e.txt")
		require.NoError(t, err)
		delete(pathToData, "b/d/e.txt")
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "b/d")
		require.NoError(t, err)
		delete(pathToData, "b/d/f.txt")
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "b")
		require.NoError(t, err)
		delete(pathToData, "b/d.txt")
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "a.txt")
		require.NoError(t, err)
		delete(pathToData, "a.txt")
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "c.d")
		require.NoError(t, err)
		delete(pathToData, "c.d/e.txt")
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "c.d")
		require.NoError(t, err)
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "c.de")
		require.NoError(t, err)
		delete(pathToData, "c.de/f.txt")
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
		err = writeBucket.DeleteAll(ctx, "")
		require.NoError(t, err)
		delete(pathToData, "g.txt")
		AssertPathToContent(
			t,
			writeBucketToReadBucket(t, writeBucket),
			"",
			pathToData,
		)
	})

	t.Run("walk-prefixed-bucket-should-not-error", func(t *testing.T) {
		t.Parallel()
		writeBucket := newWriteBucket(t, defaultProvider)
		readBucket := writeBucketToReadBucket(t, writeBucket)
		mappedReadBucket := storage.MapReadBucket(readBucket, storage.MapOnPrefix("prefix"))
		err := mappedReadBucket.Walk(context.Background(), "", func(_ storage.ObjectInfo) error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("symlink_success_no_symlinks", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, symlinkSuccessDirPath, defaultProvider)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"file.proto": testProtoContent,
			},
		)
	})
	t.Run("symlink_success_follow_symlinks", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(
			t,
			symlinkSuccessDirPath,
			storageos.NewProvider(
				storageos.ProviderWithSymlinks(),
			),
		)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"1.proto":      testProtoContent,
				"a/b/1.proto":  testProtoContent,
				"a/b/2.proto":  testProtoContent,
				"a/b/2.txt":    testTxtContent,
				"a/bar.yaml":   testYAMLContent,
				"a/file.proto": testProtoContent,
				"ab/1.proto":   testProtoContent,
				"ab/2.proto":   testProtoContent,
				"ab/2.txt":     testTxtContent,
				"file.proto":   testProtoContent,
			},
		)
	})
	t.Run("symlink_loop_no_symlinks", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(t, symlinkLoopDirPath, defaultProvider)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"file.proto": testProtoContent,
			},
		)
	})
	t.Run("symlink_loop_follow_symlinks", func(t *testing.T) {
		t.Parallel()
		readBucket, _ := newReadBucket(
			t,
			symlinkLoopDirPath,
			storageos.NewProvider(
				storageos.ProviderWithSymlinks(),
			),
		)
		AssertPathToContent(
			t,
			readBucket,
			"",
			map[string]string{
				"file.proto": testProtoContent,
			},
		)
	})
	t.Run("is_empty", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		readBucket, _ := newReadBucket(t, oneDirPath, defaultProvider)
		isEmpty, err := storage.IsEmpty(ctx, readBucket, "")
		require.NoError(t, err)
		require.False(t, isEmpty)
		isEmpty, err = storage.IsEmpty(ctx, readBucket, "root/a")
		require.NoError(t, err)
		require.False(t, isEmpty)

		tmpDir, err := tmp.NewDir()
		require.NoError(t, err)
		readBucket, _ = newReadBucket(t, tmpDir.AbsPath(), defaultProvider)
		isEmpty, err = storage.IsEmpty(ctx, readBucket, "")
		require.NoError(t, err)
		require.True(t, isEmpty)
		err = os.WriteFile(filepath.Join(tmpDir.AbsPath(), "foo.txt"), []byte("foo"), 0600)
		require.NoError(t, err)
		// need to make a new readBucket since the old one won't necessarily have the foo.txt
		// file in it, ie in-memory buckets
		readBucket, _ = newReadBucket(t, tmpDir.AbsPath(), defaultProvider)
		isEmpty, err = storage.IsEmpty(ctx, readBucket, "")
		require.NoError(t, err)
		require.False(t, isEmpty)
		isEmpty, err = storage.IsEmpty(
			ctx,
			storage.MapReadBucket(readBucket, storage.MatchPathExt(".proto")),
			"",
		)
		require.NoError(t, err)
		require.True(t, isEmpty)
		require.NoError(t, tmpDir.Close())
	})
	t.Run("limit-write-bucket", func(t *testing.T) {
		t.Parallel()
		writeBucket := newWriteBucket(t, defaultProvider)
		readBucket := writeBucketToReadBucket(t, writeBucket)
		const limit = 2048
		limitedWriteBucket := storage.LimitWriteBucket(writeBucket, limit)
		var (
			wg           sync.WaitGroup
			writtenBytes atomic.Int64
			triedBytes   atomic.Int64
		)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				data := bytes.Repeat([]byte("b"), i*100)
				path := strconv.Itoa(i)
				triedBytes.Add(int64(len(data)))
				err := storage.PutPath(context.Background(), limitedWriteBucket, path, data)
				if err != nil {
					assert.True(t, storage.IsWriteLimitReached(err))
					return
				}
				readData, err := storage.ReadPath(context.Background(), readBucket, path)
				assert.NoError(t, err)
				assert.Equal(t, readData, data)
				writtenBytes.Add(int64(len(data)))
			}(i)
		}
		wg.Wait()
		require.Greater(t, triedBytes.Load(), int64(limit))
		assert.LessOrEqual(t, writtenBytes.Load(), int64(limit))
	})
	t.Run("limit-untar-file-size", func(t *testing.T) {
		t.Parallel()
		writeBucket := newWriteBucket(t, defaultProvider)
		const limit = 2048
		files := map[string][]byte{
			"within":     bytes.Repeat([]byte{0}, limit-1),
			"at":         bytes.Repeat([]byte{0}, limit),
			"exceeds":    bytes.Repeat([]byte{0}, limit+1),
			"match-file": bytes.Repeat([]byte{0}, limit-1),
		}
		for path, data := range files {
			err := storage.PutPath(context.Background(), writeBucket, path, data)
			require.NoError(t, err)
		}
		var buffer bytes.Buffer
		err := storagearchive.Tar(context.Background(), writeBucketToReadBucket(t, writeBucket), &buffer)
		require.NoError(t, err)
		writeBucket = newWriteBucket(t, defaultProvider)
		tarball := bytes.NewReader(buffer.Bytes())
		err = storagearchive.Untar(context.Background(), tarball, writeBucket, nil, 0, storagearchive.WithMaxFileSizeUntarOption(limit))
		assert.ErrorIs(t, err, storagearchive.ErrFileSizeLimit)
		_, err = tarball.Seek(0, io.SeekStart)
		require.NoError(t, err)
		err = storagearchive.Untar(context.Background(), tarball, writeBucket, nil, 0)
		assert.NoError(t, err)
		_, err = tarball.Seek(0, io.SeekStart)
		require.NoError(t, err)
		err = storagearchive.Untar(context.Background(), tarball, writeBucket, nil, 0, storagearchive.WithMaxFileSizeUntarOption(limit+1))
		assert.NoError(t, err)
		err = storagearchive.Untar(
			context.Background(),
			tarball,
			writeBucket,
			storage.MatchPathEqual("match-file"),
			0,
			storagearchive.WithMaxFileSizeUntarOption(limit),
		)
		assert.NoError(t, err)
	})
}
