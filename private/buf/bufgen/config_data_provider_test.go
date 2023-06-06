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

package bufgen

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testConfigFileData = `version: %s
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

	provider := NewConfigDataProvider(zap.NewNop())
	_, _, err = provider.GetConfigData(context.Background(), readWriteBucket)
	require.True(t, storage.IsNotExist(err))
}

func testProvider(t *testing.T, version string) {
	storageosProvider := storageos.NewProvider()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join("testdata", version))
	require.NoError(t, err)

	provider := NewConfigDataProvider(zap.NewNop())
	actual, _, err := provider.GetConfigData(context.Background(), readWriteBucket)
	require.NoError(t, err)
	expected := []byte(fmt.Sprintf(testConfigFileData, version))
	assert.Equal(t, expected, actual)

	// TODO: write tests (not necessarily in this file) on the unmarshalled configs, for both V1 and V2.
}
