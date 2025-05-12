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

package bufmodulecache

import (
	"context"
	"log/slog"
	"sync/atomic"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

type baseProvider[K any, V any] struct {
	logger                   *slog.Logger
	delegateGetValuesForKeys func(context.Context, []K) ([]V, error)
	storeGetValuesForKeys    func(context.Context, []K) ([]V, []K, error)
	storePutValues           func(context.Context, []V) error
	keyToCommitID            func(K) uuid.UUID
	valueToCommitID          func(V) uuid.UUID

	keysRetrieved atomic.Int64
	keysHit       atomic.Int64
}

func newBaseProvider[K any, V any](
	logger *slog.Logger,
	delegateGetValuesForKeys func(context.Context, []K) ([]V, error),
	storeGetValuesForKeys func(context.Context, []K) ([]V, []K, error),
	storePutValues func(context.Context, []V) error,
	keyToCommitID func(K) uuid.UUID,
	valueToCommitID func(V) uuid.UUID,
) *baseProvider[K, V] {
	return &baseProvider[K, V]{
		logger:                   logger,
		delegateGetValuesForKeys: delegateGetValuesForKeys,
		storeGetValuesForKeys:    storeGetValuesForKeys,
		storePutValues:           storePutValues,
		keyToCommitID:            keyToCommitID,
		valueToCommitID:          valueToCommitID,
	}
}

func (p *baseProvider[K, V]) getValuesForKeys(ctx context.Context, keys []K) ([]V, error) {
	commitIDToIndexedKey, err := xslices.ToUniqueIndexedValuesMap(
		keys,
		p.keyToCommitID,
	)
	if err != nil {
		return nil, err
	}
	foundValues, notFoundKeys, err := p.storeGetValuesForKeys(ctx, keys)
	if err != nil {
		return nil, err
	}
	delegateValues, err := p.delegateGetValuesForKeys(
		ctx,
		notFoundKeys,
	)
	if err != nil {
		return nil, err
	}
	if err := p.storePutValues(
		ctx,
		delegateValues,
	); err != nil {
		return nil, err
	}
	// We are getting the values again so that we retrieve the values from the cache directly.
	// This matters for ie ModuleDatas where the storage.Bucket attached will have local paths
	// instead of empty local paths if read from the cache. We document NewModuleDataProvider
	// to return a ModuleDataProvider that will always have local paths for returned storage.Buckets,
	// if the cache is an on-disk cache.
	var delegateNotFoundKeys []K
	delegateValues, delegateNotFoundKeys, err = p.storeGetValuesForKeys(ctx, notFoundKeys)
	if err != nil {
		return nil, err
	}
	// We need to ensure that all the delegate values can be retrieved from the store. If there
	// are unfound keys, we return an error.
	if len(delegateNotFoundKeys) > 0 {
		return nil, syserror.Newf(
			"delegate keys %v not found in the store after putting in the store",
			delegateNotFoundKeys,
		)
	}

	p.keysRetrieved.Add(int64(len(keys)))
	p.keysHit.Add(int64(len(foundValues)))

	indexedValues, err := xslices.MapError(
		append(foundValues, delegateValues...),
		func(value V) (xslices.Indexed[V], error) {
			commitID := p.valueToCommitID(value)
			indexedKey, ok := commitIDToIndexedKey[commitID]
			if !ok {
				return xslices.Indexed[V]{}, syserror.Newf("did not get value from store with commitID %q", uuidutil.ToDashless(commitID))
			}
			return xslices.Indexed[V]{
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

func (p *baseProvider[K, V]) getKeysRetrieved() int {
	return int(p.keysRetrieved.Load())
}

func (p *baseProvider[K, V]) getKeysHit() int {
	return int(p.keysHit.Load())
}
