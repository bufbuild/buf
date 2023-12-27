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

func (p *moduleDataProvider) GetOptionalModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys ...bufmodule.ModuleKey,
) ([]bufmodule.OptionalModuleData, error) {
	cachedOptionalModuleDatas, err := p.store.GetOptionalModuleDatasForModuleKeys(ctx, moduleKeys...)
	if err != nil {
		return nil, err
	}
	resultOptionalModuleDatas := make([]bufmodule.OptionalModuleData, len(moduleKeys))
	// The indexes within moduleKeys of the ModuleKeys that did not have a cached ModuleData.
	// We will then fetch these specific ModuleKeys in one shot from the delegate.
	var missedModuleKeysIndexes []int
	for i, cachedOptionalModuleData := range cachedOptionalModuleDatas {
		if err := p.logDebugModuleKey(
			moduleKeys[i],
			"module cache get",
			zap.Bool("found", cachedOptionalModuleData.Found()),
		); err != nil {
			return nil, err
		}
		if cachedOptionalModuleData.Found() {
			// We put the cached ModuleData at the specific location it is expected to be returned,
			// given that the returned ModuleData order must match the input ModuleKey order.
			resultOptionalModuleDatas[i] = cachedOptionalModuleData
		} else {
			missedModuleKeysIndexes = append(missedModuleKeysIndexes, i)
		}
	}
	if len(missedModuleKeysIndexes) > 0 {
		missedOptionalModuleDatas, err := p.delegate.GetOptionalModuleDatasForModuleKeys(
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
		if len(missedOptionalModuleDatas) != len(missedModuleKeysIndexes) {
			return nil, fmt.Errorf(
				"expected %d ModuleDatas, got %d",
				len(missedModuleKeysIndexes),
				len(missedOptionalModuleDatas),
			)
		}
		// Put the found ModuleDatas into the store.
		if err := p.store.PutModuleDatas(
			ctx,
			slicesext.Map(
				// Get just the OptionalModuleDatas that were found.
				slicesext.Filter(
					missedOptionalModuleDatas,
					func(optionalModuleData bufmodule.OptionalModuleData) bool {
						return optionalModuleData.Found()
					},
				),
				// Get found OptionalModuleData -> ModuleData.
				func(optionalModuleData bufmodule.OptionalModuleData) bufmodule.ModuleData {
					return optionalModuleData.ModuleData()
				},
			)...,
		); err != nil {
			return nil, err
		}
		for i, missedModuleKeysIndex := range missedModuleKeysIndexes {
			// i is the index within missedOptionalModuleDatas, while missedModuleKeysIndex is the index
			// within missedModuleKeysIndexes, and consequently moduleKeys.
			//
			// Put in the specific location we expect the OptionalModuleData to be returned.
			// Put in regardless of whether it was found.
			resultOptionalModuleDatas[missedModuleKeysIndex] = missedOptionalModuleDatas[i]
		}
	}
	p.moduleKeysRetrieved.Add(int64(len(resultOptionalModuleDatas)))
	p.moduleKeysHit.Add(int64(len(resultOptionalModuleDatas) - len(missedModuleKeysIndexes)))
	return resultOptionalModuleDatas, nil
}

func (p *moduleDataProvider) getModuleKeysRetrieved() int {
	return int(p.moduleKeysRetrieved.Load())
}

func (p *moduleDataProvider) getModuleKeysHit() int {
	return int(p.moduleKeysHit.Load())
}

func (p *moduleDataProvider) logDebugModuleKey(
	moduleKey bufmodule.ModuleKey,
	message string,
	fields ...zap.Field,
) error {
	if checkedEntry := p.logger.Check(zap.DebugLevel, message); checkedEntry != nil {
		moduleDigest, err := moduleKey.ModuleDigest()
		if err != nil {
			return err
		}
		checkedEntry.Write(
			append(
				[]zap.Field{
					zap.String("moduleFullName", moduleKey.ModuleFullName().String()),
					zap.String("moduleDigest", moduleDigest.String()),
				},
				fields...,
			)...,
		)
	}
	return nil
}
