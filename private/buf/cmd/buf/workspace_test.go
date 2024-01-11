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

	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceDir(t *testing.T) {
	// Directory paths contained within a workspace.
	t.Parallel()
	for _, baseDirPath := range []string{
		// TODO(doria): can we move `dir` and `dir_buf_work` into a directory `v1` for symmetry with `v2`?
		"dir",          // dir contains a v1 buf.work.yaml
		"dir_buf_work", // dir_buf_work contains a v1 buf.work
		"v2/dir",       // v2/dir contains a v2 buf.yaml
	} {
		wd, err := osext.Getwd()
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
			bufctl.ExitCodeFileAnnotation,
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
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`,
			),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
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
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/a/proto/a/v1/a.proto
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto
		    testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".
		    testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
		)
		testRunStdoutStderrNoWarn(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			"",
			filepath.FromSlash(`Failure: testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto: import "request.proto": file does not exist`),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--config",
			`{"version":"v1beta1","lint": {"use": ["PACKAGE_DIRECTORY_MATCH"]}}`,
		)
		testRunStdoutStderrNoWarn(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			"",
			filepath.FromSlash(`Failure: testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto: import "request.proto": file does not exist`),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--config",
			`{"version":"v1","lint": {"use": ["PACKAGE_DIRECTORY_MATCH"]}}`,
		)
		testRunStdout(
			t,
			nil,
			1,
			"",
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--config",
			`version: v2
modules:
  - directory: a
  - directory: other/proto
    lint:
      use:
	    - PACKAGE_DIRECTORY_MATCH
  - directory: proto`,
			"--path",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
		// TODO: targeting information problem. The rpc.proto file should be the only one
		// targeted, but request.proto was targeted.
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
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
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--path",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
	}
}

func TestWorkspaceBreaking(t *testing.T) {
	t.Parallel()
	for _, dirPaths := range []struct {
		base    string
		against string
	}{
		{base: "dir", against: "breaking"},
		{base: "dir_buf_work", against: "breaking"},
		{base: "v2/dir", against: "breaking"},
		{base: "v2/dir", against: "v2/breaking"},
	} {
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPaths.against),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPaths.against+`/a/proto/a/v1/a.proto
		    testdata/workspace/success/`+dirPaths.against+`/other/proto/request.proto
		    testdata/workspace/success/`+dirPaths.against+`/proto/rpc.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPaths.against),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPaths.against+`/other/proto/request.proto:5:1:Previously present field "1" with name "name" on message "Request" was deleted.
		    testdata/workspace/success/`+dirPaths.against+`/proto/rpc.proto:8:5:Field "1" with name "request" on message "RPC" changed option "json_name" from "req" to "request".
		    testdata/workspace/success/`+dirPaths.against+`/proto/rpc.proto:8:21:Field "1" on message "RPC" changed name from "req" to "request".`),
			"breaking",
			filepath.Join("testdata", "workspace", "success", dirPaths.against),
			"--against",
			filepath.Join("testdata", "workspace", "success", dirPaths.base),
		)
	}
}

