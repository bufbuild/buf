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
	testFormatNoDiff(t, "testdata/editions/2024")
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

func TestFormatterWithDeprecation(t *testing.T) {
	t.Parallel()
	// Test basic deprecation with prefix matching
	testDeprecateNoDiff(t, "basic", "testdata/deprecate", []string{"test.deprecate"},
		[]string{"already_deprecated.proto", "nested_types.proto"})
	// Test field deprecation with exact match
	testDeprecateNoDiff(t, "field", "testdata/deprecate", []string{"test.deprecate", "test.deprecate.MyMessage.id"},
		[]string{"field_deprecation.proto"})
	// Test enum value deprecation with exact match
	testDeprecateNoDiff(t, "enum_value", "testdata/deprecate", []string{
		"test.deprecate",
		"test.deprecate.STATUS_ACTIVE",
		"test.deprecate.STATUS_INACTIVE",
		"test.deprecate.OuterMessage.NESTED_STATUS_ACTIVE",
	}, []string{"enum_value_deprecation.proto"})
}

func testDeprecateNoDiff(t *testing.T, name string, path string, deprecatePrefixes []string, files []string) {
	t.Run(name, func(t *testing.T) {
		ctx := context.Background()
		bucket, err := storageos.NewProvider().NewReadWriteBucket(path)
		require.NoError(t, err)
		var opts []FormatOption
		for _, prefix := range deprecatePrefixes {
			opts = append(opts, WithDeprecate(prefix))
		}
		var matchers []storage.Matcher
		for _, file := range files {
			matchers = append(matchers, storage.MatchPathEqual(file))
		}
		filteredBucket := storage.FilterReadBucket(bucket, storage.MatchOr(matchers...))
		assertGolden := func(formatBucket storage.ReadBucket) {
			err := storage.WalkReadObjects(
				ctx,
				formatBucket,
				"",
				func(formattedFile storage.ReadObject) error {
					formattedData, err := io.ReadAll(formattedFile)
					require.NoError(t, err)
					expectedPath := strings.Replace(formattedFile.Path(), ".proto", ".golden", 1)
					expectedData, err := storage.ReadPath(ctx, bucket, expectedPath)
					require.NoError(t, err)
					fileDiff, err := diff.Diff(ctx, expectedData, formattedData, expectedPath, formattedFile.Path()+" (formatted)")
					require.NoError(t, err)
					require.Empty(t, string(fileDiff), "formatted output differs from golden file for %s", formattedFile.Path())
					return nil
				},
			)
			require.NoError(t, err)
		}
		// First pass: format with deprecation options
		formatBucket, err := FormatBucket(ctx, filteredBucket, opts...)
		require.NoError(t, err)
		assertGolden(formatBucket)
		// Second pass: re-format the already-formatted output to verify stability
		reformatBucket, err := FormatBucket(ctx, formatBucket, opts...)
		require.NoError(t, err)
		assertGolden(reformatBucket)
	})
}

func TestFormatBucketNoTypesMatchedError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bucket, err := storageos.NewProvider().NewReadWriteBucket("testdata/deprecate")
	require.NoError(t, err)
	// Use a prefix that won't match anything in the deprecate testdata
	_, err = FormatBucket(ctx, bucket, WithDeprecate("nonexistent.package"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no types matched")
}
