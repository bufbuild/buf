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

package bufpolicycache

import (
	"context"
	"log/slog"
	"sync/atomic"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicystore"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

// NewPolicyDataProvider returns a new PolicyDataProvider that caches the results of the delegate.
//
// The PolicyDataStore is used as a cache.
func NewPolicyDataProvider(
	logger *slog.Logger,
	delegate bufpolicy.PolicyDataProvider,
	store bufpolicystore.PolicyDataStore,
) bufpolicy.PolicyDataProvider {
	return newPolicyDataProvider(logger, delegate, store)
}

/// *** PRIVATE ***

type policyDataProvider struct {
	logger   *slog.Logger
	delegate bufpolicy.PolicyDataProvider
	store    bufpolicystore.PolicyDataStore

	keysRetrieved atomic.Int64
	keysHit       atomic.Int64
}

func newPolicyDataProvider(
	logger *slog.Logger,
	delegate bufpolicy.PolicyDataProvider,
	store bufpolicystore.PolicyDataStore,
) *policyDataProvider {
	return &policyDataProvider{
		logger:   logger,
		delegate: delegate,
		store:    store,
	}
}

func (p *policyDataProvider) GetPolicyDatasForPolicyKeys(
	ctx context.Context,
	policyKeys []bufpolicy.PolicyKey,
) ([]bufpolicy.PolicyData, error) {
	foundValues, notFoundKeys, err := p.store.GetPolicyDatasForPolicyKeys(ctx, policyKeys)
	if err != nil {
		return nil, err
	}

	delegateValues, err := p.delegate.GetPolicyDatasForPolicyKeys(ctx, notFoundKeys)
	if err != nil {
		return nil, err
	}
	if err := p.store.PutPolicyDatas(ctx, delegateValues); err != nil {
		return nil, err
	}

	p.keysRetrieved.Add(int64(len(policyKeys)))
	p.keysHit.Add(int64(len(foundValues)))

	commitIDToIndexedKey, err := xslices.ToUniqueIndexedValuesMap(
		policyKeys,
		func(policyKey bufpolicy.PolicyKey) uuid.UUID {
			return policyKey.CommitID()
		},
	)
	if err != nil {
		return nil, err
	}
	indexedValues, err := xslices.MapError(
		append(foundValues, delegateValues...),
		func(value bufpolicy.PolicyData) (xslices.Indexed[bufpolicy.PolicyData], error) {
			commitID := value.PolicyKey().CommitID()
			indexedKey, ok := commitIDToIndexedKey[commitID]
			if !ok {
				return xslices.Indexed[bufpolicy.PolicyData]{}, syserror.Newf("did not get value from store with commitID %q", uuidutil.ToDashless(commitID))
			}
			return xslices.Indexed[bufpolicy.PolicyData]{
				Value: value,
				Index: indexedKey.Index,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return xslices.IndexedToSortedValues(indexedValues), nil
}
