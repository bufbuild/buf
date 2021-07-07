// Copyright 2020-2021 Buf Technologies, Inc.
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

package bufmodule

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufcoretesting"
	modulev1alpha1 "github.com/bufbuild/buf/internal/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetingModuleBasic(t *testing.T) {
	ctx := context.Background()
	module, err := NewModuleForProto(
		ctx,
		&modulev1alpha1.Module{
			Files: []*modulev1alpha1.ModuleFile{
				{
					Path:    "a/a.proto",
					Content: []byte(`syntax = "proto3"; package a;`),
				},
				{
					Path:    "a/b.proto",
					Content: []byte(`syntax = "proto3"; package a;`),
				},
				{
					Path:    "b/a.proto",
					Content: []byte(`syntax = "proto3"; package b; import "a/a.proto";`),
				},
				{
					Path:    "b/b.proto",
					Content: []byte(`syntax = "proto3"; package b; import "a/b.proto";`),
				},
				{
					Path:    "c/c.proto/a.proto",
					Content: []byte(`syntax = "proto3"; package c; import "b/a.proto";`),
				},
				{
					Path:    "c/c.proto/b.proto",
					Content: []byte(`syntax = "proto3"; package c; import "b/b.proto";`),
				},
			},
		},
	)
	require.NoError(t, err)

	fileInfos, err := module.SourceFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "a/a.proto", "a/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "a/b.proto", "a/b.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/a.proto", "b/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/b.proto", "b/b.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "c/c.proto/a.proto", "c/c.proto/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "c/c.proto/b.proto", "c/c.proto/b.proto", false), nil, ""),
		},
		fileInfos,
	)

	targetModule, err := ModuleWithTargetPaths(
		module,
		[]string{
			"b/a.proto",
			"b/b.proto",
		},
	)
	require.NoError(t, err)
	targetFileInfos, err := targetModule.TargetFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/a.proto", "b/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/b.proto", "b/b.proto", false), nil, ""),
		},
		targetFileInfos,
	)

	targetModule, err = ModuleWithTargetPaths(
		module,
		[]string{
			"b",
		},
	)
	require.NoError(t, err)
	targetFileInfos, err = targetModule.TargetFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/a.proto", "b/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/b.proto", "b/b.proto", false), nil, ""),
		},
		targetFileInfos,
	)

	targetModule, err = ModuleWithTargetPaths(
		module,
		[]string{
			"b",
			"b/a.proto",
		},
	)
	require.NoError(t, err)
	targetFileInfos, err = targetModule.TargetFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/a.proto", "b/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/b.proto", "b/b.proto", false), nil, ""),
		},
		targetFileInfos,
	)

	targetModule, err = ModuleWithTargetPaths(
		module,
		[]string{
			"b",
			"a",
		},
	)
	require.NoError(t, err)
	targetFileInfos, err = targetModule.TargetFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "a/a.proto", "a/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "a/b.proto", "a/b.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/a.proto", "b/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/b.proto", "b/b.proto", false), nil, ""),
		},
		targetFileInfos,
	)

	targetModule, err = ModuleWithTargetPaths(
		module,
		[]string{
			"b",
			// the directory is c/c.proto, not c.proto
			"c.proto",
		},
	)
	require.NoError(t, err)
	_, err = targetModule.TargetFileInfos(ctx)
	require.Error(t, err)

	targetModule, err = ModuleWithTargetPathsAllowNotExist(
		module,
		[]string{
			"b",
			// the directory is c/c.proto, not c.proto
			"c.proto",
		},
	)
	require.NoError(t, err)
	targetFileInfos, err = targetModule.TargetFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/a.proto", "b/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/b.proto", "b/b.proto", false), nil, ""),
		},
		targetFileInfos,
	)

	targetModule, err = ModuleWithTargetPaths(
		module,
		[]string{
			"b",
			"c/c.proto",
		},
	)
	require.NoError(t, err)
	targetFileInfos, err = targetModule.TargetFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/a.proto", "b/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "b/b.proto", "b/b.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "c/c.proto/a.proto", "c/c.proto/a.proto", false), nil, ""),
			NewFileInfo(bufcoretesting.NewFileInfo(t, "c/c.proto/b.proto", "c/c.proto/b.proto", false), nil, ""),
		},
		targetFileInfos,
	)

	targetModule, err = ModuleWithTargetPaths(
		module,
		[]string{
			"c/c.proto/a.proto",
		},
	)
	require.NoError(t, err)
	targetFileInfos, err = targetModule.TargetFileInfos(ctx)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]FileInfo{
			NewFileInfo(bufcoretesting.NewFileInfo(t, "c/c.proto/a.proto", "c/c.proto/a.proto", false), nil, ""),
		},
		targetFileInfos,
	)
}
