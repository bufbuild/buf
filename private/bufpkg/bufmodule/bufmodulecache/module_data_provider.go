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
	*baseProvider[bufmodule.ModuleKey, bufmodule.ModuleData]
}

func newModuleDataProvider(
	logger *zap.Logger,
	delegate bufmodule.ModuleDataProvider,
	store bufmodulestore.ModuleDataStore,
) *moduleDataProvider {
	return &moduleDataProvider{
		baseProvider: newBaseProvider(
			logger,
			delegate.GetModuleDatasForModuleKeys,
			store.GetModuleDatasForModuleKeys,
			store.PutModuleDatas,
			bufmodule.ModuleKey.CommitID,
			func(moduleData bufmodule.ModuleData) uuid.UUID {
				return moduleData.ModuleKey().CommitID()
			},
		),
	}
}

func (p *moduleDataProvider) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, error) {
	return p.baseProvider.getValuesForKeys(ctx, moduleKeys)
}
