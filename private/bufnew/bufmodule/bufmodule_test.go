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

package bufmodule_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufnew/bufmodule/bufmoduletest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	testBSRProvider, err := bufmoduletest.NewTestProviderForPathToData(
		ctx,
		map[string]map[string][]byte{
			"buf.build/foo/extdep1": map[string][]byte{
				"extdep1.proto": []byte(
					`syntax = proto3; package extdep1;`,
				),
			},
			"buf.build/foo/extdep2": map[string][]byte{
				"extdep2.proto": []byte(
					`syntax = proto3; package extdep2; import "extdep1.proto";`,
				),
			},
		},
	)

	moduleBuilder := bufmodule.NewModuleBuilder(ctx, testBSRProvider)

	// Remember, the testBSRProvider is just acting like a BSR; if we actually want to
	// say dependencies are part of our workspace, we need to add them! We do so now.
	moduleRef, err := bufmodule.NewModuleRef("buf.build", "foo", "extdep1", "")
	require.NoError(t, err)
	moduleInfo, err := testBSRProvider.GetModuleInfoForModuleRef(ctx, moduleRef)
	require.NoError(t, err)
	err = moduleBuilder.AddModuleForModuleInfo(moduleInfo)
	require.NoError(t, err)
	moduleRef, err = bufmodule.NewModuleRef("buf.build", "foo", "extdep2", "")
	require.NoError(t, err)
	moduleInfo, err = testBSRProvider.GetModuleInfoForModuleRef(ctx, moduleRef)
	require.NoError(t, err)
	err = moduleBuilder.AddModuleForModuleInfo(moduleInfo)
	require.NoError(t, err)

	// This module has no name but is part of the workspace.
	err = moduleBuilder.AddModuleForBucket(
		"path/to/module1",
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module1.proto": []byte(
					`syntax = proto3; package module1; import "extdep2.proto";`,
				),
			},
		),
	)
	require.NoError(t, err)

	// This module has a name and is part of the workspace.
	moduleFullName, err := bufmodule.NewModuleFullName("buf.build", "bar", "module2")
	require.NoError(t, err)
	err = moduleBuilder.AddModuleForBucket(
		"path/to/module2",
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module2.proto": []byte(
					`syntax = proto3; package module1; import "module1.proto"; import "extdep1.proto";`,
				),
			},
		),
		bufmodule.AddModuleForBucketWithModuleFullName(moduleFullName),
	)
	require.NoError(t, err)

	modules, err := moduleBuilder.Build()
	require.NoError(t, err)
	require.Equal(t, 4, len(modules))

	module2 := testFindModuleWithOpaqueID(t, modules, "buf.build/bar/module2")
	require.Equal(
		t,
		[]string{
			"buf.build/foo/extdep1",
			// Does not directly depend on extdep2
			//"buf.build/foo/extdep2",
			"path/to/module1",
		},
		testSortedDirectDepOpaqueIDs(t, module2),
	)

	extdep2 := testFindModuleWithOpaqueID(t, modules, "buf.build/foo/extdep2")
	require.Equal(
		t,
		[]string{
			"buf.build/foo/extdep1",
		},
		testSortedDirectDepOpaqueIDs(t, extdep2),
	)

	graph, err := bufmodule.GetModuleOpaqueIDDAG(modules...)
	require.NoError(t, err)
	topoSort, err := graph.TopoSort("buf.build/bar/module2")
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"buf.build/foo/extdep1",
			"buf.build/foo/extdep2",
			"path/to/module1",
			"buf.build/bar/module2",
		},
		topoSort,
	)
}

func testNewBucketForPathToData(t *testing.T, pathToData map[string][]byte) storage.ReadBucket {
	bucket, err := storagemem.NewReadBucket(pathToData)
	require.NoError(t, err)
	return bucket
}

func testFindModuleWithOpaqueID(t *testing.T, modules []bufmodule.Module, opaqueID string) bufmodule.Module {
	var foundModules []bufmodule.Module
	for _, module := range modules {
		if module.OpaqueID() == opaqueID {
			foundModules = append(foundModules, module)
		}
	}
	switch len(foundModules) {
	case 0:
		require.NoError(t, fmt.Errorf("no module found for opaqueID %q", opaqueID))
		return nil
	case 1:
		return foundModules[0]
	default:
		require.NoError(t, fmt.Errorf("multiple modules found for opaqueID %q", opaqueID))
		return nil
	}
}

func testSortedDirectDepOpaqueIDs(t *testing.T, module bufmodule.Module) []string {
	depModules, err := module.DepModules()
	require.NoError(t, err)
	depOpaqueIDs := make([]string, 0, len(depModules))
	for _, depModule := range depModules {
		depOpaqueIDs = append(depOpaqueIDs, depModule.OpaqueID())
	}
	sort.Strings(depOpaqueIDs)
	return depOpaqueIDs
}
