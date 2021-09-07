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

	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReaderBasic(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmodule.NewModulePin(
		"buf.build",
		"foob",
		"bar",
		"main",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigest,
		time.Now(),
	)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestData)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)

	delegateDataReadWriteBucket, delegateSumReadWriteBucket, delegateFileLocker := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), delegateDataReadWriteBucket, delegateSumReadWriteBucket, delegateFileLocker)
	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)
	// the delegate uses the cache we just populated
	delegateModuleReader := newModuleReader(
		zap.NewNop(),
		delegateDataReadWriteBucket,
		delegateSumReadWriteBucket,
		moduleCacher,
		WithFileLocker(delegateFileLocker),
	)

	// the main does not, so there will be a cache miss
	mainDataReadWriteBucket, mainSumReadWriteBucket, mainFileLocker := newTestDataSumBucketsAndLocker(t)
	moduleReader := newModuleReader(
		zap.NewNop(),
		mainDataReadWriteBucket,
		mainSumReadWriteBucket,
		delegateModuleReader,
		WithFileLocker(mainFileLocker),
	)
	getModule, err := moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	testFile1HasNoExternalPath(t, ctx, getModule)
	getReadBucketBuilder := storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err := getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	// Verify that the buf.lock file was created.
	exists, err := storage.Exists(ctx, getReadBucket, buflock.ExternalConfigFilePath)
	require.NoError(t, err)
	require.True(t, exists)

	// Exclude non-proto files for the diff check
	filteredReadBucket := storage.MapReadBucket(getReadBucket, storage.MatchPathExt(".proto"))
	diff, err := storage.DiffBytes(ctx, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))

	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	testFile1HasNoExternalPath(t, ctx, getModule)
	require.Equal(t, 2, moduleReader.getCount())
	require.Equal(t, 1, moduleReader.getCacheHits())

	// put some data that will not match the sum and make sure that we have a cache miss
	require.NoError(t, storage.PutPath(ctx, mainDataReadWriteBucket, normalpath.Join(newCacheKey(modulePin), "1234.proto"), []byte("foo")))
	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadBucketBuilder = storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err = getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	// Exclude non-proto files for the diff check
	filteredReadBucket = storage.MapReadBucket(getReadBucket, storage.MatchPathExt(".proto"))
	diff, err = storage.DiffBytes(ctx, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
	require.Equal(t, 3, moduleReader.getCount())
	require.Equal(t, 1, moduleReader.getCacheHits())

	_, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	require.Equal(t, 4, moduleReader.getCount())
	require.Equal(t, 2, moduleReader.getCacheHits())

	// overwrite the sum file and make sure that we have a cache miss
	require.NoError(t, storage.PutPath(ctx, mainSumReadWriteBucket, newCacheKey(modulePin), []byte("foo")))
	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadBucketBuilder = storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err = getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	// Exclude non-proto files for the diff check
	filteredReadBucket = storage.MapReadBucket(getReadBucket, storage.MatchPathExt(".proto"))
	diff, err = storage.DiffBytes(ctx, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
	require.Equal(t, 5, moduleReader.getCount())
	require.Equal(t, 2, moduleReader.getCacheHits())

	_, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	require.Equal(t, 6, moduleReader.getCount())
	require.Equal(t, 3, moduleReader.getCacheHits())

	// delete the sum file and make sure that we have a cache miss
	require.NoError(t, mainSumReadWriteBucket.Delete(ctx, newCacheKey(modulePin)))
	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadBucketBuilder = storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err = getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	// Exclude non-proto files for the diff check
	filteredReadBucket = storage.MapReadBucket(getReadBucket, storage.MatchPathExt(".proto"))
	diff, err = storage.DiffBytes(ctx, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
	require.Equal(t, 7, moduleReader.getCount())
	require.Equal(t, 3, moduleReader.getCacheHits())
}

func TestCacherBasic(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmodule.NewModulePin(
		"buf.build",
		"foob",
		"bar",
		"main",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigest,
		time.Now(),
	)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestData)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)

	dataReadWriteBucket, sumReadWriteBucket, fileLocker := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), dataReadWriteBucket, sumReadWriteBucket, fileLocker)
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
	exists, err := storage.Exists(ctx, getReadBucket, buflock.ExternalConfigFilePath)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestModuleReaderCacherWithDocumentation(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmodule.NewModulePin(
		"buf.build",
		"foob",
		"bar",
		"main",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigest,
		time.Now(),
	)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestDataWithDocumentation)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)

	dataReadWriteBucket, sumReadWriteBucket, fileLocker := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), dataReadWriteBucket, sumReadWriteBucket, fileLocker)
	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)
	module, err = moduleCacher.GetModule(ctx, modulePin)
	require.NoError(t, err)
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	require.NoError(t, bufmodule.ModuleToBucket(ctx, module, readBucketBuilder))
	readBucket, err = readBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	// Verify that the buf.md file was created.
	exists, err := storage.Exists(ctx, readBucket, bufmodule.DocumentationFilePath)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, bufmoduletesting.TestModuleDocumentation, module.Documentation())
}

func newTestDataSumBucketsAndLocker(t *testing.T) (storage.ReadWriteBucket, storage.ReadWriteBucket, filelock.Locker) {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	dataReadWriteBucket, err := storageosProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	sumReadWriteBucket, err := storageosProvider.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	fileLocker, err := filelock.NewLocker(t.TempDir())
	require.NoError(t, err)
	return dataReadWriteBucket, sumReadWriteBucket, fileLocker
}

// This is to make sure that if we get a file from the cache, we strip the
// external path via storage.NoExternalPathReadBucket.
func testFile1HasNoExternalPath(t *testing.T, ctx context.Context, module bufmodule.Module) {
	file1ModuleFile, err := module.GetModuleFile(ctx, bufmoduletesting.TestFile1Path)
	require.NoError(t, err)
	require.Equal(t, bufmoduletesting.TestFile1Path, file1ModuleFile.Path())
	require.Equal(t, bufmoduletesting.TestFile1Path, file1ModuleFile.ExternalPath())
	require.NoError(t, file1ModuleFile.Close())
}
