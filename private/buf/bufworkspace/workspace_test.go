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

package bufworkspace

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/dag/dagtest"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
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
	workspaceProvider := testNewWorkspaceProvider(
		t,
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/date",
			DirPath: "testdata/basic/bsr/buf.testing/acme/date",
		},
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/extension",
			DirPath: "testdata/basic/bsr/buf.testing/acme/extension",
		},
	)

	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket(normalpath.Join("testdata/basic", subDirPath))
	require.NoError(t, err)

	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		zaptest.NewLogger(t),
		bucket,
		"finance/portfolio/proto",
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NotNil(t, bucketTargeting.ControllingWorkspace())
	require.Equal(t, ".", bucketTargeting.ControllingWorkspace().Path())
	require.Equal(t, "finance/portfolio/proto", bucketTargeting.InputDir())
	require.NoError(t, err)

	workspace, err := workspaceProvider.GetWorkspaceForBucket(
		ctx,
		bucket,
		bucketTargeting,
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
		bufmodule.Module.OpaqueID,
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

	bucketTargeting, err = buftarget.NewBucketTargeting(
		ctx,
		zaptest.NewLogger(t),
		bucket,
		"common/money/proto",
		[]string{"common/money/proto/acme/money/v1/currency_code.proto"},
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NoError(t, err)
	require.NotNil(t, bucketTargeting.ControllingWorkspace())
	require.Equal(t, ".", bucketTargeting.ControllingWorkspace().Path())
	require.Equal(t, "common/money/proto", bucketTargeting.InputDir())
	require.Equal(
		t,
		[]string{"common/money/proto/acme/money/v1/currency_code.proto"},
		bucketTargeting.TargetPaths(),
	)

	workspace, err = workspaceProvider.GetWorkspaceForBucket(
		ctx,
		bucket,
		bucketTargeting,
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

func TestUnusedDep(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	workspaceProvider := testNewWorkspaceProvider(
		t,
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/date",
			DirPath: "testdata/basic/bsr/buf.testing/acme/date",
		},
		bufmoduletesting.ModuleData{
			Name:    "buf.testing/acme/extension",
			DirPath: "testdata/basic/bsr/buf.testing/acme/extension",
		},
	)

	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket("testdata/basic/workspace_unused_dep")
	require.NoError(t, err)
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		zaptest.NewLogger(t),
		bucket,
		".",
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NoError(t, err)
	require.NotNil(t, bucketTargeting.ControllingWorkspace())
	require.Equal(t, ".", bucketTargeting.ControllingWorkspace().Path())
	require.Equal(t, ".", bucketTargeting.InputDir())

	workspace, err := workspaceProvider.GetWorkspaceForBucket(
		ctx,
		bucket,
		bucketTargeting,
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

func testNewWorkspaceProvider(t *testing.T, testModuleDatas ...bufmoduletesting.ModuleData) WorkspaceProvider {
	bsrProvider, err := bufmoduletesting.NewOmniProvider(testModuleDatas...)
	require.NoError(t, err)
	return NewWorkspaceProvider(
		zap.NewNop(),
		tracing.NopTracer,
		bsrProvider,
		bsrProvider,
		bsrProvider,
	)
}
