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

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"go.uber.org/zap"
)

// NewModuleKeyProvider returns a new ModuleKeyProvider for the given API clients.
func NewModuleKeyProvider(
	logger *zap.Logger,
	clientProvider bufapi.CommitServiceClientProvider,
) bufmodule.ModuleKeyProvider {
	return newModuleKeyProvider(logger, clientProvider)
}

// *** PRIVATE ***

type moduleKeyProvider struct {
	logger         *zap.Logger
	clientProvider bufapi.CommitServiceClientProvider
}

func newModuleKeyProvider(
	logger *zap.Logger,
	clientProvider bufapi.CommitServiceClientProvider,
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

	registryToIndexedModuleRefs := getKeyToIndexedValues(
		moduleRefs,
		func(moduleRef bufmodule.ModuleRef) string {
			return moduleRef.ModuleFullName().Registry()
		},
	)
	moduleKeys := make([]bufmodule.ModuleKey, len(moduleRefs))
	for registry, indexedModuleRefs := range registryToIndexedModuleRefs {
		registryModuleKeys, err := a.getModuleKeysForRegistryAndModuleRefs(
			ctx,
			registry,
			getValuesForIndexedValues(indexedModuleRefs),
			digestType,
		)
		if err != nil {
			return nil, err
		}
		for i, registryModuleKey := range registryModuleKeys {
			moduleKeys[indexedModuleRefs[i].Index] = registryModuleKey
		}
	}
	return moduleKeys, nil
}

func (a *moduleKeyProvider) getModuleKeysForRegistryAndModuleRefs(
	ctx context.Context,
	registry string,
	moduleRefs []bufmodule.ModuleRef,
	digestType bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	protoCommits, err := a.getProtoCommitsForRegistryAndModuleRefs(ctx, registry, moduleRefs, digestType)
	if err != nil {
		return nil, err
	}
	moduleKeys := make([]bufmodule.ModuleKey, len(moduleRefs))
	for i, protoCommit := range protoCommits {
		commitID, err := ProtoToCommitID(protoCommit.Id)
		if err != nil {
			return nil, err
		}
		moduleKey, err := bufmodule.NewModuleKey(
			// Note we don't have to resolve owner_name and module_name since we already have them.
			moduleRefs[i].ModuleFullName(),
			commitID,
			func() (bufmodule.Digest, error) {
				// Do not call getModuleKeyForProtoCommit, we already have the owner and module names.
				return ProtoToDigest(protoCommit.Digest)
			},
		)
		if err != nil {
			return nil, err
		}
		moduleKeys[i] = moduleKey
	}
	return moduleKeys, nil
}

func (a *moduleKeyProvider) getProtoCommitsForRegistryAndModuleRefs(
	ctx context.Context,
	registry string,
	moduleRefs []bufmodule.ModuleRef,
	digestType bufmodule.DigestType,
) ([]*modulev1beta1.Commit, error) {
	protoDigestType, err := digestTypeToProto(digestType)
	if err != nil {
		return nil, err
	}
	response, err := a.clientProvider.CommitServiceClient(registry).GetCommits(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitsRequest{
				// TODO: chunking
				ResourceRefs: slicesext.Map(
					moduleRefs,
					func(moduleRef bufmodule.ModuleRef) *modulev1beta1.ResourceRef {
						return &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Name_{
								Name: &modulev1beta1.ResourceRef_Name{
									Owner:  moduleRef.ModuleFullName().Owner(),
									Module: moduleRef.ModuleFullName().Name(),
									Child: &modulev1beta1.ResourceRef_Name_Ref{
										// TODO: What to do about commit IDs? Need to be dashful.
										Ref: moduleRef.Ref(),
									},
								},
							},
						}
					},
				),
				DigestType: protoDigestType,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Kind of an abuse of fs.PathError. Is there a way to get a specific ModuleRef out of this?
			return nil, &fs.PathError{Op: "read", Path: err.Error(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if len(response.Msg.Commits) != len(moduleRefs) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(moduleRefs), len(response.Msg.Commits))
	}
	return response.Msg.Commits, nil
}