func TestWorkspaceArchiveDir(t *testing.T) {
	// Archive that defines a workspace at the root of the archive.
	t.Parallel()
	for _, dirPath := range []string{
		"dir",
		"v2/dir",
	} {
		zipDir := createZipFromDir(
			t,
			filepath.Join("testdata", "workspace", "success", dirPath),
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
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
			"lint",
			filepath.Join(zipDir, "archive.zip#subdir=proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`proto/rpc.proto:3:1:Files with package "example" must be within a directory "example" relative to root but were in directory ".".
        proto/rpc.proto:3:1:Package name "example" should be suffixed with a correctly formed version, such as "example.v1".`),
			"lint",
			filepath.Join(zipDir, "archive.zip#subdir=proto"),
			"--path",
			filepath.Join("proto", "rpc.proto"),
		)
	}
}

func TestWorkspaceNestedArchive(t *testing.T) {
	// Archive that defines a workspace in a sub-directory to the root.
	t.Parallel()
	for _, dirPath := range []string{
		"nested",
		"v2/nested",
	} {
		zipDir := createZipFromDir(
			t,
			filepath.Join("testdata", "workspace", "success", dirPath),
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
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`proto/internal/internal.proto:3:1:Files with package "internal" must be within a directory "internal" relative to root but were in directory ".".
        proto/internal/internal.proto:3:1:Package name "internal" should be suffixed with a correctly formed version, such as "internal.v1".`),
			"lint",
			filepath.Join(zipDir, "archive.zip#subdir=proto/internal"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`proto/internal/internal.proto:3:1:Files with package "internal" must be within a directory "internal" relative to root but were in directory ".".
        proto/internal/internal.proto:3:1:Package name "internal" should be suffixed with a correctly formed version, such as "internal.v1".`),
			"lint",
			filepath.Join(zipDir, "archive.zip#subdir=proto/internal"),
			"--path",
			filepath.Join("proto", "internal", "internal.proto"),
		)
	}
}

func TestWorkspaceDetached(t *testing.T) {
	t.Parallel()
	// The workspace doesn't include the 'proto' directory, so
	// its contents aren't included in the workspace.
	for _, dirPath := range []string{
		"detached",
		"v2/detached",
	} {
		// In the pre-refactor, this was a successful call, as the workspace was still being discovered
		// as the enclosing workspace, despite not pointing to the proto directory. In post-refactor
		// we'd consider this a bug: you specified the proto directory, and no controlling workspace
		// was discovered, therefore you build as if proto was the input directory, which results in
		// request.proto not existing as an import.
		testRunStdoutStderrNoWarn(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			``,
			filepath.FromSlash(`Failure: testdata/workspace/success/`+dirPath+`/proto/rpc.proto: import "request.proto": file does not exist`),
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/proto/rpc.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		// In the pre-refactor, this was a successful call, as the workspace was still being discovered
		// as the enclosing workspace, despite not pointing to the proto directory. In post-refactor
		// we'd consider this a bug: you specified the proto directory, and no controlling workspace
		// was discovered, therefore you build as if proto was the input directory, which results in
		// request.proto not existing as an import.
		testRunStdoutStderrNoWarn(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			``,
			filepath.FromSlash(`Failure: testdata/workspace/success/`+dirPath+`/proto/rpc.proto: import "request.proto": file does not exist`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/request.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/request.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
	}
}

func TestWorkspaceNoModuleConfig(t *testing.T) {
	// The workspace points to modules that don't contain a buf.yaml.
	//
	// This only tests for v1 workspaces, since in v2, we no longer have nested
	// buf.yaml files for workspace modules.
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
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
	//
	// This only tests for v1 workspaces, since in v2, we no longer have nested
	// buf.lock files for workspace modules and this module would already be excluded
	// from the workspace-level buf.lock.
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/lock/b/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/lock/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "lock", "b"),
	)
}

func TestWorkspaceWithTransitiveDependencies(t *testing.T) {
	t.Parallel()
	// The workspace points to a module that includes transitive
	// dependencies (i.e. a depends on b, and b depends on c).
	for _, dirPath := range []string{
		"transitive",
		"v2/transitive",
	} {
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/proto/a.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/proto/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/proto/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "private", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/private/proto/b.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "private", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/private/proto/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/private/proto/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "private", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c.proto:3:1:Files with package "c" must be within a directory "c" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/other/proto/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c.proto
        testdata/workspace/success/`+dirPath+`/private/proto/b.proto
        testdata/workspace/success/`+dirPath+`/proto/a.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c.proto:3:1:Files with package "c" must be within a directory "c" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/other/proto/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".
        testdata/workspace/success/`+dirPath+`/private/proto/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/private/proto/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".
        testdata/workspace/success/`+dirPath+`/proto/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/proto/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
	}
}

func TestWorkspaceWithDiamondDependency(t *testing.T) {
	t.Parallel()
	// The workspace points to a module that includes a diamond
	// dependency (i.e. a depends on b and c, and b depends on c).
	for _, dirPath := range []string{
		"diamond",
		"v2/diamond",
	} {
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/proto/a.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/proto/a.proto:3:1:Files with package "a" must be within a directory "a" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/proto/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "private", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/private/proto/b.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "private", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/private/proto/b.proto:3:1:Files with package "b" must be within a directory "b" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/private/proto/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "private", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c.proto:3:1:Files with package "c" must be within a directory "c" relative to root but were in directory ".".
        testdata/workspace/success/`+dirPath+`/other/proto/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
	}
}

func TestWorkspaceWKT(t *testing.T) {
	t.Parallel()
	// The workspace includes multiple images that import the same
	// well-known type (empty.proto).
	for _, dirPath := range []string{
		"wkt",
		"v2/wkt",
	} {
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c/c.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c/c.proto:6:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath, "other", "proto"),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"build",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
		testRunStdout(
			t,
			nil,
			0,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c/c.proto
        testdata/workspace/success/`+dirPath+`/proto/a/a.proto
        testdata/workspace/success/`+dirPath+`/proto/b/b.proto`),
			"ls-files",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/other/proto/c/c.proto:6:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".
        testdata/workspace/success/`+dirPath+`/proto/a/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".
        testdata/workspace/success/`+dirPath+`/proto/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
			"lint",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
		testRunStdout(
			t,
			nil,
			0,
			``,
			"breaking",
			filepath.Join("testdata", "workspace", "success", dirPath),
			"--against",
			filepath.Join("testdata", "workspace", "success", dirPath),
		)
	}
}

func TestWorkspaceRoots(t *testing.T) {
	// Workspaces should support modules with multiple roots specified in a v1beta1 buf.yaml.
	// This is only tested with v1 workspaces, since v2 workspaces does not support individual
	// buf.yaml configurations.
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
		bufctl.ExitCodeFileAnnotation,
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
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module1/a/a.proto:3:1:Package name "a" should be suffixed with a correctly formed version, such as "a.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "roots", "module1", "a"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module2/root1/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "roots", "module2", "root1", "b"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module2/root2/c/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "roots", "module2", "root2", "c"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(`testdata/workspace/success/roots/module2/root1/b/b.proto:3:1:Package name "b" should be suffixed with a correctly formed version, such as "b.v1".
testdata/workspace/success/roots/module2/root2/c/c.proto:3:1:Package name "c" should be suffixed with a correctly formed version, such as "c.v1".
testdata/workspace/success/roots/module2/root3/d/d.proto:3:1:Package name "d" should be suffixed with a correctly formed version, such as "d.v1".`),
		"lint",
		filepath.Join("testdata", "workspace", "success", "roots", "module2"),
		"--config",
		filepath.Join("testdata", "workspace", "success", "roots", "module2", "other.buf.yaml"),
	)
}

func TestWorkspaceProtoFile(t *testing.T) {
	t.Parallel()
	// The ProtoFileRef is only accepted for lint commands, currently
	// dir_buf_work contains a buf.work instead of a buf.work.yaml
	// we want to make sure this still works
	for _, baseDirPath := range []string{
		"dir",
		"dir_buf_work",
		"v2/dir",
	} {
		wd, err := osext.Getwd()
		require.NoError(t, err)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
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
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`,
			),
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
		testRunStdout(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(
				fmt.Sprintf(`%s/testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Files with package "request" must be within a directory "request" relative to root but were in directory ".".
		    %s/testdata/workspace/success/`+baseDirPath+`/other/proto/request.proto:3:1:Package name "request" should be suffixed with a correctly formed version, such as "request.v1".`, wd, wd),
			),
			"lint",
			filepath.Join(wd, "testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
	}
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
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: input contained 1 images, whereas against contained 3 images`,
		"breaking",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
		"--against",
		filepath.Join("testdata", "workspace", "success", "breaking"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: input contained 1 images, whereas against contained 3 images`,
		"breaking",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
		"--against",
		filepath.Join("testdata", "workspace", "success", "v2", "breaking"),
	)
}

