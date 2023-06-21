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
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/bufimagemodifytesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestModifySingleOption(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("..", "testdata")
	tests := []struct {
		description    string
		subDir         string
		modifyFunc     func(Marker, bufimage.ImageFile, Override) error
		file           string
		override       Override
		expectedValue  interface{}
		assertFunc     func(*testing.T, interface{}, *descriptorpb.FileDescriptorProto)
		fileOptionPath []int32
	}{
		{
			description:   "Java Package",
			subDir:        "emptyoptions",
			modifyFunc:    ModifyJavaPackage,
			file:          "a.proto",
			override:      NewValueOverride[string]("valueoverride"),
			expectedValue: "valueoverride",
			assertFunc:    assertJavaPackage,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			image := bufimagemodifytesting.GetTestImage(
				t,
				filepath.Join(baseDir, test.subDir),
				true,
			)
			bufimagemodifytesting.AssertFileOptionSourceCodeInfoEmpty(t, image, javaPackagePath, true)
			markSweeper := NewMarkSweeper(image)
			require.NotNil(t, markSweeper)
			imageFile := image.GetFile(test.file)
			require.NotNil(t, imageFile)
			err := ModifyJavaPackage(
				markSweeper,
				imageFile,
				newValueOverride[string]("valueoverride"),
			)
			require.NoError(t, err)
			require.NotNil(t, imageFile.Proto())
			test.assertFunc(t, test.expectedValue, imageFile.Proto())
		})
	}
}

func assertJavaPackage(t *testing.T, expectedValue interface{}, descriptor *descriptorpb.FileDescriptorProto) {
	assert.Equal(t, expectedValue, descriptor.GetOptions().GetJavaPackage())
}
