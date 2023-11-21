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

package gittest

import (
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/stretchr/testify/require"
)

type repository struct {
	inner   git.Repository
	repoDir string
	runner  command.Runner
}

func newRepository(
	inner git.Repository,
	repoDir string,
	runner command.Runner,
) *repository {
	return &repository{
		inner:   inner,
		repoDir: repoDir,
		runner:  runner,
	}
}

func (r *repository) Close() error {
	return r.inner.Close()
}
func (r *repository) CheckedOutBranch(options ...git.CheckedOutBranchOption) (string, error) {
	return r.inner.CheckedOutBranch(options...)
}
func (r *repository) DefaultBranch() string {
	return r.inner.DefaultBranch()
}
func (r *repository) ForEachBranch(f func(branch string, headHash git.Hash) error, options ...git.ForEachBranchOption) error {
	return r.inner.ForEachBranch(f, options...)
}
func (r *repository) ForEachCommit(f func(commit git.Commit) error, options ...git.ForEachCommitOption) error {
	return r.inner.ForEachCommit(f, options...)
}
func (r *repository) ForEachTag(f func(tag string, commitHash git.Hash) error) error {
	return r.inner.ForEachTag(f)
}
func (r *repository) HEADCommit(options ...git.HEADCommitOption) (git.Commit, error) {
	return r.inner.HEADCommit(options...)
}
func (r *repository) Objects() git.ObjectReader {
	return r.inner.Objects()
}
func (r *repository) Checkout(t *testing.T, branch string) {
	runInDir(t, r.runner, r.repoDir, "git", "checkout", branch)
}
func (r *repository) CheckoutB(t *testing.T, branch string) {
	runInDir(t, r.runner, r.repoDir, "git", "checkout", "-b", branch)
}
func (r *repository) Commit(t *testing.T, msg string, files map[string]string, opts ...CommitOption) {
	if len(files) == 0 {
		runInDir(t, r.runner, r.repoDir, "git", "commit", "--allow-empty", "-m", msg)
		return
	}
	var options commitOpts
	for _, opt := range opts {
		opt(&options)
	}
	writeFiles(t, r.repoDir, files)
	for _, path := range options.executablePaths {
		runInDir(t, r.runner, r.repoDir, "chmod", "+x", path)
	}
	runInDir(t, r.runner, r.repoDir, "git", "add", ".")
	runInDir(t, r.runner, r.repoDir, "git", "commit", "-m", msg)
}
func (r *repository) Tag(t *testing.T, name string, msg string) {
	if msg != "" {
		runInDir(t, r.runner, r.repoDir, "git", "tag", "-fm", msg, name)
	} else {
		runInDir(t, r.runner, r.repoDir, "git", "tag", "-f", name)
	}
}
func (r *repository) Push(t *testing.T) {
	currentBranch, err := r.CheckedOutBranch()
	require.NoError(t, err)
	runInDir(t, r.runner, r.repoDir, "git", "push", "--follow-tags", "origin", currentBranch)
}
func (r *repository) Merge(t *testing.T, branch string) {
	runInDir(t, r.runner, r.repoDir, "git", "merge", "--squash", branch)
}
func (r *repository) PackRefs(t *testing.T) {
	runInDir(t, r.runner, r.repoDir, "git", "pack-refs", "--all")
	runInDir(t, r.runner, r.repoDir, "git", "repack")
}
func (r *repository) ResetHard(t *testing.T, ref string) {
	runInDir(t, r.runner, r.repoDir, "git", "reset", "--hard", ref)
}

var _ Repository = (*repository)(nil)
