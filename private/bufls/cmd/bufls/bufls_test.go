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

package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
)

func TestDefinition(t *testing.T) {
	t.Parallel()
	testDefinitionExtend(t)
	testDefinitionGroup(t)
	testDefinitionMap(t)
	testDefinitionNamespace(t)
	testDefinitionNoPackage(t)
	testDefinitionWithCacheDependency(t)
	testDefinitionWithWorkspaceDependency(t)
}

func TestDefinitionError(t *testing.T) {
	t.Parallel()
	testDefintionError(
		t,
		"testdata/local/nopackage/foo.proto:3:1", // The 'message' keyword.
		"could not resolve definition for location testdata/local/nopackage/foo.proto:3:1",
	)
	testDefintionError(
		t,
		"testdata/local/nopackage/foo.proto:13:8", // A '.' delimiter.
		"could not resolve definition for location testdata/local/nopackage/foo.proto:13:8",
	)
}

func TestLocationError(t *testing.T) {
	t.Parallel()
	testDefintionError(
		t,
		"testdata/local/nopackage/foo.proto",
		"location testdata/local/nopackage/foo.proto is not structured as <filename>:<line>:<column>",
	)
	testDefintionError(
		t,
		"testdata/local/nopackage/foo:1:1",
		"location path testdata/local/nopackage/foo must be a .proto file",
	)
	testDefintionError(
		t,
		"testdata/local/nopackage/foo.proto:-2:1",
		"location line -2 must be a positive integer",
	)
	testDefintionError(
		t,
		"testdata/local/nopackage/foo.proto:1:0",
		"location column 0 must be a positive integer",
	)
}

func testDefinitionExtend(t *testing.T) {
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:11:5",
		"testdata/local/extend/extend.proto:7:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:12:5",
		"testdata/local/extend/extend.proto:9:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:13:5",
		"testdata/local/extend/extend.proto:17:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:14:5",
		"testdata/local/extend/extend.proto:9:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:14:12",
		"testdata/local/extend/extend.proto:17:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:19:17",
		"testdata/local/extend/extend.proto:7:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:20:17",
		"testdata/local/extend/extend.proto:9:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/extend/extend.proto:21:24",
		"testdata/local/extend/extend.proto:17:11",
	)
}

func testDefinitionGroup(t *testing.T) {
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:18:14",
		"testdata/local/group/group.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:18:18",
		"testdata/local/group/group.proto:6:18",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:22:21",
		"testdata/local/group/group.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:22:21",
		"testdata/local/group/group.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:22:25",
		"testdata/local/group/group.proto:6:18",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:25:12",
		"testdata/local/group/group.proto:10:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:26:12",
		"testdata/local/group/group.proto:6:18",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:30:12",
		"testdata/local/group/group.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:30:16",
		"testdata/local/group/group.proto:6:18",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:31:23",
		"testdata/local/group/group.proto:12:20",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:31:23",
		"testdata/local/group/group.proto:12:20",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:35:23",
		"testdata/local/group/group.proto:6:18",
	)
	testDefintionSuccess(
		t,
		"testdata/local/group/group.proto:35:23",
		"testdata/local/group/group.proto:6:18",
	)
}

func testDefinitionMap(t *testing.T) {
	testDefintionSuccess(
		t,
		"testdata/local/map/map.proto:6:15",
		"testdata/local/map/map.proto:18:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/map/map.proto:7:21",
		"testdata/local/map/map.proto:19:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/map/map.proto:12:19",
		"testdata/local/map/map.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/map/map.proto:14:24",
		"testdata/local/map/map.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/map/map.proto:14:31",
		"testdata/local/map/map.proto:9:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/map/map.proto:14:44",
		"testdata/local/map/map.proto:10:13",
	)
	// The map test cases defined in testdata/local/mapentry might not be something we ever
	// want to support (i.e. referencing a synthetic map message with map_entry set).
	// It might even be a bug in protoc). The testdata is included anyway for posterity.

	// For more information, see the following:
	// https://github.com/bufbuild/protobuf-grammar/pull/1#discussion_r932220444
}

func testDefinitionNamespace(t *testing.T) {
	testDefintionSuccess(
		t,
		"testdata/local/namespace/foo/foo/foo_foo.proto:8:3",
		"testdata/local/namespace/foo/foo.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/namespace/foo/foo/foo_foo.proto:8:13",
		"testdata/local/namespace/foo/foo.proto:6:11",
	)
}

