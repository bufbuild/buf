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

package bufimagemodifyv2

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal/bufimagemodifytesting"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifyJSType(t *testing.T) {
	t.Parallel()
	for _, includeSourceCodeInfo := range []bool{true, false} {
		image := bufimagemodifytesting.GetTestImage(
			t,
			filepath.Join("..", "testdata", "fieldoptions"),
			includeSourceCodeInfo,
		)
		require.NotNil(t, image)
		markSweeper := newMarkSweeper(image)
		require.NotNil(t, markSweeper)
		imageFile := image.GetFile("a.proto")
		require.NotNil(t, imageFile)
		modifier, err := NewFieldOptionModifier(imageFile, markSweeper)
		require.NoError(t, err)
		expectedNames := []string{
			"foo.bar.baz.Outer.Inner.i1",
			"foo.bar.baz.Outer.Inner.i2",
			"foo.bar.baz.Outer.Inner.i3",
			"foo.bar.baz.Outer.Inner.i4",
			"foo.bar.baz.Outer.Inner.i5",
			"foo.bar.baz.Outer.Inner.i6",
			"foo.bar.baz.Outer.o1",
			"foo.bar.baz.Outer.o2",
			"foo.bar.baz.Outer.o3",
			"foo.bar.baz.Outer.o4",
			"foo.bar.baz.Outer.o5",
			"foo.bar.baz.Outer.o6",
			"foo.bar.baz.i7", // this looks different because it's a field extension
			"foo.bar.baz.i8",
		}
		actualNames := modifier.FieldNames()
		sort.Strings(expectedNames)
		sort.Strings(actualNames)
		require.Equal(t, expectedNames, actualNames)

		messages := imageFile.Proto().GetMessageType()
		require.Len(t, messages, 1)
		outerMessage := messages[0]
		require.Len(t, outerMessage.GetNestedType(), 1)
		innerMessage := outerMessage.GetNestedType()[0]
		require.Len(t, innerMessage.GetField(), 6)

		// modify on nested field
		fieldI5 := innerMessage.GetField()[4]
		require.NotNil(t, fieldI5.GetOptions())
		require.NotNil(t, fieldI5.GetOptions().Jstype)
		require.Equal(t, descriptorpb.FieldOptions_JS_STRING, *fieldI5.GetOptions().Jstype)
		err = modifier.ModifyJSType("foo.bar.baz.Outer.Inner.i5", NewValueOverride(descriptorpb.FieldOptions_JS_NUMBER))
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FieldOptions_JS_NUMBER, *fieldI5.GetOptions().Jstype)

		// modify on a field with no option defined
		fieldI6 := innerMessage.GetField()[5]
		require.Nil(t, fieldI6.GetOptions())
		err = modifier.ModifyJSType("foo.bar.baz.Outer.Inner.i6", NewValueOverride(descriptorpb.FieldOptions_JS_STRING))
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FieldOptions_JS_STRING, *fieldI6.GetOptions().Jstype)

		// modify on a field not supported by jstype
		require.Len(t, outerMessage.GetField(), 6)
		fieldO2 := outerMessage.GetField()[1]
		require.NotNil(t, fieldO2)
		require.Nil(t, fieldO2.GetOptions())
		err = modifier.ModifyJSType("foo.bar.baz.Outer.o2", NewValueOverride(descriptorpb.FieldOptions_JS_NORMAL))
		require.NoError(t, err)
		require.Nil(t, fieldO2.GetOptions())

		// modify on a oneof field
		fieldO5 := outerMessage.GetField()[4]
		require.NotNil(t, fieldO5)
		require.NotNil(t, fieldO5.GetOptions())
		require.Equal(t, descriptorpb.FieldOptions_JS_STRING, *fieldO5.GetOptions().Jstype)
		err = modifier.ModifyJSType("foo.bar.baz.Outer.o5", NewValueOverride(descriptorpb.FieldOptions_JS_NORMAL))
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FieldOptions_JS_NORMAL, *fieldO5.GetOptions().Jstype)

		// modify the option to the exisint value from the file
		fieldO6 := outerMessage.GetField()[5]
		require.NotNil(t, fieldO6)
		require.NotNil(t, fieldO6.GetOptions())
		require.Equal(t, descriptorpb.FieldOptions_JS_NUMBER, *fieldO6.GetOptions().Jstype)
		err = modifier.ModifyJSType("foo.bar.baz.Outer.o6", NewValueOverride(descriptorpb.FieldOptions_JS_NUMBER))
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FieldOptions_JS_NUMBER, *fieldO6.GetOptions().Jstype)

		// modify on an extension field
		extensions := imageFile.Proto().GetExtension()
		require.Len(t, extensions, 2)
		fieldExtensionI7 := extensions[0]
		require.NotNil(t, fieldExtensionI7.GetOptions())
		require.Equal(t, descriptorpb.FieldOptions_JS_NUMBER, *fieldExtensionI7.GetOptions().Jstype)
		err = modifier.ModifyJSType("foo.bar.baz.i7", NewValueOverride(descriptorpb.FieldOptions_JS_NORMAL))
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FieldOptions_JS_NORMAL, *fieldO5.GetOptions().Jstype)

		// still marked even if the image is built without source code info, but
		// sweep will not take effect.
		require.Equal(
			t,
			map[string]map[string]struct{}{
				"a.proto": {
					internal.GetPathKey([]int32{4, 0, 3, 0, 2, 4, 8, 6}): struct{}{},
					internal.GetPathKey([]int32{4, 0, 3, 0, 2, 5, 8, 6}): struct{}{},
					internal.GetPathKey([]int32{4, 0, 2, 4, 8, 6}):       struct{}{},
					internal.GetPathKey([]int32{7, 0, 8, 6}):             struct{}{},
				},
			},
			markSweeper.sourceCodeInfoPaths,
		)
	}
}

func TestModifyJSTypeForWKT(t *testing.T) {
	t.Parallel()
	for _, includeSourceCodeInfo := range []bool{true, false} {
		image := bufimagemodifytesting.GetTestImage(
			t,
			filepath.Join("..", "testdata", "wktimport"),
			includeSourceCodeInfo,
		)
		require.NotNil(t, image)
		markSweeper := newMarkSweeper(image)
		require.NotNil(t, markSweeper)
		imageFile := image.GetFile("google/protobuf/timestamp.proto")
		require.NotNil(t, imageFile)
		modifier, err := NewFieldOptionModifier(imageFile, markSweeper)
		require.NoError(t, err)
		expectedNames := []string{
			"google.protobuf.Timestamp.seconds",
			"google.protobuf.Timestamp.nanos",
		}
		actualNames := modifier.FieldNames()
		sort.Strings(expectedNames)
		sort.Strings(actualNames)
		require.Equal(t, expectedNames, actualNames)

		messages := imageFile.Proto().GetMessageType()
		require.Len(t, messages, 1)
		timestamp := messages[0]
		fields := timestamp.GetField()
		require.Len(t, fields, 2)
		secondsField := fields[0]
		require.Nil(t, secondsField.GetOptions())

		err = modifier.ModifyJSType("google.protobuf.Timestamp.seconds", NewValueOverride(descriptorpb.FieldOptions_JS_NUMBER))
		require.NoError(t, err)
		// wkt should be skipped
		require.Nil(t, secondsField.GetOptions())

		// still marked even if the image is built without source code info, but
		// sweep will not take effect.
		require.Equal(
			t,
			map[string]map[string]struct{}{},
			markSweeper.sourceCodeInfoPaths,
		)
	}
}
