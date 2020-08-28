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

package storagemem_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/internal/storagetesting"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

var storagetestingDirPath = filepath.Join("..", "internal", "storagetesting")

func TestMem(t *testing.T) {
	storagetesting.RunTestSuite(
		t,
		storagetestingDirPath,
		testNewReadBucket,
		testNewWriteBucketAndCleanup,
		testWriteBucketToReadBucket,
	)
}

func testNewReadBucket(t *testing.T, dirPath string) storage.ReadBucket {
	osBucket, err := storageos.NewReadWriteBucket(dirPath)
	require.NoError(t, err)
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	_, err = storage.Copy(
		context.Background(),
		osBucket,
		readBucketBuilder,
		storage.CopyWithExternalPaths(),
	)
	require.NoError(t, err)
	readBucket, err := readBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	return readBucket
}

func testNewWriteBucketAndCleanup(*testing.T) storage.WriteBucket {
	return storagemem.NewReadBucketBuilder()
}

func testWriteBucketToReadBucket(t *testing.T, writeBucket storage.WriteBucket) storage.ReadBucket {
	// hacky
	readBucketBuilder, ok := writeBucket.(storagemem.ReadBucketBuilder)
	require.True(t, ok)
	readBucket, err := readBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	return readBucket
}
