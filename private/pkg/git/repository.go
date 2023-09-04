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

package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/filepathextended"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type openRepositoryOpts struct {
	defaultBranch string
}

type repository struct {
	gitDirPath       string
	defaultBranch    string
	checkedOutBranch string
	objectReader     *objectReader

	// packedOnce controls the fields below related to reading the `packed-refs` file
	packedOnce      sync.Once
	packedReadError error
	packedBranches  map[string]Hash
	packedTags      map[string]Hash
}

func openGitRepository(
	ctx context.Context,
	gitDirPath string,
	runner command.Runner,
	options ...OpenRepositoryOption,
) (Repository, error) {
	opts := &openRepositoryOpts{}
	for _, opt := range options {
		if err := opt(opts); err != nil {
			return nil, err
		}
	}
	gitDirPath = normalpath.Unnormalize(gitDirPath)
	if err := validateDirPathExists(gitDirPath); err != nil {
		return nil, err
	}
	gitDirPath, err := filepath.Abs(gitDirPath)
	if err != nil {
		return nil, err
	}
	reader, err := newObjectReader(gitDirPath, runner)
	if err != nil {
		return nil, err
	}
	if opts.defaultBranch == "" {
		opts.defaultBranch, err = detectDefaultBranch(gitDirPath)
		if err != nil {
			return nil, fmt.Errorf("automatically determine default branch: %w", err)
		}
	}
	checkedOutBranch, err := detectCheckedOutBranch(ctx, gitDirPath, runner)
	if err != nil {
		return nil, fmt.Errorf("automatically determine checked out branch: %w", err)
	}
	return &repository{
		gitDirPath:       gitDirPath,
		defaultBranch:    opts.defaultBranch,
		checkedOutBranch: checkedOutBranch,
		objectReader:     reader,
	}, nil
}

func (r *repository) Close() error {
	return r.objectReader.close()
}

func (r *repository) Objects() ObjectReader {
	return r.objectReader
}

func (r *repository) ForEachBranch(f func(string, Hash) error) error {
	unpackedBranches := make(map[string]struct{})
	// Read unpacked branch refs.
	dir := path.Join(r.gitDirPath, "refs", "heads")
	if err := filepathextended.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "HEAD" || info.IsDir() {
			return nil
		}
		branchName, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		branchName = normalpath.Normalize(branchName)
		hashBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hashBytes = bytes.TrimSuffix(hashBytes, []byte{'\n'})
		hash, err := parseHashFromHex(string(hashBytes))
		if err != nil {
			return err
		}
		unpackedBranches[branchName] = struct{}{}
		return f(branchName, hash)
	}); err != nil {
		return err
	}
	// Read packed branch refs that haven't been seen yet.
	if err := r.readPackedRefs(); err != nil {
		return err
	}
	for branchName, hash := range r.packedBranches {
		if _, seen := unpackedBranches[branchName]; !seen {
			if err := f(branchName, hash); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *repository) DefaultBranch() string {
	return r.defaultBranch
}

func (r *repository) CurrentBranch() string {
	return r.checkedOutBranch
}

func (r *repository) ForEachCommit(f func(Commit) error, options ...ForEachCommitOption) error {
	var config forEachCommitOpts
	for _, option := range options {
		if err := option(&config); err != nil {
			return err
		}
	}
	var startCommit Commit
	if config.start == nil {
		// if no custom start point is set, use HEAD from the default branch
		var err error
		startCommit, err = r.HEADCommit(r.DefaultBranch())
		if err != nil {
			return fmt.Errorf("get head commit for default branch %q: %w", r.DefaultBranch(), err)
		}
	} else {
		switch config.start.refType {
		case refTypeHash:
			commitID, err := NewHashFromHex(config.start.refName)
			if err != nil {
				return fmt.Errorf("new hash from %s: %w", config.start.refName, err)
			}
			startCommit, err = r.objectReader.Commit(commitID)
			if err != nil {
				return fmt.Errorf("read commit %s: %w", commitID.Hex(), err)
			}
		case refTypeBranch:
			branch := normalpath.Unnormalize(config.start.refName)
			var err error
			startCommit, err = r.HEADCommit(branch)
			if err != nil {
				return fmt.Errorf("read HEAD commit for branch %q: %w", branch, err)
			}
		default:
			return fmt.Errorf("unsupported start point reference type %s:%s", config.start.refType, config.start.refName)
		}
	}
	currentCommit := startCommit
	for {
		if err := f(currentCommit); err != nil {
			return err
		}
		if len(currentCommit.Parents()) == 0 {
			// We've reach the root of the graph.
			return nil
		}
		// When traversing a commit graph, follow only the first parent commit upon seeing a
		// merge commit. This allows us to ignore the individual commits brought in to a branch's
		// history by such a merge, as those commits are usually updating the state of the target
		// branch.
		nextCommitHash := currentCommit.Parents()[0]
		var err error
		currentCommit, err = r.objectReader.Commit(nextCommitHash)
		if err != nil {
			return fmt.Errorf("read commit %s: %w", nextCommitHash, err)
		}
	}
}

func (r *repository) ForEachTag(f func(string, Hash) error) error {
	seen := map[string]struct{}{}
	// Read unpacked tag refs.
	dir := path.Join(r.gitDirPath, "refs", "tags")
	if err := filepathextended.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		tagName, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		tagName = normalpath.Normalize(tagName)
		hashBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hashBytes = bytes.TrimSuffix(hashBytes, []byte{'\n'})
		hash, err := parseHashFromHex(string(hashBytes))
		if err != nil {
			return err
		}
		// Tags are either annotated or lightweight. Depending on the type,
		// they are stored differently. First, we try to load the tag
		// as an annnotated tag. If this fails, we try a commit.
		// Finally, we fail.
		tag, err := r.objectReader.Tag(hash)
		if err == nil {
			seen[tagName] = struct{}{}
			return f(tagName, tag.Commit())
		}
		if !errors.Is(err, errObjectTypeMismatch) {
			return err
		}
		_, err = r.objectReader.Commit(hash)
		if err == nil {
			seen[tagName] = struct{}{}
			return f(tagName, hash)
		}
		if !errors.Is(err, errObjectTypeMismatch) {
			return err
		}
		return fmt.Errorf(
			"failed to determine target of tag %q; it is neither a tag nor a commit",
			tagName,
		)
	}); err != nil {
		return err
	}
	// Read packed tag refs that haven't been seen yet.
	if err := r.readPackedRefs(); err != nil {
		return err
	}
	for tagName, commit := range r.packedTags {
		if _, found := seen[tagName]; !found {
			if err := f(tagName, commit); err != nil {
				return err
			}
		}
	}
	return nil
}

