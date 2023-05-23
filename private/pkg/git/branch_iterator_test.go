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

	"github.com/bufbuild/buf/private/pkg/git/gittest"
	"github.com/stretchr/testify/require"
)

func TestBranches(t *testing.T) {
	repo := gittest.ScaffoldGitRepository(t)
	var branches []string
	err := repo.BranchIterator.ForEachBranch(func(branch string) error {
		branches = append(branches, branch)
		return nil
	})

	require.NoError(t, err)
	require.ElementsMatch(t, branches, []string{
		"master",
		"smian/branch1",
		"smian/branch2",
	})
}
