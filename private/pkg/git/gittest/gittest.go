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
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/stretchr/testify/require"
)

const DefaultBranch = "master"

type TestGitRepository struct {
	DotGitDir         string
	DefaultBranchHead git.Commit
	Reader            git.ObjectReader
	BranchIterator    git.BranchIterator
	CommitIterator    git.CommitIterator
	TagIterator       git.TagIterator
}

func ScaffoldGitRepository(t *testing.T) TestGitRepository {
	runner := command.NewRunner()
	dir := scaffoldGitRepository(t, runner)
	dotGitPath := path.Join(dir, git.DotGitDir)
	reader, err := git.OpenObjectReader(dotGitPath, runner)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, reader.Close())
	})
	branchIterator, err := git.NewBranchIterator(dotGitPath, reader)
	require.NoError(t, err)
	commitIterator, err := git.NewCommitIterator(dotGitPath, reader, git.CommitIteratorWithBaseBranch(DefaultBranch))
	require.NoError(t, err)
	tagIterator, err := git.NewTagIterator(dotGitPath, reader)
	require.NoError(t, err)
	commitBytes, err := os.ReadFile(path.Join(dotGitPath, "refs", "heads", DefaultBranch))
	require.NoError(t, err)
	require.NoError(t, err)
	commitBytes = bytes.TrimRight(commitBytes, "\n")
	commitID, err := git.NewHashFromHex(string(commitBytes))
	require.NoError(t, err)
	commit, err := reader.Commit(commitID)
	require.NoError(t, err)
	return TestGitRepository{
		DotGitDir:         dotGitPath,
		Reader:            reader,
		BranchIterator:    branchIterator,
		CommitIterator:    commitIterator,
		TagIterator:       tagIterator,
		DefaultBranchHead: commit,
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
func scaffoldGitRepository(t *testing.T, runner command.Runner) string {
	dir := t.TempDir()

	// (0) setup local and remote
	runInDir(t, runner, dir, "mkdir", "local", "remote")
	remote := path.Join(dir, "remote")
	runInDir(t, runner, remote, "git", "init", "--bare")
	runInDir(t, runner, remote, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, remote, "git", "config", "user.email", "testbot@buf.build")
	local := path.Join(dir, "local")
	runInDir(t, runner, local, "git", "init")
	runInDir(t, runner, local, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, local, "git", "config", "user.email", "testbot@buf.build")
	runInDir(t, runner, local, "git", "remote", "add", "origin", remote)

	// (1) commit in main branch
	runInDir(t, runner, local, "touch", "randomBinary")
	runInDir(t, runner, local, "chmod", "+x", "randomBinary")
	runInDir(t, runner, local, "mkdir", "proto")
	runInDir(t, runner, path.Join(local, "proto"), "touch", "buf.yaml")
	runInDir(t, runner, local, "mkdir", "-p", "proto/acme/petstore/v1")
	runInDir(t, runner, path.Join(local, "proto", "acme", "petstore", "v1"), "touch", "a.proto", "b.proto")
	runInDir(t, runner, local, "mkdir", "-p", "proto/acme/grocerystore/v1")
	runInDir(t, runner, path.Join(local, "proto", "acme", "grocerystore", "v1"), "touch", "c.proto", "d.proto")
	runInDir(t, runner, local, "git", "add", ".")
	runInDir(t, runner, local, "git", "commit", "-m", "initial commit")
	runInDir(t, runner, local, "git", "tag", "release/v1")
	runInDir(t, runner, local, "git", "push", "--follow-tags", "-u", "-f", "origin", DefaultBranch)

	// (2) branch off main and begin work
	runInDir(t, runner, local, "git", "checkout", "-b", "smian/branch1")
	runInDir(t, runner, path.Join(local, "proto", "acme", "petstore", "v1"), "touch", "e.proto", "f.proto")
	runInDir(t, runner, local, "git", "add", ".")
	runInDir(t, runner, local, "git", "commit", "-m", "branch1")
	runInDir(t, runner, local, "git", "tag", "-m", "for testing", "branch/v1")
	runInDir(t, runner, local, "git", "push", "--follow-tags", "origin", "smian/branch1")

	// (3) branch off branch and begin work
	runInDir(t, runner, local, "git", "checkout", "-b", "smian/branch2")
	runInDir(t, runner, path.Join(local, "proto", "acme", "grocerystore", "v1"), "touch", "g.proto", "h.proto")
	runInDir(t, runner, local, "git", "add", ".")
	runInDir(t, runner, local, "git", "commit", "-m", "branch2")
	runInDir(t, runner, local, "git", "tag", "-m", "for testing", "branch/v2")
	runInDir(t, runner, local, "git", "push", "--follow-tags", "origin", "smian/branch2")

	// (4) merge first branch
	runInDir(t, runner, local, "git", "checkout", DefaultBranch)
	runInDir(t, runner, local, "git", "merge", "--squash", "smian/branch1")
	runInDir(t, runner, local, "git", "commit", "-m", "second commit")
	runInDir(t, runner, local, "git", "tag", "v2")
	runInDir(t, runner, local, "git", "push", "--follow-tags")

	// (5) merge second branch
	runInDir(t, runner, local, "git", "checkout", DefaultBranch)
	runInDir(t, runner, local, "git", "merge", "--squash", "smian/branch2")
	runInDir(t, runner, local, "git", "commit", "-m", "third commit")
	runInDir(t, runner, local, "git", "tag", "v3.0")
	runInDir(t, runner, local, "git", "push", "--follow-tags")

	return local
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
		t.Logf("run %q", strings.Join(append([]string{cmd}, args...), " "))
		_, err := io.Copy(os.Stderr, stderr)
		require.NoError(t, err)
	}
	require.NoError(t, err)
}
