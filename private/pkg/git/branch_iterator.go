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
	"io/fs"
	"path"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/filepathextended"
	"github.com/bufbuild/buf/private/pkg/git/gitobject"
	"github.com/bufbuild/buf/private/pkg/normalpath"
)

type branchIterator struct {
	gitDirPath   string
	objectReader gitobject.Reader
}

func newBranchIterator(
	gitDirPath string,
	objectReader gitobject.Reader,
) (BranchIterator, error) {
	gitDirPath = normalpath.Unnormalize(gitDirPath)
	if err := validateDirPathExists(gitDirPath); err != nil {
		return nil, err
	}
	gitDirPath, err := filepath.Abs(gitDirPath)
	if err != nil {
		return nil, err
	}
	return &branchIterator{
		gitDirPath:   gitDirPath,
		objectReader: objectReader,
	}, nil
}

func (r *branchIterator) ForEachBranch(f func(string) error) error {
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
