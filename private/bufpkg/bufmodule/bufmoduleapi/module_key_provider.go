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

package bufmoduleapi

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/zap"
)

// NewModuleKeyProvider returns a new ModuleKeyProvider for the given API clients.
func NewModuleKeyProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	},
) bufmodule.ModuleKeyProvider {
	return newModuleKeyProvider(logger, clientProvider)
}

// *** PRIVATE ***

type moduleKeyProvider struct {
	logger         *zap.Logger
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	}
}

func newModuleKeyProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	},
) *moduleKeyProvider {
	return &moduleKeyProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *moduleKeyProvider) GetModuleKeysForModuleRefs(
	ctx context.Context,
	moduleRefs []bufmodule.ModuleRef,
	digestType bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	// Check unique.
	if _, err := slicesext.ToUniqueValuesMapError(
		moduleRefs,
		func(moduleRef bufmodule.ModuleRef) (string, error) {
			return moduleRef.String(), nil
		},
	); err != nil {
		return nil, err
	}

	registryToIndexedModuleRefs := slicesext.ToIndexedValuesMap(
		moduleRefs,
		func(moduleRef bufmodule.ModuleRef) string {
			return moduleRef.ModuleFullName().Registry()
		},
	)
	indexedModuleKeys := make([]slicesext.Indexed[bufmodule.ModuleKey], 0, len(moduleRefs))
	for registry, indexedModuleRefs := range registryToIndexedModuleRefs {
		indexedRegistryModuleKeys, err := a.getIndexedModuleKeysForRegistryAndIndexedModuleRefs(
			ctx,
			registry,
			indexedModuleRefs,
			digestType,
		)
		if err != nil {
			return nil, err
		}
		indexedModuleKeys = append(indexedModuleKeys, indexedRegistryModuleKeys...)
	}
	return slicesext.IndexedToSortedValues(indexedModuleKeys), nil
}

func (a *moduleKeyProvider) getIndexedModuleKeysForRegistryAndIndexedModuleRefs(
	ctx context.Context,
	registry string,
	indexedModuleRefs []slicesext.Indexed[bufmodule.ModuleRef],
	digestType bufmodule.DigestType,
) ([]slicesext.Indexed[bufmodule.ModuleKey], error) {
	universalProtoCommits, err := getUniversalProtoCommitsForRegistryAndModuleRefs(ctx, a.clientProvider, registry, slicesext.IndexedToValues(indexedModuleRefs), digestType)
	if err != nil {
		return nil, err
	}
	indexedModuleKeys := make([]slicesext.Indexed[bufmodule.ModuleKey], len(indexedModuleRefs))
	for i, universalProtoCommit := range universalProtoCommits {
		universalProtoCommit := universalProtoCommit
		commitID, err := uuidutil.FromDashless(universalProtoCommit.ID)
		if err != nil {
			return nil, err
		}
		moduleKey, err := bufmodule.NewModuleKey(
			// Note we don't have to resolve owner_name and module_name since we already have them.
			indexedModuleRefs[i].Value.ModuleFullName(),
			commitID,
			func() (bufmodule.Digest, error) {
				// Do not call getModuleKeyForProtoCommit, we already have the owner and module names.
				return universalProtoCommit.Digest, nil
			},
		)
		if err != nil {
			return nil, err
		}
		indexedModuleKeys[i] = slicesext.Indexed[bufmodule.ModuleKey]{
			Value: moduleKey,
			Index: indexedModuleRefs[i].Index,
		}
	}
	return indexedModuleKeys, nil
}
