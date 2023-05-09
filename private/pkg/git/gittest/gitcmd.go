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
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/pkg/command"
)

type gitCmdOpts struct {
	baredir string
	indir   string
}

func (opts *gitCmdOpts) Validate() error {
	if opts.baredir != "" && opts.indir != "" {
		return errors.New("Init and InitBare are mutually exclusive")
	}
	if opts.baredir == "" && opts.indir == "" {
		return errors.New("either Init or InitBare must be provided")
	}
	return nil
}

type GitCmdOption interface {
	apply(*gitCmdOpts)
}

type baredir string

func (d baredir) apply(opts *gitCmdOpts) {
	opts.baredir = string(d)
}

// GitCmdInitBare specifies creating a bare git repository.
// Tests using git commands expecting a working tree should use GitCmdInit.
func GitCmdInitBare(dir string) GitCmdOption {
	return baredir(dir)
}

type indir string

func (d indir) apply(opts *gitCmdOpts) {
	opts.indir = string(d)
}

// GitCmdInit specifies creating a git repository with a working tree.
func GitCmdInit(dir string) GitCmdOption {
	return indir(dir)
}

// GitCmd wraps calling out to the host's "git" in a bare repository or one with
// a working dir.
//
// The style is oriented for testing. Repository discovery is prevented to
// isolate tests. Git commands that fail are considered fatal errors and passed
// to testing.T to simplify reading the test's setup.
type GitCmd struct {
	t       *testing.T
	runner  command.Runner
	gitdir  string
	workdir string
	timeout time.Duration
	env     map[string]string
}

// NewGitCmd creates a [GitCmd]. It will call `git init` to initialize the
// repository in accordance with the options provided.
func NewGitCmd(
	t *testing.T,
	runner command.Runner,
	options ...GitCmdOption,
) *GitCmd {
	var opts gitCmdOpts
	for _, option := range options {
		option.apply(&opts)
	}
	if err := opts.Validate(); err != nil {
		t.Fatalf("NewGitCmd: %s", err)
	}
	git := &GitCmd{
		t:       t,
		runner:  runner,
		timeout: 5 * time.Second,
	}
	if opts.indir != "" {
		git.gitdir = filepath.Join(opts.indir, ".git")
		git.workdir = opts.indir
		git.Cmd("init")
	} else if opts.baredir != "" {
		git.gitdir = opts.baredir
		git.Cmd("init", "--bare")
	}
	return git
}

// Env derives this [GitCmd] into a new one with the specified environment.
func (g *GitCmd) Env(env map[string]string) *GitCmd {
	git := *g
	git.env = env
	return &git
}

// Cmd executes a git command, returning standard out as a string.
// If the command fails, the test is fatally aborted.
func (g *GitCmd) Cmd(args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	var stdout strings.Builder
	err := g.runner.Run(ctx,
		"git",
		command.RunWithArgs(args...),
		command.RunWithEnv(g.cmdEnv()),
		command.RunWithStdout(io.MultiWriter(&stdout, os.Stdout)),
		command.RunWithStderr(os.Stderr),
	)
	if err != nil {
		argStr := strings.Join(args, " ")
		g.t.Fatalf("`git %s`: %s", argStr, err)
	}
	return stdout.String()
}

// cmdEnv returns default git environment variables and appends in the
// user's environment variables.
func (g *GitCmd) cmdEnv() map[string]string {
	env := make(map[string]string)
	if g.gitdir != "" {
		env["GIT_DIR"] = g.gitdir
	}
	if g.workdir != "" {
		env["GIT_WORK_TREE"] = g.workdir
	}
	for k, v := range g.env {
		env[k] = v
	}
	return env
}
