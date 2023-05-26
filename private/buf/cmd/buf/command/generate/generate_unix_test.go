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

package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtoFileRef(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "protofileref", "buf.gen.yaml"),
		filepath.Join("testdata", "protofileref", "a", "v1", "a.proto"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "B.java"))
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestOutputWithExclude(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		filepath.Join("testdata", "paths"),
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.gen.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v1"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3"),
	)

	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v2", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "b", "v1", "B.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "foo", "Foo.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "bar", "Bar.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestOutputWithPathWithinExclude(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.gen.yaml"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a"),
	)

	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v2", "A.java"))
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestOutputWithExcludeWithinPath(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.gen.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		"--path",
		filepath.Join("testdata", "paths", "a"),
		filepath.Join("testdata", "paths"),
	)

	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v2", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "b", "v1", "B.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestOutputWithNestedExcludeAndTargetPaths(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.gen.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3", "foo", "bar.proto"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v3"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v3", "foo"),
		filepath.Join("testdata", "paths"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "foo", "FooOuterClass.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "b", "v1", "B.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v2", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "foo", "BarOuterClass.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestWorkspaceGenerateWithExcludeAndTargetPaths(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "workspace", "buf.gen.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "workspace", "a", "v3", "foo", "bar.proto"),
		"--exclude-path",
		filepath.Join("testdata", "workspace", "a", "v3"),
		"--path",
		filepath.Join("testdata", "workspace", "a", "v3", "foo"),
		"--exclude-path",
		filepath.Join("testdata", "workspace", "b", "v1", "foo.proto"),
		filepath.Join("testdata", "workspace"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "foo", "FooOuterClass.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "b", "v1", "B.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v2", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "A.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v3", "foo", "BarOuterClass.java"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}
