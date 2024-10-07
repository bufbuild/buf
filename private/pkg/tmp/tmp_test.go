// Copyright 2020-2024 Buf Technologies, Inc.
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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpFile, err := NewFile(ctx, strings.NewReader("foo"))
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(tmpFile.Path()))
	data, err := os.ReadFile(tmpFile.Path())
	assert.NoError(t, err)
	assert.Equal(t, "foo", string(data))
	assert.NoError(t, tmpFile.Close())
	_, err = os.ReadFile(tmpFile.Path())
	assert.Error(t, err)
}

func TestFileCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	tmpFile, err := NewFile(ctx, strings.NewReader("foo"))
	require.NoError(t, err)
	_, err = os.ReadFile(tmpFile.Path())
	assert.NoError(t, err)
	cancel()
	time.Sleep(1 * time.Second)
	_, err = os.ReadFile(tmpFile.Path())
	assert.Error(t, err)
}

func TestDir(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir, err := NewDir(ctx)
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(tmpDir.Path()))
	fileInfo, err := os.Lstat(tmpDir.Path())
	assert.NoError(t, err)
	assert.True(t, fileInfo.IsDir())
	assert.NoError(t, tmpDir.Close())
	_, err = os.Lstat(tmpDir.Path())
	assert.Error(t, err)
}

func TestDirCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	tmpDir, err := NewDir(ctx)
	require.NoError(t, err)
	_, err = os.Lstat(tmpDir.Path())
	assert.NoError(t, err)
	cancel()
	time.Sleep(1 * time.Second)
	_, err = os.Lstat(tmpDir.Path())
	assert.Error(t, err)
}
