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
	"testing"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufnew/bufmodule/bufmoduletest"
	"github.com/bufbuild/buf/private/pkg/dag"
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
			// Adding in a module that exists remotely but we'll also have in the workspace.
			//
			// This one will import from extdep2 instead of the workspace importing from extdep1.
			"buf.build/bar/module2": map[string][]byte{
				"module2.proto": []byte(
					`syntax = proto3; package module2; import "extdep2.proto";`,
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
	moduleRef, err = bufmodule.NewModuleRef("buf.build", "bar", "module2", "")
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
	//
	// This module is also in the BSR, but we'll prefer this one.
	moduleFullName, err := bufmodule.NewModuleFullName("buf.build", "bar", "module2")
	require.NoError(t, err)
	err = moduleBuilder.AddModuleForBucket(
		"path/to/module2",
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module2.proto": []byte(
					`syntax = proto3; package module2; import "module1.proto"; import "extdep1.proto";`,
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
		map[string]bool{
			"buf.build/foo/extdep1": true,
			"buf.build/foo/extdep2": false,
			"path/to/module1":       true,
		},
		testGetDepOpaqueIDToDirect(t, module2),
	)

	extdep2 := testFindModuleWithOpaqueID(t, modules, "buf.build/foo/extdep2")
	require.Equal(
		t,
		map[string]bool{
			"buf.build/foo/extdep1": true,
		},
		testGetDepOpaqueIDToDirect(t, extdep2),
	)

	graph, err := bufmodule.GetModuleOpaqueIDDAG(modules...)
	require.NoError(t, err)
	testWalkGraphNodes(
		t,
		graph,
		[]testStringNode{
			{
				Key:     "buf.build/bar/module2",
				Inbound: []string{},
				Outbound: []string{
					"buf.build/foo/extdep1",
					"path/to/module1",
				},
			},
			{
				Key: "buf.build/foo/extdep1",
				Inbound: []string{
					"buf.build/bar/module2",
					"buf.build/foo/extdep2",
				},
				Outbound: []string{},
			},
			{
				Key: "path/to/module1",
				Inbound: []string{
					"buf.build/bar/module2",
				},
				Outbound: []string{
					"buf.build/foo/extdep2",
				},
			},
			{
				Key: "buf.build/foo/extdep2",
				Inbound: []string{
					"path/to/module1",
				},
				Outbound: []string{
					"buf.build/foo/extdep1",
				},
			},
		},
	)
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

func testGetDepOpaqueIDToDirect(t *testing.T, module bufmodule.Module) map[string]bool {
	moduleDeps, err := module.ModuleDeps()
	require.NoError(t, err)
	depOpaqueIDToDirect := make(map[string]bool)
	for _, moduleDep := range moduleDeps {
		depOpaqueIDToDirect[moduleDep.OpaqueID()] = moduleDep.IsDirect()
	}
	return depOpaqueIDToDirect
}

func testWalkGraphNodes(t *testing.T, graph *dag.Graph[string], expected []testStringNode) {
	var results []testStringNode
	err := graph.WalkNodes(
		func(key string, inbound []string, outbound []string) error {
			results = append(
				results,
				testStringNode{
					Key:      key,
					Inbound:  inbound,
					Outbound: outbound,
				},
			)
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, expected, results)
}

type testStringNode struct {
	Key      string
	Inbound  []string
	Outbound []string
}
