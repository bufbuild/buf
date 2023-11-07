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
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRubyPackageEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, true)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
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
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)
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
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, true)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "override"})

		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].FileDescriptorProto()
		assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		require.Equal(t, 1, len(image.Files()))
		descriptor := image.Files()[0].FileDescriptorProto()
		assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
	})
}

func TestRubyPackageAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "override"})

		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, map[string]string{"a.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "a.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t, "foo", descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})
}

func TestRubyPackageOptions(t *testing.T) {
	t.Parallel()
	testRubyPackageOptions(t, filepath.Join("testdata", "rubyoptions", "single"), `Acme::V1`)
	testRubyPackageOptions(t, filepath.Join("testdata", "rubyoptions", "double"), `Acme::Weather::V1`)
	testRubyPackageOptions(t, filepath.Join("testdata", "rubyoptions", "triple"), `Acme::Weather::Data::V1`)
	testRubyPackageOptions(t, filepath.Join("testdata", "rubyoptions", "underscore"), `Acme::Weather::FooBar::V1`)
}

func testRubyPackageOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, map[string]string{"override.proto": "override"})

		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, map[string]string{"override.proto": "override"})
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})
}

func TestRubyPackageWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	modifiedRubyPackage := `Acme::Weather::V1alpha1`
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)

		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if isWellKnownType(context.Background(), imageFile) {
				// php_namespace is unset for the well-known types
				assert.Empty(t, descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t,
				modifiedRubyPackage,
				descriptor.GetOptions().GetRubyPackage(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := RubyPackage(zap.NewNop(), sweeper, nil, nil, nil)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if isWellKnownType(context.Background(), imageFile) {
				// php_namespace is unset for the well-known types
				assert.Empty(t, descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t,
				modifiedRubyPackage,
				descriptor.GetOptions().GetRubyPackage(),
			)
		}
	})
}

func TestRubyPackageExcept(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "rubyoptions", "single")
	testModuleIdentity, err := bufmoduleref.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, true), image)
		// Still not empty, because the module is in except
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)
	})
	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			nil,
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})
	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		// not modified even though a file has override, because the module is in except
		assert.Equal(t, testGetImage(t, dirPath, true), image)
		// Still not empty, because the module is in except
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)
	})
	t.Run("without SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			[]bufmoduleref.ModuleIdentity{testModuleIdentity},
			nil,
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})
}

func TestRubyPackageOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "rubyoptions", "single")
	overrideRubyPackage := "MODULE"
	testModuleIdentity, err := bufmoduleref.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideRubyPackage},
			nil,
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, overrideRubyPackage, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, true)
	})
	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideRubyPackage},
			nil,
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			assert.Equal(t, overrideRubyPackage, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})
	t.Run("with SourceCodeInfo with per-file override", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, rubyPackagePath)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideRubyPackage},
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t, overrideRubyPackage, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, true)
	})
	t.Run("without SourceCodeInfo and per-file override", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)

		sweeper := NewFileOptionSweeper()
		rubyPackageModifier := RubyPackage(
			zap.NewNop(),
			sweeper,
			nil,
			map[bufmoduleref.ModuleIdentity]string{testModuleIdentity: overrideRubyPackage},
			map[string]string{"override.proto": "override"},
		)
		modifier := NewMultiModifier(rubyPackageModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.FileDescriptorProto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetRubyPackage())
				continue
			}
			assert.Equal(t, overrideRubyPackage, descriptor.GetOptions().GetRubyPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, rubyPackagePath, false)
	})
}
