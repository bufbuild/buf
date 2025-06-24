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
	"time"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiowner"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

// NewCommitProvider returns a new CommitProvider for the given API client.
func NewCommitProvider(
	logger *slog.Logger,
	moduleClientProvider interface {
		bufregistryapimodule.V1CommitServiceClientProvider
		bufregistryapimodule.V1ModuleServiceClientProvider
		bufregistryapimodule.V1Beta1CommitServiceClientProvider
	},
	ownerClientProvider bufregistryapiowner.V1OwnerServiceClientProvider,
) bufmodule.CommitProvider {
	return newCommitProvider(logger, moduleClientProvider, ownerClientProvider)
}

// *** PRIVATE ***

type commitProvider struct {
	logger               *slog.Logger
	moduleClientProvider interface {
		bufregistryapimodule.V1CommitServiceClientProvider
		bufregistryapimodule.V1ModuleServiceClientProvider
		bufregistryapimodule.V1Beta1CommitServiceClientProvider
	}
	ownerClientProvider bufregistryapiowner.V1OwnerServiceClientProvider
}

func newCommitProvider(
	logger *slog.Logger,
	moduleClientProvider interface {
		bufregistryapimodule.V1CommitServiceClientProvider
		bufregistryapimodule.V1ModuleServiceClientProvider
		bufregistryapimodule.V1Beta1CommitServiceClientProvider
	},
	ownerClientProvider bufregistryapiowner.V1OwnerServiceClientProvider,
) *commitProvider {
	return &commitProvider{
		logger:               logger,
		moduleClientProvider: moduleClientProvider,
		ownerClientProvider:  ownerClientProvider,
	}
}

func (a *commitProvider) GetCommitsForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.Commit, error) {
	if len(moduleKeys) == 0 {
		return nil, nil
	}
	digestType, err := bufmodule.UniqueDigestTypeForModuleKeys(moduleKeys)
	if err != nil {
		return nil, err
	}

	registryToIndexedModuleKeys := xslices.ToIndexedValuesMap(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) string {
			return moduleKey.FullName().Registry()
		},
	)
	indexedCommits := make([]xslices.Indexed[bufmodule.Commit], 0, len(moduleKeys))
	for registry, indexedModuleKeys := range registryToIndexedModuleKeys {
		registryIndexedCommits, err := a.getIndexedCommitsForRegistryAndIndexedModuleKeys(
			ctx,
			registry,
			indexedModuleKeys,
			digestType,
		)
		if err != nil {
			return nil, err
		}
		indexedCommits = append(indexedCommits, registryIndexedCommits...)
	}
	return xslices.IndexedToSortedValues(indexedCommits), nil
}

func (a *commitProvider) GetCommitsForCommitKeys(
	ctx context.Context,
	commitKeys []bufmodule.CommitKey,
) ([]bufmodule.Commit, error) {
	if len(commitKeys) == 0 {
		return nil, nil
	}
	digestType, err := bufmodule.UniqueDigestTypeForCommitKeys(commitKeys)
	if err != nil {
		return nil, err
	}

	// We don't want to persist these across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	v1ProtoModuleProvider := newV1ProtoModuleProvider(a.logger, a.moduleClientProvider)
	v1ProtoOwnerProvider := newV1ProtoOwnerProvider(a.logger, a.ownerClientProvider)

	registryToIndexedCommitKeys := xslices.ToIndexedValuesMap(
		commitKeys,
		func(commitKey bufmodule.CommitKey) string {
			return commitKey.Registry()
		},
	)
	indexedCommits := make([]xslices.Indexed[bufmodule.Commit], 0, len(commitKeys))
	for registry, indexedCommitKeys := range registryToIndexedCommitKeys {
		registryIndexedCommits, err := a.getIndexedCommitsForRegistryAndIndexedCommitKeys(
			ctx,
			v1ProtoModuleProvider,
			v1ProtoOwnerProvider,
			registry,
			indexedCommitKeys,
			digestType,
		)
		if err != nil {
			return nil, err
		}
		indexedCommits = append(indexedCommits, registryIndexedCommits...)
	}
	return xslices.IndexedToSortedValues(indexedCommits), nil
}

