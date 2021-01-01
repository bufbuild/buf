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

package bufmodulecache

import (
	"context"
	"testing"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulestorage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReaderBasic(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmodule.NewModulePin(
		"buf.build",
		"foo",
		"bar",
		"v1",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigest,
		time.Now(),
	)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestData)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)

	moduleStore := newTestModuleStore(t)
	moduleCacher := newModuleCacher(moduleStore)
	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)
	// the delegate uses the cache we just populated
	delegateModuleReader := newModuleReader(zap.NewNop(), moduleStore, moduleCacher)

	// the main does not, so there will be a cache miss
	moduleReader := newModuleReader(zap.NewNop(), newTestModuleStore(t), delegateModuleReader)
	getModule, err := moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadBucketBuilder := storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err := getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	// Verify that the buf.lock file was created.
	exists, err := storage.Exists(ctx, getReadBucket, bufmodule.LockFilePath)
	require.NoError(t, err)
	require.True(t, exists)

	// Exclude non-proto files for the diff check
	filteredReadBucket := storage.MapReadBucket(getReadBucket, storage.MatchPathExt(".proto"))
	diff, err := storage.DiffBytes(ctx, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))

	_, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)

	require.Equal(t, 2, moduleReader.getCount())
	require.Equal(t, 1, moduleReader.getCacheHits())
}

func TestCacherBasic(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmodule.NewModulePin(
		"buf.build",
		"foo",
		"bar",
		"v1",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigest,
		time.Now(),
	)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestData)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)

	moduleCacher := newModuleCacher(newTestModuleStore(t))
	_, err = moduleCacher.GetModule(ctx, modulePin)
	require.True(t, storage.IsNotExist(err))

	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)

	getModule, err := moduleCacher.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadBucketBuilder := storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err := getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	exists, err := storage.Exists(ctx, getReadBucket, bufmodule.LockFilePath)
	require.NoError(t, err)
	require.True(t, exists)
}

func newTestModuleStore(t *testing.T) bufmodulestorage.Store {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	return bufmodulestorage.NewStore(zap.NewNop(), readWriteBucket)
}
