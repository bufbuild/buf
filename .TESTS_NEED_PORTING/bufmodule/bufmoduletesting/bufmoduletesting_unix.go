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

// Matching the unix-like build tags in the Golang standard library based on the dependency
// on "path/filepath", i.e. https://cs.opensource.google/go/go/+/refs/tags/go1.17:src/path/filepath/path_unix.go;l=5-6

//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js,wasm linux netbsd openbsd solaris

package bufmoduletesting

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/stretchr/testify/require"
)

// NewFileInfo returns a new FileInfo for testing.
func NewFileInfo(
	t *testing.T,
	path string,
	externalPath string,
	moduleIdentity bufmoduleref.ModuleIdentity,
	commit string,
) bufmoduleref.FileInfo {
	fileInfo, err := bufmoduleref.NewFileInfo(
		path,
		externalPath,
		moduleIdentity,
		commit,
	)
	require.NoError(t, err)
	return fileInfo
}
