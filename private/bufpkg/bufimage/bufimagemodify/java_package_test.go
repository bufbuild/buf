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

const testJavaPackagePrefix = "com."

func TestJavaPackageError(t *testing.T) {
	t.Parallel()
	_, err := JavaPackage(NewFileOptionSweeper(), "")
	require.Error(t, err)
}

func TestJavaPackageEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(sweeper, testJavaPackagePrefix)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, true), image)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(sweeper, testJavaPackagePrefix)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
	})
}

func TestJavaPackageAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(sweeper, testJavaPackagePrefix)
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
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(sweeper, testJavaPackagePrefix)
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
		assertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})
}

func TestJavaPackageJavaOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "javaoptions")
	modifiedJavaPackage := "com.acme.weather"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaPackagePath)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(sweeper, testJavaPackagePrefix)
		require.NoError(t, err)

		modifier := NewMultiModifier(javaPackageModifier, ModifierFunc(sweeper.Sweep))
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(sweeper, testJavaPackagePrefix)
		require.NoError(t, err)
		err = modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, modifiedJavaPackage, descriptor.GetOptions().GetJavaPackage())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, false)
	})
}

func TestJavaPackageWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	javaPackagePrefix := "org."
	modifiedJavaPackage := "org.acme.weather.v1alpha1"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		javaPackageModifier, err := JavaPackage(sweeper, javaPackagePrefix)
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
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier, err := JavaPackage(sweeper, javaPackagePrefix)
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
