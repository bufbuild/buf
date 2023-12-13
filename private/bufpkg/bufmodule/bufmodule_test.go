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
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/dag/dagtest"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	bsrProvider, err := bufmoduletesting.NewOmniProvider(
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/extdep1",
			PathToData: map[string][]byte{
				"extdep1.proto": []byte(
					`syntax = proto3; package extdep1;`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/extdep2",
			PathToData: map[string][]byte{
				"extdep2.proto": []byte(
					`syntax = proto3; package extdep2; import "extdep1.proto";`,
				),
			},
		},
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/extdep3",
			PathToData: map[string][]byte{
				"extdep3.proto": []byte(
					`syntax = proto3; package extdep3; import "extdep4.proto";`,
				),
			},
		},
		// This module is only a transitive remote dependency. It is only depended on by extdep3.
		bufmoduletesting.ModuleData{
			Name: "buf.build/foo/extdep4",
			PathToData: map[string][]byte{
				"extdep4.proto": []byte(
					`syntax = proto3; package extdep4;`,
				),
			},
		},
		// Adding in a module that exists remotely but we'll also have in the workspace.
		//
		// This one will import from extdep2 instead of the workspace importing from extdep1.
		bufmoduletesting.ModuleData{
			Name: "buf.build/bar/module2",
			PathToData: map[string][]byte{
				"module2.proto": []byte(
					`syntax = proto3; package module2; import "extdep2.proto";`,
				),
			},
		},
	)
	require.NoError(t, err)

	// This is the ModuleSetBuilder that will build the modules that we are going to test.
	// This is replicating how a workspace would be built from remote dependencies and
	// local sources.
	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, zap.NewNop(), bsrProvider)

	// First, we add the remote dependences (adding order doesn't matter).
	//
	// Remember, the bsrProvider is just acting like a BSR; if we actually want to
	// say dependencies are part of our workspace, we need to add them! We do so now.
	moduleRefExtdep1, err := bufmodule.NewModuleRef("buf.build", "foo", "extdep1", "")
	require.NoError(t, err)
	moduleRefExtdep2, err := bufmodule.NewModuleRef("buf.build", "foo", "extdep2", "")
	require.NoError(t, err)
	moduleRefExtdep3, err := bufmodule.NewModuleRef("buf.build", "foo", "extdep3", "")
	require.NoError(t, err)
	moduleRefExtdep4, err := bufmodule.NewModuleRef("buf.build", "foo", "extdep4", "")
	require.NoError(t, err)
	moduleRefModule2, err := bufmodule.NewModuleRef("buf.build", "bar", "module2", "")
	require.NoError(t, err)
	moduleKeys, err := bufmodule.GetModuleKeysForModuleRefs(
		ctx,
		bsrProvider,
		moduleRefExtdep1,
		moduleRefExtdep2,
		moduleRefExtdep3,
		moduleRefExtdep4,
		moduleRefModule2,
	)
	require.NoError(t, err)
	for _, moduleKey := range moduleKeys {
		moduleSetBuilder.AddRemoteModule(moduleKey, false)
	}

	// Next, we add the local sources.

	// This module has no name but is part of the workspace.
	moduleSetBuilder.AddLocalModule(
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module1.proto": []byte(
					`syntax = proto3; package module1; import "extdep2.proto";`,
				),
			},
		),
		"path/to/module1",
		true,
	)

	// This module has a name and is part of the workspace.
	//
	// This module is also in the BSR, but we'll prefer the local sources when
	// we do ModuleSetBuilder.Build().
	moduleFullName, err := bufmodule.NewModuleFullName("buf.build", "bar", "module2")
	require.NoError(t, err)
	moduleSetBuilder.AddLocalModule(
		testNewBucketForPathToData(
			t,
			map[string][]byte{
				"module2.proto": []byte(
					`syntax = proto3; package module2; import "module1.proto"; import "extdep1.proto";`,
				),
				// module2 is excluded by path, but imports a Module that is not imported anywhere
				// else. We want to make sure this path is not targeted, but extdep3 is still
				// a dependency of the Module.
				"foo/module2_excluded.proto": []byte(
					`syntax = proto3; package module2; import "extdep3.proto";`,
				),
			},
		),
		"path/to/module2",
		true,
		bufmodule.LocalModuleWithModuleFullName(moduleFullName),
		// We're going to exclude the files in the foo directory from targeting,
		// ie foo/module2_excluded.proto. This file will still be in the module,
		// but will not be marked as a target.
		bufmodule.LocalModuleWithTargetPaths(nil, []string{"foo"}),
	)

	// Build our module set!
	moduleSet, err := moduleSetBuilder.Build()
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"buf.build/bar/module2",
			"buf.build/foo/extdep1",
			"buf.build/foo/extdep2",
			"buf.build/foo/extdep3",
			"buf.build/foo/extdep4",
			"path/to/module1",
		},
		bufmodule.ModuleSetOpaqueIDs(moduleSet),
	)
	require.Equal(
		t,
		[]string{
			"buf.build/bar/module2",
			"path/to/module1",
		},
		bufmodule.ModuleSetTargetOpaqueIDs(moduleSet),
	)

	module2 := moduleSet.GetModuleForOpaqueID("buf.build/bar/module2")
	require.NotNil(t, module2)
	// module2 depends on all these modules transitively. extdep1,
	// extdep3, and module1 are direct.
	require.Equal(
		t,
		map[string]bool{
			"buf.build/foo/extdep1": true,
			"buf.build/foo/extdep2": false,
			"buf.build/foo/extdep3": true,
			"buf.build/foo/extdep4": false,
			"path/to/module1":       true,
		},
		testGetDepOpaqueIDToDirect(t, module2),
	)
	// These are the files in the in the module.
	testFilePaths(
		t,
		module2,
		"foo/module2_excluded.proto",
		"module2.proto",
	)
	// These are the target files. We excluded foo, so we only have module2.proto.
	testTargetFilePaths(
		t,
		module2,
		"module2.proto",
	)
	//module2ProtoFileInfo, err := module2.StatFileInfo(ctx, "module2.proto")
	//require.NoError(t, err)
	//imports, err := module2ProtoFileInfo.protoFileImports()
	//require.NoError(t, err)
	//require.Equal(t, []string{"extdep1.proto", "module1.proto"}, imports)
	//pkg, err := module2ProtoFileInfo.protoFilePackage()
	//require.NoError(t, err)
	//require.Equal(t, "module2", pkg)

	extdep2 := moduleSet.GetModuleForOpaqueID("buf.build/foo/extdep2")
	require.NotNil(t, extdep2)
	require.Equal(
		t,
		map[string]bool{
			"buf.build/foo/extdep1": true,
		},
		testGetDepOpaqueIDToDirect(t, extdep2),
	)

	// This is a graph of OpaqueIDs. This tests that we have the full dependency tree
	// that we expect.
	graph, err := bufmodule.ModuleSetToDAG(moduleSet)
	require.NoError(t, err)
	dagtest.RequireGraphEqual(
		t,
		[]dagtest.ExpectedNode[string]{
			{
				Key: "buf.build/bar/module2",
				Outbound: []string{
					"buf.build/foo/extdep1",
					"buf.build/foo/extdep3",
					"path/to/module1",
				},
			},
			{
				Key:      "buf.build/foo/extdep1",
				Outbound: []string{},
			},
			{
				Key: "buf.build/foo/extdep3",
				Outbound: []string{
					"buf.build/foo/extdep4",
				},
			},
			{
				Key:      "buf.build/foo/extdep4",
				Outbound: []string{},
			},
			{
				Key: "path/to/module1",
				Outbound: []string{
					"buf.build/foo/extdep2",
				},
			},
			{
				Key: "buf.build/foo/extdep2",
				Outbound: []string{
					"buf.build/foo/extdep1",
				},
			},
		},
		graph,
	)
	topoSort, err := graph.TopoSort("buf.build/bar/module2")
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"buf.build/foo/extdep1",
			"buf.build/foo/extdep4",
			"buf.build/foo/extdep3",
			"buf.build/foo/extdep2",
			"path/to/module1",
			"buf.build/bar/module2",
		},
		topoSort,
	)
	remoteDeps, err := bufmodule.RemoteDepsForModuleSet(moduleSet)
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"buf.build/foo/extdep1",
			"buf.build/foo/extdep2",
			"buf.build/foo/extdep3",
			"buf.build/foo/extdep4",
		},
		slicesext.Map(remoteDeps, func(remoteDep bufmodule.RemoteDep) string { return remoteDep.OpaqueID() }),
	)
	transitiveRemoteDeps := slicesext.Filter(remoteDeps, func(remoteDep bufmodule.RemoteDep) bool { return !remoteDep.IsDirect() })
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"buf.build/foo/extdep4",
		},
		slicesext.Map(transitiveRemoteDeps, func(remoteDep bufmodule.RemoteDep) string { return remoteDep.OpaqueID() }),
	)
}

func TestProtoFileTargetPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	bucket := testNewBucketForPathToData(
		t,
		map[string][]byte{
			"a/1.proto": []byte(
				`syntax = proto3; package a;`,
			),
			"a/2.proto": []byte(
				`syntax = proto3; package a;`,
			),
			"also_a/1.proto": []byte(
				`syntax = proto3; package a;`,
			),
			"b/1.proto": []byte(
				`syntax = proto3; package b;`,
			),
			"b/2.proto": []byte(
				`syntax = proto3; package b;`,
			),
		},
	)

	moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, zap.NewNop(), bufmodule.NopModuleDataProvider)
	moduleSetBuilder.AddLocalModule(bucket, "module1", true)
	moduleSet, err := moduleSetBuilder.Build()
	require.NoError(t, err)
	module1 := moduleSet.GetModuleForOpaqueID("module1")
	require.NotNil(t, module1)
	testFilePaths(
		t,
		module1,
		"a/1.proto",
		"a/2.proto",
		"also_a/1.proto",
		"b/1.proto",
		"b/2.proto",
	)
	testTargetFilePaths(
		t,
		module1,
		"a/1.proto",
		"a/2.proto",
		"also_a/1.proto",
		"b/1.proto",
		"b/2.proto",
	)

	// The single file a/1.proto
	moduleSetBuilder = bufmodule.NewModuleSetBuilder(ctx, zap.NewNop(), bufmodule.NopModuleDataProvider)
	moduleSetBuilder.AddLocalModule(
		bucket,
		"module1",
		true,
		bufmodule.LocalModuleWithProtoFileTargetPath(
			"a/1.proto",
			false,
		),
	)
	moduleSet, err = moduleSetBuilder.Build()
	require.NoError(t, err)
	module1 = moduleSet.GetModuleForOpaqueID("module1")
	require.NotNil(t, module1)
	testFilePaths(
		t,
		module1,
		"a/1.proto",
		"a/2.proto",
		"also_a/1.proto",
		"b/1.proto",
		"b/2.proto",
	)
	testTargetFilePaths(
		t,
		module1,
		"a/1.proto",
	)

	// The single file a/1.proto with package files
	moduleSetBuilder = bufmodule.NewModuleSetBuilder(ctx, zap.NewNop(), bufmodule.NopModuleDataProvider)
	moduleSetBuilder.AddLocalModule(
		bucket,
		"module1",
		true,
		bufmodule.LocalModuleWithProtoFileTargetPath(
			"a/1.proto",
			true,
		),
	)
	moduleSet, err = moduleSetBuilder.Build()
	require.NoError(t, err)
	module1 = moduleSet.GetModuleForOpaqueID("module1")
	require.NotNil(t, module1)
	testFilePaths(
		t,
		module1,
		"a/1.proto",
		"a/2.proto",
		"also_a/1.proto",
		"b/1.proto",
		"b/2.proto",
	)
	testTargetFilePaths(
		t,
		module1,
		"a/1.proto",
		"a/2.proto",
		"also_a/1.proto",
	)
}

func testNewBucketForPathToData(t *testing.T, pathToData map[string][]byte) storage.ReadBucket {
	bucket, err := storagemem.NewReadBucket(pathToData)
	require.NoError(t, err)
	return bucket
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

func testFilePaths(t *testing.T, module bufmodule.Module, expectedFilePaths ...string) {
	ctx := context.Background()
	fileInfos, err := bufmodule.GetFileInfos(ctx, module)
	require.NoError(t, err)
	require.Equal(
		t,
		expectedFilePaths,
		bufmodule.FileInfoPaths(fileInfos),
	)
}

func testTargetFilePaths(t *testing.T, module bufmodule.Module, expectedFilePaths ...string) {
	ctx := context.Background()
	fileInfos, err := bufmodule.GetTargetFileInfos(ctx, module)
	require.NoError(t, err)
	require.Equal(
		t,
		expectedFilePaths,
		bufmodule.FileInfoPaths(fileInfos),
	)
}