// Copyright 2020-2021 Buf Technologies, Inc.
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

package storagemem_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/storage/storagetesting"
	"github.com/stretchr/testify/require"
)

var storagetestingDirPath = filepath.Join("..", "storagetesting")

func TestMemNoCompression(t *testing.T) {
	t.Parallel()
	testMem(t, false)
}

func TestMemCompression(t *testing.T) {
	t.Parallel()
	testMem(t, true)
}

func testMem(t *testing.T, compression bool) {
	storagetesting.RunTestSuite(
		t,
		storagetestingDirPath,
		func(
			t *testing.T,
			dirPath string,
			storageosProvider storageos.Provider,
		) (storage.ReadBucket, storagetesting.GetExternalPathFunc) {
			return testNewReadBucket(t, dirPath, storageosProvider, compression)
		},
		func(
			t *testing.T,
			storageosProvider storageos.Provider,
		) storage.WriteBucket {
			return testNewWriteBucket(t, storageosProvider, compression)
		},
		testWriteBucketToReadBucket,
	)
}

func testNewReadBucket(
	t *testing.T,
	dirPath string,
	storageosProvider storageos.Provider,
	compression bool,
) (storage.ReadBucket, storagetesting.GetExternalPathFunc) {
	osBucket, err := storageosProvider.NewReadWriteBucket(
		dirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	readWriteBucket := storagemem.NewReadWriteBucket(testGetReadWriteBucketOptions(compression)...)
	_, err = storage.Copy(
		context.Background(),
		osBucket,
		readWriteBucket,
		storage.CopyWithExternalPaths(),
	)
	require.NoError(t, err)
	return readWriteBucket, func(t *testing.T, rootPath string, path string) string {
		// Join calls Clean
		return normalpath.Unnormalize(normalpath.Join(rootPath, path))
	}
}

func testNewWriteBucket(
	t *testing.T,
	storageosProvider storageos.Provider,
	compression bool,
) storage.WriteBucket {
	return storagemem.NewReadWriteBucket(testGetReadWriteBucketOptions(compression)...)
}

func testWriteBucketToReadBucket(t *testing.T, writeBucket storage.WriteBucket) storage.ReadBucket {
	// hacky
	readWriteBucket, ok := writeBucket.(storage.ReadWriteBucket)
	require.True(t, ok)
	return readWriteBucket
}

func testGetReadWriteBucketOptions(compression bool) []storagemem.ReadWriteBucketOption {
	if compression {
		return []storagemem.ReadWriteBucketOption{
			storagemem.ReadWriteBucketWithCompression(),
		}
	}
	return nil
}
