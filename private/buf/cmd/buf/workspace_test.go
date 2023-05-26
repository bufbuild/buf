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

package buf

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/pkg/osextended"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceDir(t *testing.T) {
	// Directory paths contained within a workspace.
	t.Parallel()
	// dir_buf_work contains a buf.work instead of a buf.work.yaml
	// we want to make sure this still works
	for _, baseDirPath := range []string{
		"dir",
		"dir_buf_work",
	} {
		wd, err := osextended.Getwd()
		require.NoError(t, err)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(
				`testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`,
			),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`,
			),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(
				fmt.Sprintf(`%s/testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    %s/testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`, wd, wd),
			),
			"lint",
			filepath.Join(wd, "testdata", "workspace", "success", baseDirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto
		    testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".
		    testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", "breaking"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/breaking/other/proto/request.proto
		    testdata/workspace/success/breaking/proto/rpc.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", "breaking"),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/breaking/other/proto/request.proto:5:1:Previously present field "1" with name "name" on message "Request" was deleted.
		    testdata/workspace/success/breaking/proto/rpc.proto:8:5:Field "1" with name "request" on message "RPC" changed option "json_name" from "req" to "request".
		    testdata/workspace/success/breaking/proto/rpc.proto:8:21:Field "1" on message "RPC" changed name from "req" to "request".`),
			"breaking",
			filepath.Join("testdata", "workspace", "success", "breaking"),
			"--against",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
		)
		testRunStdoutStderr(
			t,
			nil,
			1,
			"", // stdout should be empty
			"Failure: the --config flag is not compatible with workspaces",
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--config",
			`{"version":"v1","lint": {"use": ["PACKAGE_DIRECTORY_MATCH"]}}`,
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--path",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "proto", "rpc.proto"),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--path",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
		testRunStdoutStderr(
			t,
			nil,
			1,
			"", // stdout should be empty
			"Failure: the --config flag is not compatible with workspaces",
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--config",
			`{"version":"v1","lint": {"use": ["PACKAGE_DIRECTORY_MATCH"]}}`,
			"--path",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
	}
}

func TestWorkspaceArchiveDir(t *testing.T) {
	// Archive that defines a workspace at the root of the archive.
	t.Parallel()
	zipDir := createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "dir"),
		"archive.zip",
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`proto/rpc.proto`),
		"ls-files",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
		"--path",
		filepath.Join("proto", "rpc.proto"),
	)
}

