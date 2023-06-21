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

package bufimagemodifyv1

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestOptimizeForEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			optimizeForModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, nil)
		require.NoError(t, err)
		err = optimizeForModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, map[string]string{"a.proto": "SPEED"})
		require.NoError(t, err)
		modifier := NewMultiModifier(
			optimizeForModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
				continue
			}
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, map[string]string{"a.proto": "SPEED"})
		require.NoError(t, err)
		err = optimizeForModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
				continue
			}
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)
	})
}

func TestOptimizeForAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, optimizeForPath)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			optimizeForModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, nil)
		require.NoError(t, err)
		err = optimizeForModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, optimizeForPath)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, map[string]string{"a.proto": "SPEED"})
		require.NoError(t, err)
		modifier := NewMultiModifier(
			optimizeForModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
				continue
			}
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, optimizeForPath)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_LITE_RUNTIME, nil, nil, map[string]string{"a.proto": "SPEED"})
		require.NoError(t, err)
		err = optimizeForModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
				continue
			}
			assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)
	})
}

func TestOptimizeForCcOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "ccoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, optimizeForPath)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_SPEED, nil, nil, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			optimizeForModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_SPEED, nil, nil, nil)
		require.NoError(t, err)
		err = optimizeForModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, optimizeForPath)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_SPEED, nil, nil, map[string]string{"a.proto": "LITE_RUNTIME"})
		require.NoError(t, err)
		modifier := NewMultiModifier(
			optimizeForModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
				continue
			}
			assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
		}
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_SPEED, nil, nil, map[string]string{"a.proto": "LITE_RUNTIME"})
		require.NoError(t, err)
		err = optimizeForModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, descriptorpb.FileOptions_LITE_RUNTIME, descriptor.GetOptions().GetOptimizeFor())
				continue
			}
			assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, false)
	})
}

func TestOptimizeForWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_SPEED, nil, nil, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			optimizeForModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(zap.NewNop(), sweeper, descriptorpb.FileOptions_SPEED, nil, nil, nil)
		require.NoError(t, err)
		err = optimizeForModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, descriptorpb.FileOptions_SPEED, descriptor.GetOptions().GetOptimizeFor())
		}
	})
}

func TestOptimizeForWithExcept(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "emptyoptions")
	testModuleIdentity, err := bufmoduleref.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, true), image)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{
				"a.proto": "LITE_RUNTIME",
			},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{
				"a.proto": "SPEED",
			},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}

func TestOptimizeForWithOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "emptyoptions")
	overrideOptimizeFor := descriptorpb.FileOptions_LITE_RUNTIME
	testModuleIdentity, err := bufmoduleref.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			nil,
			map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode{
				testModuleIdentity: overrideOptimizeFor,
			},
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				overrideOptimizeFor,
				descriptor.GetOptions().GetOptimizeFor(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			nil,
			map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode{
				testModuleIdentity: overrideOptimizeFor,
			},
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				overrideOptimizeFor,
				descriptor.GetOptions().GetOptimizeFor(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			nil,
			map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode{
				testModuleIdentity: overrideOptimizeFor,
			},
			map[string]string{
				"a.proto": "CODE_SIZE",
			},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				descriptorpb.FileOptions_CODE_SIZE,
				descriptor.GetOptions().GetOptimizeFor(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(
			zap.NewNop(),
			sweeper,
			descriptorpb.FileOptions_CODE_SIZE,
			nil,
			map[bufmoduleref.ModuleIdentity]descriptorpb.FileOptions_OptimizeMode{
				testModuleIdentity: overrideOptimizeFor,
			},
			map[string]string{
				"a.proto": "CODE_SIZE",
			},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(optimizeForModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				descriptorpb.FileOptions_CODE_SIZE,
				descriptor.GetOptions().GetOptimizeFor(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}
