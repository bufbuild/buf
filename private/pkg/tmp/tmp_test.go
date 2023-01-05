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

package tmp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	t.Parallel()
	tmpFile, err := NewFileWithData([]byte("foo"))
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(tmpFile.AbsPath()))
	data, err := os.ReadFile(tmpFile.AbsPath())
	assert.NoError(t, err)
	assert.Equal(t, "foo", string(data))
	assert.NoError(t, tmpFile.Close())
	_, err = os.ReadFile(tmpFile.AbsPath())
	assert.Error(t, err)
}

func TestDir(t *testing.T) {
	t.Parallel()
	tmpDir, err := NewDir()
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(tmpDir.AbsPath()))
	fileInfo, err := os.Lstat(tmpDir.AbsPath())
	assert.NoError(t, err)
	assert.True(t, fileInfo.IsDir())
	assert.NoError(t, tmpDir.Close())
	_, err = os.Lstat(tmpDir.AbsPath())
	assert.Error(t, err)
}
