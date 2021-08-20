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

func TestCcEnableArenasEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			CcEnableArenas(sweeper, false),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		err := CcEnableArenas(sweeper, false).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.False(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})
}

func TestCcEnableArenasAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, ccEnableArenasPath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			CcEnableArenas(sweeper, true),
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
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		err := CcEnableArenas(sweeper, true).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})
}

func TestCcEnableArenasCcOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "ccoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, ccEnableArenasPath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			CcEnableArenas(sweeper, true),
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
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)

		sweeper := NewFileOptionSweeper()
		err := CcEnableArenas(sweeper, true).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.True(t, descriptor.GetOptions().GetCcEnableArenas())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, ccEnableArenasPath, false)
	})
}

func TestCcEnableArenasWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			CcEnableArenas(sweeper, true),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetCcEnableArenas())
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		err := CcEnableArenas(sweeper, true).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			assert.True(t, imageFile.Proto().GetOptions().GetCcEnableArenas())
		}
	})
}
