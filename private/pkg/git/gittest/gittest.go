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
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/stretchr/testify/require"
)

const DefaultBranch = "master"

func ScaffoldGitRepository(t *testing.T) git.Repository {
	runner := command.NewRunner()
	dir := scaffoldGitRepository(t, runner)
	dotGitPath := path.Join(dir, git.DotGitDir)
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
	return repo
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
	repoDir := t.TempDir()

	// (0) setup repo
	runInDir(t, runner, repoDir, "git", "init")
	runInDir(t, runner, repoDir, "git", "config", "user.name", "Buf TestBot")
	runInDir(t, runner, repoDir, "git", "config", "user.email", "testbot@buf.build")

	// (1) commit in main branch
	writeFiles(t, repoDir, map[string]string{
		"randomBinary":                       "some executable",
		"proto/buf.yaml":                     "some buf.yaml",
		"proto/acme/petstore/v1/a.proto":     "cats",
		"proto/acme/petstore/v1/b.proto":     "animals",
		"proto/acme/grocerystore/v1/c.proto": "toysrus",
		"proto/acme/grocerystore/v1/d.proto": "petsrus",
	})
	runInDir(t, runner, repoDir, "chmod", "+x", "randomBinary")
	runInDir(t, runner, repoDir, "git", "add", ".")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "initial commit")
	runInDir(t, runner, repoDir, "git", "tag", "release/v1")

	// (2) branch off main and begin work
	runInDir(t, runner, repoDir, "git", "checkout", "-b", "buftest/branch1")
	writeFiles(t, repoDir, map[string]string{
		"proto/acme/petstore/v1/e.proto": "loblaws",
		"proto/acme/petstore/v1/f.proto": "merchant of venice",
	})
	runInDir(t, runner, repoDir, "git", "add", ".")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "branch1")
	runInDir(t, runner, repoDir, "git", "tag", "-m", "for testing", "branch/v1")

	// (3) branch off branch and begin work
	runInDir(t, runner, repoDir, "git", "checkout", "-b", "buftest/branch2")
	writeFiles(t, repoDir, map[string]string{
		"proto/acme/grocerystore/v1/g.proto": "hamlet",
		"proto/acme/grocerystore/v1/h.proto": "bethoven",
	})
	runInDir(t, runner, repoDir, "git", "add", ".")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "branch2")
	runInDir(t, runner, repoDir, "git", "tag", "-m", "for testing", "branch/v2")

	// (4) merge first branch
	runInDir(t, runner, repoDir, "git", "checkout", DefaultBranch)
	runInDir(t, runner, repoDir, "git", "merge", "--squash", "buftest/branch1")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "second commit")
	runInDir(t, runner, repoDir, "git", "tag", "v2")

	// (5) pack some refs
	runInDir(t, runner, repoDir, "git", "pack-refs", "--all")
	runInDir(t, runner, repoDir, "git", "repack")

	// (6) merge second branch
	runInDir(t, runner, repoDir, "git", "checkout", DefaultBranch)
	runInDir(t, runner, repoDir, "git", "merge", "--squash", "buftest/branch2")
	runInDir(t, runner, repoDir, "git", "commit", "-m", "third commit")
	runInDir(t, runner, repoDir, "git", "tag", "v3.0")

	// commit a local-only branch
	runInDir(t, runner, repoDir, "git", "checkout", "-b", "buftest/local-only")
	runInDir(t, runner, repoDir, "git", "commit", "--allow-empty", "-m", "local commit")

	// checkout to default branch
	runInDir(t, runner, repoDir, "git", "checkout", DefaultBranch)

	return repoDir
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

func writeFiles(t *testing.T, dir string, files map[string]string) {
	for path, contents := range files {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dir, path), []byte(contents), 0600))
	}
}
