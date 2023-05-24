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
	repo := gittest.ScaffoldGitRepository(t)
	var tags []string
	err := repo.TagIterator.ForEachTag(func(tag string, commitHash git.Hash) error {
		tags = append(tags, tag)

		commit, err := repo.Reader.Commit(commitHash)
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