// HEADCommit resolves the HEAD commit for a branch.
func (r *repository) HEADCommit(branch string) (Commit, error) {
	commitBytes, err := os.ReadFile(path.Join(r.gitDirPath, "refs", "heads", branch))
	if errors.Is(err, fs.ErrNotExist) {
		// it may be that the branch ref is packed; let's read the packed refs
		if err := r.readPackedRefs(); err != nil {
			return nil, err
		}
		if commitID, ok := r.packedBranches[branch]; ok {
			commit, err := r.objectReader.Commit(commitID)
			if err != nil {
				return nil, err
			}
			return commit, nil
		}
		return nil, fmt.Errorf("branch %q not found", branch)
	}
	if err != nil {
		return nil, err
	}
	commitBytes = bytes.TrimRight(commitBytes, "\n")
	commitID, err := NewHashFromHex(string(commitBytes))
	if err != nil {
		return nil, err
	}
	commit, err := r.objectReader.Commit(commitID)
	if err != nil {
		return nil, err
	}
	return commit, nil
}

func (r *repository) readPackedRefs() error {
	r.packedOnce.Do(func() {
		packedRefsPath := path.Join(r.gitDirPath, "packed-refs")
		if _, err := os.Stat(packedRefsPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				r.packedBranches = map[string]Hash{}
				r.packedTags = map[string]Hash{}
				return
			}
			r.packedReadError = err
			return
		}
		allBytes, err := os.ReadFile(packedRefsPath)
		if err != nil {
			r.packedReadError = err
			return
		}
		r.packedBranches, r.packedTags, r.packedReadError = parsePackedRefs(allBytes)
	})
	return r.packedReadError
}

// detectDefaultBranch returns the repository's default branch name. It attempts to read it from the
// `.git/refs/remotes/origin/HEAD` file, and expects it to be pointing to a branch also in the
// `origin` remote.
func detectDefaultBranch(gitDirPath string) (string, error) {
	const defaultRemoteName = "origin"
	path := path.Join(gitDirPath, "refs", "remotes", defaultRemoteName, "HEAD")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var defaultBranchRefPrefix = []byte("ref: refs/remotes/" + defaultRemoteName + "/")
	if !bytes.HasPrefix(data, defaultBranchRefPrefix) {
		return "", errors.New("invalid contents in " + path)
	}
	data = bytes.TrimPrefix(data, defaultBranchRefPrefix)
	data = bytes.TrimSuffix(data, []byte("\n"))
	return string(data), nil
}

func detectCheckedOutBranch(ctx context.Context, gitDirPath string, runner command.Runner) (string, error) {
	var (
		stdOutBuffer = bytes.NewBuffer(nil)
		stdErrBuffer = bytes.NewBuffer(nil)
	)
	if err := runner.Run(
		ctx,
		"git",
		command.RunWithArgs(
			"rev-parse",
			"--abbrev-ref",
			"HEAD",
		),
		command.RunWithStdout(stdOutBuffer),
		command.RunWithStderr(stdErrBuffer),
		command.RunWithDir(gitDirPath), // exec command at the root of the git repo
	); err != nil {
		stdErrMsg, err := io.ReadAll(stdErrBuffer)
		if err != nil {
			stdErrMsg = []byte(fmt.Sprintf("read stderr: %s", err.Error()))
		}
		return "", fmt.Errorf("git rev-parse: %w (%s)", err, string(stdErrMsg))
	}
	stdOut, err := io.ReadAll(stdOutBuffer)
	if err != nil {
		return "", fmt.Errorf("read current branch: %w", err)
	}
	currentBranch := string(bytes.TrimSuffix(stdOut, []byte("\n")))
	if currentBranch == "" {
		return "", errors.New("empty current branch")
	}
	if currentBranch == "HEAD" {
		return "", errors.New("no current branch, git HEAD is detached")
	}
	return currentBranch, nil
}

// validateDirPathExists returns a non-nil error if the given dirPath is not a valid directory path.
func validateDirPathExists(dirPath string) error {
	var fileInfo os.FileInfo
	// We do not follow symlinks
	fileInfo, err := os.Lstat(dirPath)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return normalpath.NewError(dirPath, errors.New("not a directory"))
	}
	return nil
}
