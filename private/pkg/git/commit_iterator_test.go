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

func TestCommits(t *testing.T) {
	t.Parallel()

	repo := gittest.ScaffoldGitRepository(t)
	var commits []git.Commit
	err := repo.CommitIterator.ForEachCommit(gittest.DefaultBranch, func(c git.Commit) error {
		commits = append(commits, c)
		return nil
	})

	require.NoError(t, err)
	require.Len(t, commits, 3)
	assert.Empty(t, commits[0].Parents())
	assert.Equal(t, commits[0].Message(), "initial commit")
	assert.Contains(t, commits[1].Parents(), commits[0].Hash())
	assert.Equal(t, commits[1].Message(), "second commit")
	assert.Contains(t, commits[2].Parents(), commits[1].Hash())
	assert.Equal(t, commits[2].Message(), "third commit")
}
