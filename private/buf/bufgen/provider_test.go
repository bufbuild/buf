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

package bufgen

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

const testConfigFileData = `
version: %s
plugins:
  - name: go
    out: private/gen/proto/go
    opt: paths=source_relative
`

func TestProviderV1Beta1(t *testing.T) {
	testProvider(t, "v1beta1")
}

func TestProviderV1(t *testing.T) {
	testProvider(t, "v1")
}

func TestProviderError(t *testing.T) {
	storageosProvider := storageos.NewProvider()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(".")
	require.NoError(t, err)

	provider := NewProvider(zap.NewNop())
	_, err = provider.GetConfig(context.Background(), readWriteBucket)
	require.True(t, storage.IsNotExist(err))
}

func testProvider(t *testing.T, version string) {
	storageosProvider := storageos.NewProvider()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join("testdata", version))
	require.NoError(t, err)

	nopLogger := zap.NewNop()
	provider := NewProvider(zap.NewNop())
	actual, err := provider.GetConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)

	emptyBucket, err := storagemem.NewReadBucket(nil)
	require.NoError(t, err)
	expected, err := ReadConfig(context.Background(), nopLogger, provider, emptyBucket, ReadConfigWithOverride(fmt.Sprintf(testConfigFileData, version)))
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