func testDefinitionNoPackage(t *testing.T) {
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:3:9",
		"testdata/local/nopackage/foo.proto:3:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:9:5",
		"testdata/local/nopackage/foo.proto:6:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:13:5",
		"testdata/local/nopackage/foo.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:13:9",
		"testdata/local/nopackage/foo.proto:6:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:16:4",
		"testdata/local/nopackage/foo.proto:5:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:16:8",
		"testdata/local/nopackage/foo.proto:6:11",
	)
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:20:4",
		"testdata/local/nopackage/foo.proto:3:9",
	)
	testDefintionSuccess(
		t,
		"testdata/local/nopackage/foo.proto:21:8",
		"testdata/local/nopackage/foo.proto:8:11",
	)
}

func testDefinitionWithCacheDependency(t *testing.T) {
	testDefintionSuccessWithCache(
		t,
		"testdata/cache",
		"testdata/local/withcachedependency/baz.proto:8:10",
		"testdata/cache/v1/module/data/buf.build/test-owner/test-repository/6e230f46113f498392c82d12b1a07b70/bar.proto:7:9",
	)
}

func testDefinitionWithWorkspaceDependency(t *testing.T) {
	testDefintionSuccessWithCache(
		t,
		"testdata/cache",
		"testdata/local/withworkspacedependency/workspace.proto:8:28",
		"testdata/local/withcachedependency/baz.proto:7:9",
	)
	testDefintionSuccessWithCache(
		t,
		"testdata/cache",
		"testdata/local/withworkspacedependency/workspace.proto:9:28",
		"testdata/local/withcachedependency/baz.proto:13:9",
	)
	testDefintionSuccessWithCache(
		t,
		"testdata/cache",
		"testdata/local/withworkspacedependency/workspace.proto:13:29",
		"testdata/local/withcachedependency/baz.proto:7:9",
	)
	testDefintionSuccessWithCache(
		t,
		"testdata/cache",
		"testdata/local/withworkspacedependency/workspace.proto:14:29",
		"testdata/local/withcachedependency/baz.proto:13:9",
	)
}

func testDefintionSuccess(
	t *testing.T,
	inputLocation string,
	outputLocation string,
) {
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(outputLocation),
		"definition",
		filepath.FromSlash(inputLocation),
	)
}

func testDefintionSuccessWithCache(
	t *testing.T,
	cacheDir string,
	inputLocation string,
	outputLocation string,
) {
	testRunStdoutWithCache(
		t,
		nil,
		0,
		filepath.FromSlash(outputLocation),
		filepath.FromSlash(cacheDir),
		"definition",
		filepath.FromSlash(inputLocation),
	)
}

func testDefintionError(
	t *testing.T,
	inputLocation string,
	outputError string,
) {
	testRunStdoutStderr(
		t,
		nil,
		1,
		"",
		filepath.FromSlash(outputError),
		"definition",
		filepath.FromSlash(inputLocation),
	)
}

func testRunStdout(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdout(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStdout,
		newEnvFunc(t, ""),
		stdin,
		args...,
	)
}

func testRunStdoutWithCache(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, cacheDir string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdout(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStdout,
		newEnvFunc(t, cacheDir),
		stdin,
		args...,
	)
}

func testRunStdoutStderr(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, expectedStderr string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdoutStderr(
		t,
		func(use string) *appcmd.Command { return NewRootCommand(use) },
		expectedExitCode,
		expectedStdout,
		expectedStderr,
		newEnvFunc(t, ""),
		stdin,
		// we do not want warnings to be part of our stderr test calculation
		append(
			args,
			"--no-warn",
		)...,
	)
}

func newEnvFunc(tb testing.TB, cacheDir string) func(string) map[string]string {
	if cacheDir == "" {
		cacheDir = tb.TempDir()
	}
	return func(use string) map[string]string {
		return map[string]string{
			useEnvVar(use, "CACHE_DIR"): cacheDir,
			useEnvVar(use, "HOME"):      tb.TempDir(),
			"PATH":                      os.Getenv("PATH"),
		}
	}
}

func useEnvVar(use string, suffix string) string {
	return strings.ToUpper(use) + "_" + suffix
}
