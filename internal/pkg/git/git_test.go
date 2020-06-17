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

package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TODO: refactor this into common logic shared by tests

func TestCloneBranchToBucket(t *testing.T) {
	t.Parallel()
	absGitPath, err := filepath.Abs("../../../.git")
	require.NoError(t, err)
	_, err = os.Stat(absGitPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("no .git repository")
			return
		}
		require.NoError(t, err)
	}

	absFilePathSuccess1, err := filepath.Abs("../app/app.go")
	require.NoError(t, err)
	relFilePathSuccess1, err := filepath.Rel(filepath.Dir(absGitPath), absFilePathSuccess1)
	require.NoError(t, err)
	relFilePathError1 := "Makefile"

	cloner := NewCloner(zap.NewNop(), ClonerOptions{})
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		1,
		readWriteBucketCloser,
		CloneToBucketOptions{
			Name: NewBranchName("master"),
			TransformerOptions: []normalpath.TransformerOption{
				normalpath.WithExt(".go"),
			},
		},
	)
	require.NoError(t, err)

	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))

	assert.NoError(t, readWriteBucketCloser.Close())
}

func TestCloneRefToBucket(t *testing.T) {
	t.Parallel()
	absGitPath, err := filepath.Abs("../../../.git")
	require.NoError(t, err)
	_, err = os.Stat(absGitPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("no .git repository")
			return
		}
		require.NoError(t, err)
	}

	absFilePathSuccess1, err := filepath.Abs("../app/app.go")
	require.NoError(t, err)
	relFilePathSuccess1, err := filepath.Rel(filepath.Dir(absGitPath), absFilePathSuccess1)
	require.NoError(t, err)
	relFilePathError1 := "Makefile"

	cloner := NewCloner(zap.NewNop(), ClonerOptions{})
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		1,
		readWriteBucketCloser,
		CloneToBucketOptions{
			Name: NewRefName(testGetLastGitCommit(t)),
			TransformerOptions: []normalpath.TransformerOption{
				normalpath.WithExt(".go"),
			},
		},
	)
	require.NoError(t, err)

	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))

	assert.NoError(t, readWriteBucketCloser.Close())
}

func TestCloneBranchAndRefToBucket(t *testing.T) {
	t.Parallel()
	absGitPath, err := filepath.Abs("../../../.git")
	require.NoError(t, err)
	_, err = os.Stat(absGitPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("no .git repository")
			return
		}
		require.NoError(t, err)
	}

	absFilePathSuccess1, err := filepath.Abs("../app/app.go")
	require.NoError(t, err)
	relFilePathSuccess1, err := filepath.Rel(filepath.Dir(absGitPath), absFilePathSuccess1)
	require.NoError(t, err)
	relFilePathError1 := "Makefile"

	cloner := NewCloner(zap.NewNop(), ClonerOptions{})
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		50,
		readWriteBucketCloser,
		CloneToBucketOptions{
			Name: newRefWithBranch("refs/remotes/origin/master", "master"), // Should hopefully always exist
			TransformerOptions: []normalpath.TransformerOption{
				normalpath.WithExt(".go"),
			},
		},
	)
	require.NoError(t, err)

	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))

	assert.NoError(t, readWriteBucketCloser.Close())
}

func TestCloneDefault(t *testing.T) {
	t.Parallel()
	absGitPath, err := filepath.Abs("../../../.git")
	require.NoError(t, err)
	_, err = os.Stat(absGitPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("no .git repository")
			return
		}
		require.NoError(t, err)
	}

	absFilePathSuccess1, err := filepath.Abs("../app/app.go")
	require.NoError(t, err)
	relFilePathSuccess1, err := filepath.Rel(filepath.Dir(absGitPath), absFilePathSuccess1)
	require.NoError(t, err)
	relFilePathError1 := "Makefile"

	cloner := NewCloner(zap.NewNop(), ClonerOptions{})
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		1,
		readWriteBucketCloser,
		CloneToBucketOptions{
			TransformerOptions: []normalpath.TransformerOption{
				normalpath.WithExt(".go"),
			},
		},
	)
	require.NoError(t, err)

	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readWriteBucketCloser.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))

	assert.NoError(t, readWriteBucketCloser.Close())
}

func testGetLastGitCommit(t *testing.T) string {
	envContainer, err := app.NewEnvContainerForOS()
	require.NoError(t, err)
	buffer := bytes.NewBuffer(nil)
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Env = app.Environ(envContainer)
	cmd.Stdout = buffer
	require.NoError(t, cmd.Run())
	return strings.TrimSpace(buffer.String())
}
