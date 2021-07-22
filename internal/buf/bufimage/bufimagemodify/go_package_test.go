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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoPackageError(t *testing.T) {
	t.Parallel()
	_, err := GoPackage(NewFileOptionSweeper(), "", nil, nil)
	require.Error(t, err)
}

func TestGoPackageEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(testImportPathPrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}

func TestGoPackageAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, goPackagePath)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
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
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
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
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}

func TestGoPackagePackageVersion(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "packageversion")
	packageSuffix := "weatherv1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, goPackagePath)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

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
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

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
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}

func TestGoPackageWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	packageSuffix := "weatherv1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
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
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(sweeper, testImportPathPrefix, nil, nil)
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
	dirPath := filepath.Join("testdata", "emptyoptions")
	testModuleIdentity, err := bufmodule.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(
			sweeper,
			testImportPathPrefix,
			[]bufmodule.ModuleIdentity{testModuleIdentity},
			nil,
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
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
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(
			sweeper,
			testImportPathPrefix,
			[]bufmodule.ModuleIdentity{testModuleIdentity},
			nil,
		)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}

func TestGoPackageWithOverride(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	overrideGoPackagePrefix := "github.com/foo/bar/private/internal/gen/proto/go"
	testModuleIdentity, err := bufmodule.NewModuleIdentity(
		testRemote,
		testRepositoryOwner,
		testRepositoryName,
	)
	require.NoError(t, err)

	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)

		sweeper := NewFileOptionSweeper()
		goPackageModifier, err := GoPackage(
			sweeper,
			testImportPathPrefix,
			nil,
			map[bufmodule.ModuleIdentity]string{
				testModuleIdentity: overrideGoPackagePrefix,
			},
		)
		require.NoError(t, err)

		modifier := NewMultiModifier(goPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(overrideGoPackagePrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := GoPackage(
			sweeper,
			testImportPathPrefix,
			nil,
			map[bufmodule.ModuleIdentity]string{
				testModuleIdentity: overrideGoPackagePrefix,
			},
		)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t,
				normalpath.Dir(overrideGoPackagePrefix+"/"+imageFile.Path()),
				descriptor.GetOptions().GetGoPackage(),
			)
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, goPackagePath, false)
	})
}
