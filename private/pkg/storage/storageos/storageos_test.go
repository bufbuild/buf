// Copyright 2020-2024 Buf Technologies, Inc.
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
	"context"
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
		ctx := context.Background()
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
		ctx := context.Background()
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
