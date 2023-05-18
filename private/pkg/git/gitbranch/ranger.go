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

package gitbranch

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/filepathextended"
	"github.com/bufbuild/buf/private/pkg/git/gitobject"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

const defaultRemoteName = "origin"

var baseBranchRefPrefix = []byte("ref: refs/remotes/" + defaultRemoteName + "/")

type rangerOpts struct {
	baseBranch string
}

type ranger struct {
	gitDirPath   string
	baseBranch   string
	objectReader gitobject.Reader
}

func newRanger(
	gitDirPath string,
	objectReader gitobject.Reader,
	opts rangerOpts,
) (Ranger, error) {
	gitDirPath = normalpath.Unnormalize(gitDirPath)
	if err := validateDirPathExists(gitDirPath); err != nil {
		return nil, err
	}
	gitDirPath, err := filepath.Abs(gitDirPath)
	if err != nil {
		return nil, err
	}
	// TODO: do we need to validate `gitDirPath` is not jumping context?
	if opts.baseBranch == "" {
		opts.baseBranch, err = baseBranch(gitDirPath)
		if err != nil {
			return nil, fmt.Errorf("automatically determine base branch: %w", err)
		}
	}
	return &ranger{
		gitDirPath:   gitDirPath,
		baseBranch:   opts.baseBranch,
		objectReader: objectReader,
	}, nil
}

func (r *ranger) BaseBranch() string {
	return r.baseBranch
}

func (r *ranger) Branches(f func(string) error) error {
	dir := path.Join(r.gitDirPath, "refs", "remotes", defaultRemoteName)
	return filepathextended.Walk(dir, func(path string, info fs.FileInfo, err error) error {
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
		return f(branchName)
	})
}

func (r *ranger) Commits(branch string, f func(gitobject.Commit) error) error {
	branch = normalpath.Unnormalize(branch)
	commitBytes, err := os.ReadFile(path.Join(r.gitDirPath, "refs", "remotes", defaultRemoteName, branch))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("branch not found")
		}
		return err
	}
	commitBytes = bytes.TrimRight(commitBytes, "\n")
	commitID, err := gitobject.NewIDFromHex(string(commitBytes))
	if err != nil {
		return err
	}
	commit, err := r.objectReader.Commit(commitID)
	if err != nil {
		return err
	}
	var commits []gitobject.Commit
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

func baseBranch(gitDirPath string) (string, error) {
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
