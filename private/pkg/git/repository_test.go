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
	err := repo.ForEachCommit(func(c git.Commit) error {
		commits = append(commits, c)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, commits, 3)

	assert.Equal(t, commits[0].Message(), "third commit")
	assert.Contains(t, commits[0].Parents(), commits[1].Hash())
	assert.Equal(t, commits[1].Message(), "second commit")
	assert.Contains(t, commits[1].Parents(), commits[2].Hash())
	assert.Equal(t, commits[2].Message(), "initial commit")
	assert.Empty(t, commits[2].Parents())

	t.Run("default_behavior", func(t *testing.T) {
		var commitsFromDefaultBranch []git.Commit
		err := repo.ForEachCommit(
			func(c git.Commit) error {
				commitsFromDefaultBranch = append(commitsFromDefaultBranch, c)
				return nil
			},
			git.ForEachCommitWithBranchStartPoint(repo.DefaultBranch()),
		)
		require.NoError(t, err)
		assert.Equal(t, commits, commitsFromDefaultBranch)
	})

	t.Run("set_same_starting_point_multiple_times", func(t *testing.T) {
		assert.NoError(t, repo.ForEachCommit(
			func(git.Commit) error { return nil },
			// multiple times, same starting point should be a nop
			git.ForEachCommitWithBranchStartPoint(repo.DefaultBranch()),
			git.ForEachCommitWithBranchStartPoint(repo.DefaultBranch()),
			git.ForEachCommitWithBranchStartPoint(repo.DefaultBranch()),
		))
	})

	t.Run("custom_starting_point", func(t *testing.T) {
		var commitsFromSecond []git.Commit
		err = repo.ForEachCommit(
			func(c git.Commit) error {
				commitsFromSecond = append(commitsFromSecond, c)
				return nil
			},
			git.ForEachCommitWithHashStartPoint(commits[1].Hash().Hex()),
		)
		require.NoError(t, err)
		require.Len(t, commitsFromSecond, 2)

		assert.Equal(t, commitsFromSecond[0].Message(), "second commit")
		assert.Contains(t, commitsFromSecond[0].Parents(), commitsFromSecond[1].Hash())
		assert.Equal(t, commitsFromSecond[1].Message(), "initial commit")
		assert.Empty(t, commitsFromSecond[1].Parents())
	})

	t.Run("failures", func(t *testing.T) {
		t.Parallel()
		type testCase struct {
			name string
			opts []git.ForEachCommitOption
		}
		testCases := []testCase{
			{
				name: "when_multiple_starting_points",
				opts: []git.ForEachCommitOption{
					git.ForEachCommitWithBranchStartPoint("some-branch"),
					git.ForEachCommitWithHashStartPoint("some-hash"),
				},
			},
			{
				name: "when_invalid_hash",
				opts: []git.ForEachCommitOption{
					git.ForEachCommitWithHashStartPoint("invalid-hash"),
				},
			},
			{
				name: "when_non_existent_branch",
				opts: []git.ForEachCommitOption{
					git.ForEachCommitWithBranchStartPoint("non-existent-branch"),
				},
			},
		}
		for _, tc := range testCases {
			func(tc testCase) {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					assert.Error(t, repo.ForEachCommit(
						func(git.Commit) error { return nil },
						tc.opts...,
					))
				})
			}(tc)
		}
	})
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
