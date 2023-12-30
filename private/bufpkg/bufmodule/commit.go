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
	// Digest returns the digest of the content of the Commit.
	//
	// This is the Digest as retrieved from the BSR - it relies on the BSR
	// correctly calculating digests.
	//
	// When CreateTime() or other lazy methods are called, this Digest will be checked
	// against the Digest in ModuleKey, and if there is a difference,
	// an error will be returned.
	Digest() (Digest, error)

	isCommit()
}

// NewCommit returns a new Commit.
func NewCommit(
	moduleKey ModuleKey,
	getCreateTime func() (time.Time, error),
	getDigest func() (Digest, error),
) Commit {
	return newCommit(
		moduleKey,
		getCreateTime,
		getDigest,
	)
}

// *** PRIVATE ***

type commit struct {
	moduleKey     ModuleKey
	getCreateTime func() (time.Time, error)
	getDigest     func() (Digest, error)
}

func newCommit(
	moduleKey ModuleKey,
	getCreateTime func() (time.Time, error),
	getDigest func() (Digest, error),
) *commit {
	return &commit{
		moduleKey:     moduleKey,
		getCreateTime: sync.OnceValues(getCreateTime),
		getDigest: sync.OnceValues(
			func() (Digest, error) {
				digest, err := getDigest()
				if err != nil {
					return nil, err
				}
				moduleKeyDigest, err := moduleKey.Digest()
				if err != nil {
					return nil, err
				}
				if !DigestEqual(digest, moduleKeyDigest) {
					return nil, fmt.Errorf(
						"verification failed for commit %s: expected digest %q but downloaded commit had digest %q",
						moduleKey.String(),
						moduleKeyDigest.String(),
						digest.String(),
					)
				}
				return digest, nil
			},
		),
	}
}

func (c *commit) ModuleKey() ModuleKey {
	return c.moduleKey
}

func (c *commit) CreateTime() (time.Time, error) {
	if _, err := c.getDigest(); err != nil {
		return time.Time{}, err
	}
	return c.getCreateTime()
}

func (c *commit) Digest() (Digest, error) {
	return c.getDigest()
}

func (*commit) isCommit() {}

func validateCommitID(commitID string) error {
	if err := uuidutil.ValidateDashless(commitID); err != nil {
		return fmt.Errorf("invalid commit ID %s: %w", commitID, err)
	}
	return nil
}
