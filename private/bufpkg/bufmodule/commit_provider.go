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

package bufmodule

import (
	"context"
	"io/fs"
)

// CommitProvider provides Commits for CommitIDs.
type CommitProvider interface {
	// GetCommitsForCommitIDs gets the Commits for the given CommitIDs.
	//
	// Resolution of the CommitIDs is done per the CommitID documentation.
	//
	// If there is no error, the length of the OptionalCommits returned will match the length of the CommitIDs.
	// If there is an error, no OptionalCommits will be returned.
	// If a Commit is not found, the OptionalCommit will have Found() equal to false, otherwise
	// the OptionalCommit will have Found() equal to true with non-nil Commit.
	GetOptionalCommitsForCommitIDs(context.Context, ...string) ([]OptionalCommit, error)
}

// GetCommitsForCommitIDs calls GetOptionalCommitsForCommitIDs, returning an error
// with fs.ErrNotExist if any CommitID is not found.
func GetCommitsForCommitIDs(
	ctx context.Context,
	commitProvider CommitProvider,
	commitIDs ...string,
) ([]Commit, error) {
	optionalCommits, err := commitProvider.GetOptionalCommitsForCommitIDs(ctx, commitIDs...)
	if err != nil {
		return nil, err
	}
	commits := make([]Commit, len(optionalCommits))
	for i, optionalCommit := range optionalCommits {
		if !optionalCommit.Found() {
			return nil, &fs.PathError{Op: "read", Path: commitIDs[i], Err: fs.ErrNotExist}
		}
		commits[i] = optionalCommit.Commit()
	}
	return commits, nil
}
