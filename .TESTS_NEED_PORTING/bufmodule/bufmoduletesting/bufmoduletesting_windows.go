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

//go:build windows
// +build windows

package bufmoduletesting

import (
	"path/filepath"
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
		filepath.Clean(filepath.FromSlash(externalPath)),
		moduleIdentity,
		commit,
	)
	require.NoError(t, err)
	return fileInfo
}