func TestWorkspaceDuplicateFail(t *testing.T) {
	t.Parallel()
	// The workspace includes multiple images that define the same file.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: foo.proto exists in multiple locations: testdata/workspace/fail/duplicate/other/proto/foo.proto testdata/workspace/fail/duplicate/proto/foo.proto`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "duplicate"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: foo.proto exists in multiple locations: testdata/workspace/fail/v2/duplicate/other/proto/foo.proto testdata/workspace/fail/v2/duplicate/proto/foo.proto`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "v2", "duplicate"),
	)
}

func TestWorkspaceDuplicateFailSpecificModule(t *testing.T) {
	// The workspace includes multiple images that define the same file.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: foo.proto exists in multiple locations: testdata/workspace/fail/duplicate/other/proto/foo.proto testdata/workspace/fail/duplicate/proto/foo.proto`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "duplicate", "proto"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: foo.proto exists in multiple locations: testdata/workspace/fail/v2/duplicate/other/proto/foo.proto testdata/workspace/fail/v2/duplicate/proto/foo.proto`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "v2", "duplicate", "proto"),
	)
}

func TestWorkspaceNotExistFail(t *testing.T) {
	t.Parallel()
	// The directory defined in the workspace does not exist.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: module "notexist" had no .proto files`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "notexist"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: module "notexist" had no .proto files`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "v2", "notexist"),
	)
}

