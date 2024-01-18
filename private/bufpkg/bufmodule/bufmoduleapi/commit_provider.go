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

package bufmoduleapi

import (
	"context"
	"time"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// NewCommitProvider returns a new CommitProvider for the given API client.
func NewCommitProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.CommitServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.OwnerServiceClientProvider
	},
) bufmodule.CommitProvider {
	return newCommitProvider(logger, clientProvider)
}

// *** PRIVATE ***

type commitProvider struct {
	logger         *zap.Logger
	clientProvider interface {
		bufapi.CommitServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.OwnerServiceClientProvider
	}
}

func newCommitProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.CommitServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.OwnerServiceClientProvider
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

	// We don't want to persist these across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	protoModuleProvider := newProtoModuleProvider(a.logger, a.clientProvider)
	protoOwnerProvider := newProtoOwnerProvider(a.logger, a.clientProvider)

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
			protoModuleProvider,
			protoOwnerProvider,
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

func (a *commitProvider) getIndexedCommitsForRegistryAndIndexedModuleKeys(
	ctx context.Context,
	protoModuleProvider *protoModuleProvider,
	protoOwnerProvider *protoOwnerProvider,
	registry string,
	indexedModuleKeys []slicesext.Indexed[bufmodule.ModuleKey],
	digestType bufmodule.DigestType,
) ([]slicesext.Indexed[bufmodule.Commit], error) {
	commitIDToIndexedModuleKey, err := slicesext.ToUniqueValuesMapError(
		indexedModuleKeys,
		func(indexedModuleKey slicesext.Indexed[bufmodule.ModuleKey]) (string, error) {
			return indexedModuleKey.Value.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	protoCommitIDs, err := slicesext.MapError(
		slicesext.MapKeysToSortedSlice(commitIDToIndexedModuleKey),
		func(commitID string) (string, error) {
			return CommitIDToProto(commitID)
		},
	)
	if err != nil {
		return nil, err
	}
	protoCommits, err := getProtoCommitsForRegistryAndCommitIDs(ctx, a.clientProvider, registry, protoCommitIDs, digestType)
	if err != nil {
		return nil, err
	}
	return slicesext.MapError(
		protoCommits,
		func(protoCommit *modulev1beta1.Commit) (slicesext.Indexed[bufmodule.Commit], error) {
			commitID, err := ProtoToCommitID(protoCommit.Id)
			if err != nil {
				return slicesext.Indexed[bufmodule.Commit]{}, err
			}
			indexedModuleKey, ok := commitIDToIndexedModuleKey[commitID]
			if !ok {
				return slicesext.Indexed[bufmodule.Commit]{}, syserror.Newf("no ModuleKey for proto commit ID %q", commitID)
			}
			return slicesext.Indexed[bufmodule.Commit]{
				Value: bufmodule.NewCommit(
					indexedModuleKey.Value,
					func() (time.Time, error) {
						return protoCommit.CreateTime.AsTime(), nil
					},
					func() (bufmodule.Digest, error) {
						return ProtoToDigest(protoCommit.Digest)
					},
				),
				Index: indexedModuleKey.Index,
			}, nil
		},
	)
}
