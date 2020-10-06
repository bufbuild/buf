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
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TODO: refactor this into common logic shared by tests

func TestCloneBranchToBucket(t *testing.T) {
	t.Parallel()
	// TODO: we can't assume that our CI platform will always have both
	// the master branch and the current branch, we need to reconfigure
	// this test to potentially use a remote public URL with a master branch
	//
	// CI is always set in GitHub Actions
	// https://docs.github.com/en/free-pro-team@latest/actions/reference/environment-variables#default-environment-variables
	if os.Getenv("CI") == "true" {
		t.Skip("we do not necessarily have a master branch in CI")
	}
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
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		1,
		readBucketBuilder,
		CloneToBucketOptions{
			Mapper: storage.MatchPathExt(".go"),
			Name:   NewBranchName("master"),
		},
	)
	require.NoError(t, err)
	readBucket, err := readBucketBuilder.ToReadBucket()
	require.NoError(t, err)

	_, err = readBucket.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readBucket.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))
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
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		1,
		readBucketBuilder,
		CloneToBucketOptions{
			Mapper: storage.MatchPathExt(".go"),
			Name:   NewRefName(testGetLastGitCommit(t)),
		},
	)
	require.NoError(t, err)
	readBucket, err := readBucketBuilder.ToReadBucket()
	require.NoError(t, err)

	_, err = readBucket.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readBucket.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))
}

func TestCloneBranchAndRefToBucket(t *testing.T) {
	t.Parallel()
	// TODO: we can't assume that our CI platform will always have both
	// the master branch and the current branch, we need to reconfigure
	// this test to potentially use a remote public URL with a master branch
	//
	// CI is always set in GitHub Actions
	// https://docs.github.com/en/free-pro-team@latest/actions/reference/environment-variables#default-environment-variables
	if os.Getenv("CI") == "true" {
		t.Skip("we do not necessarily have a master branch in CI")
	}
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
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		50,
		readBucketBuilder,
		CloneToBucketOptions{
			Mapper: storage.MatchPathExt(".go"),
			Name:   newRefWithBranch("refs/remotes/origin/master", "master"), // Should hopefully always exist
		},
	)
	require.NoError(t, err)
	readBucket, err := readBucketBuilder.ToReadBucket()
	require.NoError(t, err)

	_, err = readBucket.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readBucket.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))
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
	readBucketBuilder := storagemem.NewReadBucketBuilder()
	err = cloner.CloneToBucket(
		context.Background(),
		envContainer,
		"file://"+absGitPath,
		1,
		readBucketBuilder,
		CloneToBucketOptions{
			Mapper: storage.MatchPathExt(".go"),
		},
	)
	require.NoError(t, err)
	readBucket, err := readBucketBuilder.ToReadBucket()
	require.NoError(t, err)

	_, err = readBucket.Stat(context.Background(), relFilePathSuccess1)
	assert.NoError(t, err)
	_, err = readBucket.Stat(context.Background(), relFilePathError1)
	assert.True(t, storage.IsNotExist(err))
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
