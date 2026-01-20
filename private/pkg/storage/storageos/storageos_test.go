// Copyright 2020-2026 Buf Technologies, Inc.
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

package storageos_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/storage/storagetesting"
	"github.com/stretchr/testify/require"
)

var storagetestingDirPath = filepath.Join("..", "storagetesting")

func TestOS(t *testing.T) {
	t.Parallel()
	storagetesting.RunTestSuite(
		t,
		storagetestingDirPath,
		testNewReadBucket,
		testNewWriteBucket,
		testWriteBucketToReadBucket,
		true,
	)

	t.Run("get_non_existent_file", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		// Create a bucket at an absolute path.
		tempDir := t.TempDir()
		tempDir, err := filepath.Abs(tempDir)
		require.NoError(t, err)
		bucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
		require.NoError(t, err)

		// Write a file to it.
		writeObjectCloser, err := bucket.Put(ctx, "foo.txt")
		require.NoError(t, err)
		written, err := writeObjectCloser.Write([]byte(nil))
		require.NoError(t, err)
		require.Zero(t, written)
		require.NoError(t, writeObjectCloser.Close())

		// Try reading a file as if foo.txt is a directory.
		_, err = bucket.Get(ctx, "foo.txt/bar.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)
		_, err = bucket.Get(ctx, "foo.txt/bar.txt/baz.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)

		// Read a file that does not exist at all.
		_, err = bucket.Get(ctx, "baz.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("get_non_existent_file_symlink", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		// Create a bucket at an absolute path.
		actualTempDir := t.TempDir()
		actualTempDir, err := filepath.Abs(actualTempDir)
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(actualTempDir, "foo.txt"))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		tempDir := t.TempDir()
		tempDir, err = filepath.Abs(tempDir)
		require.NoError(t, err)
		tempDir = filepath.Join(tempDir, "sym")
		require.NoError(t, os.Symlink(actualTempDir, tempDir))
		provider := storageos.NewProvider(storageos.ProviderWithSymlinks())
		bucket, err := provider.NewReadWriteBucket(tempDir, storageos.ReadWriteBucketWithSymlinksIfSupported())
		require.NoError(t, err)

		foo, err := bucket.Get(ctx, "foo.txt")
		require.NoError(t, err)
		require.NoError(t, foo.Close())

		// Try reading a file as if foo.txt is a directory.
		_, err = bucket.Get(ctx, "foo.txt/bar.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)
		_, err = bucket.Get(ctx, "foo.txt/bar.txt/baz.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)

		// Read a file that does not exist at all.
		_, err = bucket.Get(ctx, "baz.txt")
		require.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("read_only_files_non_atomic", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tempDir := t.TempDir()

		// Create bucket with read-only files option
		bucket, err := storageos.NewProvider().NewReadWriteBucket(
			tempDir,
			storageos.ReadWriteBucketWithReadOnlyFiles(),
		)
		require.NoError(t, err)

		// Write a file without atomic option (non-atomic write)
		writeObjectCloser, err := bucket.Put(ctx, "test.txt")
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte("test data"))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())

		// Check file permissions
		filePath := filepath.Join(tempDir, "test.txt")
		fileInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		require.Equal(t, fs.FileMode(0444), fileInfo.Mode().Perm(), "file should have read-only permissions (0444)")

		// Verify we can still read the file through the bucket
		readObjectCloser, err := bucket.Get(ctx, "test.txt")
		require.NoError(t, err)
		require.NoError(t, readObjectCloser.Close())
	})

	t.Run("read_only_files_atomic", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tempDir := t.TempDir()

		// Create bucket with read-only files option
		bucket, err := storageos.NewProvider().NewReadWriteBucket(
			tempDir,
			storageos.ReadWriteBucketWithReadOnlyFiles(),
		)
		require.NoError(t, err)

		// Write a file with atomic option
		writeObjectCloser, err := bucket.Put(ctx, "test.txt", storage.PutWithAtomic())
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte("test data"))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())

		// Check file permissions
		filePath := filepath.Join(tempDir, "test.txt")
		fileInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		require.Equal(t, fs.FileMode(0444), fileInfo.Mode().Perm(), "file should have read-only permissions (0444)")

		// Verify we can still read the file through the bucket
		readObjectCloser, err := bucket.Get(ctx, "test.txt")
		require.NoError(t, err)
		require.NoError(t, readObjectCloser.Close())
	})

	t.Run("read_only_files_nested_directories", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tempDir := t.TempDir()

		// Create bucket with read-only files option
		bucket, err := storageos.NewProvider().NewReadWriteBucket(
			tempDir,
			storageos.ReadWriteBucketWithReadOnlyFiles(),
		)
		require.NoError(t, err)

		// Write a file in nested directories
		writeObjectCloser, err := bucket.Put(ctx, "subdir/nested/test.txt", storage.PutWithAtomic())
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte("test data"))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())

		// Check file permissions
		filePath := filepath.Join(tempDir, "subdir", "nested", "test.txt")
		fileInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		require.Equal(t, fs.FileMode(0444), fileInfo.Mode().Perm(), "file should have read-only permissions (0444)")
	})

	t.Run("normal_files_without_read_only_option", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tempDir := t.TempDir()

		// Create bucket WITHOUT read-only files option
		bucket, err := storageos.NewProvider().NewReadWriteBucket(tempDir)
		require.NoError(t, err)

		// Write a file
		writeObjectCloser, err := bucket.Put(ctx, "test.txt")
		require.NoError(t, err)
		_, err = writeObjectCloser.Write([]byte("test data"))
		require.NoError(t, err)
		require.NoError(t, writeObjectCloser.Close())

		// Check file permissions - should NOT be 0444
		filePath := filepath.Join(tempDir, "test.txt")
		fileInfo, err := os.Stat(filePath)
		require.NoError(t, err)
		require.NotEqual(t, fs.FileMode(0444), fileInfo.Mode().Perm(), "file should NOT have read-only permissions when option is not set")
	})

	t.Run("read_only_files_multiple_files", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		tempDir := t.TempDir()

		// Create bucket with read-only files option
		bucket, err := storageos.NewProvider().NewReadWriteBucket(
			tempDir,
			storageos.ReadWriteBucketWithReadOnlyFiles(),
		)
		require.NoError(t, err)

		// Write multiple files
		files := []string{"file1.txt", "dir/file2.txt", "dir/nested/file3.txt"}
		for _, file := range files {
			writeObjectCloser, err := bucket.Put(ctx, file, storage.PutWithAtomic())
			require.NoError(t, err)
			_, err = writeObjectCloser.Write([]byte("test data"))
			require.NoError(t, err)
			require.NoError(t, writeObjectCloser.Close())
		}

		// Check all files have read-only permissions
		for _, file := range files {
			filePath := filepath.Join(tempDir, filepath.FromSlash(file))
			fileInfo, err := os.Stat(filePath)
			require.NoError(t, err)
			require.Equal(t, fs.FileMode(0444), fileInfo.Mode().Perm(), "file %s should have read-only permissions (0444)", file)
		}
	})
}

func testNewReadBucket(t *testing.T, dirPath string, storageosProvider storageos.Provider) (storage.ReadBucket, storagetesting.GetExternalPathFunc) {
	osBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	return osBucket, func(t *testing.T, rootPath string, path string) string {
		// Join calls Clean
		return normalpath.Unnormalize(normalpath.Join(rootPath, path))
	}
}

func testNewWriteBucket(t *testing.T, storageosProvider storageos.Provider) storage.WriteBucket {
	tmpDir := t.TempDir()
	osBucket, err := storageosProvider.NewReadWriteBucket(
		tmpDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	return osBucket
}

func testWriteBucketToReadBucket(t *testing.T, writeBucket storage.WriteBucket) storage.ReadBucket {
	// hacky
	readWriteBucket, ok := writeBucket.(storage.ReadWriteBucket)
	require.True(t, ok)
	return readWriteBucket
}
