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

package bufmodulestorage

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	ctx := context.Background()
	moduleName, err := bufmodule.ModuleNameForString(bufmoduletesting.TestModuleNameString)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestData)
	require.NoError(t, err)
	exists, err := storage.Exists(ctx, readBucket, bufmodule.LockFilePath)
	require.NoError(t, err)
	require.False(t, exists)
	module, err := bufmodule.NewModuleForBucket(ctx, readBucket)
	require.NoError(t, err)
	resolvedModuleName, err := bufmodule.ResolvedModuleNameForModule(ctx, moduleName, module)
	require.NoError(t, err)
	require.Equal(t, resolvedModuleName.Digest(), bufmoduletesting.TestDigest)

	moduleReadWriter := newTestModuleReadWriter(t)
	_, err = moduleReadWriter.GetModule(ctx, resolvedModuleName)
	require.True(t, storage.IsNotExist(err))

	putResolvedModuleName, err := moduleReadWriter.PutModule(
		context.Background(),
		moduleName,
		module,
	)
	require.NoError(t, err)
	require.True(t, bufmodule.ModuleNameEqual(resolvedModuleName, putResolvedModuleName))

	getModule, err := moduleReadWriter.GetModule(ctx, resolvedModuleName)
	require.NoError(t, err)
	getReadBucketBuilder := storagemem.NewReadBucketBuilder()
	err = bufmodule.ModuleToBucket(ctx, getModule, getReadBucketBuilder)
	require.NoError(t, err)
	getReadBucket, err := getReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	exists, err = storage.Exists(ctx, getReadBucket, bufmodule.LockFilePath)
	require.NoError(t, err)
	require.True(t, exists)
}

func newTestModuleReadWriter(t *testing.T) bufmodule.ModuleReadWriter {
	readWriteBucket, err := storageos.NewReadWriteBucket(t.TempDir())
	require.NoError(t, err)
	return NewModuleReadWriter(readWriteBucket)
}
