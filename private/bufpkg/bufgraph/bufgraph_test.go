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

package bufgraph

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufwork"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	workspace, err := testBuildWorkspace(ctx, filepath.Join("testdata", "basic"))
	require.NoError(t, err)
	builder := NewBuilder(
		zap.NewNop(),
		bufmodule.NewNopModuleResolver(),
		bufmodule.NewNopModuleReader(),
	)
	graph, fileAnnotations, err := builder.Build(
		ctx,
		workspace.GetModules(),
		BuildWithWorkspace(workspace),
	)
	require.NoError(t, err)
	require.Empty(t, fileAnnotations)
	dotString, err := graph.DOTString(func(key Node) string { return key.String() })
	require.NoError(t, err)
	require.Equal(
		t,
		`digraph {

  1 [label="bsr.internal/foo/test-a"]
  2 [label="bsr.internal/foo/test-b"]
  3 [label="bsr.internal/foo/test-c"]
  4 [label="bsr.internal/foo/test-d"]
  5 [label="bsr.internal/foo/test-e"]
  6 [label="bsr.internal/foo/test-f"]
  7 [label="bsr.internal/foo/test-g"]

  1 -> 2
  2 -> 3
  3 -> 4
  1 -> 4
  1 -> 5
  5 -> 6
  7

}`,
		dotString,
	)
}

// TODO: This entire function is all you should need to do to build workspaces, and even
// this is overly complicated because of the wonkiness of bufmodulebuild and NewWorkspace.
// We should have this in a common place for at least testing.
func testBuildWorkspace(ctx context.Context, workspacePath string) (bufmodule.Workspace, error) {
	workspaceBucket, err := storageos.NewProvider().NewReadWriteBucket(workspacePath)
	if err != nil {
		return nil, err
	}
	workspaceConfig, err := bufwork.GetConfigForBucket(ctx, workspaceBucket, ".")
	if err != nil {
		return nil, err
	}
	moduleBucketBuilder := bufmodulebuild.NewModuleBucketBuilder(zap.NewNop())
	namedModules := make(map[string]bufmodule.Module, len(workspaceConfig.Directories))
	allModules := make([]bufmodule.Module, 0, len(workspaceConfig.Directories))
	for _, directory := range workspaceConfig.Directories {
		moduleBucket := storage.MapReadBucket(
			workspaceBucket,
			storage.MapOnPrefix(directory),
		)
		moduleConfig, err := bufconfig.GetConfigForBucket(ctx, moduleBucket)
		if err != nil {
			return nil, err
		}
		module, err := moduleBucketBuilder.BuildForBucket(
			ctx,
			moduleBucket,
			moduleConfig.Build,
			bufmodulebuild.WithModuleIdentity(
				moduleConfig.ModuleIdentity,
			),
			bufmodulebuild.WithWorkspaceDirectory(
				directory,
			),
		)
		if err != nil {
			return nil, err
		}
		if moduleConfig.ModuleIdentity != nil {
			namedModules[moduleConfig.ModuleIdentity.IdentityString()] = module
		}
		allModules = append(allModules, module)
	}
	return bufmodule.NewWorkspace(
		ctx,
		namedModules,
		allModules,
	)
}
