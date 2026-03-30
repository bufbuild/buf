// Copyright 2020-2026 Buf Technologies, Inc.
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

package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffoldSingleRoot(t *testing.T) {
	t.Parallel()
	testScaffold(t, "testdata/single_root", 0, "")
}

func TestScaffoldMultiRoot(t *testing.T) {
	t.Parallel()
	testScaffold(t, "testdata/multi_root", 0, "")
}

func TestScaffoldRootModule(t *testing.T) {
	t.Parallel()
	testScaffold(t, "testdata/root_module", 0, "")
}

func TestScaffoldNoProtoFiles(t *testing.T) {
	t.Parallel()
	testScaffold(t, "testdata/no_proto_files", 1, "no .proto files found")
}

func TestScaffoldNestedRoots(t *testing.T) {
	t.Parallel()
	testScaffold(t, "testdata/nested_roots", 0, "")
}

func TestScaffoldExistingBufYAML(t *testing.T) {
	t.Parallel()
	testScaffold(t, "testdata/existing_buf_yaml", 1, "buf.yaml already exists")
}

func TestScaffoldNotGitRoot(t *testing.T) {
	t.Parallel()
	testScaffoldWithOptions(t, "testdata/not_git_root", 1, "is not the root of a git repository", false)
}

func testScaffold(t *testing.T, dir string, expectCode int, expectStderr string) {
	t.Helper()
	testScaffoldWithOptions(t, dir, expectCode, expectStderr, true)
}

func testScaffoldWithOptions(t *testing.T, dir string, expectCode int, expectStderr string, createGitDir bool) {
	t.Helper()
	storageosProvider := storageos.NewProvider()
	inputBucket, err := storageosProvider.NewReadWriteBucket(filepath.Join(dir, "input"))
	require.NoError(t, err)
	tempDir := t.TempDir()
	tempBucket, err := storageosProvider.NewReadWriteBucket(tempDir)
	require.NoError(t, err)
	_, err = storage.Copy(t.Context(), inputBucket, tempBucket)
	require.NoError(t, err)
	if createGitDir {
		require.NoError(t, os.Mkdir(filepath.Join(tempDir, ".git"), 0o755))
	}
	appcmdtesting.Run(
		t,
		func(use string) *appcmd.Command {
			return NewCommand(use, appext.NewBuilder(use))
		},
		appcmdtesting.WithExpectedExitCode(expectCode),
		appcmdtesting.WithExpectedStderrPartials(expectStderr),
		appcmdtesting.WithArgs(tempDir),
	)
	if expectCode != 0 {
		return
	}
	got, err := os.ReadFile(filepath.Join(tempDir, "buf.yaml"))
	require.NoError(t, err)
	expected, err := os.ReadFile(filepath.Join(dir, "output", "buf.yaml"))
	require.NoError(t, err)
	assert.Equal(t, string(expected), string(got))
}
