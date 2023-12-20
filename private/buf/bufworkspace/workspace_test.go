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
	"errors"
	"io/fs"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/dag/dagtest"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBasicV1(t *testing.T) {
	t.Parallel()
	testBasic(t, "workspacev1")
}

func TestBasicV2(t *testing.T) {
	t.Parallel()
	testBasic(t, "workspacev2")
}

func testBasic(t *testing.T, subDirPath string) {
	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	bsrProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/date",
			DirPath: "testdata/basic/bsr/buf.testing/acme/date",
		},
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/extension",
			DirPath: "testdata/basic/bsr/buf.testing/acme/extension",
		},
	)
	require.NoError(t, err)

	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket(normalpath.Join("testdata/basic", subDirPath))
	require.NoError(t, err)

	workspace, err := NewWorkspaceForBucket(
		ctx,
		zap.NewNop(),
		tracing.NopTracer,
		bucket,
		bufapi.NewNopClientProvider(),
		bsrProvider,
		WithTargetSubDirPath(
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
	require.NoError(t, err)
	module = workspace.GetModuleForOpaqueID("buf.testing/acme/bond")
	require.NotNil(t, module)
	_, err = module.StatFileInfo(ctx, "acme/bond/real/v1/bond.proto")
	require.NoError(t, err)
	_, err = module.StatFileInfo(ctx, "acme/bond/v2/bond.proto")
	require.NoError(t, err)
	_, err = module.StatFileInfo(ctx, "acme/bond/excluded/v2/excluded.proto")
	require.True(t, errors.Is(err, fs.ErrNotExist))
	_, err = module.StatFileInfo(ctx, "README.md")
	require.NoError(t, err)
	_, err = module.StatFileInfo(ctx, "LICENSE")
	require.NoError(t, err)

	//malformedDeps, err := MalformedDepsForWorkspace(workspace)
	//require.NoError(t, err)
	//require.Equal(t, 1, len(malformedDeps))
	//malformedDep := malformedDeps[0]
	//require.Equal(t, "buf.testing/acme/extension", malformedDep.ModuleFullName().String())
	//require.Equal(t, MalformedDepTypeUndeclared, malformedDep.Type())

	workspace, err = NewWorkspaceForBucket(
		ctx,
		zap.NewNop(),
		tracing.NopTracer,
		bucket,
		bufapi.NewNopClientProvider(),
		bsrProvider,
		WithTargetSubDirPath(
			"common/money/proto",
		),
		WithTargetPaths(
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
}

func TestProtoc(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	workspace, err := NewWorkspaceForProtoc(
		ctx,
		zap.NewNop(),
		tracing.NopTracer,
		storageos.NewProvider(),
		[]string{
			"testdata/basic/bsr/buf.testing/acme/date",
			"testdata/basic/bsr/buf.testing/acme/extension",
			"testdata/basic/workspacev1/common/geo/proto",
			"testdata/basic/workspacev1/common/money/proto",
			"testdata/basic/workspacev1/finance/bond/proto/root1",
			"testdata/basic/workspacev1/finance/bond/proto/root2",
			"testdata/basic/workspacev1/finance/portfolio/proto",
		},
		[]string{
			"testdata/basic/workspacev1/finance/portfolio/proto/acme/portfolio/v1/portfolio.proto",
		},
	)
	require.NoError(t, err)

	modules := workspace.Modules()
	require.Equal(t, 1, len(modules))
	module := modules[0]
	require.Equal(t, ".", module.OpaqueID())
	require.True(t, module.IsTarget())

	fileInfo, err := module.StatFileInfo(ctx, "acme/money/v1/currency_code.proto")
	require.NoError(t, err)
	require.False(t, fileInfo.IsTargetFile())
	fileInfo, err = module.StatFileInfo(ctx, "acme/money/v1/money.proto")
	require.NoError(t, err)
	require.False(t, fileInfo.IsTargetFile())
	fileInfo, err = module.StatFileInfo(ctx, "acme/bond/real/v1/bond.proto")
	require.NoError(t, err)
	require.False(t, fileInfo.IsTargetFile())
	fileInfo, err = module.StatFileInfo(ctx, "acme/portfolio/v1/portfolio.proto")
	require.NoError(t, err)
	require.True(t, fileInfo.IsTargetFile())
}

func TestUnusedDep(t *testing.T) {
	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	bsrProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/date",
			DirPath: "testdata/basic/bsr/buf.testing/acme/date",
		},
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/extension",
			DirPath: "testdata/basic/bsr/buf.testing/acme/extension",
		},
	)
	require.NoError(t, err)

	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket("testdata/basic/workspace_unused_dep")
	require.NoError(t, err)

	workspace, err := NewWorkspaceForBucket(
		ctx,
		zap.NewNop(),
		tracing.NopTracer,
		bucket,
		bufapi.NewNopClientProvider(),
		bsrProvider,
	)
	require.NoError(t, err)

	malformedDeps, err := MalformedDepsForWorkspace(workspace)
	require.NoError(t, err)
	require.Equal(t, 2, len(malformedDeps))
	require.Equal(t, "buf.testing/acme/date", malformedDeps[0].ModuleFullName().String())
	require.Equal(t, MalformedDepTypeUnused, malformedDeps[0].Type())
	require.Equal(t, "buf.testing/acme/extension", malformedDeps[1].ModuleFullName().String())
	require.Equal(t, MalformedDepTypeUnused, malformedDeps[1].Type())
}

func TestUndeclaredDep(t *testing.T) {
	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	bsrProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/date",
			DirPath: "testdata/basic/bsr/buf.testing/acme/date",
		},
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/extension",
			DirPath: "testdata/basic/bsr/buf.testing/acme/extension",
		},
	)
	require.NoError(t, err)

	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket("testdata/basic/workspace_undeclared_dep")
	require.NoError(t, err)

	workspace, err := NewWorkspaceForBucket(
		ctx,
		zap.NewNop(),
		tracing.NopTracer,
		bucket,
		bufapi.NewNopClientProvider(),
		bsrProvider,
	)
	require.NoError(t, err)

	malformedDeps, err := MalformedDepsForWorkspace(workspace)
	require.NoError(t, err)
	require.Equal(t, 1, len(malformedDeps))
	require.Equal(t, "buf.testing/acme/extension", malformedDeps[0].ModuleFullName().String())
	require.Equal(t, MalformedDepTypeUndeclared, malformedDeps[0].Type())
}
