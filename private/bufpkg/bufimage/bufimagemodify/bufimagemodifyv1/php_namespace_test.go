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

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifytesting"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPhpNamespaceEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(zap.NewNop(), sweeper, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		assert.Equal(t, "override", descriptor.GetOptions().GetPhpNamespace())
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		assert.Equal(t, "override", descriptor.GetOptions().GetPhpNamespace())
	})
}

func TestPhpNamespaceAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.PhpNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.PhpNamespacePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(zap.NewNop(), sweeper, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.PhpNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpNamespace())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpNamespace())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)
	})
}

func TestPhpNamespaceOptions(t *testing.T) {
	t.Parallel()
	testPhpNamespaceOptions(t, filepath.Join("..", "testdata", "phpoptions", "single"), `Acme\V1`)
	testPhpNamespaceOptions(t, filepath.Join("..", "testdata", "phpoptions", "double"), `Acme\Weather\V1`)
	testPhpNamespaceOptions(t, filepath.Join("..", "testdata", "phpoptions", "triple"), `Acme\Weather\Data\V1`)
	testPhpNamespaceOptions(t, filepath.Join("..", "testdata", "phpoptions", "reserved"), `Acme\Error_\V1`)
	testPhpNamespaceOptions(t, filepath.Join("..", "testdata", "phpoptions", "underscore"), `Acme\Weather\FooBar\V1`)
}

func testPhpNamespaceOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.PhpNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(zap.NewNop(), sweeper, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.PhpNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(zap.NewNop(), sweeper, map[string]string{"override.proto": "override"})

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpNamespace())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(zap.NewNop(), sweeper, map[string]string{"override.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpNamespace())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.PhpNamespacePath, false)
	})
}

func TestPhpNamespaceWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "wktimport")
	modifiedPhpNamespace := `Acme\Weather\V1alpha1`
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if internal.IsWellKnownType(imageFile) {
				// php_namespace is unset for the well-known types
				assert.Empty(t, descriptor.GetOptions().GetPhpNamespace())
				continue
			}
			assert.Equal(t,
				modifiedPhpNamespace,
				descriptor.GetOptions().GetPhpNamespace(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(zap.NewNop(), sweeper, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if internal.IsWellKnownType(imageFile) {
				// php_namespace is unset for the well-known types
				assert.Empty(t, descriptor.GetOptions().GetPhpNamespace())
				continue
			}
			assert.Equal(t,
				modifiedPhpNamespace,
				descriptor.GetOptions().GetPhpNamespace(),
			)
		}
	})
}
