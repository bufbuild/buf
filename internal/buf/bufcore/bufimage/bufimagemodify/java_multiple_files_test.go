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

func TestJavaMultipleFilesEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaMultipleFiles(sweeper, true),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaMultipleFiles(sweeper, true).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaMultipleFilesPath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaMultipleFiles(sweeper, true),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, true), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaMultipleFiles(sweeper, true).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesJavaOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "javaoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaMultipleFilesPath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaMultipleFiles(sweeper, false),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaMultipleFiles(sweeper, false).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetJavaMultipleFiles())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaMultipleFilesPath, false)
	})
}

func TestJavaMultipleFilesWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaMultipleFiles(sweeper, true),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetJavaMultipleFiles())
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaMultipleFiles(sweeper, true).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetJavaMultipleFiles())
		}
	})
}