func TestWorkspaceNestedArchive(t *testing.T) {
	// Archive that defines a workspace in a sub-directory to the root.
	t.Parallel()
	zipDir := createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "nested"),
		"archive.zip",
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join(zipDir, "archive.zip#subdir=proto/internal"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`proto/internal/internal.proto`),
		"ls-files",
		filepath.Join(zipDir, "archive.zip#subdir=proto/internal"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`proto/internal/internal.proto:3:1:Files with package "internal" must be within a directory "internal" relative to root but were in directory ".".
        proto/internal/internal.proto:3:1:Package name "internal" should be suffixed with a correctly formed version, such as "internal.v1".`),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto/internal"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`proto/internal/internal.proto:3:1:Files with package "internal" must be within a directory "internal" relative to root but were in directory ".".
        proto/internal/internal.proto:3:1:Package name "internal" should be suffixed with a correctly formed version, such as "internal.v1".`),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto/internal"),
		"--path",
		filepath.Join("proto", "internal", "internal.proto"),
	)
}

func TestWorkspaceGit(t *testing.T) {
	t.Skip("skip until the move to private/buf is merged")
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
		filepath.Join("internal", "buf", "cmd", "buf", "testdata", "workspace", "success", "dir", "proto", "rpc.proto"),
	)
}

func TestWorkspaceDetached(t *testing.T) {
	// The workspace doesn't include the 'proto' directory, so
	// its contents aren't included in the workspace.
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "detached", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/detached/proto/rpc.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "detached", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/detached/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        testdata/workspace/success/detached/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "detached", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "detached", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/detached/other/proto/request.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "detached", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/detached/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
        testdata/workspace/success/detached/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "detached", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "detached"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/detached/other/proto/request.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "detached"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/detached/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
        testdata/workspace/success/detached/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "detached"),
	)
}

func TestWorkspaceNoModuleConfig(t *testing.T) {
	// The workspace points to modules that don't contain a buf.yaml.
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "noconfig", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/noconfig/proto/rpc.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "noconfig", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/noconfig/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        testdata/workspace/success/noconfig/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "noconfig", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "noconfig", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/noconfig/other/proto/request.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "noconfig", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/noconfig/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
        testdata/workspace/success/noconfig/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "noconfig", "other", "proto"),
	)
}

func TestWorkspaceWithLock(t *testing.T) {
	// The workspace points to a module that includes a buf.lock, but
	// the listed dependency is defined in the workspace so the module
	// cache is unused.
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "lock", "a"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/lock/a/a.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "lock", "a"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/lock/a/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/lock/a/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "lock", "a"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "lock", "b"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/lock/b/b.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "lock", "b"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/lock/b/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/lock/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "lock", "b"),
	)
}

func TestWorkspaceWithTransitiveDependencies(t *testing.T) {
	// The workspace points to a module that includes transitive
	// dependencies (i.e. a depends on b, and b depends on c).
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "transitive", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/transitive/proto/a.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "transitive", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/transitive/proto/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/transitive/proto/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "transitive", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "transitive", "private", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/transitive/private/proto/b.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "transitive", "private", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/transitive/private/proto/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/transitive/private/proto/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "transitive", "private", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "transitive", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/transitive/other/proto/c.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "transitive", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/transitive/other/proto/c.proto:3:1:Files with package "c" must be within a directory "c" relative to root but were in directory ".".
        testdata/workspace/success/transitive/other/proto/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "transitive", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "transitive"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/transitive/other/proto/c.proto
        testdata/workspace/success/transitive/private/proto/b.proto
        testdata/workspace/success/transitive/proto/a.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "transitive"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/transitive/other/proto/c.proto:3:1:Files with package "c" must be within a directory "c" relative to root but were in directory ".".
        testdata/workspace/success/transitive/other/proto/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".
        testdata/workspace/success/transitive/private/proto/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/transitive/private/proto/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".
        testdata/workspace/success/transitive/proto/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/transitive/proto/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "transitive"),
	)
}

func TestWorkspaceWithDiamondDependency(t *testing.T) {
	// The workspace points to a module that includes a diamond
	// dependency (i.e. a depends on b and c, and b depends on c).
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "diamond", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/diamond/proto/a.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "diamond", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/diamond/proto/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/diamond/proto/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "diamond", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "diamond", "private", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/diamond/private/proto/b.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "diamond", "private", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/diamond/private/proto/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/diamond/private/proto/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "diamond", "private", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "diamond", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/diamond/other/proto/c.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "diamond", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/diamond/other/proto/c.proto:3:1:Files with package "c" must be within a directory "c" relative to root but were in directory ".".
        testdata/workspace/success/diamond/other/proto/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "diamond", "other", "proto"),
	)
}

func TestWorkspaceWKT(t *testing.T) {
	// The workspace includes multiple images that import the same
	// well-known type (empty.proto).
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "wkt", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/wkt/other/proto/c/c.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "wkt", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/wkt/other/proto/c/c.proto:6:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "wkt", "other", "proto"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "wkt"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/wkt/other/proto/c/c.proto
        testdata/workspace/success/wkt/proto/a/a.proto
        testdata/workspace/success/wkt/proto/b/b.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "wkt"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/wkt/other/proto/c/c.proto:6:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".
        testdata/workspace/success/wkt/proto/a/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".
        testdata/workspace/success/wkt/proto/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "wkt"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"breaking",
		filepath.Join("testdata", "workspace", "success", "wkt"),
		"--against",
		filepath.Join("testdata", "workspace", "success", "wkt"),
	)
}

func TestWorkspaceRoots(t *testing.T) {
	// Workspaces should support modules with multiple roots specified in a v1beta1 buf.yaml.
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "success", "roots"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/workspace/success/roots/module1/a/a.proto
        testdata/workspace/success/roots/module2/root1/b/b.proto
        testdata/workspace/success/roots/module2/root2/c/c.proto`),
		"ls-files",
		filepath.Join("testdata", "workspace", "success", "roots"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module1/a/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".
        testdata/workspace/success/roots/module2/root1/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".
		testdata/workspace/success/roots/module2/root2/c/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"breaking",
		filepath.Join("testdata", "workspace", "success", "roots"),
		"--against",
		filepath.Join("testdata", "workspace", "success", "roots"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module1/a/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "roots", "module1", "a"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module2/root1/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "roots", "module2", "root1", "b"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module2/root2/c/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "roots", "module2", "root2", "c"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module2/root1/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".
testdata/workspace/success/roots/module2/root2/c/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".
testdata/workspace/success/roots/module2/root3/d/d.proto:3:1:Package name "d" should be suffixed with a correctly formed version, such as "d.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots", "module2"),
		"--config",
		filepath.Join("testdata", "workspace", "success", "roots", "module2", "other.buf.yaml"),
	)
}

