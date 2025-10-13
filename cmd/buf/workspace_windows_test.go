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

//go:build windows

package buf

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceAbsoluteFail(t *testing.T) {
	// The workspace file (v1: buf.work.yaml, v2: buf.yaml) file cannot specify absolute paths.
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: decode testdata/workspace/fail/absolute/windows/buf.work.yaml: directory "C:\\buf" is invalid: C:\buf: expected to be relative`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "absolute", "windows"),
	)
	testRunStdoutStderrNoWarn(
		t,
		nil,
		1,
		``,
		`Failure: decode testdata/workspace/fail/v2/absolute/windows/buf.yaml: invalid module path: C:\buf: expected to be relative`,
		"build",
		filepath.Join("testdata", "workspace", "fail", "v2", "absolute", "windows"),
	)
}
