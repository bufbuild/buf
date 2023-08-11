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

package buflock_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfigV1Beta1(t *testing.T) {
	t.Parallel()
	testReadConfig(t, "v1beta1")
}

func TestReadConfigV1(t *testing.T) {
	t.Parallel()
	testReadConfig(t, "v1")
}

func testReadConfig(t *testing.T, version string) {
	successConfig := &buflock.Config{
		Dependencies: []buflock.Dependency{
			{
				Remote:     "bufbuild.test",
				Owner:      "acme",
				Repository: "weather",
				Commit:     "e9191fcdc2294e2f8f3b82c528fc90a8",
			},
		},
	}
	ctx := context.Background()
	provider := storageos.NewProvider()
	readBucket, err := provider.NewReadWriteBucket(filepath.Join("testdata", version, "success"))
	require.NoError(t, err)
	config, err := buflock.ReadConfig(ctx, readBucket)
	require.NoError(t, err)
	require.Equal(t, successConfig, config)

	readBucket, err = provider.NewReadWriteBucket(filepath.Join("testdata", version, "failure"))
	require.NoError(t, err)
	_, err = buflock.ReadConfig(ctx, readBucket)
	require.Error(t, err)
}

func TestWriteReadConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	testConfig := &buflock.Config{
		Dependencies: []buflock.Dependency{
			{
				Remote:     "buf.build",
				Owner:      "test1",
				Repository: "foob1",
				Commit:     bufmoduletesting.TestCommit,
			},
			{
				Remote:     "buf.build",
				Owner:      "test2",
				Repository: "foob2",
				Commit:     bufmoduletesting.TestCommit,
			},
		},
	}
	err = buflock.WriteConfig(context.Background(), readWriteBucket, testConfig)
	require.NoError(t, err)

	readConfig, err := buflock.ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)
}

func TestWriteReadEmptyConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	err = buflock.WriteConfig(context.Background(), readWriteBucket, &buflock.Config{})
	require.NoError(t, err)

	readConfig, err := buflock.ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, &buflock.Config{}, readConfig)
}

// TODO: Write fuzz tester for the invariant ReadConfig(WriteConfig(file)) == file.

func TestParseV1Beta1Config(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	testConfig := &buflock.Config{
		Dependencies: []buflock.Dependency{
			{
				Remote:     "buf.build",
				Owner:      "test1",
				Repository: "foob1",
				Commit:     bufmoduletesting.TestCommit,
			},
			{
				Remote:     "buf.build",
				Owner:      "test2",
				Repository: "foob2",
				Commit:     bufmoduletesting.TestCommit,
			},
		},
	}
	writeObjectCloser, err := readWriteBucket.Put(context.Background(), buflock.ExternalConfigFilePath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, writeObjectCloser.Close())
	})
	v1beta1Config := &buflock.ExternalConfigV1Beta1{
		Deps: []buflock.ExternalConfigDependencyV1Beta1{
			buflock.ExternalConfigDependencyV1Beta1ForDependency(testConfig.Dependencies[0]),
			buflock.ExternalConfigDependencyV1Beta1ForDependency(testConfig.Dependencies[1]),
		},
	}
	err = encoding.NewYAMLEncoder(writeObjectCloser).Encode(v1beta1Config)
	require.NoError(t, err)

	readConfig, err := buflock.ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)
}

func TestParseNoConfig(t *testing.T) {
	t.Parallel()
	emptyReadBucket := storagemem.NewReadWriteBucket()
	readConfig, err := buflock.ReadConfig(context.Background(), emptyReadBucket)
	require.NoError(t, err)
	require.Empty(t, readConfig)
}

func TestParseIncompleteConfig(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	readWriteBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDir)
	require.NoError(t, err)
	testConfig := &buflock.Config{
		Dependencies: []buflock.Dependency{
			{
				Remote:     "buf.build",
				Owner:      "test1",
				Repository: "foob1",
				Commit:     bufmoduletesting.TestCommit,
			},
		},
	}
	configBytes, err := encoding.MarshalYAML(&buflock.ExternalConfigV1{
		Version: buflock.V1Version,
		Deps: []buflock.ExternalConfigDependencyV1{
			buflock.ExternalConfigDependencyV1ForDependency(testConfig.Dependencies[0]),
		},
	})
	require.NoError(t, err)
	err = storage.PutPath(context.Background(), readWriteBucket, buflock.ExternalConfigFilePath, configBytes)
	require.NoError(t, err)

	readConfig, err := buflock.ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)

	err = buflock.WriteConfig(context.Background(), readWriteBucket, readConfig)
	require.NoError(t, err)

	// And again after using the proper WriteConfig
	readConfig, err = buflock.ReadConfig(context.Background(), readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, testConfig, readConfig)
}

func TestDependencyForExternalConfigDependencyV1(t *testing.T) {
	t.Parallel()
	digester, err := manifest.NewDigester(manifest.DigestTypeShake256)
	require.NoError(t, err)
	nullDigest, err := digester.Digest(&bytes.Buffer{})
	require.NoError(t, err)
	testDependencyForExternalConfigDependencyV1(
		t,
		"typical",
		buflock.ExternalConfigDependencyV1{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
			Digest:     nullDigest.String(),
		},
		buflock.Dependency{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
			Digest:     nullDigest.String(),
		},
	)
	testDependencyForExternalConfigDependencyV1(
		t,
		"no digest",
		buflock.ExternalConfigDependencyV1{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
		},
		buflock.Dependency{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
		},
	)
	testDependencyForExternalConfigDependencyV1(
		t,
		"filter out b1 hash",
		buflock.ExternalConfigDependencyV1{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
			Digest:     "b1-gLO3B_5ClhdU52w1gMOxk4GokvCoM1OqjarxMfjStGQ=",
		},
		buflock.Dependency{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
		},
	)
	testDependencyForExternalConfigDependencyV1(
		t,
		"filter out b3 hash",
		buflock.ExternalConfigDependencyV1{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
			Digest:     "b3-j7iu4iVzYQUFr97mbq2PNAlM5UjEnjtwEas0q7g4DVM",
		},
		buflock.Dependency{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
		},
	)
	testDependencyForExternalConfigDependencyV1(
		t,
		"don't filter out non-b1/b3 invalid hashes",
		buflock.ExternalConfigDependencyV1{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
			Digest:     "shake256:abcd",
		},
		buflock.Dependency{
			Remote:     "remote",
			Owner:      "owner",
			Repository: "repository",
			Commit:     "aabbccd",
			Digest:     "shake256:abcd",
		},
	)
}

func testDependencyForExternalConfigDependencyV1(
	t *testing.T,
	desc string,
	in buflock.ExternalConfigDependencyV1,
	expect buflock.Dependency,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		actual := buflock.DependencyForExternalConfigDependencyV1(in)
		assert.Equal(t, expect, actual)
	})
}
