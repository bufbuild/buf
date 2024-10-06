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
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestore"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/stretchr/testify/require"
)

func TestCommitProviderForModuleKeyBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	bsrProvider, moduleKeys := testGetBSRProviderAndModuleKeys(t, ctx)
	logger := slogtestext.NewLogger(t)

	cacheProvider := newCommitProvider(
		logger,
		bsrProvider,
		bufmodulestore.NewCommitStore(
			logger,
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
	logger := slogtestext.NewLogger(t)
	commitKeys, err := slicesext.MapError(moduleKeys, bufmodule.ModuleKeyToCommitKey)
	require.NoError(t, err)

	cacheProvider := newCommitProvider(
		logger,
		bsrProvider,
		bufmodulestore.NewCommitStore(
			logger,
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
	logger := slogtestext.NewLogger(t)

	cacheProvider := newModuleDataProvider(
		logger,
		bsrProvider,
		bufmodulestore.NewModuleDataStore(
			logger,
			storagemem.NewReadWriteBucket(),
			filelock.NewNopLocker(),
		),
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
	t.Parallel()

	bsrProvider, moduleKeys := testGetBSRProviderAndModuleKeys(t, context.Background())
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	logger := slogtestext.NewLogger(t)

	for i := 0; i < 20; i++ {
		require.NoError(t, os.MkdirAll(cacheDir, 0755))
		jobs, err := slicesext.MapError(
			[]int{0, 1, 2, 3, 4},
			func(i int) (func(ctx context.Context) error, error) {
				bucket, err := storageos.NewProvider().NewReadWriteBucket(cacheDir)
				if err != nil {
					return nil, err
				}
				filelocker, err := filelock.NewLocker(
					cacheDir,
					filelock.LockerWithLockRetryDelay(10*time.Millisecond), // Drops test time from ~16s to ~1s
				)
				if err != nil {
					return nil, err
				}
				cacheProvider := newModuleDataProvider(
					logger,
					bsrProvider,
					bufmodulestore.NewModuleDataStore(
						logger,
						bucket,
						filelocker,
					),
				)
				return func(ctx context.Context) error {
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
				}, nil
			},
		)
		require.NoError(t, err)
		require.NoError(t, thread.Parallelize(context.Background(), jobs))
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
