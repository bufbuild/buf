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

func TestJavaMultipleFilesEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDataDir, "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, false)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			javaMultipleFilesModifier,
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
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"}, false)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			javaMultipleFilesModifier,
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
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"}, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDataDir, "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaMultipleFilesPath)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, false)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			javaMultipleFilesModifier,
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
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaMultipleFilesPath)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"}, false)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			javaMultipleFilesModifier,
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
				assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaMultipleFilesPath)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"}, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})

	t.Run("with preserveExistingValue", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, true)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesJavaOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDataDir, "javaoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaMultipleFilesPath)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, false, nil, false)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			javaMultipleFilesModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, false, nil, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
				continue
			}
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaMultipleFilesPath)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, false, map[string]string{"override.proto": "true"}, false)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			javaMultipleFilesModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
				continue
			}
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, false, map[string]string{"override.proto": "true"}, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
				continue
			}
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})

	t.Run("with preserveExistingValue", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, false, nil, true)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join(testDataDir, "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, false)
		require.NoError(t, err)
		modifier := NewMultiModifier(
			javaMultipleFilesModifier,
			ModifierFunc(sweeper.Sweep),
		)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetJavaMultipleFiles())
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetJavaMultipleFiles())
		}
	})
}
