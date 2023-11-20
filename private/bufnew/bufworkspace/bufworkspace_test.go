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

package bufworkspace

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufnew/bufmodule/bufmoduletest"
	"github.com/bufbuild/buf/private/pkg/dag/dagtest"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func init() {
	bufconfig.AllowV2ForTesting()
}

func TestBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	bsrProvider, err := bufmoduletest.NewOmniProvider(
		bufmoduletest.ModuleData{
			Name:    "buf.testing/acme/date",
			DirPath: "testdata/basic/bsr/buf.testing/acme/date",
		},
		bufmoduletest.ModuleData{
			Name:    "buf.testing/acme/extension",
			DirPath: "testdata/basic/bsr/buf.testing/acme/extension",
		},
	)
	require.NoError(t, err)

	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket(
		"testdata/basic/workspace",
	)
	require.NoError(t, err)

	workspace, err := NewWorkspaceForBucket(
		ctx,
		bucket,
		bsrProvider,
		WorkspaceWithTargetSubDirPath(
			"finance/portfolio/proto",
		),
	)
	require.NoError(t, err)
	module := workspace.GetModuleForOpaqueID("buf.testing/acme/bond")
	require.NotNil(t, module)
	require.False(t, module.IsTarget())
	module = workspace.GetModuleForOpaqueID("finance/portfolio/proto")
	require.NotNil(t, module)
	require.True(t, module.IsTarget())
	graph, err := bufmodule.ModuleSetToDAG(workspace)
	require.NoError(t, err)
	dagtest.RequireGraphEqual(
		t,
		[]dagtest.ExpectedNode[string]{
			{
				Key: "buf.testing/acme/extension",
			},
			{
				Key: "buf.testing/acme/date",
				Outbound: []string{
					"buf.testing/acme/extension",
				},
			},
			{
				Key: "buf.testing/acme/geo",
			},
			{
				Key: "buf.testing/acme/money",
			},
			{
				Key: "buf.testing/acme/bond",
				Outbound: []string{
					"buf.testing/acme/date",
					"buf.testing/acme/geo",
					"buf.testing/acme/money",
				},
			},
			{
				Key: "finance/portfolio/proto",
				Outbound: []string{
					"buf.testing/acme/bond",
					"buf.testing/acme/extension",
				},
			},
		},
		graph,
	)

	workspace, err = NewWorkspaceForBucket(
		ctx,
		bucket,
		bsrProvider,
		WorkspaceWithTargetSubDirPath(
			"common/money/proto",
		),
		WorkspaceWithTargetPaths(
			[]string{"common/money/proto/acme/money/v1/currency_code.proto"},
			nil,
		),
	)
	require.NoError(t, err)
	module = workspace.GetModuleForOpaqueID("buf.testing/acme/money")
	require.NotNil(t, module)
	require.True(t, module.IsTarget())
	fileInfo, err := module.StatFileInfo(ctx, "acme/money/v1/currency_code.proto")
	require.NoError(t, err)
	require.True(t, fileInfo.IsTargetFile())
	fileInfo, err = module.StatFileInfo(ctx, "acme/money/v1/money.proto")
	require.NoError(t, err)
	require.False(t, fileInfo.IsTargetFile())

	//workspace, err = NewWorkspaceForBucket(
	//ctx,
	//bucket,
	//bsrProvider,
	//WorkspaceWithTargetSubDirPath(
	//"common/geo/proto",
	//),
	//WorkspaceWithTargetPaths(
	//[]string{"common/money/proto/acme/money/v1/currency_code.proto"},
	//nil,
	//),
	//)
	//require.NoError(t, err)
	//module = workspace.GetModuleForOpaqueID("buf.testing/acme/money")
	//require.NotNil(t, module)
	//require.False(t, module.IsTarget())
	//fileInfo, err = module.StatFileInfo(ctx, "acme/money/v1/currency_code.proto")
	//require.NoError(t, err)
	//require.False(t, fileInfo.IsTargetFile())
	//fileInfo, err = module.StatFileInfo(ctx, "acme/money/v1/money.proto")
	//require.NoError(t, err)
	//require.False(t, fileInfo.IsTargetFile())
}
