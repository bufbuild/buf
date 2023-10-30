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

package bufformat

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/diff"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestFormatter(t *testing.T) {
	t.Parallel()
	testFormatCustomOptions(t)
	testFormatProto2(t)
	testFormatProto3(t)
}

func testFormatCustomOptions(t *testing.T) {
	testFormatNoDiff(t, "testdata/customoptions")
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
	testFormatNoDiff(t, "testdata/proto3/file/v1")
	testFormatNoDiff(t, "testdata/proto3/header/v1")
	testFormatNoDiff(t, "testdata/proto3/literal/v1")
	testFormatNoDiff(t, "testdata/proto3/oneof/v1")
	testFormatNoDiff(t, "testdata/proto3/range/v1")
	testFormatNoDiff(t, "testdata/proto3/service/v1")
	testFormatNoDiff(t, "testdata/proto3/block/v1")
}

func testFormatNoDiff(t *testing.T, path string) {
	t.Run(path, func(t *testing.T) {
		ctx := context.Background()
		runner := command.NewRunner()
		moduleBucket, err := storageos.NewProvider().NewReadWriteBucket(path)
		require.NoError(t, err)
		module, err := bufmodule.NewModuleForBucket(ctx, moduleBucket)
		require.NoError(t, err)
		readBucket, err := Format(ctx, module)
		require.NoError(t, err)
		require.NoError(
			t,
			storage.WalkReadObjects(
				ctx,
				readBucket,
				"",
				func(formattedFile storage.ReadObject) error {
					expectedPath := formattedFile.Path()
					t.Run(expectedPath, func(t *testing.T) {
						// The expecetd format result is the golden file. If
						// this file IS a golden file, it is expected to not
						// change.
						if !strings.HasSuffix(expectedPath, ".golden.proto") {
							expectedPath = strings.Replace(expectedPath, ".proto", ".golden.proto", 1)
						}
						formattedData, err := io.ReadAll(formattedFile)
						require.NoError(t, err)
						expectedFile, err := moduleBucket.Get(ctx, expectedPath)
						require.NoError(t, err)
						expectedData, err := io.ReadAll(expectedFile)
						require.NoError(t, err)
						fileDiff, err := diff.Diff(ctx, runner, expectedData, formattedData, expectedPath, formattedFile.Path()+" (formatted)")
						require.NoError(t, err)
						require.Empty(t, string(fileDiff))
					})
					return nil
				},
			),
		)
	})
}
