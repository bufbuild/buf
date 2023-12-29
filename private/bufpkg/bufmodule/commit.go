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
	"fmt"
	"sync"
	"time"

	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

// Commit represents a Commit on the BSR.
type Commit interface {
	// ModuleKey returns the ModuleKey for the Commit.
	ModuleKey() ModuleKey
	// CreateTime returns the time the Commit was created on the BSR.
	CreateTime() (time.Time, error)

	isCommit()
}

// NewCommit returns a new Commit.
func NewCommit(
	moduleKey ModuleKey,
	getCreateTime func() (time.Time, error),
	options ...CommitOption,
) (Commit, error) {
	return newCommit(
		moduleKey,
		getCreateTime,
		options...,
	)
}

// CommitOption is an option for a new Commit.
type CommitOption func(*commitOptions)

// CommitWithReceivedDigest returns a new CommitOption that specifies the Digest
// that was received when creating the Commit.
//
// When CreateTime() or other lazy methods are called, if this Digest is specified, it
// will be checked against the Digest in ModuleKey, and if there is a difference,
// an error will be returned.
func CommitWithReceivedDigest(receivedDigest Digest) CommitOption {
	return func(commitOptions *commitOptions) {
		commitOptions.receivedDigest = receivedDigest
	}
}

// OptionalCommit is a result from a CommitProvider.
//
// It returns whether or not the Commit was found, and a non-nil
// Commit if the Commit was found.
type OptionalCommit interface {
	Commit() Commit
	Found() bool

	isOptionalCommit()
}

// NewOptionalCommit returns a new OptionalCommit.
//
// As opposed to most functions in this codebase, the input Commit can be nil.
// If it is nil, then Found() will return false.
func NewOptionalCommit(commit Commit) OptionalCommit {
	return newOptionalCommit(commit)
}

// *** PRIVATE ***

type commit struct {
	moduleKey     ModuleKey
	getCreateTime func() (time.Time, error)

	checkDigest func() error
}

func newCommit(
	moduleKey ModuleKey,
	getCreateTime func() (time.Time, error),
	options ...CommitOption,
) (*commit, error) {
	commitOptions := newCommitOptions()
	for _, option := range options {
		option(commitOptions)
	}
	commit := &commit{
		moduleKey:     moduleKey,
		getCreateTime: sync.OnceValues(getCreateTime),
	}
	if commitOptions.receivedDigest != nil {
		commit.checkDigest = sync.OnceValue(
			func() error {
				digest, err := moduleKey.Digest()
				if err != nil {
					return err
				}
				if !DigestEqual(digest, commitOptions.receivedDigest) {
					return fmt.Errorf(
						"verification failed for commit %s: expected digest %q but downloaded commit had digest %q",
						moduleKey.String(),
						digest.String(),
						commitOptions.receivedDigest.String(),
					)
				}
				return nil
			},
		)
	}
	return commit, nil
}

func (c *commit) ModuleKey() ModuleKey {
	return c.moduleKey
}

func (c *commit) CreateTime() (time.Time, error) {
	if c.checkDigest != nil {
		if err := c.checkDigest(); err != nil {
			return time.Time{}, err
		}
	}
	return c.getCreateTime()
}

func (*commit) isCommit() {}

type optionalCommit struct {
	commit Commit
}

func newOptionalCommit(commit Commit) *optionalCommit {
	return &optionalCommit{
		commit: commit,
	}
}

func (o *optionalCommit) Commit() Commit {
	return o.commit
}

func (o *optionalCommit) Found() bool {
	return o.commit != nil
}

func (*optionalCommit) isOptionalCommit() {}

type commitOptions struct {
	receivedDigest Digest
}

func newCommitOptions() *commitOptions {
	return &commitOptions{}
}

func validateCommitID(commitID string) error {
	if err := uuidutil.ValidateDashless(commitID); err != nil {
		return fmt.Errorf("invalid commit ID %s: %w", commitID, err)
	}
	return nil
}
