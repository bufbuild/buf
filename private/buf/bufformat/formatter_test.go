// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufformat

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/diff"
	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestFormatter(t *testing.T) {
	t.Parallel()
	testFormatCustomOptions(t)
	testFormatEditions(t)
	testFormatProto2(t)
	testFormatProto3(t)
	testFormatHeader(t)
}

func testFormatCustomOptions(t *testing.T) {
	testFormatNoDiff(t, "testdata/customoptions")
}

func testFormatEditions(t *testing.T) {
	testFormatNoDiff(t, "testdata/editions/2023")
	testFormatError(t, "testdata/editions/2024", `edition "2024" not yet fully supported; latest supported edition "2023"`)
}

func testFormatProto2(t *testing.T) {
	testFormatNoDiff(t, "testdata/proto2/enum/v1")
	testFormatNoDiff(t, "testdata/proto2/extend/v1")
	testFormatNoDiff(t, "testdata/proto2/field/v1")
	testFormatNoDiff(t, "testdata/proto2/group/v1")
	testFormatNoDiff(t, "testdata/proto2/header/v1")
	testFormatNoDiff(t, "testdata/proto2/license/v1")
	testFormatNoDiff(t, "testdata/proto2/message/v1")
	testFormatNoDiff(t, "testdata/proto2/option/v1")
	testFormatNoDiff(t, "testdata/proto2/quotes/v1")
	testFormatNoDiff(t, "testdata/proto2/utf8/v1")
}

func testFormatProto3(t *testing.T) {
	testFormatNoDiff(t, "testdata/proto3/all/v1")
	testFormatNoDiff(t, "testdata/proto3/block/v1")
	testFormatNoDiff(t, "testdata/proto3/file/v1")
	testFormatNoDiff(t, "testdata/proto3/header/v1")
	testFormatNoDiff(t, "testdata/proto3/literal/v1")
	testFormatNoDiff(t, "testdata/proto3/oneof/v1")
	testFormatNoDiff(t, "testdata/proto3/range/v1")
	testFormatNoDiff(t, "testdata/proto3/service/v1")
}

func testFormatHeader(t *testing.T) {
	testFormatNoDiff(t, "testdata/header")
}

func testFormatNoDiff(t *testing.T, path string) {
	t.Run(path, func(t *testing.T) {
		ctx := context.Background()
		bucket, err := storageos.NewProvider().NewReadWriteBucket(path)
		require.NoError(t, err)

		moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, slogtestext.NewLogger(t), bufmodule.NopModuleDataProvider, bufmodule.NopCommitProvider)
		moduleSetBuilder.AddLocalModule(bucket, path, true)
		moduleSet, err := moduleSetBuilder.Build()
		require.NoError(t, err)
		formatBucket, err := FormatModuleSet(ctx, moduleSet)
		require.NoError(t, err)
		assertGolden := func(formatBucket storage.ReadBucket) {
			err := storage.WalkReadObjects(
				ctx,
				formatBucket,
				"",
				func(formattedFile storage.ReadObject) error {
					formattedData, err := io.ReadAll(formattedFile)
					require.NoError(t, err)
					expectedPath := strings.Replace(formattedFile.Path(), ".proto", ".golden", 1)
					t.Log("expectedPath", expectedPath, formattedFile.Path())
					expectedData, err := storage.ReadPath(ctx, bucket, expectedPath)
					require.NoError(t, err)
					fileDiff, err := diff.Diff(ctx, expectedData, formattedData, expectedPath, formattedFile.Path()+" (formatted)")
					require.NoError(t, err)
					require.Empty(t, string(fileDiff))
					return nil
				},
			)
			require.NoError(t, err)
		}
		assertGolden(formatBucket)

		moduleSetBuilder = bufmodule.NewModuleSetBuilder(ctx, slogtestext.NewLogger(t), bufmodule.NopModuleDataProvider, bufmodule.NopCommitProvider)
		moduleSetBuilder.AddLocalModule(formatBucket, path, true)
		moduleSet, err = moduleSetBuilder.Build()
		require.NoError(t, err)
		reformattedBucket, err := FormatModuleSet(ctx, moduleSet)
		require.NoError(t, err)
		assertGolden(reformattedBucket)
	})
}

func testFormatError(t *testing.T, path string, errContains string) {
	t.Run(path, func(t *testing.T) {
		ctx := context.Background()
		bucket, err := storageos.NewProvider().NewReadWriteBucket(path)
		require.NoError(t, err)
		moduleSetBuilder := bufmodule.NewModuleSetBuilder(ctx, slogtestext.NewLogger(t), bufmodule.NopModuleDataProvider, bufmodule.NopCommitProvider)
		moduleSetBuilder.AddLocalModule(bucket, path, true)
		moduleSet, err := moduleSetBuilder.Build()
		require.NoError(t, err)
		_, err = FormatModuleSet(ctx, moduleSet)
		require.ErrorContains(t, err, errContains)
	})
}
