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
	"fmt"
	"io/fs"
	"testing"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/dag/dagtest"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestBasicV1(t *testing.T) {
	t.Parallel()
	testBasic(t, "workspacev1", false)
}

func TestBasicV2(t *testing.T) {
	t.Parallel()
	testBasic(t, "workspacev2", true)
}

func testBasic(t *testing.T, subDirPath string, isV2 bool) {
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
		slogtestext.NewLogger(t),
		bucket,
		"finance/portfolio/proto",
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NotNil(t, bucketTargeting.ControllingWorkspace())
	require.Equal(t, ".", bucketTargeting.ControllingWorkspace().Path())
	require.Equal(t, "finance/portfolio/proto", bucketTargeting.SubDirPath())
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
	graphRemoteOnly, err := bufmodule.ModuleSetToDAG(workspace, bufmodule.ModuleSetToDAGWithRemoteOnly())
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
		},
		graphRemoteOnly,
		bufmodule.Module.OpaqueID,
	)
	module = workspace.GetModuleForOpaqueID("buf.testing/acme/bond")
	require.NotNil(t, module)
	_, err = module.StatFileInfo(ctx, "acme/bond/real/v1/bond.proto")
	require.NoError(t, err)
	_, err = module.StatFileInfo(ctx, "acme/bond/v2/bond.proto")
	require.NoError(t, err)
	_, err = module.StatFileInfo(ctx, "acme/bond/excluded/v2/excluded.proto")
	require.True(t, errors.Is(err, fs.ErrNotExist))

	testLicenseAndDoc(t, ctx, workspace, isV2)

	bucketTargeting, err = buftarget.NewBucketTargeting(
		ctx,
		slogtestext.NewLogger(t),
		bucket,
		"common/money/proto",
		[]string{"common/money/proto/acme/money/v1/currency_code.proto"},
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NoError(t, err)
	require.NotNil(t, bucketTargeting.ControllingWorkspace())
	require.Equal(t, ".", bucketTargeting.ControllingWorkspace().Path())
	require.Equal(t, "common/money/proto", bucketTargeting.SubDirPath())
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

	testLicenseAndDoc(t, ctx, workspace, isV2)
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
		slogtestext.NewLogger(t),
		bucket,
		".",
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NoError(t, err)
	require.NotNil(t, bucketTargeting.ControllingWorkspace())
	require.Equal(t, ".", bucketTargeting.ControllingWorkspace().Path())
	require.Equal(t, ".", bucketTargeting.SubDirPath())

	workspace, err := workspaceProvider.GetWorkspaceForBucket(
		ctx,
		bucket,
		bucketTargeting,
	)
	require.NoError(t, err)

	malformedDeps, err := MalformedDepsForWorkspace(workspace)
	require.NoError(t, err)
	require.Equal(t, 2, len(malformedDeps))
	require.Equal(t, "buf.testing/acme/date", malformedDeps[0].ModuleRef().ModuleFullName().String())
	require.Equal(t, MalformedDepTypeUnused, malformedDeps[0].Type())
	require.Equal(t, "buf.testing/acme/extension", malformedDeps[1].ModuleRef().ModuleFullName().String())
	require.Equal(t, MalformedDepTypeUnused, malformedDeps[1].Type())
}

func TestDuplicatePath(t *testing.T) {
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
	bucket, err := storageosProvider.NewReadWriteBucket("testdata/basic/workspacev2_duplicate_path")
	require.NoError(t, err)
	bucketTargeting, err := buftarget.NewBucketTargeting(
		ctx,
		slogtestext.NewLogger(t),
		bucket,
		".",
		nil,
		nil,
		buftarget.TerminateAtControllingWorkspace,
	)
	require.NoError(t, err)
	require.NotNil(t, bucketTargeting.ControllingWorkspace())
	require.Equal(t, ".", bucketTargeting.ControllingWorkspace().Path())
	require.Equal(t, ".", bucketTargeting.SubDirPath())

	workspace, err := workspaceProvider.GetWorkspaceForBucket(
		ctx,
		bucket,
		bucketTargeting,
	)
	require.NoError(t, err)
	require.NotNil(t, workspace)

	require.Len(t, workspace.Modules(), 7) // 5 local + 2 remote
	require.NotNil(t, workspace.GetModuleForOpaqueID("buf.testing/acme/date"))
	require.NotNil(t, workspace.GetModuleForOpaqueID("buf.testing/acme/extension"))

	module := workspace.GetModuleForOpaqueID("proto/shared")
	require.NotNil(t, module)
	requireModuleContainFileNames(t, module, "prefix/bar/v1/bar.proto")

	module = workspace.GetModuleForOpaqueID("proto/shared-2")
	require.NotNil(t, module)
	requireModuleContainFileNames(t, module, "prefix/foo/v1/foo.proto")

	module = workspace.GetModuleForOpaqueID("proto/shared1")
	require.NotNil(t, module)
	requireModuleContainFileNames(t, module, "prefix/x/x.proto")

	module = workspace.GetModuleForOpaqueID("proto/shared1-2")
	require.NotNil(t, module)
	requireModuleContainFileNames(t, module, "prefix/y/y.proto")

	module = workspace.GetModuleForOpaqueID("separate")
	require.NotNil(t, module)
	requireModuleContainFileNames(t, module, "v1/separate.proto")
}

