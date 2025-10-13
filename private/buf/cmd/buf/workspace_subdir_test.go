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

package buf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test if workspaces work when buf is executed in a sub-directory specified
// in the workspace.

func TestWorkspaceSubDirectory(t *testing.T) {
	// Cannot run in parallel since we chdir
	defer chdirToSubDir(t, "testdata/workspace_subdir/other/proto/subdir")()
	// Execute buf within a workspace directory.
	wd, err := osext.Getwd()
	require.NoError(t, err)
	parentDirectory := filepath.Join(wd, "..")
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("..", "..", ".."),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`../one/a.proto
        ../one/b.proto
        ../two/c.proto`),
		"ls-files",
		filepath.Join("..", "..", ".."),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`../one/a.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        ../one/b.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        ../two/c.proto:17:1:Files with package "two.v1" must be within a directory "two/v1" relative to root but were in directory "two".`),
		"lint",
		filepath.Join("..", "..", ".."),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`../one/a.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        ../one/b.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".`),
		"lint",
		filepath.Join("..", "..", ".."),
		"--path",
		filepath.Join("..", "one"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`../two/c.proto:17:1:Files with package "two.v1" must be within a directory "two/v1" relative to root but were in directory "two".`),
		"lint",
		filepath.Join("..", "..", ".."),
		"--path",
		filepath.Join("..", "two"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join(wd, "..", "..", ".."),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(fmt.Sprintf(`%s/one/a.proto
        %s/one/b.proto
        %s/two/c.proto`, parentDirectory, parentDirectory, parentDirectory)),
		"ls-files",
		filepath.Join(wd, "..", "..", ".."),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(fmt.Sprintf(`%s/one/a.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        %s/one/b.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        %s/two/c.proto:17:1:Files with package "two.v1" must be within a directory "two/v1" relative to root but were in directory "two".`,
			parentDirectory, parentDirectory, parentDirectory,
		)),
		"lint",
		filepath.Join(wd, "..", "..", ".."),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("..", "..", "..", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`../one/a.proto
        ../one/b.proto
        ../two/c.proto`),
		"ls-files",
		filepath.Join("..", "..", "..", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"breaking",
		filepath.Join("..", "..", ".."),
		"--against",
		filepath.Join("..", "..", "..", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(fmt.Sprintf(`%s/one/a.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        %s/one/b.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".`,
			parentDirectory, parentDirectory,
		)),
		"lint",
		filepath.Join(wd, "..", "..", ".."),
		"--path",
		filepath.Join(wd, "..", "one"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(fmt.Sprintf(`%s/two/c.proto:17:1:Files with package "two.v1" must be within a directory "two/v1" relative to root but were in directory "two".`,
			parentDirectory,
		)),
		"lint",
		filepath.Join(wd, "..", "..", ".."),
		"--path",
		filepath.Join(wd, "..", "two"),
	)
}

func TestWorkspaceOverlapSubDirectory(t *testing.T) {
	// Cannot run in parallel since we chdir
	defer chdirToSubDir(t, "testdata/workspace_subdir/other/proto/subdir")()
	// Specify an overlapping input in a sub-directory.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: failed to build input "other/proto/one" because it is contained by module at path "other/proto" specified in your configuration, you must provide the workspace or module as the input, and filter to this path using --path`,
		"build",
		filepath.Join("..", "one"),
	)
}

func TestWorkspaceWithProtoFileRef(t *testing.T) {
	// Cannot run in parallel since we chdir
	defer chdirToSubDir(t, "testdata/workspace_subdir/other/proto/subdir")()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		0,
		filepath.FromSlash("../../../../workspace/success/protofileref/another/foo/foo.proto"),
		``,
		"ls-files",
		filepath.Join("..", "..", "..", "..", "workspace", "success", "protofileref", "another", "foo", "foo.proto"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		../../../../workspace/success/protofileref/another/foo/foo.proto:3:1:Package name "foo" should be suffixed with a correctly formed version, such as "foo.v1".
		`),
		"lint",
		filepath.Join("..", "..", "..", "..", "workspace", "success", "protofileref", "another", "foo", "foo.proto"),
	)
}

// Change the the subdirectory and then return a function to undo the chdir.
//
// Using this prevents tests being able to be run in parallel
func chdirToSubDir(t *testing.T, relSubDirPath string) func() {
	pwd, err := osext.Getwd()
	require.NoError(t, err)
	subDirPath := normalpath.Unnormalize(normalpath.Join(pwd, relSubDirPath))
	require.NoError(t, os.MkdirAll(subDirPath, os.ModePerm))
	require.NoError(t, osext.Chdir(subDirPath))
	return func() {
		r := recover()
		assert.NoError(t, osext.Chdir(pwd))
		if r != nil {
			panic(r)
		}
	}
}
