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

package bufimagemodify

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestOptimizeForEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, optimizeForPath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_LITE_RUNTIME, map[string]string{})
		assert.NoError(t, err)
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
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_LITE_RUNTIME, map[string]string{})
		assert.NoError(t, err)
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
}

func TestOptimizeForAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, optimizeForPath)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_LITE_RUNTIME, map[string]string{})
		assert.NoError(t, err)
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
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_LITE_RUNTIME, map[string]string{})
		assert.NoError(t, err)
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
}

func TestOptimizeForCcOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "ccoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, optimizeForPath)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_SPEED, map[string]string{})
		assert.NoError(t, err)
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
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_SPEED, map[string]string{})
		assert.NoError(t, err)
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
}

func TestOptimizeForWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_SPEED, map[string]string{})
		assert.NoError(t, err)
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
		optimizeForModifier, err := OptimizeFor(sweeper, descriptorpb.FileOptions_SPEED, map[string]string{})
		assert.NoError(t, err)
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
