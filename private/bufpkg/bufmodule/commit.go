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

package bufmodule

import (
	"time"

	"github.com/bufbuild/buf/private/pkg/syncext"
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
) Commit {
	return newCommit(
		moduleKey,
		getCreateTime,
		options...,
	)
}

// CommitOption is an option for a new Commit.
type CommitOption func(*commitOptions)

// CommitWithExpectedDigest returns a new CommitOption that will compare the Digest
// of the ModuleKey provided at construction with this digest whenever any lazy method is called.
// If the digests do not match, an error is returned
//
// This is used in situations where we have a Digest from our read location (such as the BSR
// or the cache), and we want to compare it with a ModuleKey we were provided from a local location.
func CommitWithExpectedDigest(expectedDigest Digest) CommitOption {
	return func(commitOptions *commitOptions) {
		commitOptions.expectedDigest = expectedDigest
	}
}

// *** PRIVATE ***

type commit struct {
	moduleKey     ModuleKey
	getCreateTime func() (time.Time, error)
}

func newCommit(
	moduleKey ModuleKey,
	getCreateTime func() (time.Time, error),
	options ...CommitOption,
) *commit {
	commitOptions := newCommitOptions()
	for _, option := range options {
		option(commitOptions)
	}
	if commitOptions.expectedDigest != nil {
		// We need to preserve this, as if we do not, the new value for moduleKey
		// will end up recursively calling itself when moduleKey.Digest() is called
		// in the anonymous function. We could just extract moduleKeyDigestFunc := moduleKey.Digest
		// and call that, but we make a variable to reference the original ModuleKey just for constency.
		originalModuleKey := moduleKey
		moduleKey = newModuleKeyNoValidate(
			originalModuleKey.ModuleFullName(),
			originalModuleKey.CommitID(),
			func() (Digest, error) {
				moduleKeyDigest, err := originalModuleKey.Digest()
				if err != nil {
					return nil, err
				}
				if !DigestEqual(commitOptions.expectedDigest, moduleKeyDigest) {
					return nil, &DigestMismatchError{
						ModuleFullName: originalModuleKey.ModuleFullName(),
						CommitID:       originalModuleKey.CommitID(),
						ExpectedDigest: commitOptions.expectedDigest,
						ActualDigest:   moduleKeyDigest,
					}
				}
				return moduleKeyDigest, nil
			},
		)
	}
	return &commit{
		moduleKey:     moduleKey,
		getCreateTime: syncext.OnceValues(getCreateTime),
	}
}

func (c *commit) ModuleKey() ModuleKey {
	return c.moduleKey
}

func (c *commit) CreateTime() (time.Time, error) {
	// This may invoke tamper-proofing per newCommit construction.
	if _, err := c.moduleKey.Digest(); err != nil {
		return time.Time{}, err
	}
	return c.getCreateTime()
}

func (*commit) isCommit() {}

type commitOptions struct {
	expectedDigest Digest
}

func newCommitOptions() *commitOptions {
	return &commitOptions{}
}
