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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/filepathextended"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

const defaultRemoteName = "origin"

var baseBranchRefPrefix = []byte("ref: refs/remotes/" + defaultRemoteName + "/")

type openRepositoryOpts struct {
	baseBranch string
}

type repository struct {
	gitDirPath   string
	baseBranch   string
	objectReader *objectReader

	// packedOnce controls the fields below related to reading the `packed-refs` file
	packedOnce      sync.Once
	packedReadError error
	packedBranches  map[string]Hash
	packedTags      map[string]Hash
}

func openGitRepository(
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
	if opts.baseBranch == "" {
		opts.baseBranch, err = detectBaseBranch(gitDirPath)
		if err != nil {
			return nil, fmt.Errorf("automatically determine base branch: %w", err)
		}
	}
	return &repository{
		gitDirPath:   gitDirPath,
		baseBranch:   opts.baseBranch,
		objectReader: reader,
	}, nil
}

func (r *repository) Close() error {
	return r.objectReader.close()
}

func (r *repository) Objects() ObjectReader {
	return r.objectReader
}

func (r *repository) ForEachBranch(f func(string, Hash) error) error {
	seen := map[string]struct{}{}
	// Read unpacked branch refs.
	dir := path.Join(r.gitDirPath, "refs", "remotes", defaultRemoteName)
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
		seen[branchName] = struct{}{}
		return f(branchName, hash)
	}); err != nil {
		return err
	}
	// Read packed branch refs that haven't been seen yet.
	if err := r.readPackedRefs(); err != nil {
		return err
	}
	for branchName, hash := range r.packedBranches {
		if _, found := seen[branchName]; !found {
			if err := f(branchName, hash); err != nil {
				return err
			}
		}
	}
	return nil
}
func (r *repository) BaseBranch() string {
	return r.baseBranch
}

func (r *repository) ForEachCommit(branch string, f func(Commit) error) error {
	branch = normalpath.Unnormalize(branch)
	commit, err := r.resolveBranch(branch)
	if err != nil {
		return err
	}
	var commits []Commit
	// TODO: this only works for the base branch; for non-base branches,
	// we have to be much more careful about not ranging over commits belonging
	// to other branches (i.e., running past the origin of our branch).
	// In order to do this, we will want to preload the HEADs of all known branches,
	// and halt iteration for a given branch when we encounter the head of another branch.
	for {
		commits = append(commits, commit)
		if len(commit.Parents()) == 0 {
			// We've reach the root of the graph.
			break
		}
		// When traversing a commit graph, follow only the first parent commit upon seeing a
		// merge commit. This allows us to ignore the individual commits brought in to a branch's
		// history by such a merge, as those commits are usually updating the state of the target
		// branch.
		commit, err = r.objectReader.Commit(commit.Parents()[0])
		if err != nil {
			return err
		}
	}
	// Visit in reverse order, starting with the root of the graph first.
	for i := len(commits) - 1; i >= 0; i-- {
		if err := f(commits[i]); err != nil {
			return err
		}
	}
	return nil
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

func (r *repository) resolveBranch(branch string) (Commit, error) {
	commitBytes, err := os.ReadFile(path.Join(r.gitDirPath, "refs", "remotes", defaultRemoteName, branch))
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

func detectBaseBranch(gitDirPath string) (string, error) {
	path := path.Join(gitDirPath, "refs", "remotes", defaultRemoteName, "HEAD")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(data, baseBranchRefPrefix) {
		return "", errors.New("invalid contents in " + path)
	}
	data = bytes.TrimPrefix(data, baseBranchRefPrefix)
	data = bytes.TrimSuffix(data, []byte("\n"))
	return string(data), nil
}

// validateDirPathExists returns a non-nil error if the given dirPath
// is not a valid directory path.
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
