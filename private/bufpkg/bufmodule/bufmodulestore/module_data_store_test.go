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

package bufmodulestore

import (
	"context"
	"os"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/zaputil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestModuleDataStoreBasicDir(t *testing.T) {
	t.Parallel()
	testModuleDataStoreBasic(t, false)
}

// TODO(doria): add test using storageos bucket and filelocker

func TestModuleDataStoreBasicTar(t *testing.T) {
	t.Parallel()
	testModuleDataStoreBasic(t, true)
}

func testModuleDataStoreBasic(t *testing.T, tar bool) {
	ctx := context.Background()
	bucket := storagemem.NewReadWriteBucket()
	var moduleDataStoreOptions []ModuleDataStoreOption
	if tar {
		moduleDataStoreOptions = append(moduleDataStoreOptions, ModuleDataStoreWithTar())
	}
	logger := zaputil.NewLogger(os.Stderr, zapcore.DebugLevel, zaputil.NewTextEncoder())
	moduleDataStore := NewModuleDataStore(logger, bucket, filelock.NewNopLocker(), moduleDataStoreOptions...)
	moduleKeys, moduleDatas := testGetModuleKeysAndModuleDatas(t, ctx)

	foundModuleDatas, notFoundModuleKeys, err := moduleDataStore.GetModuleDatasForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	testRequireModuleDataNamesEqual(t, nil, foundModuleDatas)
	testRequireModuleKeyNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		notFoundModuleKeys,
	)

	err = moduleDataStore.PutModuleDatas(ctx, moduleDatas)
	require.NoError(t, err)

	foundModuleDatas, notFoundModuleKeys, err = moduleDataStore.GetModuleDatasForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	testRequireModuleDataNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		foundModuleDatas,
	)
	testRequireModuleKeyNamesEqual(t, nil, notFoundModuleKeys)

	// Corrupt the cache.
	if tar {
		tarPath, err := getModuleDataStoreTarPath(moduleKeys[0])
		require.NoError(t, err)
		require.NoError(t, storage.PutPath(ctx, bucket, tarPath, []byte("invalid_tar")))
	} else {
		dirPath, err := getModuleDataStoreDirPath(moduleKeys[0])
		require.NoError(t, err)
		require.NoError(
			t,
			storage.PutPath(
				ctx,
				bucket,
				normalpath.Join(
					dirPath,
					externalModuleDataFileName,
				),
				[]byte("invalid_info_json"),
			),
		)
	}
	foundModuleDatas, notFoundModuleKeys, err = moduleDataStore.GetModuleDatasForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	testRequireModuleDataNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		foundModuleDatas,
	)
	testRequireModuleKeyNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
		},
		notFoundModuleKeys,
	)
}

func testGetModuleKeysAndModuleDatas(t *testing.T, ctx context.Context) ([]bufmodule.ModuleKey, []bufmodule.ModuleData) {
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
				"mod3.proto": []byte(
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
	testRequireModuleKeyNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		moduleKeys,
	)
	moduleDatas, err := bsrProvider.GetModuleDatasForModuleKeys(
		ctx,
		moduleKeys,
	)
	require.NoError(t, err)
	testRequireModuleDataNamesEqual(
		t,
		[]string{
			"buf.build/foo/mod1",
			"buf.build/foo/mod3",
			"buf.build/foo/mod2",
		},
		moduleDatas,
	)
	return moduleKeys, moduleDatas
}

func testRequireModuleKeyNamesEqual(t *testing.T, expected []string, actual []bufmodule.ModuleKey) {
	if len(expected) == 0 {
		require.Equal(t, 0, len(actual))
	} else {
		require.Equal(
			t,
			expected,
			slicesext.Map(
				actual,
				func(value bufmodule.ModuleKey) string {
					return value.ModuleFullName().String()
				},
			),
		)
	}
}

func testRequireModuleDataNamesEqual(t *testing.T, expected []string, actual []bufmodule.ModuleData) {
	if len(expected) == 0 {
		require.Equal(t, 0, len(actual))
	} else {
		require.Equal(
			t,
			expected,
			slicesext.Map(
				actual,
				func(value bufmodule.ModuleData) string {
					return value.ModuleKey().ModuleFullName().String()
				},
			),
		)
	}
}
