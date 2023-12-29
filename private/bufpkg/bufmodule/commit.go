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
	"errors"
	"fmt"
	"time"

	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

// Commit represents a Commit on the BSR.
type Commit interface {
	// ID returns the ID of the Commit.
	//
	// A CommitID is always a dashless UUID.
	// The CommitID converted to using dashes is the ID of the Commit on the BSR.
	ID() string
	// ModuleKey returns the ModuleKey for the Commit.
	ModuleKey() (ModuleKey, error)
	// CreateTime returns the time the Commit was created on the BSR.
	CreateTime() (time.Time, error)

	isCommit()
}

// NewCommit returns a new Commit.
func NewCommit(
	id string,
	getModuleKey func() (ModuleKey, error),
	getCreateTime func() (time.Time, error),
) (Commit, error) {
	return newCommit(
		id,
		getModuleKey,
		getCreateTime,
	)
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
	id            string
	getModuleKey  func() (ModuleKey, error)
	getCreateTime func() (time.Time, error)
}

func newCommit(
	id string,
	getModuleKey func() (ModuleKey, error),
	getCreateTime func() (time.Time, error),
) (*commit, error) {
	if id == "" {
		return nil, errors.New("empty commitID when constructing Commit")
	}
	if err := validateCommitID(id); err != nil {
		return nil, err
	}
	return &commit{
		id:            id,
		getModuleKey:  getModuleKey,
		getCreateTime: getCreateTime,
	}, nil
}

func (c *commit) ID() string {
	return c.id
}

func (c *commit) ModuleKey() (ModuleKey, error) {
	return c.getModuleKey()
}

func (c *commit) CreateTime() (time.Time, error) {
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

func validateCommitID(commitID string) error {
	if err := uuidutil.ValidateDashless(commitID); err != nil {
		return fmt.Errorf("invalid commit ID %s: %w", commitID, err)
	}
	return nil
}
