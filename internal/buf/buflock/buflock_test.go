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

package buflock

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/internal/pkg/encoding"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestReadConfigV1Beta1(t *testing.T) {
	testReadConfig(t, "v1beta1")
}

func TestReadConfigV1(t *testing.T) {
	testReadConfig(t, "v1")
}

func testReadConfig(t *testing.T, version string) {
	successConfig := &Config{
		Dependencies: []Dependency{
			{
				Remote:     "bufbuild.test",
				Owner:      "acme",
				Repository: "weather",
				Commit:     "e9191fcdc2294e2f8f3b82c528fc90a8",
				Digest:     bufmoduletesting.TestDigest,
				Branch:     "main",
				CreateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	ctx := context.Background()
	provider := storageos.NewProvider()
	readBucket, err := provider.NewReadWriteBucket(filepath.Join("testdata", version, "success"))
	require.NoError(t, err)
	config, err := ReadConfig(ctx, readBucket)
	require.NoError(t, err)
	require.Equal(t, successConfig, config)

	readBucket, err = provider.NewReadWriteBucket(filepath.Join("testdata", version, "failure"))
	require.NoError(t, err)
	_, err = ReadConfig(ctx, readBucket)
	require.Error(t, err)
}

func TestWriteReadConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	testConfig := &Config{
		Dependencies: []Dependency{
			{
				Remote:     "buf.build",
				Owner:      "test1",
				Repository: "foob1",
				Commit:     bufmoduletesting.TestCommit,
				Digest:     bufmoduletesting.TestDigest,
				Branch:     "barr",
				CreateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				Remote:     "buf.build",
				Owner:      "test2",
				Repository: "foob2",
				Commit:     bufmoduletesting.TestCommit,
				Digest:     bufmoduletesting.TestDigest,
				Branch:     "main", // Can't import bufmodule here without causing an import cycle
				CreateTime: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	err = WriteConfig(context.Background(), readWriteBucket, testConfig)
	require.NoError(t, err)

	readConfig, err := ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)
}

func TestWriteReadEmptyConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	err = WriteConfig(context.Background(), readWriteBucket, &Config{})
	require.NoError(t, err)

	readConfig, err := ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, &Config{}, readConfig)
}

// TODO: Write fuzz tester for the invariant ReadConfig(WriteConfig(file)) == file.

func TestParseV1Beta1Config(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	testConfig := &Config{
		Dependencies: []Dependency{
			{
				Remote:     "buf.build",
				Owner:      "test1",
				Repository: "foob1",
				Commit:     bufmoduletesting.TestCommit,
				Digest:     bufmoduletesting.TestDigest,
				Branch:     "barr",
				CreateTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				Remote:     "buf.build",
				Owner:      "test2",
				Repository: "foob2",
				Commit:     bufmoduletesting.TestCommit,
				Digest:     bufmoduletesting.TestDigest,
				Branch:     "main", // Can't import bufmodule here without causing an import cycle
				CreateTime: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	writeObjectCloser, err := readWriteBucket.Put(context.Background(), ExternalConfigFilePath)
	require.NoError(t, err)
	v1beta1Config := &ExternalConfigV1Beta1{
		Deps: []ExternalConfigDependencyV1Beta1{
			ExternalConfigDependencyV1Beta1(testConfig.Dependencies[0]),
			ExternalConfigDependencyV1Beta1(testConfig.Dependencies[1]),
		},
	}
	err = encoding.NewYAMLEncoder(writeObjectCloser).Encode(v1beta1Config)
	require.NoError(t, err)

	readConfig, err := ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)
}

func TestParseNoConfig(t *testing.T) {
	t.Parallel()
	emptyReadBucket, err := storagemem.NewReadBucketBuilder().ToReadBucket()
	require.NoError(t, err)
	readConfig, err := ReadConfig(context.Background(), emptyReadBucket)
	require.NoError(t, err)
	require.Empty(t, readConfig)
}

func TestParseIncompleteConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	testConfig := &Config{
		Dependencies: []Dependency{
			{
				Remote:     "buf.build",
				Owner:      "test1",
				Repository: "foob1",
				Commit:     bufmoduletesting.TestCommit,
			},
		},
	}
	configBytes, err := encoding.MarshalYAML(&ExternalConfigV1{
		Version: V1Version,
		Deps: []ExternalConfigDependencyV1{
			ExternalConfigDependencyV1(testConfig.Dependencies[0]),
		},
	})
	require.NoError(t, err)
	err = storage.PutPath(context.Background(), readWriteBucket, ExternalConfigFilePath, configBytes)
	require.NoError(t, err)

	readConfig, err := ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)

	err = WriteConfig(context.Background(), readWriteBucket, readConfig)
	require.NoError(t, err)

	// And again after using the proper WriteConfig
	readConfig, err = ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)
}
