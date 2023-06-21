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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testJavaPackagePrefix = "com"

func TestJavaPackageError(t *testing.T) {
	t.Parallel()
	_, err := JavaPackage(zap.NewNop(), NewFileOptionSweeper(), "", nil, nil, nil)
	require.Error(t, err)
}

func TestJavaPackageEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
	})
}

func TestJavaPackageAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, map[string]string{"a.proto": "override"})
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})
}

func TestJavaPackageJavaOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "javaoptions")
	modifiedJavaPackage := "com.acme.weather"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, map[string]string{"override.proto": "override"})
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
				continue
			}
			assert.Equal(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(zap.NewNop(), sweeper, testJavaPackagePrefix, nil, nil, map[string]string{"override.proto": "override"})
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
				continue
			}
			assert.Equal(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})
}

func TestJavaPackageWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "wktimport")
	javaPackagePrefix := "org"
	modifiedJavaPackage := "org.acme.weather.v1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(zap.NewNop(), sweeper, javaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetJavaPackage())
				assert.NotEqual(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
				continue
			}
			assert.Equal(t,
				modifiedJavaPackage,
				descriptor.GetOptions().GetJavaPackage(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(zap.NewNop(), sweeper, javaPackagePrefix, nil, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetJavaPackage())
				assert.NotEqual(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
				continue
			}
			assert.Equal(t,
				modifiedJavaPackage,
				descriptor.GetOptions().GetJavaPackage(),
			)
		}
	})
}

func TestJavaPackageWithExcept(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "javaemptyoptions")
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{"a.proto": "override"},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})
}

func TestJavaPackageWithOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "javaemptyoptions")
	overrideJavaPackagePrefix := "foo.bar"
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideJavaPackagePrefix,
			},
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				overrideJavaPackagePrefix+"."+descriptor.GetPackage(),
				descriptor.GetOptions().GetJavaPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideJavaPackagePrefix,
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
				overrideJavaPackagePrefix+"."+descriptor.GetPackage(),
				descriptor.GetOptions().GetJavaPackage(),
			)
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideJavaPackagePrefix,
			},
			map[string]string{"a.proto": "override"},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(
			zap.NewNop(),
			sweeper,
			testJavaPackagePrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{
				testModuleIdentity: overrideJavaPackagePrefix,
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
			assert.Equal(t, "override", descriptor.GetOptions().GetJavaPackage())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})
}
