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

// NewModuleDataProvider returns a new ModuleDataProvider that caches the results of the delegate.
//
// The ModuleDataStore is used as a cache.
func NewModuleDataProvider(
	logger *zap.Logger,
	delegate bufmodule.ModuleDataProvider,
	store bufmodulestore.ModuleDataStore,
) bufmodule.ModuleDataProvider {
	return newModuleDataProvider(logger, delegate, store)
}

/// *** PRIVATE ***

type moduleDataProvider struct {
	logger   *zap.Logger
	delegate bufmodule.ModuleDataProvider
	store    bufmodulestore.ModuleDataStore

	moduleKeysRetrieved atomic.Int64
	moduleKeysHit       atomic.Int64
}

func newModuleDataProvider(
	logger *zap.Logger,
	delegate bufmodule.ModuleDataProvider,
	store bufmodulestore.ModuleDataStore,
) *moduleDataProvider {
	return &moduleDataProvider{
		logger:   logger,
		delegate: delegate,
		store:    store,
	}
}

func (p *moduleDataProvider) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
	options ...bufmodule.GetModuleDatasForModuleKeysOption,
) ([]bufmodule.ModuleData, error) {
	getModuleDatasForModuleKeysOptions := bufmodule.NewGetModuleDatasForModuleKeysOptions(options)
	if _, err := bufmodule.ModuleFullNameStringToUniqueValue(moduleKeys); err != nil {
		return nil, err
	}
	_ = getModuleDatasForModuleKeysOptions

	storeModuleDatasResult, err := p.store.GetModuleDatasForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	foundModuleDatas := storeModuleDatasResult.FoundModuleDatas()
	notFoundModuleKeys := storeModuleDatasResult.NotFoundModuleKeys()
	delegateModuleDatas, err := p.delegate.GetModuleDatasForModuleKeys(
		ctx,
		notFoundModuleKeys,
		options...,
	)
	if err != nil {
		return nil, err
	}
	if err := p.store.PutModuleDatas(
		ctx,
		delegateModuleDatas,
	); err != nil {
		return nil, err
	}

	p.moduleKeysRetrieved.Add(int64(len(moduleKeys)))
	p.moduleKeysHit.Add(int64(len(foundModuleDatas)))

	moduleDatas := append(foundModuleDatas, delegateModuleDatas...)
	sort.Slice(
		moduleDatas,
		func(i int, j int) bool {
			return moduleDatas[i].ModuleKey().ModuleFullName().String() < moduleDatas[j].ModuleKey().ModuleFullName().String()
		},
	)
	return moduleDatas, nil
}

func (p *moduleDataProvider) getModuleKeysRetrieved() int {
	return int(p.moduleKeysRetrieved.Load())
}

func (p *moduleDataProvider) getModuleKeysHit() int {
	return int(p.moduleKeysHit.Load())
}
