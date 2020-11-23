// Copyright 2020 Buf Technologies, Inc.
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

package filepathextended

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWalkSymlinkSuccessNoSymlinks(t *testing.T) {
	t.Parallel()
	filePaths, err := testWalkGetRegularFilePaths(
		filepath.Join("testdata", "symlink_success"),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"file.proto",
		},
		filePaths,
	)
}

func TestWalkSymlinkSuccessFollowSymlinks(t *testing.T) {
	t.Parallel()
	filePaths, err := testWalkGetRegularFilePaths(
		filepath.Join("testdata", "symlink_success"),
		WalkWithFollowSymlinks(),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{
			"1.proto",
			"a/b/1.proto",
			"a/b/2.proto",
			"a/b/2.txt",
			"a/bar.yaml",
			"a/file.proto",
			"ab/1.proto",
			"ab/2.proto",
			"ab/2.txt",
			"file.proto",
		},
		filePaths,
	)
}

func TestWalkSymlinkErrorLoopNoSymlinks(t *testing.T) {
	t.Parallel()
	filePaths, err := testWalkGetRegularFilePaths(
		filepath.Join("testdata", "symlink_error_loop"),
	)
	require.NoError(t, err)
	require.Equal(t, 0, len(filePaths))
}

func TestWalkSymlinkErrorLoopFollowSymlinks(t *testing.T) {
	t.Parallel()
	_, err := testWalkGetRegularFilePaths(
		filepath.Join("testdata", "symlink_error_loop"),
		WalkWithFollowSymlinks(),
	)
	require.True(t, errors.Is(err, &symlinkLoopError{}))
}

func testWalkGetRegularFilePaths(dirPath string, options ...WalkOption) ([]string, error) {
	var filePaths []string
	if err := Walk(
		dirPath,
		func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fileInfo.Mode().IsRegular() {
				relPath, err := filepath.Rel(dirPath, path)
				if err != nil {
					return err
				}
				filePaths = append(filePaths, relPath)
			}
			return nil
		},
		options...,
	); err != nil {
		return nil, err
	}
	sort.Strings(filePaths)
	return filePaths, nil
}
