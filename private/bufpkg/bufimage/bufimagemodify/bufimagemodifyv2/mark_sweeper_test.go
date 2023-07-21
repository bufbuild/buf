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
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal/bufimagemodifytesting"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestSweepWithSourceCodeInfo(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description       string
		fileToPathsToMark map[string][][]int32
		// paths not marked but should be removed
		fileToExpectedAdditionalRemovedPaths map[string][][]int32
	}{
		{
			description: "mark and sweep a single field option path",
			fileToPathsToMark: map[string][][]int32{
				"a.proto": {
					{4, 0, 2, 3, 8, 6}, // Outer.o4.jstype, the only field option on this field
				},
			},
			fileToExpectedAdditionalRemovedPaths: map[string][][]int32{
				"a.proto": {
					{4, 0, 2, 3, 8},
				},
			},
		},
		{
			description: "mark and sweep some field option paths for a field",
			fileToPathsToMark: map[string][][]int32{
				"a.proto": {
					{4, 0, 2, 4, 8, 1}, // Outer.Inner.o5.ctype, but this field also has jstype option
				},
			},
		},
		{
			description: "mark and sweep some field option paths for a field",
			fileToPathsToMark: map[string][][]int32{
				"a.proto": {
					{7, 1, 8, 17}, // Outer.Inner.o6.retention, but this field has 4 options.
					{7, 1, 8, 16}, // Outer.Inner.o6.debug_redact, but this field has 4 options.
					{7, 1, 8, 5},  // Outer.Inner.o6.lazy, but this field has 4 options.
				},
			},
		},
		{
			description: "mark and sweep  all field options paths for a field",
			fileToPathsToMark: map[string][][]int32{
				"a.proto": {
					{7, 1, 8, 6},  // Outer.Inner.o6.jstype
					{7, 1, 8, 17}, // Outer.Inner.o6.retention
					{7, 1, 8, 16}, // Outer.Inner.o6.debug_redact
					{7, 1, 8, 5},  // Outer.Inner.o6.lazy
				},
			},
			fileToExpectedAdditionalRemovedPaths: map[string][][]int32{
				"a.proto": {
					{7, 1, 8},
				},
			},
		},
		{
			description: "mark and sweep a single file option path",
			fileToPathsToMark: map[string][][]int32{
				"a.proto": {
					{8, 11},
				},
			},
			fileToExpectedAdditionalRemovedPaths: map[string][][]int32{
				"a.proto": {
					{8},
				},
			},
		},
		{
			description: "mark and sweep multiple file options and field options for multiple files",
			fileToPathsToMark: map[string][][]int32{
				"a.proto": {
					{4, 0, 2, 3, 8, 6},       // Outer.o4.jstype, the only field option
					{4, 0, 3, 0, 2, 4, 8, 1}, // Outer.Inner.o5.ctype
					{8, 1},
					{4, 0, 3, 0, 2, 4, 8, 6}, // Outer.Inner.o5.jstype
					{8, 11},
					{4, 0, 2, 4, 8, 6}, // Outer.o5.jstype, there is still ctype option, so do not remove parent
					{7, 0, 8, 6},       // i7.jstype, the only field option
				},
				"b.proto": {
					{8, 16},
					{8, 17},
					{8, 37},
					{4, 0, 2, 0, 8, 6},
				},
			},
			fileToExpectedAdditionalRemovedPaths: map[string][][]int32{
				"a.proto": {
					{4, 0, 2, 3, 8},
					{4, 0, 3, 0, 2, 4, 8},
					{7, 0, 8},
					{8},
					{8},
				},
				"b.proto": {
					{8},
					{8},
					{8},
					{4, 0, 2, 0, 8},
				},
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			image := bufimagemodifytesting.GetTestImage(
				t,
				filepath.Join("..", "testdata", "fieldoptions"),
				true,
			)
			require.NotNil(t, image)
			markSweeper := newMarkSweeper(image)
			require.NotNil(t, markSweeper)
			nameToFile := make(map[string]bufimage.ImageFile, len(testcase.fileToPathsToMark))
			nameToLocationCountBeforeSweep := make(map[string]int, len(testcase.fileToPathsToMark))

			// mark paths for each file, and remember how many source code locations there are for each file.
			for _, imageFile := range image.Files() {
				fileName := imageFile.Path()
				nameToFile[fileName] = imageFile
				sourceLocations := imageFile.FileDescriptor().GetSourceCodeInfo().GetLocation()
				nameToLocationCountBeforeSweep[fileName] = len(sourceLocations)
				require.True(t, len(sourceLocations) > 0)
				for _, pathToMark := range testcase.fileToPathsToMark[fileName] {
					markSweeper.Mark(imageFile, pathToMark)
				}
				requirePathsExist(
					t,
					imageFile.Proto().GetSourceCodeInfo().GetLocation(),
					append(
						testcase.fileToPathsToMark[fileName],
						testcase.fileToExpectedAdditionalRemovedPaths[fileName]...,
					),
				)
			}
			err := markSweeper.Sweep()
			require.NoError(t, err)
			// check paths are removed correctly
			for _, imageFile := range image.Files() {
				fileName := imageFile.Path()
				require.NotNil(t, imageFile.Proto().GetSourceCodeInfo())
				locationCount := len(imageFile.Proto().GetSourceCodeInfo().GetLocation())
				expectedLocationCount := nameToLocationCountBeforeSweep[fileName] - len(testcase.fileToPathsToMark[fileName])
				if additionalExpectedPathsRemoved, ok := testcase.fileToExpectedAdditionalRemovedPaths[fileName]; ok {
					expectedLocationCount = expectedLocationCount - len(additionalExpectedPathsRemoved)
				}
				require.Equal(t, expectedLocationCount, locationCount)
				sourcePaths := make(map[string]struct{}, len(imageFile.Proto().GetSourceCodeInfo().GetLocation()))
				for _, location := range imageFile.Proto().GetSourceCodeInfo().GetLocation() {
					sourcePaths[internal.GetPathKey(location.Path)] = struct{}{}
				}
				for _, path := range testcase.fileToPathsToMark[fileName] {
					_, ok := sourcePaths[internal.GetPathKey(path)]
					require.False(t, ok, "%v should not exist among source code locations", path)
				}
				for _, path := range testcase.fileToExpectedAdditionalRemovedPaths[fileName] {
					if len(path) == 1 && path[0] == 8 {
						// there can be multiple {8}'s
						continue
					}
					_, ok := sourcePaths[internal.GetPathKey(path)]
					require.False(t, ok, "%v should not exist among source code locations", path)
				}
			}
		})
	}
}

