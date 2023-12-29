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
	"fmt"
	"sync/atomic"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestore"
	"github.com/bufbuild/buf/private/pkg/slicesext"
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

// There has to be some way to make this common with moduleDataProvider, but it's too complicated
// for now. The optional types not being generic doesn't help. But we should be able to find
// commonality.

func (p *commitProvider) GetOptionalCommitsForModuleKeys(
	ctx context.Context,
	moduleKeys ...bufmodule.ModuleKey,
) ([]bufmodule.OptionalCommit, error) {
	cachedOptionalCommits, err := p.store.GetOptionalCommitsForModuleKeys(ctx, moduleKeys...)
	if err != nil {
		return nil, err
	}
	resultOptionalCommits := make([]bufmodule.OptionalCommit, len(moduleKeys))
	// The indexes within moduleKeys of the ModuleKeys that did not have a cached Commit.
	// We will then fetch these specific ModuleKeys in one shot from the delegate.
	var missedModuleKeysIndexes []int
	for i, cachedOptionalCommit := range cachedOptionalCommits {
		p.logDebugModuleKey(
			moduleKeys[i],
			"module commits cache get",
			zap.Bool("found", cachedOptionalCommit.Found()),
		)
		if cachedOptionalCommit.Found() {
			// We put the cached Commit at the specific location it is expected to be returned,
			// given that the returned Commit order must match the input ModuleKey order.
			resultOptionalCommits[i] = cachedOptionalCommit
		} else {
			missedModuleKeysIndexes = append(missedModuleKeysIndexes, i)
		}
	}
	if len(missedModuleKeysIndexes) > 0 {
		missedOptionalCommits, err := p.delegate.GetOptionalCommitsForModuleKeys(
			ctx,
			// Map the indexes of to the actual ModuleKeys.
			slicesext.Map(
				missedModuleKeysIndexes,
				func(i int) bufmodule.ModuleKey { return moduleKeys[i] },
			)...,
		)
		if err != nil {
			return nil, err
		}
		// Just a sanity check.
		if len(missedOptionalCommits) != len(missedModuleKeysIndexes) {
			return nil, fmt.Errorf(
				"expected %d Commits, got %d",
				len(missedModuleKeysIndexes),
				len(missedOptionalCommits),
			)
		}
		// Put the found Commits into the store.
		if err := p.store.PutCommits(
			ctx,
			slicesext.Map(
				// Get just the OptionalCommits that were found.
				slicesext.Filter(
					missedOptionalCommits,
					func(optionalCommit bufmodule.OptionalCommit) bool {
						return optionalCommit.Found()
					},
				),
				// Get found OptionalCommit -> Commit.
				func(optionalCommit bufmodule.OptionalCommit) bufmodule.Commit {
					return optionalCommit.Commit()
				},
			)...,
		); err != nil {
			return nil, err
		}
		for i, missedModuleKeysIndex := range missedModuleKeysIndexes {
			// i is the index within missedOptionalCommits, while missedModuleKeysIndex is the index
			// within missedModuleKeysIndexes, and consequently moduleKeys.
			//
			// Put in the specific location we expect the OptionalCommit to be returned.
			// Put in regardless of whether it was found.
			resultOptionalCommits[missedModuleKeysIndex] = missedOptionalCommits[i]
		}
	}
	p.moduleKeysRetrieved.Add(int64(len(resultOptionalCommits)))
	p.moduleKeysHit.Add(int64(len(resultOptionalCommits) - len(missedModuleKeysIndexes)))
	return resultOptionalCommits, nil
}

func (p *commitProvider) getModuleKeysRetrieved() int {
	return int(p.moduleKeysRetrieved.Load())
}

func (p *commitProvider) getModuleKeysHit() int {
	return int(p.moduleKeysHit.Load())
}

func (p *commitProvider) logDebugModuleKey(moduleKey bufmodule.ModuleKey, message string, fields ...zap.Field) {
	logDebugModuleKey(p.logger, moduleKey, message, fields...)
}
