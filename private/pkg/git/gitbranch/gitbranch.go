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
	"errors"

	"github.com/bufbuild/buf/private/pkg/git/gitobject"
)

// Ranger ranges over branches and commits for a git repository.
//
// Only branches and commits from the remote named `origin` are ranged.
type Ranger interface {
	// BaseBranch is the base branch of the repository. This is either
	// configured via the `WithBaseBranch` option, or discovered via the
	// remote named `origin`. Therefore, discovery requires that the repository
	// is pushed to the remote.
	BaseBranch() string
	// Branches ranges over branches in the repository in an undefined order.
	Branches(func(branch string) error) error
	// Commits ranges over commits for the target branch in reverse topological order.
	//
	// Parents are visited before children, and only left parents are visited (i.e.,
	// commits from branches merged into the target branch are not visited).
	Commits(branch string, f func(commit gitobject.Commit) error) error
}

type RangerOption func(*rangerOpts) error

// WithBaseBranch configures the base branch for this ranger.
func WithBaseBranch(name string) RangerOption {
	return func(r *rangerOpts) error {
		if name == "" {
			return errors.New("default branch cannot be empty")
		}
		r.baseBranch = name
		return nil
	}
}

// NewRanger creates a new Ranger that can range over commits and branches.
//
// By default, NewRanger will attempt to detect the base branch if the repository
// has been pushed. This may fail. TODO: we probably want to remove this and
// force the use of the `WithBaseBranch` option.
func NewRanger(
	gitDirPath string,
	objectReader gitobject.Reader,
	options ...RangerOption,
) (Ranger, error) {
	var opts rangerOpts
	for _, option := range options {
		if err := option(&opts); err != nil {
			return nil, err
		}
	}
	return newRanger(gitDirPath, objectReader, opts)
}
