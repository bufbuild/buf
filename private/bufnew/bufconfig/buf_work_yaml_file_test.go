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

package bufconfig

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
)

func TestPutAndGetBufWorkYAMLFileForPrefix(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bufWorkYAMLFile, err := NewBufWorkYAMLFile(FileVersionV1, []string{"foo", "bar"})
	require.NoError(t, err)
	readWriteBucket := storagemem.NewReadWriteBucket()
	err = PutBufWorkYAMLFileForPrefix(ctx, readWriteBucket, "pre", bufWorkYAMLFile)
	require.NoError(t, err)
	_, err = GetBufWorkYAMLFileForPrefix(ctx, readWriteBucket, ".")
	require.Error(t, err)
	readBufWorkYAMLFile, err := GetBufWorkYAMLFileForPrefix(ctx, readWriteBucket, "pre")
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{"bar", "foo"},
		readBufWorkYAMLFile.DirPaths(),
	)
}

func TestReadBufWorkYAMLFileValidateVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	testcases := []struct {
		description      string
		prefix           string
		content          string
		isErr            bool
		expectedDirPaths []string
	}{
		{
			description: "current_directory",
			prefix:      ".",
			content: `version: v1
directories:
  - foo
  - bar
`,
			expectedDirPaths: []string{"bar", "foo"},
		},
		{
			description: "sub_directory",
			prefix:      "path",
			content: `version: v1
directories:
  - foo
`,
			expectedDirPaths: []string{"bar"},
		},
		{
			description: "invalid_version",
			prefix:      ".",
			content: `version: 1
directories:
  - foo
`,
			isErr: true,
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			readBucket, err := storagemem.NewReadBucket(
				map[string][]byte{
					filepath.Join(testcase.prefix, "buf.work.yaml"): []byte(testcase.content),
				},
			)
			require.NoError(t, err)
			bufWorkYAMLFile, err := GetBufWorkYAMLFileForPrefix(ctx, readBucket, testcase.prefix)
			if testcase.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testcase.expectedDirPaths, bufWorkYAMLFile.DirPaths())
		})
	}
}

func TestNewBufWorkYAMLFile(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description      string
		version          FileVersion
		dirPaths         []string
		expectedDirPaths []string
	}{
		{
			description:      "one_dir_path",
			version:          FileVersionV1,
			dirPaths:         []string{"foo"},
			expectedDirPaths: []string{"foo"},
		},
		{
			description:      "sort",
			version:          FileVersionV1,
			dirPaths:         []string{"foo", "baz", "bat", "bar"},
			expectedDirPaths: []string{"bar", "bat", "baz", "foo"},
		},
		{
			description:      "sort_and_normalize",
			version:          FileVersionV1,
			dirPaths:         []string{"bat", "./baz", "./bar/../bar", "foo"},
			expectedDirPaths: []string{"bar", "bat", "baz", "foo"},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			bufWorkYAMLFile, err := NewBufWorkYAMLFile(testcase.version, testcase.dirPaths)
			require.NoError(t, err)
			require.Equal(t, testcase.expectedDirPaths, bufWorkYAMLFile.DirPaths())
		})
	}
}

func TestNewWorkYAMLFileFail(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		description string
		version     FileVersion
		dirPaths    []string
	}{
		{
			description: "empty_dirPaths",
			version:     FileVersionV1,
			dirPaths:    []string{},
		},
		{
			description: "duplicate_dirPaths",
			version:     FileVersionV1,
			dirPaths:    []string{"foo", "bar", "foo"},
		},
		{
			description: "duplicate_dirPaths_different_forms",
			version:     FileVersionV1,
			dirPaths:    []string{"foo", "./foo"},
		},
		{
			description: "overlapping_dirPaths",
			version:     FileVersionV1,
			dirPaths:    []string{"foo", "bar", "foo/baz"},
		},
		{
			description: "overlapping_dirPaths_with_dot",
			version:     FileVersionV1,
			dirPaths:    []string{"foo", "bar", "./foo/baz"},
		},
		{
			description: "current_directory",
			version:     FileVersionV1,
			dirPaths:    []string{"foo", "."},
		},
		{
			description: "invalid_version_v1beta1",
			version:     FileVersionV1Beta1,
			dirPaths:    []string{"foo", "bar"},
		},
		{
			description: "invalid_version_v2",
			version:     FileVersionV2,
			dirPaths:    []string{"foo", "bar"},
		},
	}
	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.description, func(t *testing.T) {
			t.Parallel()
			_, err := NewBufWorkYAMLFile(testcase.version, testcase.dirPaths)
			require.Error(t, err)
		})
	}
}
