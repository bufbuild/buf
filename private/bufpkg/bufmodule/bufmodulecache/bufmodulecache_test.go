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

package bufmodulecache

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestore"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCacheBasicDir(t *testing.T) {
	testCacheBasic(t, false)
}

func TestCacheBasicTar(t *testing.T) {
	testCacheBasic(t, true)
}

func testCacheBasic(t *testing.T, tar bool) {
	ctx := context.Background()

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
	var moduleDataStoreOptions []bufmodulestore.ModuleDataStoreOption
	if tar {
		moduleDataStoreOptions = append(
			moduleDataStoreOptions,
			bufmodulestore.ModuleDataStoreWithTar(),
		)
	}
	cacheProvider := newModuleDataProvider(
		zap.NewNop(),
		bsrProvider,
		bufmodulestore.NewModuleDataStore(
			zap.NewNop(),
			storagemem.NewReadWriteBucket(),
			moduleDataStoreOptions...,
		),
	)

	moduleRefMod1, err := bufmodule.NewModuleRef("buf.build", "foo", "mod1", "")
	require.NoError(t, err)
	moduleRefMod2, err := bufmodule.NewModuleRef("buf.build", "foo", "mod2", "")
	require.NoError(t, err)
	moduleRefMod3, err := bufmodule.NewModuleRef("buf.build", "foo", "mod3", "")
	require.NoError(t, err)
	moduleKeys, err := bufmodule.GetModuleKeysForModuleRefs(
		ctx,
		bsrProvider,
		moduleRefMod1,
		// Switching order on purpose.
		moduleRefMod3,
		moduleRefMod2,
	)
	require.NoError(t, err)

	moduleDatas, err := bufmodule.GetModuleDatasForModuleKeys(
		ctx,
		cacheProvider,
		moduleKeys...,
	)
	require.NoError(t, err)
	require.Equal(t, 3, cacheProvider.getModuleKeysRetrieved())
	require.Equal(t, 0, cacheProvider.getModuleKeysHit())
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
	moduleDatas, err = bufmodule.GetModuleDatasForModuleKeys(
		ctx,
		cacheProvider,
		moduleKeys...,
	)
	require.NoError(t, err)
	require.Equal(t, 6, cacheProvider.getModuleKeysRetrieved())
	require.Equal(t, 3, cacheProvider.getModuleKeysHit())
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