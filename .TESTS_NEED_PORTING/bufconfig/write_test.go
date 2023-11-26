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

package bufconfig

import (
	"context"
	"io"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestWriteConfigSuccess(t *testing.T) {
	t.Parallel()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	writeConfigOptions := []WriteConfigOption{
		WriteConfigWithVersion(V1Version),
		WriteConfigWithBreakingConfig(
			&bufbreakingconfig.Config{
				Version: V1Version,
				Use:     []string{"FILE"},
			},
		),
		WriteConfigWithLintConfig(
			&buflintconfig.Config{
				Version: V1Version,
				Use:     []string{"DEFAULT"},
			},
		),
	}
	err = WriteConfig(context.Background(), readWriteBucket, writeConfigOptions...)
	require.NoError(t, err)
	configReadObjectCloser, err := readWriteBucket.Get(context.Background(), ExternalConfigV1FilePath)
	require.NoError(t, err)
	configBytes, err := io.ReadAll(configReadObjectCloser)
	require.NoError(t, err)
	require.Equal(
		t,
		string(configBytes),
		`version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
`,
	)
	require.NoError(t, configReadObjectCloser.Close())
}

func TestWriteConfigMismatchedConfigVersions(t *testing.T) {
	t.Parallel()
	t.Run("invalid breaking config", func(t *testing.T) {
		t.Parallel()
		storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
		readWriteBucket, err := storageosProvider.NewReadWriteBucket(t.TempDir())
		require.NoError(t, err)
		writeConfigOptions := []WriteConfigOption{
			WriteConfigWithVersion(V1Beta1Version),
			WriteConfigWithBreakingConfig(&bufbreakingconfig.Config{
				Version: V1Version,
				Use:     []string{"FILE"},
			}),
		}
		err = WriteConfig(context.Background(), readWriteBucket, writeConfigOptions...)
		require.Error(t, err)
		require.Equal(t, err.Error(), `version "v1" found for breaking config, does not match top level config version: "v1beta1"`)
	})
	t.Run("invalid lint config", func(t *testing.T) {
		t.Parallel()
		storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
		readWriteBucket, err := storageosProvider.NewReadWriteBucket(t.TempDir())
		require.NoError(t, err)
		writeConfigOptions := []WriteConfigOption{
			WriteConfigWithVersion(V1Beta1Version),
			WriteConfigWithLintConfig(&buflintconfig.Config{
				Version: V1Version,
				Use:     []string{"DEFAULT"},
			}),
		}
		err = WriteConfig(context.Background(), readWriteBucket, writeConfigOptions...)
		require.Error(t, err)
		require.Equal(t, err.Error(), `version "v1" found for lint config, does not match top level config version: "v1beta1"`)
	})
}
