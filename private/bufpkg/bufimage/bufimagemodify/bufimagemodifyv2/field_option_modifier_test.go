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

func TestGetFields(t *testing.T) {
	t.Parallel()
	image := bufimagemodifytesting.GetTestImage(
		t,
		filepath.Join("..", "testdata", "fieldoptions"),
		true, // TODO: test when this is false
	)
	require.NotNil(t, image)
	markSweeper := newMarkSweeper(image)
	require.NotNil(t, markSweeper)
	imageFile := image.GetFile("a.proto")
	require.NotNil(t, imageFile)
	m, err := NewFieldOptionModifier(imageFile, markSweeper)
	require.NoError(t, err)
	// TODO: remove
	// printImage(image)
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
	}
	actualNames := m.FieldNames()
	sort.Strings(expectedNames)
	sort.Strings(actualNames)
	require.Equal(t, expectedNames, actualNames)

	messages := imageFile.Proto().GetMessageType()
	require.Len(t, messages, 1)
	outerMessage := messages[0]
	require.Len(t, outerMessage.GetNestedType(), 1)
	innerMessage := outerMessage.GetNestedType()[0]
	require.Len(t, innerMessage.GetField(), 6)
	fifthField := innerMessage.GetField()[4]
	require.NotNil(t, fifthField.GetOptions())
	require.NotNil(t, fifthField.GetOptions().Jstype)
	require.Equal(t, descriptorpb.FieldOptions_JS_STRING, *fifthField.GetOptions().Jstype)

	err = m.ModifyJSType("foo.bar.baz.Outer.Inner.i5", newValueOverride(descriptorpb.FieldOptions_JS_NUMBER))
	require.NoError(t, err)

	require.Equal(t, descriptorpb.FieldOptions_JS_NUMBER, *fifthField.GetOptions().Jstype)

	require.Equal(
		t,
		map[string]map[string]struct{}{
			"a.proto": {
				internal.GetPathKey([]int32{4, 0, 3, 0, 2, 4, 8, 6}): struct{}{},
			},
		},
		markSweeper.sourceCodeInfoPaths,
	)
}