func TestWorkspaceBreakingFail(t *testing.T) {
	t.Parallel()
	// The two workspaces define a different number of
	// images, so it's impossible to verify compatibility.
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
	)
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: input contained 1 images, whereas against contained 2 images`,
		"breaking",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
		"--against",
		filepath.Join("testdata", "workspace", "success", "breaking"),
	)
}

func TestWorkspaceDuplicateFail(t *testing.T) {
	t.Parallel()
	// The workspace includes multiple images that define the same file.
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: foo.proto exists in multiple locations: testdata/workspace/fail/duplicate/other/proto/foo.proto testdata/workspace/fail/duplicate/proto/foo.proto`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "duplicate"),
	)
}

func TestWorkspaceNotExistFail(t *testing.T) {
	t.Parallel()
	// The directory defined in the workspace does not exist.
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: directory "notexist" listed in testdata/workspace/fail/notexist/buf.work.yaml contains no .proto files`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "notexist"),
	)
}

func TestWorkspaceJumpContextFail(t *testing.T) {
	t.Parallel()
	// The workspace directories cannot jump context.
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		fmt.Sprintf(
			"%s: %s",
			filepath.FromSlash(`Failure: directory "../breaking/other/proto" listed in testdata/workspace/fail/jumpcontext/buf.work.yaml is invalid`),
			"../breaking/other/proto: is outside the context directory",
		),
		"build",
		filepath.Join("testdata", "workspace", "fail", "jumpcontext"),
	)
}

func TestWorkspaceDirOverlapFail(t *testing.T) {
	t.Parallel()
	// The buf.work.yaml file cannot specify overlapping diretories.
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: directory "foo" contains directory "foo/bar" in testdata/workspace/fail/diroverlap/buf.work.yaml`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "diroverlap"),
	)
}

func TestWorkspaceInputOverlapFail(t *testing.T) {
	// The target input cannot overlap with any of the directories defined
	// in the workspace.
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: failed to build input "proto/buf" because it is contained by directory "proto" listed in testdata/workspace/fail/overlap/buf.work.yaml`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "overlap", "proto", "buf"),
	)
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: failed to build input "other" because it contains directory "other/proto" listed in testdata/workspace/success/dir/buf.work.yaml`),
		"build",
		filepath.Join("testdata", "workspace", "success", "dir", "other"),
	)
}

func TestWorkspaceNoVersionFail(t *testing.T) {
	// The buf.work.yaml must specify a version.
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: testdata/workspace/fail/noversion/buf.work.yaml has no version set. Please add "version: v1"`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "noversion"),
	)
}

func TestWorkspaceInvalidVersionFail(t *testing.T) {
	// The buf.work.yaml must specify a valid version.
	t.Parallel()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: testdata/workspace/fail/invalidversion/buf.work.yaml has an invalid "version: v9" set. Please add "version: v1"`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "invalidversion"),
	)
}

