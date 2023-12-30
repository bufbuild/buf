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

package bufmodulecache

import (
	"context"
	"sort"
	"sync/atomic"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestore"
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
	logger   *zap.Logger
	delegate bufmodule.CommitProvider
	store    bufmodulestore.CommitStore

	moduleKeysRetrieved atomic.Int64
	moduleKeysHit       atomic.Int64
}

func newCommitProvider(
	logger *zap.Logger,
	delegate bufmodule.CommitProvider,
	store bufmodulestore.CommitStore,
) *commitProvider {
	return &commitProvider{
		logger:   logger,
		delegate: delegate,
		store:    store,
	}
}

func (p *commitProvider) GetCommitsForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.Commit, error) {
	foundCommits, notFoundModuleKeys, err := p.store.GetCommitsForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	delegateCommits, err := p.delegate.GetCommitsForModuleKeys(
		ctx,
		notFoundModuleKeys,
	)
	if err != nil {
		return nil, err
	}
	if err := p.store.PutCommits(
		ctx,
		delegateCommits,
	); err != nil {
		return nil, err
	}

	p.moduleKeysRetrieved.Add(int64(len(moduleKeys)))
	p.moduleKeysHit.Add(int64(len(foundCommits)))

	commits := append(foundCommits, delegateCommits...)
	sort.Slice(
		commits,
		func(i int, j int) bool {
			return commits[i].ModuleKey().ModuleFullName().String() < commits[j].ModuleKey().ModuleFullName().String()
		},
	)
	return commits, nil
}

func (p *commitProvider) getModuleKeysRetrieved() int {
	return int(p.moduleKeysRetrieved.Load())
}

func (p *commitProvider) getModuleKeysHit() int {
	return int(p.moduleKeysHit.Load())
}
