// Copyright 2020-2022 Buf Technologies, Inc.
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
	"go.uber.org/zap"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

func TestJavaOuterClassnameEmptyOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "emptyoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaOuterClassname(zap.NewNop(), sweeper, nil),
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
			assert.Equal(t, "AProto", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaOuterClassname(zap.NewNop(), sweeper, nil).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "AProto", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaOuterClassname(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"}),
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
			assert.Equal(t, "override", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaOuterClassname(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"}).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)
		assert.NotEqual(t, testGetImage(t, dirPath, false), image)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)
	})
}

func TestJavaOuterClassnameAllOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "alloptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaOuterClassnamePath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaOuterClassname(zap.NewNop(), sweeper, nil),
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
			assert.Equal(t, "AProto", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaOuterClassname(zap.NewNop(), sweeper, nil).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "AProto", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaOuterClassnamePath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaOuterClassname(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"}),
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
			assert.Equal(t, "override", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaOuterClassname(zap.NewNop(), sweeper, map[string]string{"a.proto": "override"}).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, "override", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)
	})
}

func TestJavaOuterClassnameJavaOptions(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "javaoptions")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaOuterClassnamePath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaOuterClassname(zap.NewNop(), sweeper, nil),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, stringutil.ToPascalCase(normalpath.Base(imageFile.Path())), descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaOuterClassname(zap.NewNop(), sweeper, nil).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			assert.Equal(t, stringutil.ToPascalCase(normalpath.Base(imageFile.Path())), descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)
	})

	t.Run("with SourceCodeInfo and per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)
		assertFileOptionSourceCodeInfoNotEmpty(t, image, javaOuterClassnamePath)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaOuterClassname(zap.NewNop(), sweeper, map[string]string{"override.proto": "override"}),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetJavaOuterClassname())
				continue
			}
			assert.Equal(t, "JavaFileProto", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, true)
	})

	t.Run("without SourceCodeInfo and with per-file overrides", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaOuterClassname(zap.NewNop(), sweeper, map[string]string{"override.proto": "override"}).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if imageFile.Path() == "override.proto" {
				assert.Equal(t, "override", descriptor.GetOptions().GetJavaOuterClassname())
				continue
			}
			assert.Equal(t, "JavaFileProto", descriptor.GetOptions().GetJavaOuterClassname())
		}
		assertFileOptionSourceCodeInfoEmpty(t, image, javaOuterClassnamePath, false)
	})
}

func TestJavaOuterClassnameWellKnownTypes(t *testing.T) {
	t.Parallel()
	dirPath := filepath.Join("testdata", "wktimport")
	t.Run("with SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, true)

		sweeper := NewFileOptionSweeper()
		modifier := NewMultiModifier(
			JavaOuterClassname(zap.NewNop(), sweeper, nil),
			ModifierFunc(sweeper.Sweep),
		)
		err := modifier.Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.Equal(t, javaOuterClassnameValue(imageFile), descriptor.GetOptions().GetJavaOuterClassname())
				continue
			}
			assert.Equal(t,
				"AProto",
				descriptor.GetOptions().GetJavaOuterClassname(),
			)
		}
	})

	t.Run("without SourceCodeInfo", func(t *testing.T) {
		t.Parallel()
		image := testGetImage(t, dirPath, false)

		sweeper := NewFileOptionSweeper()
		err := JavaOuterClassname(zap.NewNop(), sweeper, nil).Modify(
			context.Background(),
			image,
		)
		require.NoError(t, err)

		for _, imageFile := range image.Files() {
			descriptor := imageFile.Proto()
			if isWellKnownType(context.Background(), imageFile) {
				assert.Equal(t, javaOuterClassnameValue(imageFile), descriptor.GetOptions().GetJavaOuterClassname())
				continue
			}
			assert.Equal(t,
				"AProto",
				descriptor.GetOptions().GetJavaOuterClassname(),
			)
		}
	})
}
