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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCcEnableArenasEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, false, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			ccEnableArenasModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, false, nil)
		require.NoError(t, err)
		err = ccEnableArenasModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, false, map[string]string{"a.proto": "true"})
		require.NoError(t, err)
		modifier := NewMultiModifier(
			ccEnableArenasModifier,
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
				assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
				continue
			}
			assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, false, map[string]string{"a.proto": "true"})
		require.NoError(t, err)
		err = ccEnableArenasModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
				continue
			}
			assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})
}

func TestCcEnableArenasAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, ccEnableArenasPath)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			ccEnableArenasModifier,
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
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, nil)
		require.NoError(t, err)
		err = ccEnableArenasModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, ccEnableArenasPath)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"})
		require.NoError(t, err)
		modifier := NewMultiModifier(
			ccEnableArenasModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, ccEnableArenasPath)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"})
		require.NoError(t, err)
		err = ccEnableArenasModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})
}

func TestCcEnableArenasCcOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "ccoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, ccEnableArenasPath)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			ccEnableArenasModifier,
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
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, nil)
		require.NoError(t, err)
		err = ccEnableArenasModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, ccEnableArenasPath)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"})
		require.NoError(t, err)
		modifier := NewMultiModifier(
			ccEnableArenasModifier,
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
				assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"})
		require.NoError(t, err)
		err = ccEnableArenasModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})
}

func TestCcEnableArenasWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDir, "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, nil)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			ccEnableArenasModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetCcEnableArenas())
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		ccEnableArenasModifier, err := CcEnableArenas(zap.NewNop(), sweeper, true, nil)
		require.NoError(t, err)
		err = ccEnableArenasModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetCcEnableArenas())
		}
	})
}
