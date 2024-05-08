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
	"sync/atomic"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/zap"
)

type baseProvider[K any, V any] struct {
	logger                   *zap.Logger
	delegateGetValuesForKeys func(context.Context, []K) ([]V, error)
	storeGetValuesForKeys    func(context.Context, []K) ([]V, []K, error)
	storePutValues           func(context.Context, []V) error
	keyToCommitID            func(K) uuid.UUID
	valueToCommitID          func(V) uuid.UUID

	keysRetrieved atomic.Int64
	keysHit       atomic.Int64
}

func newBaseProvider[K any, V any](
	logger *zap.Logger,
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
	commitIDToIndexedKey, err := slicesext.ToUniqueIndexedValuesMap(
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

	p.keysRetrieved.Add(int64(len(keys)))
	p.keysHit.Add(int64(len(foundValues)))

	indexedValues, err := slicesext.MapError(
		append(foundValues, delegateValues...),
		func(value V) (slicesext.Indexed[V], error) {
			commitID := p.valueToCommitID(value)
			indexedKey, ok := commitIDToIndexedKey[commitID]
			if !ok {
				return slicesext.Indexed[V]{}, syserror.Newf("did not get value from store with commitID %q", uuidutil.ToDashless(commitID))
			}
			return slicesext.Indexed[V]{
				Value: value,
				Index: indexedKey.Index,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return slicesext.IndexedToSortedValues(indexedValues), nil
}

func (p *baseProvider[K, V]) getKeysRetrieved() int {
	return int(p.keysRetrieved.Load())
}

func (p *baseProvider[K, V]) getKeysHit() int {
	return int(p.keysHit.Load())
}
