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

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	imagev1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceDir(t *testing.T) {
	// Directory paths contained within a workspace.
	t.Parallel()
	for _, baseDirPath := range []string{
		// TODO FUTURE(doria): can we move `dir` and `dir_buf_work` into a directory `v1` for symmetry with `v2`?
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
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:5:8:import "request.proto": file does not exist`),
			"",
			"lint",
			filepath.Join("testdata", "workspace", "success", baseDirPath),
			"--config",
			`{"version":"v1beta1","lint": {"use": ["PACKAGE_DIRECTORY_MATCH"]}}`,
		)
		testRunStdoutStderrNoWarn(
			t,
			nil,
			bufctl.ExitCodeFileAnnotation,
			filepath.FromSlash(`testdata/workspace/success/`+baseDirPath+`/proto/rpc.proto:5:8:import "request.proto": file does not exist`),
			"",
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
  - path: a
  - path: other/proto
    lint:
      use:
	    - PACKAGE_DIRECTORY_MATCH
  - path: proto`,
			"--path",
			filepath.Join("testdata", "workspace", "success", baseDirPath, "other", "proto", "request.proto"),
		)
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
			filepath.Join("rpc.proto"),
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
			filepath.Join("internal.proto"),
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
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/proto/rpc.proto:5:8:import "request.proto": file does not exist`),
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
			filepath.FromSlash(`testdata/workspace/success/`+dirPath+`/proto/rpc.proto:5:8:import "request.proto": file does not exist`),
			``,
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
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
	)
	// If the against contained more images than the input, we return a mismatch.
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
	// If the input contains more images than the against, we attempt to map against images
	// if there are no unique module-specific breaking configs for the input.
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(
			`testdata/workspace/success/dir/other/proto/request.proto:6:3:Field "1" with name "name" on message "Request" changed type from "int64" to "string".`,
		),
		"breaking",
		filepath.Join("testdata", "workspace", "success", "dir"),
		"--against",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
	)
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(
			`testdata/workspace/success/v2/dir/other/proto/request.proto:6:3:Field "1" with name "name" on message "Request" changed type from "int64" to "string".`,
		),
		"breaking",
		filepath.Join("testdata", "workspace", "success", "v2", "dir"),
		"--against",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
	)
	// If we find unique module-specific breaking changes, then we still fail with a mismatch error.
	testRunStdout(
		t,
		nil,
		1,
		``,
		"breaking",
		filepath.Join("testdata", "workspace", "fail", "modulebreakingconfig"),
		"--against",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
	)
	testRunStdout(
		t,
		nil,
		1,
		``,
		"breaking",
		filepath.Join("testdata", "workspace", "fail", "v2", "modulebreakingconfig"),
		"--against",
		filepath.Join("testdata", "workspace", "fail", "breaking"),
	)
}

func TestWorkspaceDuplicateDirPathSuccess(t *testing.T) {
	t.Parallel()
	workspaceDir := filepath.Join("testdata", "workspace", "success", "duplicate_dir_path")
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"prefix/bar/v1/bar.proto": {},
			"prefix/foo/v1/foo.proto": {moduleFullName: "buf.build/shared/zero"},
			"prefix/x/x.proto":        {moduleFullName: "buf.build/shared/one"},
			"prefix/y/y.proto":        {},
			"v1/separate.proto":       {},
		},
		workspaceDir,
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"prefix/bar/v1/bar.proto": {},
			"prefix/foo/v1/foo.proto": {moduleFullName: "buf.build/shared/zero"},
		},
		filepath.Join(workspaceDir, "proto", "shared"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"prefix/x/x.proto": {moduleFullName: "buf.build/shared/one"},
			"prefix/y/y.proto": {},
		},
		filepath.Join(workspaceDir, "proto", "shared1"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"prefix/x/x.proto":        {moduleFullName: "buf.build/shared/one"},
			"prefix/y/y.proto":        {},
			"prefix/bar/v1/bar.proto": {},
			"prefix/foo/v1/foo.proto": {moduleFullName: "buf.build/shared/zero"},
		},
		filepath.Join(workspaceDir, "proto"),
	)
}

func TestWorkspaceDuplicateDirPathOverlappingIncludeSuccess(t *testing.T) {
	t.Parallel()
	workspaceDir := filepath.Join("testdata", "workspace", "success", "duplicate_dir_path_overlapping_include")
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/v1/foo.proto":         {},
			"foo/v2/foo.proto":         {},
			"foo/bar/v1/bar.proto":     {},
			"foo/bar/v2/bar.proto":     {},
			"foo/bar/baz/v1/baz.proto": {},
			"foo/bar/baz/v2/baz.proto": {},
		},
		workspaceDir,
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/v1/foo.proto": {},
			"foo/v2/foo.proto": {},
		},
		workspaceDir,
		// exclude a module contained within
		"--exclude-path",
		filepath.Join(workspaceDir, "proto", "foo", "bar"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/v1/foo.proto": {},
		},
		workspaceDir,
		// filter within a module
		"--path",
		filepath.Join(workspaceDir, "proto", "foo", "v1", "foo.proto"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/bar/v2/bar.proto": {},
		},
		workspaceDir,
		// filter within another module
		"--path",
		filepath.Join(workspaceDir, "proto", "foo", "bar", "v2"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/bar/v1/bar.proto":     {},
			"foo/bar/v2/bar.proto":     {},
			"foo/bar/baz/v2/baz.proto": {},
		},
		workspaceDir,
		// filter and exclude
		"--path",
		filepath.Join(workspaceDir, "proto", "foo", "bar"),
		"--exclude-path",
		filepath.Join(workspaceDir, "proto", "foo", "bar", "baz", "v1"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/v1/foo.proto":         {},
			"foo/bar/v1/bar.proto":     {},
			"foo/bar/baz/v1/baz.proto": {},
		},
		workspaceDir,
		// filter within each module
		"--path",
		filepath.Join(workspaceDir, "proto", "foo", "v1"),
		"--path",
		filepath.Join(workspaceDir, "proto", "foo", "bar", "v1"),
		"--path",
		filepath.Join(workspaceDir, "proto", "foo", "bar", "baz", "v1"),
	)
	// Test each module is linted with the correct config.
	testRunStdout(
		t,
		nil,
		bufctl.ExitCodeFileAnnotation,
		filepath.FromSlash(
			`testdata/workspace/success/duplicate_dir_path_overlapping_include/proto/foo/bar/baz/v2/baz.proto:1:1:Files must have a syntax explicitly specified. If no syntax is specified, the file defaults to "proto2".
testdata/workspace/success/duplicate_dir_path_overlapping_include/proto/foo/bar/v2/bar.proto:1:1:Files must have a package defined.`,
		),
		"lint",
		workspaceDir,
	)
}

func TestWorkspaceOverlappingModuleDirPaths(t *testing.T) {
	t.Parallel()
	workspaceDir := filepath.Join("testdata", "workspace", "success", "overlapping_dir_path")
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/foo.proto":      {moduleFullName: "buf.test/acme/foobar"},
			"bar/bar.proto":      {moduleFullName: "buf.test/acme/foobar"},
			"foo_internal.proto": {},
			"bar_internal.proto": {},
		},
		workspaceDir,
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo_internal.proto": {},
		},
		workspaceDir,
		"--path",
		filepath.Join(workspaceDir, "proto", "foo", "internal", "foo_internal.proto"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo/foo.proto": {moduleFullName: "buf.test/acme/foobar"},
			"bar/bar.proto": {moduleFullName: "buf.test/acme/foobar"},
		},
		workspaceDir,
		"--exclude-path",
		filepath.Join(workspaceDir, "proto", "foo", "internal", "foo_internal.proto"),
		"--exclude-path",
		filepath.Join(workspaceDir, "proto", "bar", "internal", "bar_internal.proto"),
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
		`Failure: foo.proto is contained in multiple modules:
  path: "other/proto"
  path: "proto"`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "duplicate"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: v1/foo.proto is contained in multiple modules:
  path: "other/proto", includes: "other/proto/v1", excludes: "other/proto/v1/inner"
  path: "proto", includes: ["proto/v1", "proto/v2"], excludes: ["proto/v1/inner", "proto/v2/inner"]`,
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
		`Failure: foo.proto is contained in multiple modules:
  path: "other/proto"
  path: "proto"`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "duplicate", "proto"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: v1/foo.proto is contained in multiple modules:
  path: "other/proto", includes: "other/proto/v1", excludes: "other/proto/v1/inner"
  path: "proto", includes: ["proto/v1", "proto/v2"], excludes: ["proto/v1/inner", "proto/v2/inner"]`,

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
		filepath.FromSlash(`Failure: Module "path: "notexist"" had no .proto files`),
		"build",
		filepath.Join("testdata", "workspace", "fail", "notexist"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: Module "path: "notexist"" had no .proto files`),
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
		`Failure: decode testdata/workspace/fail/jumpcontext/buf.work.yaml: directory "../breaking/other/proto" is invalid: ../breaking/other/proto: is outside the context directory`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "jumpcontext"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: decode testdata/workspace/fail/v2/jumpcontext/buf.yaml: invalid module path: ../breaking/other/proto: is outside the context directory`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "v2", "jumpcontext"),
	)
}

func TestWorkspaceDirOverlapFail(t *testing.T) {
	t.Parallel()
	// The buf.work.yaml file cannot specify overlapping directories.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: decode testdata/workspace/fail/diroverlap/buf.work.yaml: directory "foo" contains directory "foo/bar"`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "diroverlap"),
	)
}

