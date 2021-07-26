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

func TestObjcClassPrefixEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, objcClassPrefixPath, true)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(sweeper)

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
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
		assertFileOptionSourceCodeInfoEmpty(t, image, objcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(sweeper)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.Equal(t, testGetImage(t, dirPath, false), image)
	})
}

func TestObjcClassPrefixAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, objcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(sweeper)

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
		assertFileOptionSourceCodeInfoNotEmpty(t, image, objcClassPrefixPath)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, objcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(sweeper)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "foo", descriptor.GetOptions().GetObjcClassPrefix())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, objcClassPrefixPath, false)
	})
}

func TestObjcClassPrefixObjcOptions(t *testing.T) {
	t.Parallel()
	testObjcClassPrefixOptions(t, filepath.Join("testdata", "objcoptions", "single"), "AXX")
	testObjcClassPrefixOptions(t, filepath.Join("testdata", "objcoptions", "double"), "AWX")
	testObjcClassPrefixOptions(t, filepath.Join("testdata", "objcoptions", "triple"), "AWD")
	testObjcClassPrefixOptions(t, filepath.Join("testdata", "objcoptions", "unversioned"), "AWD")
	testObjcClassPrefixOptions(t, filepath.Join("testdata", "objcoptions", "gpb"), "GPX")
}

func testObjcClassPrefixOptions(t *testing.T, dirPath string, classPrefix string) {
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, objcClassPrefixPath)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(sweeper)

		modifier := NewMultiModifier(objcClassPrefixModifier, ModifierFunc(sweeper.Sweep))
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, objcClassPrefixPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, objcClassPrefixPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(sweeper)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, classPrefix, descriptor.GetOptions().GetObjcClassPrefix())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, objcClassPrefixPath, false)
	})
}

func TestObjcClassPrefixWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	modifiedObjcClassPrefix := "AWX"
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		objcClassPrefixModifier := ObjcClassPrefix(sweeper)

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
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		modifier := ObjcClassPrefix(sweeper)
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
