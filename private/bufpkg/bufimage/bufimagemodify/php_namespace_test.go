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
)

func TestPhpNamespaceEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(sweeper, map[string]string{})

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
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
		assertFileOptionSourceCodeInfoEmpty(t, image, phpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(sweeper, map[string]string{})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
	})
}

func TestPhpNamespaceAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(sweeper, map[string]string{})

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
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpNamespacePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(sweeper, map[string]string{})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetPhpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpNamespacePath, false)
	})
}

func TestPhpNamespaceOptions(t *testing.T) {
	t.Parallel()
	testPhpNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "single"), `Acme\V1`)
	testPhpNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "double"), `Acme\Weather\V1`)
	testPhpNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "triple"), `Acme\Weather\Data\V1`)
	testPhpNamespaceOptions(t, filepath.Join("testdata", "phpoptions", "reserved"), `Acme\Error_\V1`)
}

func testPhpNamespaceOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, phpNamespacePath)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(sweeper, map[string]string{})

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, phpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(sweeper, map[string]string{})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetPhpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, phpNamespacePath, false)
	})
}

func TestPhpNamespaceWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	modifiedPhpNamespace := `Acme\Weather\V1alpha1`
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		phpNamespaceModifier := PhpNamespace(sweeper, map[string]string{})

		modifier := NewMultiModifier(phpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
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
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := PhpNamespace(sweeper, map[string]string{})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
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