func (a *commitProvider) getIndexedCommitsForRegistryAndIndexedModuleKeys(
	ctx context.Context,
	registry string,
	indexedModuleKeys []xslices.Indexed[bufmodule.ModuleKey],
	digestType bufmodule.DigestType,
) ([]xslices.Indexed[bufmodule.Commit], error) {
	commitIDToIndexedModuleKey, err := xslices.ToUniqueValuesMapError(
		indexedModuleKeys,
		func(indexedModuleKey xslices.Indexed[bufmodule.ModuleKey]) (uuid.UUID, error) {
			return indexedModuleKey.Value.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	commitIDs := xslices.MapKeysToSlice(commitIDToIndexedModuleKey)
	universalProtoCommits, err := getUniversalProtoCommitsForRegistryAndCommitIDs(ctx, a.moduleClientProvider, registry, commitIDs, digestType)
	if err != nil {
		return nil, err
	}
	return xslices.MapError(
		universalProtoCommits,
		func(universalProtoCommit *universalProtoCommit) (xslices.Indexed[bufmodule.Commit], error) {
			commitID, err := uuidutil.FromDashless(universalProtoCommit.ID)
			if err != nil {
				return xslices.Indexed[bufmodule.Commit]{}, err
			}
			indexedModuleKey, ok := commitIDToIndexedModuleKey[commitID]
			if !ok {
				return xslices.Indexed[bufmodule.Commit]{}, syserror.Newf("no ModuleKey for proto commit ID %q", commitID)
			}
			// This is actually backwards - this is not the expected digest, this is the actual digest.
			// TODO FUTURE: It doesn't matter too much, but we should switch around CommitWithExpectedDigest
			// to be CommitWithActualDigest.
			expectedDigest := universalProtoCommit.Digest
			return xslices.Indexed[bufmodule.Commit]{
				Value: bufmodule.NewCommit(
					indexedModuleKey.Value,
					func() (time.Time, error) {
						return universalProtoCommit.CreateTime, nil
					},
					bufmodule.CommitWithExpectedDigest(expectedDigest),
				),
				Index: indexedModuleKey.Index,
			}, nil
		},
	)
}

func (a *commitProvider) getIndexedCommitsForRegistryAndIndexedCommitKeys(
	ctx context.Context,
	v1ProtoModuleProvider *v1ProtoModuleProvider,
	v1ProtoOwnerProvider *v1ProtoOwnerProvider,
	registry string,
	indexedCommitKeys []xslices.Indexed[bufmodule.CommitKey],
	digestType bufmodule.DigestType,
) ([]xslices.Indexed[bufmodule.Commit], error) {
	commitIDToIndexedCommitKey, err := xslices.ToUniqueValuesMapError(
		indexedCommitKeys,
		func(indexedCommitKey xslices.Indexed[bufmodule.CommitKey]) (uuid.UUID, error) {
			return indexedCommitKey.Value.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	commitIDs := xslices.MapKeysToSlice(commitIDToIndexedCommitKey)
	universalProtoCommits, err := getUniversalProtoCommitsForRegistryAndCommitIDs(ctx, a.moduleClientProvider, registry, commitIDs, digestType)
	if err != nil {
		return nil, err
	}
	return xslices.MapError(
		universalProtoCommits,
		func(universalProtoCommit *universalProtoCommit) (xslices.Indexed[bufmodule.Commit], error) {
			commitID, err := uuidutil.FromDashless(universalProtoCommit.ID)
			if err != nil {
				return xslices.Indexed[bufmodule.Commit]{}, err
			}
			indexedCommitKey, ok := commitIDToIndexedCommitKey[commitID]
			if !ok {
				return xslices.Indexed[bufmodule.Commit]{}, syserror.Newf("no CommitKey for proto commit ID %q", commitID)
			}
			moduleKey, err := getModuleKeyForUniversalProtoCommit(
				ctx,
				v1ProtoModuleProvider,
				v1ProtoOwnerProvider,
				registry,
				universalProtoCommit,
			)
			if err != nil {
				return xslices.Indexed[bufmodule.Commit]{}, err
			}
			return xslices.Indexed[bufmodule.Commit]{
				// No digest to compare against to add as CommitOption.
				Value: bufmodule.NewCommit(
					moduleKey,
					func() (time.Time, error) {
						return universalProtoCommit.CreateTime, nil
					},
				),
				Index: indexedCommitKey.Index,
			}, nil
		},
	)
}