func TestWorkspaceInputOverlapFail(t *testing.T) {
	// The target input cannot overlap with any of the directories defined
	// in the workspace.
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: failed to build input "proto/buf" because it is contained by module at path "proto" specified in your configuration, you must provide the workspace or module as the input, and filter to this path using --path`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "overlap", "proto", "buf"),
	)
	// This works because of our fallback logic, so we build the workspace at testdata/workspace/success/dir
	testRunStdout(t, nil, 0, ``, "build", filepath.Join("testdata", "workspace", "success", "dir", "other"))
}

func TestWorkspaceInputOverlapNonExistentDirFail(t *testing.T) {
	// The target input is a non-existent directory and the workspace contains a single
	// module at the root, ".".
	t.Parallel()
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: no .proto files were targeted. This can occur if no .proto files are found in your input, --path points to files that do not exist, or --exclude-path excludes all files.`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "overlap", "proto", "fake-dir"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: no .proto files were targeted. This can occur if no .proto files are found in your input, --path points to files that do not exist, or --exclude-path excludes all files.`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "v2", "overlap", "fake-dir"),
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
		`Failure: given input is equal to a value of --path, this has no effect and is disallowed`,
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
		`Failure: given input is equal to a value of --path, this has no effect and is disallowed`,
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
		`Failure: given input "." is equal to a value of --exclude-path ".", this would exclude everything`,
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
		`Failure: given input "." is equal to a value of --exclude-path ".", this would exclude everything`,
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
		`Failure: module "proto" was specified with --path, specify this module path directly as an input`,
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
		`Failure: module "proto" was specified with --path, specify this module path directly as an input`,
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
	if runtime.GOOS == "windows" {
		// TODO FUTURE: failing test, fix on windows, there is temp dir clean-up fail, a reference to archive.zip not closed
		t.Skip()
	}
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
	if runtime.GOOS == "windows" {
		// TODO FUTURE: failing test, fix on windows, there is temp dir clean-up fail, a reference to archive.zip not closed
		t.Skip()
	}
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
			`Failure: %s/proto/rpc.proto: expected to be relative`,
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
			`Failure: %s/proto/rpc.proto: expected to be relative`,
			wd,
		)),
		"lint",
		filepath.Join(zipDir, "archive.zip#subdir=proto"),
		"--path",
		filepath.Join(wd, "proto", "rpc.proto"),
	)
}

func TestWorkspaceWithTargetingModuleCommonParentDir(t *testing.T) {
	workspaceDir := filepath.Join("testdata", "workspace", "success", "shared_parent_dir")
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo.proto":        {},
			"bar.proto":        {},
			"baz.proto":        {},
			"imported.proto":   {},
			"standalone.proto": {},
		},
		workspaceDir,
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"imported.proto":   {},
			"standalone.proto": {},
		},
		filepath.Join(workspaceDir, "standalone"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo.proto":      {},
			"bar.proto":      {},
			"baz.proto":      {},
			"imported.proto": {isImport: true},
		},
		filepath.Join(workspaceDir, "parent"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo.proto":      {isImport: true},
			"bar.proto":      {},
			"baz.proto":      {},
			"imported.proto": {isImport: true},
		},
		filepath.Join(workspaceDir, "parent/nextlayer"),
	)
	requireBuildOutputFilePaths(
		t,
		map[string]expectedFileInfo{
			"foo.proto":      {isImport: true},
			"bar.proto":      {},
			"imported.proto": {isImport: true},
		},
		filepath.Join(workspaceDir, "parent/nextlayer/bar"),
	)
}

func createZipFromDir(t *testing.T, rootPath string, archiveName string) string {
	zipDir := filepath.Join(t.TempDir(), rootPath)
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
	require.NoError(t, storage.PutPath(
		context.Background(),
		zipBucket,
		archiveName,
		buffer.Bytes(),
	))
	return zipDir
}

type expectedFileInfo struct {
	isImport       bool
	moduleFullName string
}

func requireBuildOutputFilePaths(t *testing.T, expectedFilePathToInfo map[string]expectedFileInfo, buildArgs ...string) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	appcmdtesting.Run(
		t,
		func(use string) *appcmd.Command { return newRootCommand(use) },
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdout(stdout),
		appcmdtesting.WithStderr(stderr),
		appcmdtesting.WithArgs(append(
			[]string{
				"build",
				"-o=-#format=binpb",
			},
			buildArgs...,
		)...),
	)
	outputImage := &imagev1.Image{}
	require.NoError(t, protoencoding.NewWireUnmarshaler(nil).Unmarshal(stdout.Bytes(), outputImage))

	filesToCheck := xslices.ToStructMap(xslices.MapKeysToSlice(expectedFilePathToInfo))

	for _, imageFile := range outputImage.GetFile() {
		filePath := imageFile.GetName()
		expectedFileInfo, ok := expectedFilePathToInfo[filePath]
		require.Truef(t, ok, "unexpected file in the image built: %s", filePath)
		require.Equal(t, expectedFileInfo.isImport, imageFile.GetBufExtension().GetIsImport())
		if expectedFileInfo.moduleFullName != "" {
			moduleName := imageFile.GetBufExtension().GetModuleInfo().GetName()
			require.NotNil(t, moduleName)
			require.Equal(t, expectedFileInfo.moduleFullName, fmt.Sprintf("%s/%s/%s", moduleName.GetRemote(), moduleName.GetOwner(), moduleName.GetRepository()))
		} else {
			require.Nil(t, imageFile.GetBufExtension().GetModuleInfo().GetName())
		}
		delete(filesToCheck, filePath)
	}
	require.Zerof(t, len(filesToCheck), "expected files missing from image built: %v", xslices.MapKeysToSortedSlice(filesToCheck))
}
