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

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package buf

import (
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufcli"
)

func TestWorkspaceSymlinkFail(t *testing.T) {
	t.Parallel()
	// The workspace includes a symlink that isn't buildable.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		``,
		filepath.FromSlash(`testdata/workspace/fail/symlink/b/b.proto:5:8:c.proto: does not exist`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "symlink"),
	)
}

func TestWorkspaceSymlink(t *testing.T) {
	// The workspace includes valid symlinks.
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "symlink"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/symlink/a/a.proto
        testdata/workspace/success/symlink/b/b.proto
        testdata/workspace/success/symlink/c/c.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "symlink"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/symlink/a/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/symlink/a/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".
        testdata/workspace/success/symlink/b/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/symlink/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".
        testdata/workspace/success/symlink/c/c.proto:3:1:Files with package "c" must be within a directory "c" relative to root but were in directory ".".
        testdata/workspace/success/symlink/c/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "symlink"),
	)
}

func TestWorkspaceAbsoluteFail(t *testing.T) {
	t.Parallel()
	// The buf.work.yaml file cannot specify absolute paths.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: directory "/home/buf" listed in testdata/workspace/fail/absolute/buf.work.yaml is invalid: /home/buf: expected to be relative`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "absolute"),
	)
}

// TODO: Move this back to workspace_test.go. after resolving the issue where git
// clone failed with "unable to create file filename too long" on Windows CI.
// Workflow run: https://github.com/bufbuild/buf/actions/runs/6510804063/job/17685247791.
// Potential fix: https://stackoverflow.com/questions/22575662/filename-too-long-in-git-for-windows.
func TestWorkspaceGit(t *testing.T) {
	// Directory paths specified as a git reference within a workspace.
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		"../../../../.git#ref=HEAD,subdir=private/buf/cmd/buf/testdata/workspace/success/dir/proto",
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		"../../../../.git#ref=HEAD,subdir=private/buf/cmd/buf/testdata/workspace/success/dir/proto",
		"--path",
		filepath.Join("private", "buf", "cmd", "buf", "testdata", "workspace", "success", "dir", "proto", "rpc.proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`private/buf/cmd/buf/testdata/workspace/success/dir/proto/rpc.proto`),
		"ls-files",
		"../../../../.git#ref=HEAD,subdir=private/buf/cmd/buf/testdata/workspace/success/dir/proto",
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`private/buf/cmd/buf/testdata/workspace/success/dir/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        private/buf/cmd/buf/testdata/workspace/success/dir/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
		"lint",
		"../../../../.git#ref=HEAD,subdir=private/buf/cmd/buf/testdata/workspace/success/dir/proto",
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`private/buf/cmd/buf/testdata/workspace/success/dir/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        private/buf/cmd/buf/testdata/workspace/success/dir/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
		"lint",
		"../../../../.git#ref=HEAD,subdir=private/buf/cmd/buf/testdata/workspace/success/dir/proto",
		"--path",
		filepath.Join("private", "buf", "cmd", "buf", "testdata", "workspace", "success", "dir", "proto", "rpc.proto"),
	)
}
