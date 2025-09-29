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

package storage_test

import (
	"context"
	"io/fs"
	"sort"
	"strings"
	"testing"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskReadBucket(t *testing.T) {
	t.Parallel()

	// Create a test bucket with various files
	testFiles := map[string]string{
		"proto/v1/user.proto":      "syntax = \"proto3\";",
		"proto/v1/admin.proto":     "syntax = \"proto3\";",
		"proto/v2/user.proto":      "syntax = \"proto3\";",
		"proto/v2/admin.proto":     "syntax = \"proto3\";",
		"proto/internal/log.proto": "syntax = \"proto3\";",
		"docs/README.md":           "# Documentation",
		"src/main.go":              "package main",
		"node_modules/pkg/mod.js":  "module.exports = {};",
		"api/v1/service.proto":     "syntax = \"proto3\";",
		"api/v2/service.proto":     "syntax = \"proto3\";",
	}
	delegate := storagemem.NewReadWriteBucket()
	ctx := t.Context()
	for path, content := range testFiles {
		writeObjectCloser, err := delegate.Put(ctx, path)
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())
	}
	// When no filters specified, all files should be returned
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{},
		[]string{},
		"",
		xslices.MapKeysToSlice(testFiles),
	)
	// Only files under proto/ should be returned
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{"proto"},
		[]string{},
		"",
		[]string{"proto/internal/log.proto", "proto/v1/admin.proto", "proto/v1/user.proto", "proto/v2/admin.proto", "proto/v2/user.proto"},
	)
	// Proto files should be included except those in internal/
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{"proto"},
		[]string{"proto/internal"},
		"",
		[]string{"proto/v1/admin.proto", "proto/v1/user.proto", "proto/v2/admin.proto", "proto/v2/user.proto"},
	)
	// Only files from proto/v1/ and api/v2/ should be returned
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{"proto/v1", "api/v2"},
		[]string{},
		"",
		[]string{"api/v2/service.proto", "proto/v1/admin.proto", "proto/v1/user.proto"},
	)
	// All files except node_modules should be returned
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{},
		[]string{"node_modules"},
		"",
		xslices.Filter(
			xslices.MapKeysToSlice(testFiles),
			func(s string) bool { return !strings.HasPrefix(s, "node_modules") },
		),
	)
	// Walking proto/v1 with include filters should only return proto/v1 files
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{"proto/v1", "proto/v2"},
		[]string{},
		"proto/v1",
		[]string{"proto/v1/admin.proto", "proto/v1/user.proto"},
	)
	// Walking proto with specific includes should only return matching files
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{"proto/v1", "proto/v2"},
		[]string{},
		"proto",
		[]string{"proto/v1/admin.proto", "proto/v1/user.proto", "proto/v2/admin.proto", "proto/v2/user.proto"},
	)
	// Walking src with proto includes should return no files
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{"proto"},
		[]string{},
		"src",
		nil,
	)
	// Walking proto/v1 with proto include should return proto/v1 files
	testMaskReadBucket(
		t,
		delegate,
		testFiles,
		[]string{"proto"},
		[]string{},
		"proto/v1",
		[]string{"proto/v1/admin.proto", "proto/v1/user.proto"},
	)
}

func TestMaskReadBucket_InvalidPrefixes(t *testing.T) {
	t.Parallel()
	readWriteBucket := storagemem.NewReadWriteBucket()
	_, err := storage.MaskReadBucket(readWriteBucket, []string{"../invalid"}, []string{})
	assert.Error(t, err, "Should error on invalid include prefix")
	_, err = storage.MaskReadBucket(readWriteBucket, []string{}, []string{"../invalid"})
	assert.Error(t, err, "Should error on invalid exclude prefix")
}

func TestMaskReadBucket_PrefixCompaction(t *testing.T) {
	t.Parallel()

	// Test that redundant child prefixes are removed
	readWriteBucket := storagemem.NewReadWriteBucket()
	ctx := t.Context()

	// Create test files
	testFiles := map[string]string{
		"foo/file1.txt":       "content1",
		"foo/v1/file2.txt":    "content2",
		"foo/v1/v2/file3.txt": "content3",
		"bar/file4.txt":       "content4",
		"baz/file5.txt":       "content5",
	}
	for path, content := range testFiles {
		writeObjectCloser, err := readWriteBucket.Put(ctx, path)
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())
	}

	// Test redundant includes: ["foo", "foo/v1", "foo/v1/v2"] should become just ["foo"]
	filteredBucket, err := storage.MaskReadBucket(
		readWriteBucket,
		[]string{"foo", "foo/v1", "foo/v1/v2"},
		[]string{},
	)
	require.NoError(t, err)

	// Walk should return all files under "foo" (since "foo/v1" and "foo/v1/v2" are redundant)
	var actualFiles []string
	err = filteredBucket.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
		actualFiles = append(actualFiles, objectInfo.Path())
		return nil
	})
	require.NoError(t, err)

	sort.Strings(actualFiles)
	expectedFiles := []string{"foo/file1.txt", "foo/v1/file2.txt", "foo/v1/v2/file3.txt"}
	sort.Strings(expectedFiles)
	assert.Equal(t, expectedFiles, actualFiles, "Should include all files under foo prefix")

	// Test mixed includes: ["foo", "bar", "foo/v1"] should become ["foo", "bar"]
	filteredBucket2, err := storage.MaskReadBucket(
		readWriteBucket,
		[]string{"foo", "bar", "foo/v1"},
		[]string{},
	)
	require.NoError(t, err)

	var actualFiles2 []string
	err = filteredBucket2.Walk(ctx, "", func(objectInfo storage.ObjectInfo) error {
		actualFiles2 = append(actualFiles2, objectInfo.Path())
		return nil
	})
	require.NoError(t, err)

	sort.Strings(actualFiles2)
	expectedFiles2 := []string{"bar/file4.txt", "foo/file1.txt", "foo/v1/file2.txt", "foo/v1/v2/file3.txt"}
	sort.Strings(expectedFiles2)
	assert.Equal(t, expectedFiles2, actualFiles2, "Should include all files under foo and bar prefixes")
}

