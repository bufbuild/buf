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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/filepathext"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type openRepositoryOpts struct {
	defaultBranch string
}

type repository struct {
	gitDirPath    string
	defaultBranch string
	objectReader  *objectReader
	runner        command.Runner

	// packedOnce controls the fields below related to reading the `packed-refs` file
	packedOnce      sync.Once
	packedReadError error
	packedBranches  map[string]map[string]Hash // remote:branch:hash (empty remote means local)
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
	return &repository{
		gitDirPath:    gitDirPath,
		defaultBranch: opts.defaultBranch,
		objectReader:  reader,
		runner:        runner,
	}, nil
}

func (r *repository) Close() error {
	return r.objectReader.close()
}

func (r *repository) Objects() ObjectReader {
	return r.objectReader
}

func (r *repository) ForEachBranch(f func(string, Hash) error, options ...ForEachBranchOption) error {
	var config forEachBranchOpts
	for _, option := range options {
		option(&config)
	}
	unpackedBranches := make(map[string]struct{})
	// Read unpacked branch refs.
	var branchesDir string
	if config.remote == "" {
		branchesDir = filepath.Join(r.gitDirPath, "refs", "heads") // all local branches
	} else {
		branchesDir = filepath.Join(r.gitDirPath, "refs", "remotes", normalpath.Unnormalize(config.remote)) // only branches in this remote
	}
	if err := filepathext.Walk(branchesDir, func(branchPath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "HEAD" || info.IsDir() {
			return nil
		}
		branchRelDir, err := filepath.Rel(branchesDir, branchPath)
		if err != nil {
			return err
		}
		branchName := normalpath.Normalize(branchRelDir)
		hashBytes, err := os.ReadFile(branchPath)
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
	}); err != nil && !errors.Is(err, fs.ErrNotExist) {
		if errors.Is(err, ErrStopForEach) {
			return nil
		}
		return err
	}
	// Read packed branch refs that haven't been seen yet.
	if err := r.readPackedRefs(); err != nil {
		return err
	}
	remotePackedBranches, ok := r.packedBranches[config.remote]
	if ok {
		for branchName, hash := range remotePackedBranches {
			if _, seen := unpackedBranches[branchName]; !seen {
				if err := f(branchName, hash); err != nil {
					if errors.Is(err, ErrStopForEach) {
						return nil
					}
					return err
				}
			}
		}
	}
	return nil
}

func (r *repository) DefaultBranch() string {
	return r.defaultBranch
}

func (r *repository) CheckedOutBranch(options ...CheckedOutBranchOption) (string, error) {
	var config checkedOutBranchOpts
	for _, option := range options {
		option(&config)
	}
	headBytes, err := os.ReadFile(filepath.Join(r.gitDirPath, "HEAD"))
	if err != nil {
		return "", fmt.Errorf("read HEAD bytes: %w", err)
	}
	headBytes = bytes.TrimSuffix(headBytes, []byte{'\n'})
	// .git/HEAD could point to a named ref, or could be in a dettached state, ie pointing to a git
	// hash. Possible values:
	//
	// "ref: refs/heads/somelocalbranch"
	// "ref: refs/remotes/someremote/somebranch"
	// "somegithash"
	const refPrefix = "ref: refs/"
	if strings.HasPrefix(string(headBytes), refPrefix) {
		refRelDir := strings.TrimPrefix(string(headBytes), refPrefix)
		if config.remote == "" {
			// only match local branches
			const localBranchPrefix = "heads/"
			if !strings.HasPrefix(refRelDir, localBranchPrefix) {
				return "", fmt.Errorf("git HEAD %s is not pointing to a local branch", string(headBytes))
			}
			return strings.TrimPrefix(refRelDir, localBranchPrefix), nil
		}
		// only match branches from the specific remote
		remoteBranchPrefix := "remotes/" + config.remote + "/"
		if !strings.HasPrefix(refRelDir, remoteBranchPrefix) {
			return "", fmt.Errorf("git HEAD %s is not pointing to branch in remote %s", string(headBytes), config.remote)
		}
		return strings.TrimPrefix(refRelDir, remoteBranchPrefix), nil
	}
	// if HEAD is not a named ref, it could be a dettached HEAD, ie a git hash
	headHash, err := parseHashFromHex(string(headBytes))
	if err != nil {
		return "", fmt.Errorf(".git/HEAD is not a named ref nor a git hash: %w", err)
	}
	// we can compare that hash with all repo branches' heads
	var currentBranch string
	if err := r.ForEachBranch(
		func(branch string, branchHEAD Hash) error {
			if headHash == branchHEAD {
				currentBranch = branch
				return ErrStopForEach
			}
			return nil
		},
		ForEachBranchWithRemote(config.remote),
	); err != nil {
		return "", fmt.Errorf("for each branch: %w", err)
	}
	if currentBranch == "" {
		return "", errors.New("git HEAD is detached, no matches with any branch")
	}
	return currentBranch, nil
}

