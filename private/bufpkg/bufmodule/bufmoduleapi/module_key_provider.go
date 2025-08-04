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

package bufmoduleapi

import (
	"context"
	"log/slog"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

// NewModuleKeyProvider returns a new ModuleKeyProvider for the given API clients.
func NewModuleKeyProvider(
	logger *slog.Logger,
	moduleClientProvider interface {
		bufregistryapimodule.V1CommitServiceClientProvider
		bufregistryapimodule.V1Beta1CommitServiceClientProvider
	},
) bufmodule.ModuleKeyProvider {
	return newModuleKeyProvider(logger, moduleClientProvider)
}

// *** PRIVATE ***

type moduleKeyProvider struct {
	logger               *slog.Logger
	moduleClientProvider interface {
		bufregistryapimodule.V1CommitServiceClientProvider
		bufregistryapimodule.V1Beta1CommitServiceClientProvider
	}
}

func newModuleKeyProvider(
	logger *slog.Logger,
	moduleClientProvider interface {
		bufregistryapimodule.V1CommitServiceClientProvider
		bufregistryapimodule.V1Beta1CommitServiceClientProvider
	},
) *moduleKeyProvider {
	return &moduleKeyProvider{
		logger:               logger,
		moduleClientProvider: moduleClientProvider,
	}
}

func (a *moduleKeyProvider) GetModuleKeysForModuleRefs(
	ctx context.Context,
	moduleRefs []bufparse.Ref,
	digestType bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	// Check unique.
	if _, err := xslices.ToUniqueValuesMapError(
		moduleRefs,
		func(moduleRef bufparse.Ref) (string, error) {
			return moduleRef.String(), nil
		},
	); err != nil {
		return nil, err
	}

	registryToIndexedModuleRefs := xslices.ToIndexedValuesMap(
		moduleRefs,
		func(moduleRef bufparse.Ref) string {
			return moduleRef.FullName().Registry()
		},
	)
	indexedModuleKeys := make([]xslices.Indexed[bufmodule.ModuleKey], 0, len(moduleRefs))
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
	return xslices.IndexedToSortedValues(indexedModuleKeys), nil
}

func (a *moduleKeyProvider) getIndexedModuleKeysForRegistryAndIndexedModuleRefs(
	ctx context.Context,
	registry string,
	indexedModuleRefs []xslices.Indexed[bufparse.Ref],
	digestType bufmodule.DigestType,
) ([]xslices.Indexed[bufmodule.ModuleKey], error) {
	universalProtoCommits, err := getUniversalProtoCommitsForRegistryAndModuleRefs(ctx, a.moduleClientProvider, registry, xslices.IndexedToValues(indexedModuleRefs), digestType)
	if err != nil {
		return nil, err
	}
	indexedModuleKeys := make([]xslices.Indexed[bufmodule.ModuleKey], len(indexedModuleRefs))
	for i, universalProtoCommit := range universalProtoCommits {
		commitID, err := uuidutil.FromDashless(universalProtoCommit.ID)
		if err != nil {
			return nil, err
		}
		moduleKey, err := bufmodule.NewModuleKey(
			// Note we don't have to resolve owner_name and module_name since we already have them.
			indexedModuleRefs[i].Value.FullName(),
			commitID,
			func() (bufmodule.Digest, error) {
				// Do not call getModuleKeyForProtoCommit, we already have the owner and module names.
				return universalProtoCommit.Digest, nil
			},
		)
		if err != nil {
			return nil, err
		}
		indexedModuleKeys[i] = xslices.Indexed[bufmodule.ModuleKey]{
			Value: moduleKey,
			Index: indexedModuleRefs[i].Index,
		}
	}
	return indexedModuleKeys, nil
}
