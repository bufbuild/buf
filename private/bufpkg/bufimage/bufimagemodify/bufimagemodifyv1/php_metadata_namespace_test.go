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

func TestPhpMetadataNamespaceEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		phpMetadataNamespaceModifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpMetadataNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, true), image)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		phpMetadataNamespaceModifier := PhpMetadataNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})

		modifier := NewMultiModifier(phpMetadataNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		require.Equal(t, "override", descriptor.GetOptions().GetPhpMetadataNamespace())
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpMetadataNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		require.Equal(t, "override", descriptor.GetOptions().GetPhpMetadataNamespace())
	})
}

func TestPhpMetadataNamespaceAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpMetadataNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpMetadataNamespaceModifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpMetadataNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpMetadataNamespacePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpMetadataNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpMetadataNamespace())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpMetadataNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpMetadataNamespaceModifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpMetadataNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpMetadataNamespacePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpMetadataNamespace(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpMetadataNamespace())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)
	})
}

func TestPhpMetadataNamespaceOptions(t *testing.T) {
	t.Parallel()
	testPhpMetadataNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "single"), `Acme\V1\GPBMetadata`)
	testPhpMetadataNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "double"), `Acme\Weather\V1\GPBMetadata`)
	testPhpMetadataNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "triple"), `Acme\Weather\Data\V1\GPBMetadata`)
	testPhpMetadataNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "reserved"), `Acme\Error_\V1\GPBMetadata`)
}

func testPhpMetadataNamespaceOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpMetadataNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpMetadataNamespaceModifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpMetadataNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpMetadataNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpMetadataNamespaceModifier := PhpMetadataNamespace(zap.NewNop(), sweeper, map[string]string{"override.proto": "override"})

		modifier := NewMultiModifier(phpMetadataNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpMetadataNamespace())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpMetadataNamespace(zap.NewNop(), sweeper, map[string]string{"override.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetPhpMetadataNamespace())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpMetadataNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpMetadataNamespacePath, false)
	})
}

func TestPhpMetadataNamespaceWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	modifiedPhpMetadataNamespace := `Acme\Weather\V1alpha1\GPBMetadata`
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		phpMetadataNamespaceModifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)

		modifier := NewMultiModifier(phpMetadataNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				// php_namespace is unset for the well-known types
				assert.Empty(t, descriptor.GetOptions().GetPhpMetadataNamespace())
				continue
			}
			assert.Equal(t,
				modifiedPhpMetadataNamespace,
				descriptor.GetOptions().GetPhpMetadataNamespace(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpMetadataNamespace(zap.NewNop(), sweeper, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				// php_namespace is unset for the well-known types
				assert.Empty(t, descriptor.GetOptions().GetPhpMetadataNamespace())
				continue
			}
			assert.Equal(t,
				modifiedPhpMetadataNamespace,
				descriptor.GetOptions().GetPhpMetadataNamespace(),
			)
		}
	})
}
