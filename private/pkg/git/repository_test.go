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

package git_test

import (
	"testing"

	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTags(t *testing.T) {
	t.Parallel()

	repo := gittest.ScaffoldGitRepository(t)
	var tags []string
	err := repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		tags = append(tags, tag)

		commit, err := repo.Objects().Commit(commitHash)
		require.NoError(t, err)
		switch tag {
		case "release/v1":
			assert.Equal(t, commit.Message(), "initial commit")
		case "branch/v1":
			assert.Equal(t, commit.Message(), "branch1")
		case "branch/v2":
			assert.Equal(t, commit.Message(), "branch2")
		case "v2":
			assert.Equal(t, commit.Message(), "second commit")
		case "v3.0":
			assert.Equal(t, commit.Message(), "third commit")
		default:
			assert.Failf(t, "unknown tag", tag)
		}

		return nil
	})

	require.NoError(t, err)
	require.ElementsMatch(t, tags, []string{
		"release/v1",
		"branch/v1",
		"branch/v2",
		"v2",
		"v3.0",
	})
}

func TestCommits(t *testing.T) {
	t.Parallel()

	repo := gittest.ScaffoldGitRepository(t)
	var commits []git.Commit
	err := repo.ForEachCommit(
		func(c git.Commit) error {
			commits = append(commits, c)
			return nil
		},
		git.ForEachCommitWithBranchStartPoint(gittest.DefaultBranch),
	)

	require.NoError(t, err)
	require.Len(t, commits, 3)
	assert.Equal(t, commits[0].Message(), "third commit")
	assert.Contains(t, commits[0].Parents(), commits[1].Hash())
	assert.Equal(t, commits[1].Message(), "second commit")
	assert.Contains(t, commits[1].Parents(), commits[2].Hash())
	assert.Equal(t, commits[2].Message(), "initial commit")
	assert.Empty(t, commits[2].Parents())
}

func TestBranches(t *testing.T) {
	t.Parallel()

	repo := gittest.ScaffoldGitRepository(t)
	assert.Equal(t, gittest.DefaultBranch, repo.CurrentBranch())

	var branches []string
	err := repo.ForEachBranch(func(branch string, headHash git.Hash) error {
		branches = append(branches, branch)

		headCommit, err := repo.HEADCommit(branch)
		require.NoError(t, err)
		assert.Equal(t, headHash, headCommit.Hash())

		commit, err := repo.Objects().Commit(headHash)
		require.NoError(t, err)
		switch branch {
		case "master":
			assert.Equal(t, commit.Message(), "third commit")
		case "smian/branch1":
			assert.Equal(t, commit.Message(), "branch1")
		case "smian/branch2":
			assert.Equal(t, commit.Message(), "branch2")
		default:
			assert.Failf(t, "unknown branch", branch)
		}

		return nil
	})

	require.NoError(t, err)
	require.ElementsMatch(t, branches, []string{
		"master",
		"smian/branch1",
		"smian/branch2",
	})
}
