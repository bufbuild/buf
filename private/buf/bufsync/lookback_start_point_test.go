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

package bufsync

import (
	"context"
	"errors"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookbackStartingPoints(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                 string
		moduleDirsForSync    map[string]struct{}
		commitsModulesToSync []mockSyncableCommit
		branchHeadHash       string
		expectedStartPoints  map[string]string
	}
	testCases := []testCase{
		{
			name: "when_no_modules",
		},
		{
			name: "when_all_modules_have_starting_points",
			moduleDirsForSync: map[string]struct{}{
				"proto/mod-a": {},
				"proto/mod-b": {},
				"proto/mod-c": {},
			},
			commitsModulesToSync: []mockSyncableCommit{
				{
					commitHash: "git1",
					modules: map[string]struct{}{
						"proto/mod-a": {},
					},
				},
				{
					commitHash: "git2",
					modules: map[string]struct{}{
						"proto/mod-a": {},
						"proto/mod-b": {},
					},
				},
				{
					commitHash: "git3",
					modules: map[string]struct{}{
						"proto/mod-a": {},
						"proto/mod-b": {},
						"proto/mod-c": {},
					},
				},
				{
					commitHash: "gitHEAD",
					modules: map[string]struct{}{
						"proto/mod-a": {},
						"proto/mod-b": {},
						"proto/mod-c": {},
					},
				},
			},
			branchHeadHash: "gitHEAD",
			expectedStartPoints: map[string]string{
				"proto/mod-a": "git1",
				"proto/mod-b": "git2",
				"proto/mod-c": "git3",
			},
		},
		{
			name: "when_some_modules_have_starting_points",
			moduleDirsForSync: map[string]struct{}{
				"proto/mod-a": {},
				"proto/mod-b": {},
				"proto/mod-c": {}, // will never appear in commits modules to sync
			},
			commitsModulesToSync: []mockSyncableCommit{
				{
					commitHash: "git1",
					modules: map[string]struct{}{
						"proto/mod-a": {},
					},
				},
				{
					commitHash: "gitHEAD",
					modules: map[string]struct{}{
						"proto/mod-a": {},
						"proto/mod-b": {},
					},
				},
			},
			branchHeadHash: "gitHEAD",
			expectedStartPoints: map[string]string{
				"proto/mod-a": "git1",
				"proto/mod-b": "gitHEAD",
				"proto/mod-c": "gitHEAD",
			},
		},
		{
			name: "when_there_is_no_commits_module_for_sync",
			moduleDirsForSync: map[string]struct{}{
				"proto/mod-a": {},
				"proto/mod-b": {},
				"proto/mod-c": {},
			},
			commitsModulesToSync: nil,
			branchHeadHash:       "gitHEAD",
			expectedStartPoints: map[string]string{
				"proto/mod-a": "gitHEAD",
				"proto/mod-b": "gitHEAD",
				"proto/mod-c": "gitHEAD",
			},
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				const branchName = "my-branch"
				modulesDirsForSync := make(map[string]bufmoduleref.ModuleIdentity)
				for moduleDir := range tc.moduleDirsForSync {
					modulesDirsForSync[moduleDir] = nil // no need for a module identity in this test
				}
				s := &syncer{
					modulesDirsToBranchesToIdentities: map[string]map[string]bufmoduleref.ModuleIdentity{
						branchName: modulesDirsForSync,
					},
					repo: &mockRepo{headHash: tc.branchHeadHash},
				}
				var syncableCommits []*syncableCommit
				for _, commitModulesToSync := range tc.commitsModulesToSync {
					modulesToSyncInCommit := make(map[string]*bufmodulebuild.BuiltModule)
					for moduleDir := range commitModulesToSync.modules {
						modulesToSyncInCommit[moduleDir] = nil // no need for a built module in this test
					}
					syncableCommits = append(syncableCommits, &syncableCommit{
						commit:  &mockCommit{commitModulesToSync.commitHash},
						modules: modulesToSyncInCommit,
					})
				}
				gotStartPoints, err := s.lookbackStartingPoints(context.Background(), branchName, syncableCommits)
				require.NoError(t, err)
				require.Len(t, gotStartPoints, len(tc.expectedStartPoints))
				for moduleDir, gotStartPoint := range gotStartPoints {
					expectedStartPoint, present := tc.expectedStartPoints[moduleDir]
					assert.Truef(t, present, "unexpected module dir %s with start point %s", moduleDir, gotStartPoint.Hex())
					assert.Equal(t, expectedStartPoint, gotStartPoint.Hex())
				}
			})
		}(tc)
	}
}

type mockRepo struct {
	headHash string
}

func (r *mockRepo) HEADCommit(string) (git.Commit, error) { return &mockCommit{hash: r.headHash}, nil }
func (*mockRepo) DefaultBranch() string                   { return "" }
func (*mockRepo) CurrentBranch() string                   { return "" }
func (*mockRepo) Objects() git.ObjectReader               { return nil }
func (*mockRepo) ForEachBranch(func(string, git.Hash) error) error {
	return errors.New("unimplemented")
}
func (*mockRepo) ForEachCommit(func(git.Commit) error, ...git.ForEachCommitOption) error {
	return errors.New("unimplemented")
}
func (*mockRepo) ForEachTag(func(string, git.Hash) error) error {
	return errors.New("unimplemented")
}
func (*mockRepo) Close() error {
	return errors.New("unimplemented")
}

type mockSyncableCommit struct {
	commitHash string
	modules    map[string]struct{}
}

type mockCommit struct {
	hash string
}

func (c *mockCommit) Hash() git.Hash     { return &mockHash{c.hash} }
func (*mockCommit) Tree() git.Hash       { return nil }
func (*mockCommit) Parents() []git.Hash  { return nil }
func (*mockCommit) Author() git.Ident    { return nil }
func (*mockCommit) Committer() git.Ident { return nil }
func (*mockCommit) Message() string      { return "" }
func (*mockCommit) String() string       { return "" }

type mockHash struct {
	hash string
}

func (h *mockHash) Hex() string    { return h.hash }
func (h *mockHash) String() string { return h.hash }