func TestSweepOnImageWithoutSourceCodeInfo(t *testing.T) {
	t.Parallel()
	image := bufimagemodifytesting.GetTestImage(
		t,
		filepath.Join("..", "testdata", "fieldoptions"),
		false,
	)
	require.NotNil(t, image)
	markSweeper := newMarkSweeper(image)
	require.NotNil(t, markSweeper)
	imageFile := image.GetFile("a.proto")
	require.NotNil(t, imageFile)
	require.Nil(t, imageFile.FileDescriptor().GetSourceCodeInfo())
	markSweeper.Mark(
		imageFile,
		[]int32{4, 0, 2, 3, 8, 6},
	)
	err := markSweeper.Sweep()
	require.NoError(t, err)
	require.Nil(t, imageFile.FileDescriptor().GetSourceCodeInfo())
}

func requirePathsExist(
	t *testing.T,
	sourceLocations []*descriptorpb.SourceCodeInfo_Location,
	paths [][]int32,
) {
	sourcePaths := make(map[string]struct{}, len(sourceLocations))
	for _, location := range sourceLocations {
		sourcePaths[internal.GetPathKey(location.Path)] = struct{}{}
	}
	for _, path := range paths {
		_, ok := sourcePaths[internal.GetPathKey(path)]
		require.True(t, ok)
	}
}
