// Copyright 2020-2022 Buf Technologies, Inc.
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

package internaltesting

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

// CopyReadBucketToTempDir copies the ReadBucket to a new temporary directory.
//
// Returns the temporary directory path and a ReadWriteBucket of the temporary directory.
func CopyReadBucketToTempDir(
	ctx context.Context,
	tb testing.TB,
	storageosProvider storageos.Provider,
	readBucket storage.ReadBucket,
) (string, storage.ReadWriteBucket) {
	// Copy to temporary directory to avoid writing to filesystem
	tempDirPath := tb.TempDir()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(tempDirPath)
	require.NoError(tb, err)
	_, err = storage.Copy(ctx, readBucket, readWriteBucket)
	require.NoError(tb, err)
	return tempDirPath, readWriteBucket
}

// NewEnvFunc returns a new env func for testing.
func NewEnvFunc(tb testing.TB) func(string) map[string]string {
	tempDirPath := tb.TempDir()
	return func(use string) map[string]string {
		return map[string]string{
			useEnvVar(use, "CACHE_DIR"): tempDirPath,
			"PATH":                      os.Getenv("PATH"),
		}
	}
}

func useEnvVar(use string, suffix string) string {
	return strings.ToUpper(use) + "_" + suffix
}
