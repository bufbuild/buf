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
	"context"
	"testing"

	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTags(t *testing.T) {
	t.Parallel()

	repo := gittest.ScaffoldGitRepository(t)
	writeModuleWithSampleCommits(t, context.Background(), repo)
	var tags []string
	err := repo.ForEachTag(func(tag string, commitHash git.Hash) error {
		tags = append(tags, tag)

		commit, err := repo.Objects().Commit(commitHash)
		require.NoError(t, err)
		switch tag {
		case "release/v1":
			assert.Equal(t, commit.Message(), "first commit")
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
	writeModuleWithSampleCommits(t, context.Background(), repo)
	var commitsByDefault []git.Commit
	err := repo.ForEachCommit(func(c git.Commit) error {
		commitsByDefault = append(commitsByDefault, c) // by default we loop from HEAD at the local default branch
		return nil
	})
	require.NoError(t, err)
	require.Len(t, commitsByDefault, 4)

	assert.Equal(t, commitsByDefault[0].Message(), "third commit")
	assert.Contains(t, commitsByDefault[0].Parents(), commitsByDefault[1].Hash())
	assert.Equal(t, commitsByDefault[1].Message(), "second commit")
	assert.Contains(t, commitsByDefault[1].Parents(), commitsByDefault[2].Hash())
	assert.Equal(t, commitsByDefault[2].Message(), "first commit")
	assert.Contains(t, commitsByDefault[2].Parents(), commitsByDefault[3].Hash())
	assert.Equal(t, commitsByDefault[3].Message(), "initial commit")
	assert.Empty(t, commitsByDefault[3].Parents())

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
		assert.Equal(t, commitsByDefault, commitsFromDefaultBranch)
	})

	t.Run("hash_starting_point", func(t *testing.T) {
		var commitsFromSecond []git.Commit
		err = repo.ForEachCommit(
			func(c git.Commit) error {
				commitsFromSecond = append(commitsFromSecond, c)
				return nil
			},
			git.ForEachCommitWithHashStartPoint(commitsByDefault[1].Hash().Hex()),
		)
		require.NoError(t, err)
		require.Len(t, commitsFromSecond, 3)

		assert.Equal(t, commitsFromSecond[0].Message(), "second commit")
		assert.Contains(t, commitsFromSecond[0].Parents(), commitsFromSecond[1].Hash())
		assert.Equal(t, commitsFromSecond[1].Message(), "first commit")
		assert.Contains(t, commitsFromSecond[1].Parents(), commitsFromSecond[2].Hash())
		assert.Equal(t, commitsFromSecond[2].Message(), "initial commit")
		assert.Empty(t, commitsFromSecond[2].Parents())
	})

	t.Run("branch_starting_point", func(t *testing.T) {
		const branchName = "buftest/branch1" // this branch has pushed and unpushed commits
		t.Run("when_local", func(t *testing.T) {
			var commitsFromLocalBranch []git.Commit
			err = repo.ForEachCommit(
				func(c git.Commit) error {
					commitsFromLocalBranch = append(commitsFromLocalBranch, c)
					return nil
				},
				git.ForEachCommitWithBranchStartPoint(branchName),
			)
			require.NoError(t, err)
			require.Len(t, commitsFromLocalBranch, 4)

			assert.Equal(t, commitsFromLocalBranch[0].Message(), "local commit on pushed branch")
			assert.Contains(t, commitsFromLocalBranch[0].Parents(), commitsFromLocalBranch[1].Hash())
			assert.Equal(t, commitsFromLocalBranch[1].Message(), "branch1")
			assert.Contains(t, commitsFromLocalBranch[1].Parents(), commitsFromLocalBranch[2].Hash())
			assert.Equal(t, commitsFromLocalBranch[2].Message(), "first commit")
			assert.Contains(t, commitsFromLocalBranch[2].Parents(), commitsFromLocalBranch[3].Hash())
			assert.Equal(t, commitsFromLocalBranch[3].Message(), "initial commit")
			assert.Empty(t, commitsFromLocalBranch[3].Parents())
		})

		t.Run("when_remote", func(t *testing.T) {
			var commitsFromRemoteBranch []git.Commit
			err = repo.ForEachCommit(
				func(c git.Commit) error {
					commitsFromRemoteBranch = append(commitsFromRemoteBranch, c)
					return nil
				},
				git.ForEachCommitWithBranchStartPoint(
					branchName,
					git.ForEachCommitWithBranchStartPointWithRemote(gittest.DefaultRemote),
				),
			)
			require.NoError(t, err)
			require.Len(t, commitsFromRemoteBranch, 3)

			assert.Equal(t, commitsFromRemoteBranch[0].Message(), "branch1")
			assert.Contains(t, commitsFromRemoteBranch[0].Parents(), commitsFromRemoteBranch[1].Hash())
			assert.Equal(t, commitsFromRemoteBranch[1].Message(), "first commit")
			assert.Contains(t, commitsFromRemoteBranch[1].Parents(), commitsFromRemoteBranch[2].Hash())
			assert.Equal(t, commitsFromRemoteBranch[2].Message(), "initial commit")
			assert.Empty(t, commitsFromRemoteBranch[2].Parents())
		})
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

func TestForEachBranch(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                        string
		remote                      string
		expectedBranchesToCommitMsg map[string]string
	}
	testCases := []testCase{
		{
			name:   "when_local",
			remote: "",
			expectedBranchesToCommitMsg: map[string]string{
				"master":             "third commit",
				"buftest/branch1":    "local commit on pushed branch",
				"buftest/branch2":    "branch2",
				"buftest/local-only": "local commit on local branch",
			},
		},
		{
			name:   "when_remote_exists",
			remote: gittest.DefaultRemote,
			expectedBranchesToCommitMsg: map[string]string{
				"master":          "third commit",
				"buftest/branch1": "branch1",
				"buftest/branch2": "branch2",
			},
		},
		{
			name:                        "when_remote_does_not_exists",
			remote:                      "randomremote",
			expectedBranchesToCommitMsg: nil,
			// we won't error when the remote does not exist, we just don't return any branch.
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				repo := gittest.ScaffoldGitRepository(t)
				writeModuleWithSampleCommits(t, context.Background(), repo)
				currentBranch, err := repo.CheckedOutBranch()
				require.NoError(t, err)
				assert.Equal(t, gittest.DefaultBranch, currentBranch)
				branches := make(map[string]struct{})
				err = repo.ForEachBranch(func(branch string, headHash git.Hash) error {
					require.NotEmpty(t, branch)
					if _, alreadySeen := branches[branch]; alreadySeen {
						assert.Fail(t, "duplicate branch", branch)
					}
					branches[branch] = struct{}{}
					headCommit, err := repo.HEADCommit(
						git.HEADCommitWithBranch(branch),
						git.HEADCommitWithRemote(tc.remote),
					)
					require.NoError(t, err)
					assert.Equal(t, headHash, headCommit.Hash())

					commit, err := repo.Objects().Commit(headHash)
					require.NoError(t, err)
					expectedMsg, ok := tc.expectedBranchesToCommitMsg[branch]
					require.True(t, ok, "unexpected branch", branch)
					assert.Equal(t, expectedMsg, commit.Message())
					return nil
				}, git.ForEachBranchWithRemote(tc.remote))
				assert.NoError(t, err)
				for expectedBranch := range tc.expectedBranchesToCommitMsg {
					_, seen := branches[expectedBranch]
					assert.True(t, seen, "expected branch not seen", expectedBranch)
				}
				for seenBranch := range branches {
					_, expected := tc.expectedBranchesToCommitMsg[seenBranch]
					assert.True(t, expected, "unexpected branch seen", seenBranch)
				}
			})
		}(tc)
	}
}

// the resulting Git repo looks like so:
//
//	.
//	├── proto
//	│   ├── acme
//	│   │   ├── grocerystore
//	│   │   │   └── v1
//	│   │   │       ├── c.proto
//	│   │   │       ├── d.proto
//	│   │   │       ├── g.proto
//	│   │   │       └── h.proto
//	│   │   └── petstore
//	│   │       └── v1
//	│   │           ├── a.proto
//	│   │           ├── b.proto
//	│   │           ├── e.proto
//	│   │           └── f.proto
//	│   └── buf.yaml
//	└── randomBinary (+x)
func writeModuleWithSampleCommits(t *testing.T, ctx context.Context, repo gittest.Repository) {
	currentBranch, err := repo.CheckedOutBranch()
	require.NoError(t, err)

	// (1) commit in main branch
	repo.Commit(t, "first commit", map[string]string{
		"randomBinary":                       "some executable",
		"proto/buf.yaml":                     "some buf.yaml",
		"proto/acme/petstore/v1/a.proto":     "cats",
		"proto/acme/petstore/v1/b.proto":     "animals",
		"proto/acme/grocerystore/v1/c.proto": "toysrus",
		"proto/acme/grocerystore/v1/d.proto": "petsrus",
	}, gittest.CommitWithExecutableFile("randomBinary"))
	repo.Tag(t, "release/v1", "")
	repo.Push(t)

	// (2) branch off main and begin work
	repo.CheckoutB(t, "buftest/branch1")
	repo.Commit(t, "branch1", map[string]string{
		"proto/acme/petstore/v1/e.proto": "loblaws",
		"proto/acme/petstore/v1/f.proto": "merchant of venice",
	})
	repo.Tag(t, "branch/v1", "for testing")
	repo.Push(t)

	// (3) branch off branch and begin work
	repo.CheckoutB(t, "buftest/branch2")
	repo.Commit(t, "branch2", map[string]string{
		"proto/acme/grocerystore/v1/g.proto": "hamlet",
		"proto/acme/grocerystore/v1/h.proto": "bethoven",
	})
	repo.Tag(t, "branch/v2", "for testing")
	repo.Push(t)

	// (4) merge first branch
	repo.Checkout(t, currentBranch)
	repo.Merge(t, "buftest/branch1")
	repo.Commit(t, "second commit", nil)
	repo.Tag(t, "v2", "")
	repo.Push(t)

	// (5) pack some refs
	repo.PackRefs(t)

	// (6) merge second branch
	repo.Checkout(t, currentBranch)
	repo.Merge(t, "buftest/branch2")
	repo.Commit(t, "third commit", nil)
	repo.Tag(t, "v3.0", "")
	repo.Push(t)

	// commit a local-only branch
	repo.CheckoutB(t, "buftest/local-only")
	repo.Commit(t, "local commit on local branch", nil)

	// make a local-only commit on top of a pushed branch
	repo.Checkout(t, "buftest/branch1")
	repo.Commit(t, "local commit on pushed branch", nil)

	// checkout to default branch
	repo.Checkout(t, currentBranch)
}
