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

package bufwire

import (
	"context"
	"io"
	"sort"
	"testing"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExcludePathsForModule(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description              string
		moduleContent            map[string][]byte
		targetPaths              []string
		excludePaths             []string
		expectedFilesAfterFilter []string
	}{
		{
			description: "target nil exclude nil",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths:  nil,
			excludePaths: nil,
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
				"dir/bar.proto",
			},
		},
		{
			description: "target empty exclude nil",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths:  make([]string, 0),
			excludePaths: nil,
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
				"dir/bar.proto",
			},
		},
		{
			description: "target empty with exclude",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths: make([]string, 0),
			excludePaths: []string{
				"dir/bar.proto",
			},
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
			},
		},
		{
			description: "target nil with exclude",
			moduleContent: map[string][]byte{
				"dir/foo.proto": nil,
				"dir/bar.proto": nil,
			},
			targetPaths: nil,
			excludePaths: []string{
				"dir/bar.proto",
			},
			expectedFilesAfterFilter: []string{
				"dir/foo.proto",
			},
		},
		{
			description: "target paths with exclude nil",
			moduleContent: map[string][]byte{
				"dir/foo.proto":  nil,
				"dir/bar.proto":  nil,
				"dir2/foo.proto": nil,
				"dir2/bar.proto": nil,
			},
			targetPaths: []string{
				"dir2",
				"dir/bar.proto",
			},
			excludePaths: nil,
			expectedFilesAfterFilter: []string{
				"dir/bar.proto",
				"dir2/foo.proto",
				"dir2/bar.proto",
			},
		},
		{
			description: "target paths with exclude paths",
			moduleContent: map[string][]byte{
				"dir/foo/bar.proto":  nil,
				"dir/foo/baz.proto":  []byte(""),
				"dir/foo/qux.proto":  []byte(""),
				"dir/foo2/bar.proto": []byte(""),
				"dir/foo2/baz.proto": []byte(""),
				"dir2/bar.proto":     nil,
				"dir2/baz.proto":     nil,
			},
			targetPaths: []string{
				"dir/",
				"dir2/baz.proto",
			},
			excludePaths: []string{
				"dir/foo2",
				"dir/foo/qux.proto",
			},
			expectedFilesAfterFilter: []string{
				"dir/foo/bar.proto",
				"dir/foo/baz.proto",
				"dir2/baz.proto",
			},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			moduleConfigReader := NewModuleConfigReader(
				zap.NewNop(),
				storageos.NewProvider(),
				&fakeModuleFetcher{
					fileContent: testcase.moduleContent,
				},
				nil,
			)
			someModuleRef, err := buffetch.NewModuleRefParser(zap.NewNop()).GetModuleRef(context.Background(), "buf.build/foo/bar")
			require.NoError(t, err)
			moduleConfigSet, err := moduleConfigReader.GetModuleConfigSet(
				context.Background(),
				nil,
				someModuleRef, // the fake module fetcher doesn't care what ref this is
				"",
				testcase.targetPaths,
				testcase.excludePaths,
				true,
			)
			require.NoError(t, err)
			require.Len(t, moduleConfigSet.ModuleConfigs(), 1)
			moduleConfig := moduleConfigSet.ModuleConfigs()[0]
			require.NotNil(t, moduleConfig)
			module := moduleConfig.Module()
			require.NotNil(t, module)
			targetFileInfos, err := module.TargetFileInfos(context.Background())
			require.NoError(t, err)
			targetFilePaths := make([]string, len(targetFileInfos))
			for i := range targetFileInfos {
				targetFilePaths[i] = targetFileInfos[i].Path()
			}
			sort.Strings(targetFilePaths)
			sort.Strings(testcase.expectedFilesAfterFilter)
			require.Equal(t, testcase.expectedFilesAfterFilter, targetFilePaths)
		})
	}
}

type fakeModuleFetcher struct {
	fileContent map[string][]byte
}

func (r *fakeModuleFetcher) GetModule(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef buffetch.ModuleRef,
) (bufmodule.Module, error) {
	moduleBucket, err := storagemem.NewReadBucket(
		r.fileContent,
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleForBucket(
		context.Background(),
		moduleBucket,
	)
}

func (r *fakeModuleFetcher) GetMessageFile(
	ctx context.Context,
	container app.EnvStdinContainer,
	messageRef buffetch.MessageRef,
) (io.ReadCloser, error) {
	return nil, nil
}

func (r *fakeModuleFetcher) GetSourceBucket(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	options ...buffetch.GetSourceBucketOption,
) (buffetch.ReadBucketCloserWithTerminateFileProvider, error) {
	return nil, nil
}
