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

	values := append(foundValues, delegateValues...)
	sort.Slice(
		values,
		func(i int, j int) bool {
			return values[i].ModuleKey().ModuleFullName().String() < values[j].ModuleKey().ModuleFullName().String()
		},
	)
	return values, nil
}

func (p *baseProvider[T]) getModuleKeysRetrieved() int {
	return int(p.moduleKeysRetrieved.Load())
}

func (p *baseProvider[T]) getModuleKeysHit() int {
	return int(p.moduleKeysHit.Load())
}
