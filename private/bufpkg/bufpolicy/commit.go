// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufpolicy

import (
	"sync"
	"time"
)

// Commit represents a Commit for a Policy on the BSR.
type Commit interface {
	// PolicyKey returns the PolicyKey for the Commit.
	PolicyKey() PolicyKey
	// CreateTime returns the time the Commit was created on the BSR.
	CreateTime() (time.Time, error)

	isCommit()
}

// NewCommit returns a new Commit.
func NewCommit(
	policyKey PolicyKey,
	getCreateTime func() (time.Time, error),
) Commit {
	return newCommit(
		policyKey,
		getCreateTime,
	)
}

// *** PRIVATE ***

type commit struct {
	policyKey     PolicyKey
	getCreateTime func() (time.Time, error)
}

func newCommit(
	policyKey PolicyKey,
	getCreateTime func() (time.Time, error),
) *commit {
	return &commit{
		policyKey:     policyKey,
		getCreateTime: sync.OnceValues(getCreateTime),
	}
}

func (c *commit) PolicyKey() PolicyKey {
	return c.policyKey
}

func (c *commit) CreateTime() (time.Time, error) {
	// This may invoke tamper-proofing per newCommit construction.
	if _, err := c.policyKey.Digest(); err != nil {
		return time.Time{}, err
	}
	return c.getCreateTime()
}

func (*commit) isCommit() {}
