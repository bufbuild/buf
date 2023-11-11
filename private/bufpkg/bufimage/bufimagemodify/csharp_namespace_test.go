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

package bufimagemodify

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCsharpNamespaceEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

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
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
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
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "foo"})

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		// Overwritten with "foo" in the namespace
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].FileDescriptorProto()
		assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "foo"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		// Overwritten with "foo" in the namespace
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].FileDescriptorProto()
		assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
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
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, csharpNamespacePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, csharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "bar"})

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			// Overwritten with "bar" in the namespace
			assert.Equal(t, "bar", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "bar"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			// Overwritten with "bar" in the namespace
			assert.Equal(t, "bar", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)
	})
}

func TestCsharpNamespaceOptions(t *testing.T) {
	t.Parallel()
	testCsharpNamespaceOptions(t, filepath.Join("testdata", "csharpoptions", "single"), "Acme.V1")
	testCsharpNamespaceOptions(t, filepath.Join("testdata", "csharpoptions", "double"), "Acme.Weather.V1")
	testCsharpNamespaceOptions(t, filepath.Join("testdata", "csharpoptions", "triple"), "Acme.Weather.Data.V1")
	testCsharpNamespaceOptions(t, filepath.Join("testdata", "csharpoptions", "underscore"), "Acme.Weather.FooBar.V1")
}

func testCsharpNamespaceOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, csharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file options", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, csharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"override.proto": "Acme.Override.V1"})

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "Acme.Override.V1", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"override.proto": "Acme.Override.V1"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "Acme.Override.V1", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
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
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
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
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
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

func TestCsharpNamespaceWithExcept(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	testModuleFullName, err := bufmoduleref.NewModuleFullName(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmodule.ModuleFullName{testModuleFullName},
			nil,
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
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
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmodule.ModuleFullName{testModuleFullName},
			nil,
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
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
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmodule.ModuleFullName{testModuleFullName},
			nil,
			map[string]string{"a.proto": "override"},
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, "", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmodule.ModuleFullName{testModuleFullName},
			nil,
			map[string]string{"a.proto": "override"},
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, "", descriptor.GetOptions().GetCsharpNamespace())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}

func TestCsharpNamespaceWithOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	overrideCsharpNamespacePrefix := "x.y.z"
	testModuleFullName, err := bufmoduleref.NewModuleFullName(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmodule.ModuleFullName]string{
				testModuleFullName: overrideCsharpNamespacePrefix,
			},
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(overrideCsharpNamespacePrefix+"/"+imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmodule.ModuleFullName]string{
				testModuleFullName: overrideCsharpNamespacePrefix,
			},
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(overrideCsharpNamespacePrefix+"/"+imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmodule.ModuleFullName]string{
				testModuleFullName: overrideCsharpNamespacePrefix,
			},
			map[string]string{"a.proto": "override"},
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.FileDescriptorProto().Name != nil && *imageFile.FileDescriptorProto().Name == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, csharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmodule.ModuleFullName]string{
				testModuleFullName: overrideCsharpNamespacePrefix,
			},
			map[string]string{"a.proto": "override"},
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.FileDescriptorProto().Name != nil && *imageFile.FileDescriptorProto().Name == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}
