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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestObjcClassPrefixEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)
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
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, map[string]string{"a.proto": "override"})

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		require.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].Proto()
		require.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
	})
}

func TestObjcClassPrefixAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, map[string]string{"a.proto": "override"})

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})
}

func TestObjcClassPrefixObjcOptions(t *testing.T) {
	t.Parallel()
	testObjcClassPrefixOptions(t, filepath.Join("..", "testdata", "objcoptions", "single"), "AXX")
	testObjcClassPrefixOptions(t, filepath.Join("..", "testdata", "objcoptions", "double"), "AWX")
	testObjcClassPrefixOptions(t, filepath.Join("..", "testdata", "objcoptions", "triple"), "AWD")
	testObjcClassPrefixOptions(t, filepath.Join("..", "testdata", "objcoptions", "unversioned"), "AWD")
	testObjcClassPrefixOptions(t, filepath.Join("..", "testdata", "objcoptions", "gpb"), "GPX")
}

func testObjcClassPrefixOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, map[string]string{"override.proto": "override"})

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, map[string]string{"override.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})
}

func TestObjcClassPrefixWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "wktimport")
	modifiedObjcClassPrefix := "AWX"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetObjcClassPrefix())
				assert.NotEqual(t, modifiedObjcClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t,
				modifiedObjcClassPrefix,
				descriptor.GetOptions().GetObjcClassPrefix(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(zap.NewNop(), sweeper, "", nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.NotEmpty(t, descriptor.GetOptions().GetObjcClassPrefix())
				assert.NotEqual(t, modifiedObjcClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t,
				modifiedObjcClassPrefix,
				descriptor.GetOptions().GetObjcClassPrefix(),
			)
		}
	})
}

func TestObjcClassPrefixWithDefault(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "objcoptions", "single")
	defaultClassPrefix := "DEFAULT"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			nil,
			nil,
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, defaultClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			nil,
			nil,
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, defaultClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			nil,
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, defaultClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			nil,
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, defaultClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})
}

func TestObjcClassPrefixWithExcept(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "objcoptions", "single")
	defaultClassPrefix := "DEFAULT"
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)
		// Should still be non-empty because the module is skipped.
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, true), image)
		// Should still be non-empty because the module is skipped.
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, bufimagemodifytesting.GetTestImage(t, dirPath, false), image)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})
}

func TestObjcClassPrefixWithOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("..", "testdata", "objcoptions", "single")
	testModuleIdentity := bufimagemodifytesting.GetTestModuleIdentity(t)
	defaultClassPrefix := "DEFAULT"
	overrideClassPrefix := "MODULE_OVERRIDE"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideClassPrefix},
			nil,
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, overrideClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideClassPrefix},
			nil,
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, overrideClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, true)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoNotEmpty(t, image, internal.ObjcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideClassPrefix},
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, overrideClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := bufimagemodifytesting.GetTestImage(t, dirPath, false)
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(
			zap.NewNop(),
			sweeper,
			defaultClassPrefix,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideClassPrefix},
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetObjcClassPrefix())
				continue
			}
			assert.Equal(t, overrideClassPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, internal.ObjcClassPrefixPath, false)
	})
}
