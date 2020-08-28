// Copyright 2020 Buf Technologies, Inc.
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

package bufcoretesting

import (
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewFileInfo returns a new FileInfo for testing.
func NewFileInfo(
	t *testing.T,
	path string,
	externalPath string,
	isImport bool,
) bufcore.FileInfo {
	fileInfo, err := bufcore.NewFileInfo(
		path,
		externalPath,
		isImport,
	)
	require.NoError(t, err)
	return fileInfo
}

// AssertFileInfosEqual asserts the expected FileInfos equal the actual FileInfos.
func AssertFileInfosEqual(t *testing.T, expected []bufcore.FileInfo, actual []bufcore.FileInfo) {
	assert.Equal(t, expected, actual)
}

// FileInfosToAbs converts the external paths to absolute.
func FileInfosToAbs(t *testing.T, fileInfos []bufcore.FileInfo) []bufcore.FileInfo {
	newFileInfos := make([]bufcore.FileInfo, len(fileInfos))
	for i, fileInfo := range fileInfos {
		newFileInfos[i] = FileInfoToAbs(t, fileInfo)
	}
	return newFileInfos
}

// FileInfoToAbs converts the external path to absolute.
func FileInfoToAbs(t *testing.T, fileInfo bufcore.FileInfo) bufcore.FileInfo {
	absExternalPath, err := normalpath.NormalizeAndAbsolute(fileInfo.ExternalPath())
	require.NoError(t, err)
	newFileInfo, err := bufcore.NewFileInfo(
		fileInfo.Path(),
		absExternalPath,
		fileInfo.IsImport(),
	)
	require.NoError(t, err)
	return newFileInfo
}