func testNewWorkspaceProvider(t *testing.T, testModuleDatas ...bufmoduletesting.ModuleData) WorkspaceProvider {
	bsrProvider, err := bufmoduletesting.NewOmniProvider(testModuleDatas...)
	require.NoError(t, err)
	return NewWorkspaceProvider(
		slogtestext.NewLogger(t),
		bsrProvider,
		bsrProvider,
		bsrProvider,
	)
}

func requireModuleContainFileNames(t *testing.T, module bufmodule.Module, expectedFileNames ...string) {
	fileNamesToBeSeen := slicesext.ToStructMap(expectedFileNames)
	require.NoError(t, module.WalkFileInfos(context.Background(), func(fi bufmodule.FileInfo) error {
		path := fi.Path()
		if _, ok := fileNamesToBeSeen[path]; !ok {
			return fmt.Errorf("module has unexpected file: %s", path)
		}
		delete(fileNamesToBeSeen, path)
		return nil
	}))
	require.Emptyf(t, fileNamesToBeSeen, "expect %s from module", stringutil.JoinSliceQuoted(slicesext.MapKeysToSlice(fileNamesToBeSeen), ","))
}

func requireModuleFileContent(
	t *testing.T,
	ctx context.Context,
	module bufmodule.Module,
	path string,
	expectedContent string,
) {
	file, err := module.GetFile(ctx, path)
	require.NoError(t, err)
	content, err := ioext.ReadAllAndClose(file)
	require.NoError(t, err)
	require.Equal(t, expectedContent, string(content))
}

func testLicenseAndDoc(t *testing.T, ctx context.Context, workspace Workspace, isV2 bool) {
	// bond has its own license and doc
	module := workspace.GetModuleForOpaqueID("buf.testing/acme/bond")
	require.NotNil(t, module)
	requireModuleFileContent(t, ctx, module, "README.md", "bond doc\n")
	requireModuleFileContent(t, ctx, module, "LICENSE", "bond license\n")

	module = workspace.GetModuleForOpaqueID("buf.testing/acme/geo")
	require.NotNil(t, module)
	// geo has its own license
	requireModuleFileContent(t, ctx, module, "LICENSE", "geo license\n")
	// geo falls back to top-level doc if it's a v2 workspace
	if isV2 {
		requireModuleFileContent(t, ctx, module, "README.md", "workspace doc\n")
	} else {
		_, err := module.StatFileInfo(ctx, "README.md")
		require.ErrorIs(t, err, fs.ErrNotExist)
	}
	_, err := module.StatFileInfo(ctx, "buf.md")
	require.ErrorIs(t, err, fs.ErrNotExist)

	module = workspace.GetModuleForOpaqueID("buf.testing/acme/money")
	require.NotNil(t, module)
	// money has its own doc
	requireModuleFileContent(t, ctx, module, "buf.md", "money doc\n")
	// money does not have README.md
	_, err = module.StatFileInfo(ctx, "README.md")
	require.ErrorIs(t, err, fs.ErrNotExist)
	// money falls back to top-level license if it's a v2 workspace
	if isV2 {
		requireModuleFileContent(t, ctx, module, "LICENSE", "workspace license\n")
	} else {
		_, err = module.StatFileInfo(ctx, "LICENSE")
		require.ErrorIs(t, err, fs.ErrNotExist)
	}

	module = workspace.GetModuleForOpaqueID("finance/portfolio/proto")
	require.NotNil(t, module)
	// portfolio does not have its own license or doc
	if isV2 {
		requireModuleFileContent(t, ctx, module, "LICENSE", "workspace license\n")
		requireModuleFileContent(t, ctx, module, "README.md", "workspace doc\n")
	} else {
		_, err = module.StatFileInfo(ctx, "LICENSE")
		require.ErrorIs(t, err, fs.ErrNotExist)
		_, err = module.StatFileInfo(ctx, "README.md")
		require.ErrorIs(t, err, fs.ErrNotExist)
	}
}
