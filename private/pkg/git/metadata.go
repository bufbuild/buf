// Copyright 2020-2024 Buf Technologies, Inc.
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

package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/bufbuild/buf/private/pkg/command"
)

const (
	gitCommand      = "git"
	gitOriginRemote = "origin"
	tagsPrefix      = "refs/tags/"
	headsPrefix     = "refs/heads/"
)

// CheckForUncommittedGitChanges checks if there are any uncommitted and/or unchecked
// changes from git based on the given directory.
func CheckForUncommittedGitChanges(
	ctx context.Context,
	runner command.Runner,
	dir string,
) ([]string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	var modifiedFiles []string
	// Unstaged changes
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("diff", "--name-only"),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
	); err != nil {
		return nil, err
	}
	modifiedFiles = append(modifiedFiles, getAllTrimmedLinesFromBuffer(stdout)...)

	stdout = bytes.NewBuffer(nil)
	stderr = bytes.NewBuffer(nil)
	// Staged changes
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("diff", "--name-only", "--cached"),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
	); err != nil {
		return nil, err
	}

	modifiedFiles = append(modifiedFiles, getAllTrimmedLinesFromBuffer(stdout)...)
	return modifiedFiles, nil
}

// GetGitRemotes returns all git remotes based on the given directory.
func GetGitRemotes(
	ctx context.Context,
	runner command.Runner,
	dir string,
) ([]string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("remote"),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
	); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	var remotes []string
	for scanner.Scan() {
		remotes = append(remotes, strings.TrimSpace(scanner.Text()))
	}
	return remotes, nil
}

// GetCurrentHEADGitCommit returns the current HEAD commit based on the given directory.
func GetCurrentHEADGitCommit(
	ctx context.Context,
	runner command.Runner,
	dir string,
) (string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("rev-parse", "HEAD"),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
	); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetRefsForGitCommitAndRemote returns all refs pointing to a given commit based on the
// given remote for the given directory.
func GetRefsForGitCommitAndRemote(
	ctx context.Context,
	runner command.Runner,
	dir string,
	remote string,
	gitCommitSha string,
) ([]string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("ls-remote", "--heads", "--tags", "--refs", remote),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
	); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	var refs []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if ref, found := strings.CutPrefix(line, gitCommitSha); found {
			ref = strings.TrimSpace(ref)
			if tag, isTag := strings.CutPrefix(ref, tagsPrefix); isTag {
				refs = append(refs, tag)
				continue
			}
			if branch, isBranchHead := strings.CutPrefix(ref, headsPrefix); isBranchHead {
				refs = append(refs, branch)
			}
		}
	}
	return refs, nil
}

// GetRemoteHEADBranch returns the HEAD branch based on the given remote and given
// directory. Querying the remote for the HEAD branch requires the passing the
// environment for permissions.
func GetRemoteHEADBranch(
	ctx context.Context,
	runner command.Runner,
	env map[string]string,
	dir string,
	remote string,
) (string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	if err := runner.Run(
		ctx,
		gitCommand,
		command.RunWithArgs("remote", "show", remote),
		command.RunWithStdout(stdout),
		command.RunWithStderr(stderr),
		command.RunWithDir(dir),
		command.RunWithEnv(env),
	); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if branch, isHEADBranch := strings.CutPrefix(line, "HEAD branch: "); isHEADBranch {
			return branch, nil
		}
	}
	return "", errors.New("no HEAD branch information found")
}

func getAllTrimmedLinesFromBuffer(buffer *bytes.Buffer) []string {
	scanner := bufio.NewScanner(buffer)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}
	return lines
}
