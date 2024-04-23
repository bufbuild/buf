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
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/zap"
)

// NewCommitProvider returns a new CommitProvider for the given API client.
func NewCommitProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1ModuleServiceClientProvider
		bufapi.V1OwnerServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	},
) bufmodule.CommitProvider {
	return newCommitProvider(logger, clientProvider)
}

// *** PRIVATE ***

type commitProvider struct {
	logger         *zap.Logger
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1ModuleServiceClientProvider
		bufapi.V1OwnerServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	}
}

func newCommitProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1ModuleServiceClientProvider
		bufapi.V1OwnerServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	},
) *commitProvider {
	return &commitProvider{
		logger:         logger,
		clientProvider: clientProvider,
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

	registryToIndexedModuleKeys := slicesext.ToIndexedValuesMap(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) string {
			return moduleKey.ModuleFullName().Registry()
		},
	)
	indexedCommits := make([]slicesext.Indexed[bufmodule.Commit], 0, len(moduleKeys))
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
	return slicesext.IndexedToSortedValues(indexedCommits), nil
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
	v1ProtoModuleProvider := newV1ProtoModuleProvider(a.logger, a.clientProvider)
	v1ProtoOwnerProvider := newV1ProtoOwnerProvider(a.logger, a.clientProvider)

	registryToIndexedCommitKeys := slicesext.ToIndexedValuesMap(
		commitKeys,
		func(commitKey bufmodule.CommitKey) string {
			return commitKey.Registry()
		},
	)
	indexedCommits := make([]slicesext.Indexed[bufmodule.Commit], 0, len(commitKeys))
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
	return slicesext.IndexedToSortedValues(indexedCommits), nil
}

func (a *commitProvider) getIndexedCommitsForRegistryAndIndexedModuleKeys(
	ctx context.Context,
	registry string,
	indexedModuleKeys []slicesext.Indexed[bufmodule.ModuleKey],
	digestType bufmodule.DigestType,
) ([]slicesext.Indexed[bufmodule.Commit], error) {
	commitIDToIndexedModuleKey, err := slicesext.ToUniqueValuesMapError(
		indexedModuleKeys,
		func(indexedModuleKey slicesext.Indexed[bufmodule.ModuleKey]) (uuid.UUID, error) {
			return indexedModuleKey.Value.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	commitIDs := slicesext.MapKeysToSlice(commitIDToIndexedModuleKey)
	universalProtoCommits, err := getUniversalProtoCommitsForRegistryAndCommitIDs(ctx, a.clientProvider, registry, commitIDs, digestType)
	if err != nil {
		return nil, err
	}
	return slicesext.MapError(
		universalProtoCommits,
		func(universalProtoCommit *universalProtoCommit) (slicesext.Indexed[bufmodule.Commit], error) {
			commitID, err := uuidutil.FromDashless(universalProtoCommit.ID)
			if err != nil {
				return slicesext.Indexed[bufmodule.Commit]{}, err
			}
			indexedModuleKey, ok := commitIDToIndexedModuleKey[commitID]
			if !ok {
				return slicesext.Indexed[bufmodule.Commit]{}, syserror.Newf("no ModuleKey for proto commit ID %q", commitID)
			}
			// This is actually backwards - this is not the expected digest, this is the actual digest.
			// TODO FUTURE: It doesn't matter too much, but we should switch around CommitWithExpectedDigest
			// to be CommitWithActualDigest.
			expectedDigest := universalProtoCommit.Digest
			return slicesext.Indexed[bufmodule.Commit]{
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
	indexedCommitKeys []slicesext.Indexed[bufmodule.CommitKey],
	digestType bufmodule.DigestType,
) ([]slicesext.Indexed[bufmodule.Commit], error) {
	commitIDToIndexedCommitKey, err := slicesext.ToUniqueValuesMapError(
		indexedCommitKeys,
		func(indexedCommitKey slicesext.Indexed[bufmodule.CommitKey]) (uuid.UUID, error) {
			return indexedCommitKey.Value.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	commitIDs := slicesext.MapKeysToSlice(commitIDToIndexedCommitKey)
	universalProtoCommits, err := getUniversalProtoCommitsForRegistryAndCommitIDs(ctx, a.clientProvider, registry, commitIDs, digestType)
	if err != nil {
		return nil, err
	}
	return slicesext.MapError(
		universalProtoCommits,
		func(universalProtoCommit *universalProtoCommit) (slicesext.Indexed[bufmodule.Commit], error) {
			commitID, err := uuidutil.FromDashless(universalProtoCommit.ID)
			if err != nil {
				return slicesext.Indexed[bufmodule.Commit]{}, err
			}
			indexedCommitKey, ok := commitIDToIndexedCommitKey[commitID]
			if !ok {
				return slicesext.Indexed[bufmodule.Commit]{}, syserror.Newf("no CommitKey for proto commit ID %q", commitID)
			}
			moduleKey, err := getModuleKeyForUniversalProtoCommit(
				ctx,
				v1ProtoModuleProvider,
				v1ProtoOwnerProvider,
				registry,
				universalProtoCommit,
			)
			if err != nil {
				return slicesext.Indexed[bufmodule.Commit]{}, err
			}
			return slicesext.Indexed[bufmodule.Commit]{
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