func TestWorkspaceJumpContextFail(t *testing.T) {
	t.Parallel()
	// The workspace directories cannot jump context.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		// TODO: figure out why even on windows, the cleaned, unnormalised path is "/"-separated from decode error
		`Failure: decode testdata/workspace/fail/jumpcontext/buf.work.yaml: directory "../breaking/other/proto" is invalid: ../breaking/other/proto: is outside the context directory`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "jumpcontext"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		// TODO: figure out why even on windows, the cleaned, unnormalised path is "/"-separated from decode error
		`Failure: decode testdata/workspace/fail/v2/jumpcontext/buf.yaml: invalid module directory: ../breaking/other/proto: is outside the context directory`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "v2", "jumpcontext"),
	)
}

func TestWorkspaceDirOverlapFail(t *testing.T) {
	t.Parallel()
	// The buf.work.yaml file cannot specify overlapping diretories.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		// TODO: figure out why even on windows, the cleaned, unnormalised path is "/"-separated from decode error
		`Failure: decode testdata/workspace/fail/diroverlap/buf.work.yaml: directory "foo" contains directory "foo/bar"`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "diroverlap"),
	)
}

func TestWorkspaceInputOverlapFail(t *testing.T) {
	// The target input cannot overlap with any of the directories defined
	// in the workspace.
	t.Parallel()
	// TODO
	t.Skip("TODO")
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: failed to build input "proto/buf" because it is contained by directory "proto" listed in testdata/workspace/fail/overlap/buf.work.yaml`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "overlap", "proto", "buf"),
	)
	testRunStdoutStderrNoWarn(
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
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		// TODO: figure out why even on windows, the cleaned, unnormalised path is "/"-separated from decode error
		`Failure: decode testdata/workspace/fail/noversion/buf.work.yaml: "version" is not set. Please add "version: v1"`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "noversion"),
	)
}

func TestWorkspaceInvalidVersionFail(t *testing.T) {
	// The buf.work.yaml must specify a valid version.
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		// TODO: figure out why even on windows, the cleaned, unnormalised path is "/"-separated from decode error
		`Failure: decode testdata/workspace/fail/invalidversion/buf.work.yaml: unknown file version: "v9"`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "invalidversion"),
	)
}

func TestWorkspaceNoDirectoriesFail(t *testing.T) {
	t.Parallel()
	// The buf.work.yaml must specify at least one directory.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		// TODO: figure out why even on windows, the cleaned, unnormalised path is "/"-separated from decode error
		`Failure: decode testdata/workspace/fail/nodirectories/buf.work.yaml: directories is empty`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "nodirectories"),
	)
}

func TestWorkspaceWithWorkspacePathFail(t *testing.T) {
	t.Parallel()
	// The --path flag cannot match the workspace directory (i.e. root requirements).
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: given input is equal to a value of --path - this has no effect and is disallowed`,
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "dir"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: given input is equal to a value of --path - this has no effect and is disallowed`,
		"lint",
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
	)
}

func TestWorkspaceWithWorkspaceExcludePathFail(t *testing.T) {
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: given input is equal to a value of --exclude-path - this would exclude everything`,
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--exclude-path",
		filepath.Join("testdata", "workspace", "success", "dir"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: given input is equal to a value of --exclude-path - this would exclude everything`,
		"lint",
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
		"--exclude-path",
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
	)
}

func TestWorkspaceWithWorkspaceDirectoryPathFail(t *testing.T) {
	t.Parallel()
	// The --path flag cannot match one of the workspace directories (i.e. root requirements).
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: module "proto" was specified with --path - specify this module path directly as an input`,
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "dir", "proto"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: module "proto" was specified with --path - specify this module path directly as an input`,
		"lint",
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "v2", "dir", "proto"),
	)
}

func TestWorkspaceWithInvalidWorkspaceDirectoryPathFail(t *testing.T) {
	t.Parallel()
	// The --path flag did not reference a file found in either of the
	// workspace directories.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: `+bufmodule.ErrNoTargetProtoFiles.Error(),
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "dir", "notexist"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: `+bufmodule.ErrNoTargetProtoFiles.Error(),
		"lint",
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "v2", "dir", "notexist"),
	)
}

func TestWorkspaceWithInvalidDirPathFail(t *testing.T) {
	t.Parallel()
	// The --path flag did not reference a file found outside of
	// one of the workspace directories.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: `+bufmodule.ErrNoTargetProtoFiles.Error(),
		"lint",
		filepath.Join("testdata", "workspace", "success", "dir", "proto"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "dir", "proto", "notexist"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: `+bufmodule.ErrNoTargetProtoFiles.Error(),
		"lint",
		filepath.Join("testdata", "workspace", "success", "v2", "dir", "proto"),
		"--path",
		filepath.Join("testdata", "workspace", "success", "v2", "dir", "proto", "notexist"),
	)
}

func TestWorkspaceWithInvalidArchivePathFail(t *testing.T) {
	// The --path flag did not reference a file found in the archive.
	zipDir := createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "dir"),
		"archive.zip",
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: `+bufmodule.ErrNoTargetProtoFiles.Error(),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
		"--path",
		filepath.Join("proto", "notexist"),
	)
	zipDir = createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
		"archive.zip",
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: `+bufmodule.ErrNoTargetProtoFiles.Error(),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
		"--path",
		filepath.Join("proto", "notexist"),
	)
}

func TestWorkspaceWithInvalidArchiveAbsolutePathFail(t *testing.T) {
	// The --path flag did not reference an absolute file patfound in the archive.
	zipDir := createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "dir"),
		"archive.zip",
	)
	wd, err := osext.Getwd()
	require.NoError(t, err)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(fmt.Sprintf(
			`Failure: %s/proto/rpc.proto: absolute paths cannot be used for this input type`,
			wd,
		)),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
		"--path",
		filepath.Join(wd, "proto", "rpc.proto"),
	)
	zipDir = createZipFromDir(
		t,
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
		"archive.zip",
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(fmt.Sprintf(
			`Failure: %s/proto/rpc.proto: absolute paths cannot be used for this input type`,
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
