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
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifytesting"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCsharpNamespaceEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "foo"})

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		// Overwritten with "foo" in the namespace
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "foo"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		// Overwritten with "foo" in the namespace
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
	})
}

func TestCsharpNamespaceAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.CsharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.CsharpNamespacePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.CsharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "bar"})

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			// Overwritten with "bar" in the namespace
			assert.Equal(t, "bar", descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "bar"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			// Overwritten with "bar" in the namespace
			assert.Equal(t, "bar", descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)
	})
}

func TestCsharpNamespaceOptions(t *testing.T) {
	t.Parallel()
	testCsharpNamespaceOptions(t, filepath.Join("..", "testdata", "csharpoptions", "single"), "Acme.V1")
	testCsharpNamespaceOptions(t, filepath.Join("..", "testdata", "csharpoptions", "double"), "Acme.Weather.V1")
	testCsharpNamespaceOptions(t, filepath.Join("..", "testdata", "csharpoptions", "triple"), "Acme.Weather.Data.V1")
	testCsharpNamespaceOptions(t, filepath.Join("..", "testdata", "csharpoptions", "underscore"), "Acme.Weather.FooBar.V1")
}

func testCsharpNamespaceOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.CsharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)
	})

	t.Run("with SourceCodeInfo and per-file options", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.CsharpNamespacePath)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"override.proto": "Acme.Override.V1"})

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "Acme.Override.V1", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, map[string]string{"override.proto": "Acme.Override.V1"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "Acme.Override.V1", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)
	})
}

func TestCsharpNamespaceWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "wktimport")
	modifiedCsharpNamespace := "Acme.Weather.V1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if internal.IsWellKnownType(imageFile) {
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
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := CsharpNamespace(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if internal.IsWellKnownType(imageFile) {
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
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
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
			descriptor := imageFile.Proto()
			assert.Equal(t, "", descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
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
			descriptor := imageFile.Proto()
			assert.Equal(t, "", descriptor.GetOptions().GetCsharpNamespace())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})
}

func TestCsharpNamespaceWithOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	overrideCsharpNamespacePrefix := "x.y.z"
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideCsharpNamespacePrefix,
			},
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(overrideCsharpNamespacePrefix+"/"+imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideCsharpNamespacePrefix,
			},
			nil,
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(overrideCsharpNamespacePrefix+"/"+imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, true)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideCsharpNamespacePrefix,
			},
			map[string]string{"a.proto": "override"},
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Proto().Name != nil && *imageFile.Proto().Name == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.CsharpNamespacePath, false)

		sweeper := NewFileOptionSweeper()
		csharpNamespaceModifier := CsharpNamespace(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideCsharpNamespacePrefix,
			},
			map[string]string{"a.proto": "override"},
		)

		modifier := NewMultiModifier(csharpNamespaceModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Proto().Name != nil && *imageFile.Proto().Name == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetCsharpNamespace())
				continue
			}
			assert.Equal(t,
				strings.ReplaceAll(normalpath.Dir(imageFile.Path()), "/", "."),
				descriptor.GetOptions().GetCsharpNamespace(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})
}