func TestWorkspaceNoDirectoriesFail(t *testing.T) {
	t.Parallel()
	// The buf.work.yaml must specify at least one directory.
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: testdata/workspace/fail/nodirectories/buf.work.yaml has no directories set. Please add "directories: [...]"`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "nodirectories"),
	)
}

func TestWorkspaceWithWorkspacePathFail(t *testing.T) {
	t.Parallel()
	// The --path flag cannot match the workspace directory (i.e. root requirements).
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash("Failure: path \"testdata/workspace/success/dir\" is equal to the workspace defined in \"testdata/workspace/success/dir/buf.work.yaml\""),
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "dir"),
	)
}

func TestWorkspaceWithWorkspaceDirectoryPathFail(t *testing.T) {
	t.Parallel()
	// The --path flag cannot match one of the workspace directories (i.e. root requirements).
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		fmt.Sprintf(
			"Failure: path \"%v\" is equal to workspace directory \"proto\" defined in \"%v\"",
			filepath.FromSlash("testdata/workspace/success/dir/proto"),
			filepath.FromSlash("testdata/workspace/success/dir/buf.work.yaml"),
		),
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "dir", "proto"),
	)
}

func TestWorkspaceWithInvalidWorkspaceDirectoryPathFail(t *testing.T) {
	t.Parallel()
	// The --path flag did not reference a file found in either of the
	// workspace directories.
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: path does not exist: testdata/workspace/success/dir/notexist`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "dir", "notexist"),
	)
}

func TestWorkspaceWithInvalidDirPathFail(t *testing.T) {
	t.Parallel()
	// The --path flag did not reference a file found outside of
	// one of the workspace directories.
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: path "notexist" has no matching file in the module`,
		"lint",
		filepath.Join("testdata", "workspace", "success", "detached", "proto"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "detached", "proto", "notexist"),
	)
}

func TestWorkspaceWithInvalidArchivePathFail(t *testing.T) {
	t.Parallel()
	// The --path flag did not reference a file found in the archive.
	zipDir := createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "dir"),
		"archive.zip",
	)
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: path "notexist" has no matching file in the module`,
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
		"--path",
		filepath.Join("proto", "notexist"),
	)
}

func TestWorkspaceWithInvalidArchiveAbsolutePathFail(t *testing.T) {
	t.Parallel()
	// The --path flag did not reference an absolute file patfound in the archive.
	zipDir := createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "dir"),
		"archive.zip",
	)
	wd, err := osextended.Getwd()
	require.NoError(t, err)
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(fmt.Sprintf(
			`Failure: %s/proto/rpc.proto: expected to be relative`,
			wd,
		)),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
		"--path",
		filepath.Join(wd, "proto", "rpc.proto"),
	)
}

func createZipFromDir(t *testing.T, rootPath string, archiveName string) string {
	zipDir := filepath.Join(os.TempDir(), rootPath)
	t.Cleanup(
		func() {
			require.NoError(t, os.RemoveAll(zipDir))
		},
	)
	require.NoError(t, os.MkdirAll(zipDir, 0755))

	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	testdataBucket, err := storageosProvider.NewReadWriteBucket(
		rootPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	buffer := bytes.NewBuffer(nil)
	require.NoError(t, storagearchive.Zip(
		context.Background(),
		testdataBucket,
		buffer,
		true,
	))

	zipBucket, err := storageosProvider.NewReadWriteBucket(
		zipDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	zipCloser, err := zipBucket.Put(
		context.Background(),
		archiveName,
	)
	require.NoError(t, err)
	t.Cleanup(
		func() {
			require.NoError(t, zipCloser.Close())
		},
	)
	_, err = zipCloser.Write(buffer.Bytes())
	require.NoError(t, err)
	return zipDir
}

func TestWorkspaceProtoFile(t *testing.T) {
	t.Parallel()
	// The ProtoFileRef is only accepted for lint commands, currently
	// dir_buf_work contains a buf.work instead of a buf.work.yaml
	// we want to make sure this still works
	for _, baseDirPath := range []string{
		"dir",
		"dir_buf_work",
	} {
		wd, err := osextended.Getwd()
		require.NoError(t, err)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(
				`testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`,
			),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "proto", "rpc.proto"),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`,
			),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
		testRunStdout(
			t,
			nil,
			bufcli.ExitCodeFileAnnotation,
			filepath.FromSlash(
				fmt.Sprintf(`%s/testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    %s/testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`, wd, wd),
			),
			"lint",
			filepath.Join(wd, "testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
	}
	testRunStdout(
		t,
		nil,
		0,
		"", // We are not expecting an output for stdout for a successful build
		"build",
		filepath.Join("testdata", "workspace", "success", "protofileref", "another", "foo", "foo.proto"),
	)
	testRunStdout(
		t,
		nil,
		bufcli.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/protofileref/another/foo/foo.proto:3:1:Package name "foo" should be suffixed with a correctly formed version, such as "foo.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "protofileref", "another", "foo", "foo.proto"),
	)
}
