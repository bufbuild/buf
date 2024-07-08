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

package bufmodulecache

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestore"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func TestCommitProviderForModuleKeyBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	bsrProvider, moduleKeys := testGetBSRProviderAndModuleKeys(t, ctx)

	cacheProvider := newCommitProvider(
		zap.NewNop(),
		bsrProvider,
		bufmodulestore.NewCommitStore(
			zap.NewNop(),
			storagemem.NewReadWriteBucket(),
		),
	)

	commits, err := cacheProvider.GetCommitsForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	require.Equal(t, 3, cacheProvider.byModuleKey.getKeysRetrieved())
	require.Equal(t, 0, cacheProvider.byModuleKey.getKeysHit())
	require.Equal(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		slicesext.Map(
			commits,
			func(commit bufmodule.Commit) string {
				return commit.ModuleKey().ModuleFullName().String()
			},
		),
	)

	moduleKeys[0], moduleKeys[1] = moduleKeys[1], moduleKeys[0]
	commits, err = cacheProvider.GetCommitsForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	require.Equal(t, 6, cacheProvider.byModuleKey.getKeysRetrieved())
	require.Equal(t, 3, cacheProvider.byModuleKey.getKeysHit())
	require.Equal(
		t,
		[]string{
			"buf.build/foo/mod3",
			"buf.build/foo/mod1",
			"buf.build/foo/mod2",
		},
		slicesext.Map(
			commits,
			func(commit bufmodule.Commit) string {
				return commit.ModuleKey().ModuleFullName().String()
			},
		),
	)
}

func TestCommitProviderForCommitKeyBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	bsrProvider, moduleKeys := testGetBSRProviderAndModuleKeys(t, ctx)
	commitKeys, err := slicesext.MapError(moduleKeys, bufmodule.ModuleKeyToCommitKey)
	require.NoError(t, err)

	cacheProvider := newCommitProvider(
		zap.NewNop(),
		bsrProvider,
		bufmodulestore.NewCommitStore(
			zap.NewNop(),
			storagemem.NewReadWriteBucket(),
		),
	)

	commits, err := cacheProvider.GetCommitsForCommitKeys(
		ctx,
		commitKeys,
	)
	require.NoError(t, err)
	require.Equal(t, 3, cacheProvider.byCommitKey.getKeysRetrieved())
	require.Equal(t, 0, cacheProvider.byCommitKey.getKeysHit())
	require.Equal(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		slicesext.Map(
			commits,
			func(commit bufmodule.Commit) string {
				return commit.ModuleKey().ModuleFullName().String()
			},
		),
	)

	commitKeys[0], commitKeys[1] = commitKeys[1], commitKeys[0]
	commits, err = cacheProvider.GetCommitsForCommitKeys(
		ctx,
		commitKeys,
	)
	require.NoError(t, err)
	require.Equal(t, 6, cacheProvider.byCommitKey.getKeysRetrieved())
	require.Equal(t, 3, cacheProvider.byCommitKey.getKeysHit())
	require.Equal(
		t,
		[]string{
			"buf.build/foo/mod3",
			"buf.build/foo/mod1",
			"buf.build/foo/mod2",
		},
		slicesext.Map(
			commits,
			func(commit bufmodule.Commit) string {
				return commit.ModuleKey().ModuleFullName().String()
			},
		),
	)
}

