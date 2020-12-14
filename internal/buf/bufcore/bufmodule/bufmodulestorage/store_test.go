// Copyright 2020 Buf Technologies, Inc.
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

package bufmodulestorage_test

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulestorage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBasic(t *testing.T) {
	ctx := context.Background()

	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestData)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)

	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	moduleStore := bufmodulestorage.NewStore(zap.NewNop(), readWriteBucket)

	keys, err := moduleStore.AllKeys(ctx)
	require.NoError(t, err)
	require.Empty(t, keys)

	key := bufmodulestorage.Key{"some", "path"}
	err = moduleStore.Put(
		context.Background(),
		key,
		module,
	)
	require.NoError(t, err)
	paths, err := storage.AllPaths(ctx, readWriteBucket, "")
	require.NoError(t, err)
	require.Equal(t, []string{"v1/some/path/module.bin.zst"}, paths)
	keys, err = moduleStore.AllKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, []bufmodulestorage.Key{key}, keys)

	getModule, err := moduleStore.Get(ctx, key)
	require.NoError(t, err)
	getReadBucketBuilder := storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err := getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)

	// Exclude non-proto files for the diff check
	filteredReadBucket := storage.MapReadBucket(getReadBucket, storage.MatchPathExt(".proto"))
	diff, err := storage.DiffBytes(ctx, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}
