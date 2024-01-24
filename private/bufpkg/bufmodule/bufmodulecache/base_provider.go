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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/zap"
)

type hasModuleKey interface {
	ModuleKey() bufmodule.ModuleKey
}

type baseProvider[T hasModuleKey] struct {
	logger                         *zap.Logger
	delegateGetValuesForModuleKeys func(context.Context, []bufmodule.ModuleKey) ([]T, error)
	storeGetValuesForModuleKeys    func(context.Context, []bufmodule.ModuleKey) ([]T, []bufmodule.ModuleKey, error)
	storePutValues                 func(context.Context, []T) error

	moduleKeysRetrieved atomic.Int64
	moduleKeysHit       atomic.Int64
}

func newBaseProvider[T hasModuleKey](
	logger *zap.Logger,
	delegateGetValuesForModuleKeys func(context.Context, []bufmodule.ModuleKey) ([]T, error),
	storeGetValuesForModuleKeys func(context.Context, []bufmodule.ModuleKey) ([]T, []bufmodule.ModuleKey, error),
	storePutValues func(context.Context, []T) error,
) *baseProvider[T] {
	return &baseProvider[T]{
		logger:                         logger,
		delegateGetValuesForModuleKeys: delegateGetValuesForModuleKeys,
		storeGetValuesForModuleKeys:    storeGetValuesForModuleKeys,
		storePutValues:                 storePutValues,
	}
}

func (p *baseProvider[T]) getValuesForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]T, error) {
	commitIDToIndexedModuleKey, err := slicesext.ToUniqueIndexedValuesMap(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) uuid.UUID {
			return moduleKey.CommitID()
		},
	)
	if err != nil {
		return nil, err
	}
	foundValues, notFoundModuleKeys, err := p.storeGetValuesForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	delegateValues, err := p.delegateGetValuesForModuleKeys(
		ctx,
		notFoundModuleKeys,
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

	p.moduleKeysRetrieved.Add(int64(len(moduleKeys)))
	p.moduleKeysHit.Add(int64(len(foundValues)))

	indexedValues, err := slicesext.MapError(
		append(foundValues, delegateValues...),
		func(value T) (slicesext.Indexed[T], error) {
			indexedModuleKey, ok := commitIDToIndexedModuleKey[value.ModuleKey().CommitID()]
			if !ok {
				return slicesext.Indexed[T]{}, syserror.Newf("did not get value from store with commitID %q", value.ModuleKey().CommitID())
			}
			return slicesext.Indexed[T]{
				Value: value,
				Index: indexedModuleKey.Index,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return slicesext.IndexedToSortedValues(indexedValues), nil
}

func (p *baseProvider[T]) getModuleKeysRetrieved() int {
	return int(p.moduleKeysRetrieved.Load())
}

func (p *baseProvider[T]) getModuleKeysHit() int {
	return int(p.moduleKeysHit.Load())
}