func (r *repository) ForEachCommit(f func(Commit) error, options ...ForEachCommitOption) error {
	var config forEachCommitOpts
	for _, option := range options {
		option(&config)
	}
	currentCommit, err := r.commitAt(config.start)
	if err != nil {
		return fmt.Errorf("find start commit: %w", err)
	}
	for {
		if err := f(currentCommit); err != nil {
			if errors.Is(err, ErrStopForEach) {
				return nil
			}
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
	tagsDir := filepath.Join(r.gitDirPath, "refs", "tags")
	if err := filepathext.Walk(tagsDir, func(tagPath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		tagRelPath, err := filepath.Rel(tagsDir, tagPath)
		if err != nil {
			return err
		}
		tagName := normalpath.Normalize(tagRelPath)
		hashBytes, err := os.ReadFile(tagPath)
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
		if errors.Is(err, ErrStopForEach) {
			return nil
		}
		return err
	}
	// Read packed tag refs that haven't been seen yet.
	if err := r.readPackedRefs(); err != nil {
		return err
	}
	for tagName, commit := range r.packedTags {
		if _, found := seen[tagName]; !found {
			if err := f(tagName, commit); err != nil {
				if errors.Is(err, ErrStopForEach) {
					return nil
				}
				return err
			}
		}
	}
	return nil
}

func (r *repository) HEADCommit(options ...HEADCommitOption) (Commit, error) {
	var config headCommitOpts
	for _, option := range options {
		option(&config)
	}
	var branch = r.DefaultBranch()
	if config.branch != "" {
		branch = config.branch
	}
	var branchPath string
	if config.remote == "" {
		branchPath = filepath.Join(r.gitDirPath, "refs", "heads", normalpath.Unnormalize(branch))
	} else {
		branchPath = filepath.Join(r.gitDirPath, "refs", "remotes", normalpath.Unnormalize(config.remote), normalpath.Unnormalize(branch))
	}
	commitBytes, err := os.ReadFile(branchPath)
	if errors.Is(err, fs.ErrNotExist) {
		// it may be that the branch ref is packed; let's read the packed refs
		if err := r.readPackedRefs(); err != nil {
			return nil, err
		}
		if remotePackedRefs, ok := r.packedBranches[config.remote]; ok {
			if commitID, ok := remotePackedRefs[branch]; ok {
				commit, err := r.objectReader.Commit(commitID)
				if err != nil {
					return nil, err
				}
				return commit, nil
			}
		}
		return nil, fmt.Errorf("branch %s not found", branch)
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
		packedRefsPath := filepath.Join(r.gitDirPath, "packed-refs")
		if _, err := os.Stat(packedRefsPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				r.packedBranches = make(map[string]map[string]Hash)
				r.packedTags = make(map[string]Hash)
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

// commitAt returns the commit at the passed reference.
func (r *repository) commitAt(ref reference) (Commit, error) {
	if ref == nil {
		// if a ref is not passed, use HEAD with its default behavior.
		commit, err := r.HEADCommit()
		if err != nil {
			return nil, fmt.Errorf("get HEAD commit: %w", err)
		}
		return commit, nil
	}
	if hashRef, ok := ref.(*hashReference); ok {
		commitID, err := NewHashFromHex(hashRef.name)
		if err != nil {
			return nil, fmt.Errorf("new hash from %s: %w", hashRef.name, err)
		}
		commit, err := r.objectReader.Commit(commitID)
		if err != nil {
			return nil, fmt.Errorf("read commit %s: %w", commitID.Hex(), err)
		}
		return commit, nil
	}
	if branchRef, ok := ref.(*branchReference); ok {
		commit, err := r.HEADCommit(
			HEADCommitWithBranch(branchRef.name),
			HEADCommitWithRemote(branchRef.remote),
		)
		if err != nil {
			return nil, fmt.Errorf("read HEAD commit for branch %q: %w", branchRef.refName(), err)
		}
		return commit, nil
	}
	return nil, fmt.Errorf("unsupported reference %s:%s", ref.refType(), ref.refName())
}

// detectDefaultBranch returns the repository's default branch name. It attempts to read it from the
// `.git/refs/remotes/origin/HEAD` file, and expects it to be pointing to a branch also in the
// `origin` remote.
func detectDefaultBranch(gitDirPath string) (string, error) {
	const defaultRemoteName = "origin"
	path := filepath.Join(gitDirPath, "refs", "remotes", defaultRemoteName, "HEAD")
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
