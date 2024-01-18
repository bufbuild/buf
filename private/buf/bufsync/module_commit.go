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

package bufsync

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type moduleCommit struct {
	commit    git.Commit
	tags      []string
	getBucket func(ctx context.Context) (storage.ReadBucket, error)
}

func newModuleCommit(
	commit git.Commit,
	tags []string,
	getBucket func(ctx context.Context) (storage.ReadBucket, error),
) *moduleCommit {
	return &moduleCommit{
		commit:    commit,
		tags:      tags,
		getBucket: getBucket,
	}
}

func (m *moduleCommit) Commit() git.Commit {
	return m.commit
}

func (m *moduleCommit) Tags() []string {
	return m.tags
}

func (m *moduleCommit) Bucket(ctx context.Context) (storage.ReadBucket, error) {
	return m.getBucket(ctx)
}

var _ ModuleCommit = (*moduleCommit)(nil)
