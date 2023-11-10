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
	"context"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
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
func (r *repository) CurrentBranch(ctx context.Context) (string, error) {
	return r.inner.CurrentBranch(ctx)
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
func (r *repository) Checkout(ctx context.Context, t *testing.T, branch string) {
	runInDir(t, r.runner, r.repoDir, "git", "checkout", branch)
}
func (r *repository) CheckoutB(ctx context.Context, t *testing.T, branch string) {
	runInDir(t, r.runner, r.repoDir, "git", "checkout", "-b", branch)
}
func (r *repository) Commit(ctx context.Context, t *testing.T, msg string, files map[string]string) {
	if len(files) == 0 {
		runInDir(t, r.runner, r.repoDir, "git", "commit", "--allow-empty", "-m", msg)
		return
	}
	writeFiles(t, r.repoDir, files)
	runInDir(t, r.runner, r.repoDir, "git", "add", ".")
	runInDir(t, r.runner, r.repoDir, "git", "commit", "-m", msg)
}
func (r *repository) Tag(ctx context.Context, t *testing.T, msg string) {
	runInDir(t, r.runner, r.repoDir, "git", "tag", msg)
}

var _ Repository = (*repository)(nil)
