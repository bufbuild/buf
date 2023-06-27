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

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal/bufimagemodifytesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestJavaMultipleFilesEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, true)

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
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, nil, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, true)

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
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		javaMultipleFilesModifier, err := JavaMultipleFiles(zap.NewNop(), sweeper, true, map[string]string{"a.proto": "false"}, false)
		require.NoError(t, err)
		err = javaMultipleFilesModifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.JavaMultipleFilesPath)

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
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.JavaMultipleFilesPath)

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
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
				continue
			}
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.JavaMultipleFilesPath)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})

	t.Run("with preserveExistingValue", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesJavaOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "javaoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.JavaMultipleFilesPath)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.JavaMultipleFilesPath)

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
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})

	t.Run("with preserveExistingValue", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.JavaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)

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
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)

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
