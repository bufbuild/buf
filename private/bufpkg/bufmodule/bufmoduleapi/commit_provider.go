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
	"fmt"
	"io/fs"
	"time"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// NewCommitProvider returns a new CommitProvider for the given API client.
func NewCommitProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) bufmodule.CommitProvider {
	return newCommitProvider(logger, clientProvider)
}

// *** PRIVATE ***

type commitProvider struct {
	logger         *zap.Logger
	clientProvider bufapi.ClientProvider
}

func newCommitProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
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
	// We don't want to persist these across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	protoModuleProvider := newProtoModuleProvider(a.logger, a.clientProvider)
	protoOwnerProvider := newProtoOwnerProvider(a.logger, a.clientProvider)

	registryToIndexedModuleKeys := getKeyToIndexedValues(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) string {
			return moduleKey.ModuleFullName().Registry()
		},
	)
	commits := make([]bufmodule.Commit, len(moduleKeys))
	for registry, indexedModuleKeys := range registryToIndexedModuleKeys {
		registryCommits, err := a.getCommitsForRegistryAndModuleKeys(
			ctx,
			protoModuleProvider,
			protoOwnerProvider,
			registry,
			getValuesForIndexedValues(indexedModuleKeys),
		)
		if err != nil {
			return nil, err
		}
		for i, registryCommit := range registryCommits {
			commits[indexedModuleKeys[i].Index] = registryCommit
		}
	}
	return commits, nil
}

func (a *commitProvider) getCommitsForRegistryAndModuleKeys(
	ctx context.Context,
	protoModuleProvider *protoModuleProvider,
	protoOwnerProvider *protoOwnerProvider,
	registry string,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.Commit, error) {
	protoCommitIDToModuleKey, err := slicesext.ToUniqueValuesMapError(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) (string, error) {
			return CommitIDToProto(moduleKey.CommitID())
		},
	)
	if err != nil {
		return nil, err
	}
	protoCommitIDs := slicesext.MapKeysToSortedSlice(protoCommitIDToModuleKey)

	response, err := a.clientProvider.CommitServiceClient(registry).GetCommits(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitsRequest{
				// TODO: chunking
				ResourceRefs: slicesext.Map(
					protoCommitIDs,
					func(protoCommitID string) *modulev1beta1.ResourceRef {
						return &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: protoCommitID,
							},
						}
					},
				),
				DigestType: modulev1beta1.DigestType_DIGEST_TYPE_B5,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Kind of an abuse of fs.PathError. Is there a way to get a specific ModuleKey out of this?
			return nil, &fs.PathError{Op: "read", Path: err.Error(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if len(response.Msg.Commits) != len(moduleKeys) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(moduleKeys), len(response.Msg.Commits))
	}
	return slicesext.MapError(
		response.Msg.Commits,
		func(protoCommit *modulev1beta1.Commit) (bufmodule.Commit, error) {
			moduleKey, ok := protoCommitIDToModuleKey[protoCommit.Id]
			if !ok {
				return nil, syserror.Newf("no ModuleKey for proto commit ID %q", protoCommit.Id)
			}
			return bufmodule.NewCommit(
				moduleKey,
				func() (time.Time, error) {
					return protoCommit.CreateTime.AsTime(), nil
				},
				func() (bufmodule.Digest, error) {
					return ProtoToDigest(protoCommit.Digest)
				},
			), nil
		},
	)
}
