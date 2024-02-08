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

package bufmodulecache

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestore"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/zap"
)

// NewCommitProvider returns a new CommitProvider that caches the results of the delegate.
//
// The CommitStore is used as a cache.
func NewCommitProvider(
	logger *zap.Logger,
	delegate bufmodule.CommitProvider,
	store bufmodulestore.CommitStore,
) bufmodule.CommitProvider {
	return newCommitProvider(logger, delegate, store)
}

/// *** PRIVATE ***

type commitProvider struct {
	byModuleKey *baseProvider[bufmodule.ModuleKey, bufmodule.Commit]
	byCommitKey *baseProvider[bufmodule.CommitKey, bufmodule.Commit]
}

func newCommitProvider(
	logger *zap.Logger,
	delegate bufmodule.CommitProvider,
	store bufmodulestore.CommitStore,
) *commitProvider {
	return &commitProvider{
		byModuleKey: newBaseProvider(
			logger,
			delegate.GetCommitsForModuleKeys,
			store.GetCommitsForModuleKeys,
			store.PutCommits,
			bufmodule.ModuleKey.CommitID,
			func(commit bufmodule.Commit) uuid.UUID {
				return commit.ModuleKey().CommitID()
			},
		),
		byCommitKey: newBaseProvider(
			logger,
			delegate.GetCommitsForCommitKeys,
			store.GetCommitsForCommitKeys,
			store.PutCommits,
			bufmodule.CommitKey.CommitID,
			func(commit bufmodule.Commit) uuid.UUID {
				return commit.ModuleKey().CommitID()
			},
		),
	}
}

func (p *commitProvider) GetCommitsForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.Commit, error) {
	return p.byModuleKey.getValuesForKeys(ctx, moduleKeys)
}

func (p *commitProvider) GetCommitsForCommitKeys(
	ctx context.Context,
	commitKeys []bufmodule.CommitKey,
) ([]bufmodule.Commit, error) {
	return p.byCommitKey.getValuesForKeys(ctx, commitKeys)
}