func TestMaskReadBucket_WalkOptimization(t *testing.T) {
	t.Parallel()

	// Add some test files
	ctx := t.Context()
	testFiles := map[string]string{
		"proto/v1/user.proto": "content",
		"proto/v2/user.proto": "content",
		"docs/README.md":      "content",
		"src/main.go":         "content",
	}
	delegate := storagemem.NewReadWriteBucket()
	for path, content := range testFiles {
		writeObjectCloser, err := delegate.Put(ctx, path)
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())
	}
	// Should walk with original prefix when no includes
	testMaskReadBucket_WalkOptimization(
		t,
		delegate,
		[]string{},
		"",
		[]string{""},
	)
	// Should walk only include prefixes
	testMaskReadBucket_WalkOptimization(
		t,
		delegate,
		[]string{"proto/v1", "proto/v2"},
		"",
		[]string{"proto/v1", "proto/v2"},
	)
	// Should walk only the matching include prefix
	testMaskReadBucket_WalkOptimization(
		t,
		delegate,
		[]string{"proto/v1", "proto/v2"},
		"proto/v1",
		[]string{"proto/v1"},
	)
	// Should narrow to include prefixes under walk prefix
	testMaskReadBucket_WalkOptimization(
		t,
		delegate,
		[]string{"proto/v1", "proto/v2"},
		"proto",
		[]string{"proto/v1", "proto/v2"},
	)
}

func testMaskReadBucket(
	t *testing.T,
	delegate storage.ReadWriteBucket,
	testFiles map[string]string,
	includePrefixes []string,
	excludePrefixes []string,
	walkPrefix string,
	expectedFiles []string,
) {
	ctx := t.Context()

	// Create prefix filter bucket
	filteredBucket, err := storage.MaskReadBucket(delegate, includePrefixes, excludePrefixes)
	require.NoError(t, err)

	// Test Walk operation
	var actualFiles []string
	err = filteredBucket.Walk(ctx, walkPrefix, func(objectInfo storage.ObjectInfo) error {
		actualFiles = append(actualFiles, objectInfo.Path())
		return nil
	})
	require.NoError(t, err)
	sort.Strings(actualFiles)
	sort.Strings(expectedFiles)
	assert.Equal(t, expectedFiles, actualFiles)

	// Test Get/Stat operations based on include/exclude filters (not walk prefix)
	// Calculate which files should be accessible based on the include/exclude filters
	accessibleFiles := make(map[string]bool)
	for file := range testFiles {
		shouldInclude := len(includePrefixes) == 0 // if no includes, include all by default
		if !shouldInclude {
			// Check if any include prefix matches
			for _, includePrefix := range includePrefixes {
				if file == includePrefix || len(file) > len(includePrefix) && strings.HasPrefix(file, includePrefix) && file[len(includePrefix)] == '/' {
					shouldInclude = true
					break
				}
			}
		}
		if shouldInclude {
			// Check excludes
			shouldExclude := false
			for _, excludePrefix := range excludePrefixes {
				if file == excludePrefix || len(file) > len(excludePrefix) && strings.HasPrefix(file, excludePrefix) && file[len(excludePrefix)] == '/' {
					shouldExclude = true
					break
				}
			}
			if !shouldExclude {
				accessibleFiles[file] = true
			}
		}
	}
	// Test Get operation for accessible files
	for file := range accessibleFiles {
		readObjectCloser, err := filteredBucket.Get(ctx, file)
		assert.NoError(t, err, "Should be able to get accessible file: %s", file)
		if err == nil {
			require.NoError(t, readObjectCloser.Close())
		}
	}
	// Test that non-accessible files return ErrNotExist
	for file := range testFiles {
		if !accessibleFiles[file] {
			_, err := filteredBucket.Get(ctx, file)
			assert.ErrorIs(t, err, fs.ErrNotExist, "Non-accessible file %s should return ErrNotExist", file)
			_, err = filteredBucket.Stat(ctx, file)
			assert.ErrorIs(t, err, fs.ErrNotExist, "Non-accessible file %s should return ErrNotExist for Stat", file)
		}
	}
}

func testMaskReadBucket_WalkOptimization(t *testing.T,
	bucket storage.ReadWriteBucket,
	includePrefixes []string,
	walkPrefix string,
	expectedWalkCalls []string,
) {
	trackingBucket := &walkTrackingBucket{delegate: bucket}
	filteredBucket, err := storage.MaskReadBucket(trackingBucket, includePrefixes, []string{})
	require.NoError(t, err)
	err = filteredBucket.Walk(t.Context(), walkPrefix, func(storage.ObjectInfo) error {
		return nil
	})
	require.NoError(t, err)
	sort.Strings(trackingBucket.walkCalls)
	sort.Strings(expectedWalkCalls)
	assert.Equal(t, expectedWalkCalls, trackingBucket.walkCalls)
}

// walkTrackingBucket wraps a ReadWriteBucket to track Walk calls
type walkTrackingBucket struct {
	delegate  storage.ReadWriteBucket
	walkCalls []string
}

func (w *walkTrackingBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	return w.delegate.Get(ctx, path)
}

func (w *walkTrackingBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	return w.delegate.Stat(ctx, path)
}

func (w *walkTrackingBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	w.walkCalls = append(w.walkCalls, prefix)
	return w.delegate.Walk(ctx, prefix, f)
}
