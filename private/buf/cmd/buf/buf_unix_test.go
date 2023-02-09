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
)

func TestLsFilesSymlinks(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/symlinks/a.proto
testdata/symlinks/b.proto`),
		"ls-files",
		filepath.Join("testdata", "symlinks"),
	)
	testRunStdout(
		t,
		nil,
		0,
		filepath.FromSlash(`testdata/symlinks/a.proto`),
		"ls-files",
		"--disable-symlinks",
		filepath.Join("testdata", "symlinks"),
	)
}

func TestBuildSymlinks(t *testing.T) {
	t.Parallel()
	testRunStdout(
		t,
		nil,
		100,
		``,
		"build",
		filepath.Join("testdata", "symlinks"),
	)
	testRunStdout(
		t,
		nil,
		0,
		``,
		"build",
		"--disable-symlinks",
		filepath.Join("testdata", "symlinks"),
	)
}
