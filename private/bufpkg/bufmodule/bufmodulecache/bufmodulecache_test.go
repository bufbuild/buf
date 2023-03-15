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
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/verbose"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestReaderBasic(t *testing.T) {
	ctx := context.Background()
	runner := command.NewRunner()

	modulePin, err := bufmoduleref.NewModulePin(
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
	module, err := bufmodule.NewModuleForProto(
		ctx,
		bufmoduletesting.TestDataProto,
		bufmodule.ModuleWithModuleIdentity(modulePin),
	)
	require.NoError(t, err)

	delegateDataReadWriteBucket, delegateSumReadWriteBucket, delegateFileLocker := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), delegateDataReadWriteBucket, delegateSumReadWriteBucket, false)
	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)

	deprecationMessage := "this is the deprecation message"

	repositoryClientFactory := fakeRepositoryServiceClientFactory(
		&fakeRepositoryService{
			repository: &registryv1alpha1.Repository{
				Deprecated:         true,
				DeprecationMessage: deprecationMessage,
			},
		},
	)

	// the delegate uses the cache we just populated
	delegateModuleReader := newModuleReader(
		zap.NewNop(),
		verbose.NopPrinter,
		delegateFileLocker,
		delegateDataReadWriteBucket,
		delegateSumReadWriteBucket,
		moduleCacher,
		repositoryClientFactory,
	)

	core, observedLogs := observer.New(zapcore.WarnLevel)
	// the main does not, so there will be a cache miss
	mainDataReadWriteBucket, mainSumReadWriteBucket, mainFileLocker := newTestDataSumBucketsAndLocker(t)
	moduleReader := newModuleReader(
		zap.New(core),
		verbose.NopPrinter,
		mainFileLocker,
		mainDataReadWriteBucket,
		mainSumReadWriteBucket,
		delegateModuleReader,
		repositoryClientFactory,
	)
	getModule, err := moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	testFile1HasNoExternalPath(t, ctx, getModule)
	getReadWriteBucket := storagemem.NewReadWriteBucket()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadWriteBucket)
	require.NoError(t, err)
	// Verify that the buf.lock file was created.
	exists, err := storage.Exists(ctx, getReadWriteBucket, buflock.ExternalConfigFilePath)
	require.NoError(t, err)
	require.True(t, exists)

	// Exclude non-proto files for the diff check
	filteredReadBucket := storage.MapReadBucket(getReadWriteBucket, storage.MatchPathExt(".proto"))
	diff, err := storage.DiffBytes(ctx, runner, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))

	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	testFile1HasNoExternalPath(t, ctx, getModule)
	require.Equal(t, 2, moduleReader.stats.Count())
	require.Equal(t, 1, moduleReader.stats.Hits())

	// put some data that will not match the sum and make sure that we have a cache miss
	require.NoError(t, storage.PutPath(ctx, mainDataReadWriteBucket, normalpath.Join(newCacheKey(modulePin), "1234.proto"), []byte("foo")))
	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadWriteBucket = storagemem.NewReadWriteBucket()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadWriteBucket)
	require.NoError(t, err)
	// Exclude non-proto files for the diff check
	filteredReadBucket = storage.MapReadBucket(getReadWriteBucket, storage.MatchPathExt(".proto"))
	diff, err = storage.DiffBytes(ctx, runner, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
	require.Equal(t, 3, moduleReader.stats.Count())
	require.Equal(t, 1, moduleReader.stats.Hits())

	_, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	require.Equal(t, 4, moduleReader.stats.Count())
	require.Equal(t, 2, moduleReader.stats.Hits())

	// overwrite the sum file and make sure that we have a cache miss
	require.NoError(t, storage.PutPath(ctx, mainSumReadWriteBucket, newCacheKey(modulePin), []byte("foo")))
	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadWriteBucket = storagemem.NewReadWriteBucket()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadWriteBucket)
	require.NoError(t, err)
	// Exclude non-proto files for the diff check
	filteredReadBucket = storage.MapReadBucket(getReadWriteBucket, storage.MatchPathExt(".proto"))
	diff, err = storage.DiffBytes(ctx, runner, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
	require.Equal(t, 5, moduleReader.stats.Count())
	require.Equal(t, 2, moduleReader.stats.Hits())

	_, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	require.Equal(t, 6, moduleReader.stats.Count())
	require.Equal(t, 3, moduleReader.stats.Hits())

	// delete the sum file and make sure that we have a cache miss
	require.NoError(t, mainSumReadWriteBucket.Delete(ctx, newCacheKey(modulePin)))
	getModule, err = moduleReader.GetModule(ctx, modulePin)
	require.NoError(t, err)
	getReadWriteBucket = storagemem.NewReadWriteBucket()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadWriteBucket)
	require.NoError(t, err)
	// Exclude non-proto files for the diff check
	filteredReadBucket = storage.MapReadBucket(getReadWriteBucket, storage.MatchPathExt(".proto"))
	diff, err = storage.DiffBytes(ctx, runner, readBucket, filteredReadBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
	require.Equal(t, 7, moduleReader.stats.Count())
	require.Equal(t, 3, moduleReader.stats.Hits())
	require.Equal(t, 4, observedLogs.Filter(func(entry observer.LoggedEntry) bool {
		return strings.Contains(entry.Message, deprecationMessage)
	}).Len())
}

