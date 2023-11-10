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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/stretchr/testify/require"
)

const (
	DefaultBranch = "master"
	DefaultRemote = "origin"
)

type Repository interface {
	git.Repository
	Commit(ctx context.Context, t *testing.T, msg string, files map[string]string)
	Checkout(ctx context.Context, t *testing.T, branch string)
	CheckoutB(ctx context.Context, t *testing.T, branch string)
	Tag(ctx context.Context, t *testing.T, msg string)
}

type scaffoldOptions struct {
	withOnlyInitialCommit bool
}

type ScaffoldGitRepositoryOption func(*scaffoldOptions)

func ScaffoldGitRepositoryWithOnlyInitialCommit() ScaffoldGitRepositoryOption {
	return func(opts *scaffoldOptions) {
		opts.withOnlyInitialCommit = true
	}
}

func ScaffoldGitRepository(t *testing.T, opts ...ScaffoldGitRepositoryOption) Repository {
	var options scaffoldOptions
	for _, opt := range opts {
		opt(&options)
	}
	runner := command.NewRunner()
	dir := scaffoldGitRepository(t, runner, DefaultBranch, options.withOnlyInitialCommit)
	dotGitPath := filepath.Join(dir, git.DotGitDir)
	repo, err := git.OpenRepository(
		context.Background(),
		dotGitPath,
		runner,
		git.OpenRepositoryWithDefaultBranch(DefaultBranch),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})
	return newRepository(
		repo,
		dir,
		runner,
	)
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
func scaffoldGitRepository(t *testing.T, runner command.Runner, defaultBranch string, onlyInitialCommit bool) string {
	dir := t.TempDir()

	// (0) setup local and remote
	runInDir(t, runner, dir, "mkdir", "local", "remote")
	remoteDir := filepath.Join(dir, "remote")
	runInDir(t, runner, remoteDir, "git", "init", "--bare")
	runInDir(t, runner, remoteDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, remoteDir, "git", "config", "user.email", "testbot@buf.build")
	localDir := filepath.Join(dir, "local")
	runInDir(t, runner, localDir, "git", "init")
	runInDir(t, runner, localDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, localDir, "git", "config", "user.email", "testbot@buf.build")
	runInDir(t, runner, localDir, "git", "remote", "add", DefaultRemote, remoteDir)

	// (1) initial commit and push
	writeFiles(t, localDir, map[string]string{"README.md": "This is a scaffold repository.\n"})
	runInDir(t, runner, localDir, "git", "add", ".")
	runInDir(t, runner, localDir, "git", "commit", "-m", "initial commit")
	runInDir(t, runner, localDir, "git", "push", "-u", "-f", "origin", defaultBranch)
	if onlyInitialCommit {
		return localDir
	}

	// (1) commit in main branch
	writeFiles(t, localDir, map[string]string{
		"randomBinary":                       "some executable",
		"proto/buf.yaml":                     "some buf.yaml",
		"proto/acme/petstore/v1/a.proto":     "cats",
		"proto/acme/petstore/v1/b.proto":     "animals",
		"proto/acme/grocerystore/v1/c.proto": "toysrus",
		"proto/acme/grocerystore/v1/d.proto": "petsrus",
	})
	runInDir(t, runner, localDir, "chmod", "+x", "randomBinary")
	runInDir(t, runner, localDir, "git", "add", ".")
	runInDir(t, runner, localDir, "git", "commit", "-m", "first commit")
	runInDir(t, runner, localDir, "git", "tag", "release/v1")
	runInDir(t, runner, localDir, "git", "push", "--follow-tags", "origin", defaultBranch)

	// (2) branch off main and begin work
	runInDir(t, runner, localDir, "git", "checkout", "-b", "buftest/branch1")
	writeFiles(t, localDir, map[string]string{
		"proto/acme/petstore/v1/e.proto": "loblaws",
		"proto/acme/petstore/v1/f.proto": "merchant of venice",
	})
	runInDir(t, runner, localDir, "git", "add", ".")
	runInDir(t, runner, localDir, "git", "commit", "-m", "branch1")
	runInDir(t, runner, localDir, "git", "tag", "-m", "for testing", "branch/v1")
	runInDir(t, runner, localDir, "git", "push", "--follow-tags", "origin", "buftest/branch1")

	// (3) branch off branch and begin work
	runInDir(t, runner, localDir, "git", "checkout", "-b", "buftest/branch2")
	writeFiles(t, localDir, map[string]string{
		"proto/acme/grocerystore/v1/g.proto": "hamlet",
		"proto/acme/grocerystore/v1/h.proto": "bethoven",
	})
	runInDir(t, runner, localDir, "git", "add", ".")
	runInDir(t, runner, localDir, "git", "commit", "-m", "branch2")
	runInDir(t, runner, localDir, "git", "tag", "-m", "for testing", "branch/v2")
	runInDir(t, runner, localDir, "git", "push", "--follow-tags", "origin", "buftest/branch2")

	// (4) merge first branch
	runInDir(t, runner, localDir, "git", "checkout", defaultBranch)
	runInDir(t, runner, localDir, "git", "merge", "--squash", "buftest/branch1")
	runInDir(t, runner, localDir, "git", "commit", "-m", "second commit")
	runInDir(t, runner, localDir, "git", "tag", "v2")
	runInDir(t, runner, localDir, "git", "push", "--follow-tags")

	// (5) pack some refs
	runInDir(t, runner, localDir, "git", "pack-refs", "--all")
	runInDir(t, runner, localDir, "git", "repack")

	// (6) merge second branch
	runInDir(t, runner, localDir, "git", "checkout", defaultBranch)
	runInDir(t, runner, localDir, "git", "merge", "--squash", "buftest/branch2")
	runInDir(t, runner, localDir, "git", "commit", "-m", "third commit")
	runInDir(t, runner, localDir, "git", "tag", "v3.0")
	runInDir(t, runner, localDir, "git", "push", "--follow-tags")

	// commit a local-only branch
	runInDir(t, runner, localDir, "git", "checkout", "-b", "buftest/local-only")
	runInDir(t, runner, localDir, "git", "commit", "--allow-empty", "-m", "local commit on local branch")

	// make a local-only commit on top of a pushed branch
	runInDir(t, runner, localDir, "git", "checkout", "buftest/branch1")
	runInDir(t, runner, localDir, "git", "commit", "--allow-empty", "-m", "local commit on pushed branch")

	// checkout to default branch
	runInDir(t, runner, localDir, "git", "checkout", defaultBranch)

	return localDir
}

func runInDir(t *testing.T, runner command.Runner, dir string, cmd string, args ...string) {
	stderr := bytes.NewBuffer(nil)
	err := runner.Run(
		context.Background(),
		cmd,
		command.RunWithArgs(args...),
		command.RunWithDir(dir),
		command.RunWithStderr(stderr),
	)
	if err != nil {
		require.FailNow(
			t,
			"git command failed",
			"run %q, error %q",
			strings.Join(append([]string{cmd}, args...), " "),
			stderr,
		)
	}
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	for path, contents := range files {
		path := normalpath.Unnormalize(path)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, path), []byte(contents), 0600))
	}
}
