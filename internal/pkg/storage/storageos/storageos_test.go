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

package storageos_test

import (
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/internal/storagetesting"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

var storagetestingDirPath = filepath.Join("..", "internal", "storagetesting")

func TestOS(t *testing.T) {
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
	return osBucket
}

func testNewWriteBucketAndCleanup(t *testing.T) storage.WriteBucket {
	tmpDir := t.TempDir()
	osBucket, err := storageos.NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	return osBucket
}

func testWriteBucketToReadBucket(t *testing.T, writeBucket storage.WriteBucket) storage.ReadBucket {
	// hacky
	readWriteBucket, ok := writeBucket.(storage.ReadWriteBucket)
	require.True(t, ok)
	return readWriteBucket
}
