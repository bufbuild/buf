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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifytesting"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testImportPathPrefix = "github.com/foo/bar/private/gen/proto/go"

func TestGoPackageError(t *testing.T) {
	t.Parallel()
	_, err := GoPackage(zap.NewNop(), NewFileOptionSweeper(), "", nil, nil, nil)
	require.Error(t, err)
}

func TestGoPackageEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})
}

func TestGoPackageAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.GoPackagePath)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, map[string]string{})
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.GoPackagePath)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})
}

func TestGoPackagePackageVersion(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "packageversion")
	packageSuffix := "weatherv1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.GoPackagePath)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				fmt.Sprintf("%s;%s",
					normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
					packageSuffix,
				),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				fmt.Sprintf("%s;%s",
					normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
					packageSuffix,
				),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.GoPackagePath)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
				continue
			}
			assert.Equal(t,
				fmt.Sprintf("%s;%s",
					normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
					packageSuffix,
				),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
				continue
			}
			assert.Equal(t,
				fmt.Sprintf("%s;%s",
					normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
					packageSuffix,
				),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})
}

func TestGoPackageWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "wktimport")
	packageSuffix := "weatherv1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			modifiedGoPackage := fmt.Sprintf("%s;%s",
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				packageSuffix,
			)
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetGoPackage())
				assert.NotEqual(t, modifiedGoPackage, descriptor.GetOptions().GetGoPackage())
				continue
			}
			assert.Equal(t,
				modifiedGoPackage,
				descriptor.GetOptions().GetGoPackage(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(zap.NewNop(), sweeper, testImportPathPrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			modifiedGoPackage := fmt.Sprintf("%s;%s",
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				packageSuffix,
			)
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetGoPackage())
				assert.NotEqual(t, modifiedGoPackage, descriptor.GetOptions().GetGoPackage())
				continue
			}
			assert.Equal(t,
				modifiedGoPackage,
				descriptor.GetOptions().GetGoPackage(),
			)
		}
	})
}

func TestGoPackageWithExcept(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		require.NoError(t, err)
		err = modifier.Modify(
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{"a.proto": "override"},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{"a.proto": "override"},
		)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})
}

func TestGoPackageWithOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	overrideGoPackagePrefix := "github.com/foo/bar/private/private/gen/proto/go"
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideGoPackagePrefix,
			},
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(overrideGoPackagePrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideGoPackagePrefix,
			},
			nil,
		)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(overrideGoPackagePrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideGoPackagePrefix,
			},
			map[string]string{"a.proto": "override"},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(
			zap.NewNop(),
			sweeper,
			testImportPathPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideGoPackagePrefix,
			},
			map[string]string{"a.proto": "override"},
		)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetGoPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.GoPackagePath, false)
	})
}
