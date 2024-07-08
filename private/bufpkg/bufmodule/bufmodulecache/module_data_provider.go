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
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulestore"
	"github.com/bufbuild/buf/private/pkg/filelock"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// NewModuleDataProvider returns a new ModuleDataProvider that caches the results of the delegate.
//
// The ModuleDataStore is used as a cache.
func NewModuleDataProvider(
	logger *zap.Logger,
	delegate bufmodule.ModuleDataProvider,
	store bufmodulestore.ModuleDataStore,
	filelocker filelock.Locker,
) bufmodule.ModuleDataProvider {
	return newModuleDataProvider(logger, delegate, store, filelocker)
}

/// *** PRIVATE ***

type moduleDataProvider struct {
	*baseProvider[bufmodule.ModuleKey, bufmodule.ModuleData]
	filelocker filelock.Locker
}

func newModuleDataProvider(
	logger *zap.Logger,
	delegate bufmodule.ModuleDataProvider,
	store bufmodulestore.ModuleDataStore,
	filelocker filelock.Locker,
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
		filelocker: filelocker,
	}
}

func (p *moduleDataProvider) GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) (_ []bufmodule.ModuleData, retErr error) {
	if p.filelocker != nil {
		// If a file locker exists, take a file lock for all requested module keys
		for _, moduleKey := range moduleKeys {
			digest, err := moduleKey.Digest()
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf(
				"%s/%s/%s/%s/%s/module.yaml", //TODO(doria): may want to export for bufmodulestore
				digest.Type().String(),
				moduleKey.ModuleFullName().Registry(),
				moduleKey.ModuleFullName().Owner(),
				moduleKey.ModuleFullName().Name(),
				moduleKey.CommitID().String(),
			)
			unlocker, err := p.filelocker.Lock(ctx, path)
			if err != nil {
				return nil, err
			}
			defer func() {
				retErr = multierr.Append(retErr, unlocker.Unlock())
			}()
		}
	}

	return p.baseProvider.getValuesForKeys(ctx, moduleKeys)
}