func TestCacherBasic(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"foob",
		"bar",
		"main",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigest,
		time.Now(),
	)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForProto(
		ctx,
		bufmoduletesting.TestDataProto,
		bufmodule.ModuleWithModuleIdentityAndCommit(modulePin, modulePin.Commit()),
	)
	require.NoError(t, err)

	dataReadWriteBucket, sumReadWriteBucket, _ := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), dataReadWriteBucket, sumReadWriteBucket, false)
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
	getReadWriteBucket := storagemem.NewReadWriteBucket()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadWriteBucket)
	require.NoError(t, err)
	exists, err := storage.Exists(ctx, getReadWriteBucket, buflock.ExternalConfigFilePath)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestModuleReaderCacherWithDocumentation(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"foob",
		"bar",
		"main",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigest,
		time.Now(),
	)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForProto(
		ctx,
		bufmoduletesting.TestDataWithDocumentationProto,
		bufmodule.ModuleWithModuleIdentity(modulePin),
	)
	require.NoError(t, err)

	dataReadWriteBucket, sumReadWriteBucket, _ := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), dataReadWriteBucket, sumReadWriteBucket, false)
	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)
	module, err = moduleCacher.GetModule(ctx, modulePin)
	require.NoError(t, err)
	readWriteBucket := storagemem.NewReadWriteBucket()
	require.NoError(t, bufmodule.ModuleToBucket(ctx, module, readWriteBucket))
	// Verify that the buf.md file was created.
	exists, err := storage.Exists(ctx, readWriteBucket, bufmodule.DocumentationFilePath)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, bufmoduletesting.TestModuleDocumentation, module.Documentation())
}
func TestModuleReaderCacherWithConfiguration(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"acme",
		"weather",
		"main",
		bufmoduletesting.TestCommit,
		bufmoduletesting.TestDigestB3WithConfiguration,
		time.Now(),
	)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestDataWithConfiguration)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)

	dataReadWriteBucket, sumReadWriteBucket, _ := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), dataReadWriteBucket, sumReadWriteBucket, false)
	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)
	module, err = moduleCacher.GetModule(ctx, modulePin)
	require.NoError(t, err)
	readWriteBucket := storagemem.NewReadWriteBucket()
	require.NoError(t, bufmodule.ModuleToBucket(ctx, module, readWriteBucket))
	// Verify that the buf.md file was created.
	exists, err := storage.Exists(ctx, readWriteBucket, bufmodule.DocumentationFilePath)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, bufmoduletesting.TestModuleDocumentation, module.Documentation())
	// Parse config from original data
	config, err := bufconfig.GetConfigForData(ctx, []byte(bufmoduletesting.TestModuleConfiguration))
	require.NoError(t, err)
	cachedConfig, err := bufconfig.GetConfigForBucket(ctx, readWriteBucket)
	require.NoError(t, err)
	require.Equal(t, config.Breaking, cachedConfig.Breaking)
	require.Equal(t, config.Lint, cachedConfig.Lint)
}

func TestModuleReaderCacherWithLicense(t *testing.T) {
	ctx := context.Background()

	modulePin, err := bufmoduleref.NewModulePin(
		"buf.build",
		"foo",
		"bar",
		"main",
		bufmoduletesting.TestCommit,
		"",
		time.Now(),
	)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestDataWithLicense)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(
		ctx,
		readBucket,
		bufmodule.ModuleWithModuleIdentity(modulePin),
	)
	require.NoError(t, err)

	dataReadWriteBucket, sumReadWriteBucket, _ := newTestDataSumBucketsAndLocker(t)
	moduleCacher := newModuleCacher(zap.NewNop(), dataReadWriteBucket, sumReadWriteBucket, false)
	err = moduleCacher.PutModule(
		context.Background(),
		modulePin,
		module,
	)
	require.NoError(t, err)
	module, err = moduleCacher.GetModule(ctx, modulePin)
	require.NoError(t, err)
	readWriteBucket := storagemem.NewReadWriteBucket()
	require.NoError(t, bufmodule.ModuleToBucket(ctx, module, readWriteBucket))
	// Verify that the license file was created.
	exists, err := storage.Exists(ctx, readWriteBucket, bufmodule.LicenseFilePath)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, bufmoduletesting.TestModuleLicense, module.License())
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

func fakeRepositoryServiceClientFactory(repositoryService registryv1alpha1connect.RepositoryServiceClient) RepositoryServiceClientFactory {
	return func(string) registryv1alpha1connect.RepositoryServiceClient {
		return repositoryService
	}
}

type fakeRepositoryService struct {
	registryv1alpha1connect.RepositoryServiceClient
	repository *registryv1alpha1.Repository
}

func (f *fakeRepositoryService) GetRepositoryByFullName(
	_ context.Context,
	_ *connect.Request[registryv1alpha1.GetRepositoryByFullNameRequest],
) (*connect.Response[registryv1alpha1.GetRepositoryByFullNameResponse], error) {
	return connect.NewResponse(&registryv1alpha1.GetRepositoryByFullNameResponse{
		Repository: f.repository,
	}), nil
}
