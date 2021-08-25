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

func TestCsharpNamespaceEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(sweeper)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
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
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(sweeper)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
	})
}

func TestCsharpNamespaceAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, csharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(sweeper)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, csharpNamespacePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(sweeper)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)
	})
}

func TestCsharpNamespaceOptions(t *testing.T) {
	t.Parallel()
	testCsharpNamespaceOptions(t, filepath.Join("testdata", "csharpoptions", "single"), "Acme.V1")
	testCsharpNamespaceOptions(t, filepath.Join("testdata", "csharpoptions", "double"), "Acme.Weather.V1")
	testCsharpNamespaceOptions(t, filepath.Join("testdata", "csharpoptions", "triple"), "Acme.Weather.Data.V1")
}

func testCsharpNamespaceOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, csharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(sweeper)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(sweeper)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)
	})
}

func TestCsharpNamespaceWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	modifiedCsharpNamespace := "Acme.Weather.V1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(sweeper)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetCsharpNamespace())
				assert.NotEqual(t, modifiedCsharpNamespace, descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t,
				modifiedCsharpNamespace,
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(sweeper)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetCsharpNamespace())
				assert.NotEqual(t, modifiedCsharpNamespace, descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t,
				modifiedCsharpNamespace,
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
	})
}
