// Copyright 2020-2022 Buf Technologies, Inc.
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

package workspacetest

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/cmd/buf"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/private/pkg/osextended"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceSubDirectory(t *testing.T) {
	// Execute buf within a workspace directory.
	t.Parallel()
	wd, err := osextended.Getwd()
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
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`../one/a.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        ../one/b.proto:17:1:Files with package "one.v1" must be within a directory "one/v1" relative to root but were in directory "one".
        ../two/c.proto:17:1:Files with package "two.v1" must be within a directory "two/v1" relative to root but were in directory "two".`),
		"lint",
		filepath.Join("..", "..", ".."),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
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
		bufcli.ExitCodeFileAnnotation,
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
		bufcli.ExitCodeFileAnnotation,
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
		bufcli.ExitCodeFileAnnotation,
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
		bufcli.ExitCodeFileAnnotation,
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
	// Specify an overlapping input in a sub-directory.
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: failed to build input "other/proto/one" because it is contained by directory "other/proto" listed in ../../../buf.work.yaml`),
		"build",
		filepath.Join("..", "one"),
	)
}

func TestWorkspaceWithProtoFileRef(t *testing.T) {
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		0,
		filepath.FromSlash("../../../../testdata/workspace/success/protofileref/another/foo/foo.proto"),
		``,
		"ls-files",
		filepath.Join("..", "..", "..", "..", "testdata", "workspace", "success", "protofileref", "another", "foo", "foo.proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`
		../../../../testdata/workspace/success/protofileref/another/foo/foo.proto:3:1:Package name "foo" should be suffixed with a correctly formed version, such as "foo.v1".
		`),
		"lint",
		filepath.Join("..", "..", "..", "..", "testdata", "workspace", "success", "protofileref", "another", "foo", "foo.proto"),
	)
}

func testRunStdout(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdout(
		t,
		func(use string) *appcmd.Command { return buf.NewRootCommand(use) },
		expectedExitCode,
		expectedStdout,
		func(use string) map[string]string {
			return map[string]string{
				useEnvVar(use, "CACHE_DIR"): "cache",
			}
		},
		stdin,
		args...,
	)
}

func testRunStdoutStderr(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, expectedStderr string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdoutStderr(
		t,
		func(use string) *appcmd.Command { return buf.NewRootCommand(use) },
		expectedExitCode,
		expectedStdout,
		expectedStderr,
		func(use string) map[string]string {
			return map[string]string{
				useEnvVar(use, "CACHE_DIR"): "cache",
			}
		},
		stdin,
		args...,
	)
}

func useEnvVar(use string, suffix string) string {
	return strings.ToUpper(use) + "_" + suffix
}