func TestModuleDataProviderBasic(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	bsrProvider, moduleKeys := testGetBSRProviderAndModuleKeys(t, ctx)

	cacheProvider := newModuleDataProvider(
		zap.NewNop(),
		bsrProvider,
		bufmodulestore.NewModuleDataStore(
			zap.NewNop(),
			storagemem.NewReadWriteBucket(),
		),
		nil, // Do not set a file locker for in-mem storage bucket
	)

	moduleDatas, err := cacheProvider.GetModuleDatasForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	require.Equal(t, 3, cacheProvider.getKeysRetrieved())
	require.Equal(t, 0, cacheProvider.getKeysHit())
	require.Equal(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		slicesext.Map(
			moduleDatas,
			func(moduleData bufmodule.ModuleData) string {
				return moduleData.ModuleKey().ModuleFullName().String()
			},
		),
	)

	moduleKeys[0], moduleKeys[1] = moduleKeys[1], moduleKeys[0]
	moduleDatas, err = cacheProvider.GetModuleDatasForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	require.Equal(t, 6, cacheProvider.getKeysRetrieved())
	require.Equal(t, 3, cacheProvider.getKeysHit())
	require.Equal(
		t,
		[]string{
			"buf.build/foo/mod3",
			"buf.build/foo/mod1",
			"buf.build/foo/mod2",
		},
		slicesext.Map(
			moduleDatas,
			func(moduleData bufmodule.ModuleData) string {
				return moduleData.ModuleKey().ModuleFullName().String()
			},
		),
	)
}

func TestConcurrentCacheReadWrite(t *testing.T) {
	t.Skip("expensive cache concurrent test")
	t.Parallel()

	bsrProvider, moduleKeys := testGetBSRProviderAndModuleKeys(t, context.Background())
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	for i := 0; i < 20; i++ {
		require.NoError(t, os.MkdirAll(cacheDir, 0755))
		errs, ctx := errgroup.WithContext(context.Background())

		for j := 0; j < 5; j++ {
			bucket, err := storageos.NewProvider().NewReadWriteBucket(cacheDir)
			require.NoError(t, err)
			filelocker, err := filelock.NewLocker(cacheDir)
			require.NoError(t, err)

			cacheProvider := newModuleDataProvider(
				zap.NewNop(),
				bsrProvider,
				bufmodulestore.NewModuleDataStore(
					zap.NewNop(),
					bucket,
				),
				filelocker,
			)

			errs.Go(func() error {
				moduleDatas, err := cacheProvider.GetModuleDatasForModuleKeys(
					ctx,
					moduleKeys,
				)
				if err != nil {
					return err
				}
				for _, moduleData := range moduleDatas {
					// Calling moduleData.Bucket() checks the digest
					if _, err := moduleData.Bucket(); err != nil {
						return err
					}
				}
				return nil
			})
		}

		assert.NoError(t, errs.Wait()) // Waits for all go routines to finish and returns the first error, if any
		require.NoError(t, os.RemoveAll(cacheDir))
	}
}

func testGetBSRProviderAndModuleKeys(t *testing.T, ctx context.Context) (bufmoduletesting.OmniProvider, []bufmodule.ModuleKey) {
	bsrProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/mod1",
			PathToData: map[string][]byte{
				"mod1.proto": []byte(
					`syntax = proto3; package mod1;`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/mod2",
			PathToData: map[string][]byte{
				"mod2.proto": []byte(
					`syntax = proto3; package mod2; import "mod1.proto";`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/mod3",
			PathToData: map[string][]byte{
				"mod3a.proto": []byte(
					`syntax = proto3; package mod3;`,
				),
				"mod3b.proto": []byte(
					`syntax = proto3; package mod3;`,
				),
			},
		},
	)
	require.NoError(t, err)
	moduleRefMod1, err := bufmodule.NewModuleRef("buf.build", "foo", "mod1", "")
	require.NoError(t, err)
	moduleRefMod2, err := bufmodule.NewModuleRef("buf.build", "foo", "mod2", "")
	require.NoError(t, err)
	moduleRefMod3, err := bufmodule.NewModuleRef("buf.build", "foo", "mod3", "")
	require.NoError(t, err)
	moduleKeys, err := bsrProvider.GetModuleKeysForModuleRefs(
		ctx,
		[]bufmodule.ModuleRef{
			moduleRefMod1,
			// Switching order on purpose.
			moduleRefMod3,
			moduleRefMod2,
		},
		bufmodule.DigestTypeB5,
	)
	require.NoError(t, err)
	require.Equal(t, 3, len(moduleKeys))
	return bsrProvider, moduleKeys
}
