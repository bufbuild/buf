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
	"fmt"
	"io/fs"
	"strings"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
	"go.uber.org/zap"
)

// NewGraphProvider returns a new GraphProvider for the given API client.
func NewGraphProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.GraphServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.OwnerServiceClientProvider
	},
) bufmodule.GraphProvider {
	return newGraphProvider(logger, clientProvider)
}

// *** PRIVATE ***

type graphProvider struct {
	logger         *zap.Logger
	clientProvider interface {
		bufapi.GraphServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.OwnerServiceClientProvider
	}
}

func newGraphProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.GraphServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.OwnerServiceClientProvider
	},
) *graphProvider {
	return &graphProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *graphProvider) GetGraphForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) (*dag.Graph[uuid.UUID, bufmodule.ModuleKey], error) {
	graph := dag.NewGraph[uuid.UUID, bufmodule.ModuleKey](bufmodule.ModuleKey.CommitID)
	if len(moduleKeys) == 0 {
		return graph, nil
	}
	digestType, err := bufmodule.UniqueDigestTypeForModuleKeys(moduleKeys)
	if err != nil {
		return nil, err
	}

	// We don't want to persist these across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	protoModuleProvider := newProtoModuleProvider(a.logger, a.clientProvider)
	protoOwnerProvider := newProtoOwnerProvider(a.logger, a.clientProvider)
	registries := slicesext.ToUniqueSorted(
		slicesext.Map(
			moduleKeys,
			func(moduleKey bufmodule.ModuleKey) string { return moduleKey.ModuleFullName().Registry() },
		),
	)
	// Validate we're all within one registry for now.
	if len(registries) != 1 {
		// TODO: This messes up legacy federation.
		return nil, fmt.Errorf("multiple registries detected: %s", strings.Join(registries, ", "))
	}
	registry := registries[0]
	protoGraph, err := a.getProtoGraphForRegistryAndModuleKeys(ctx, registry, moduleKeys, digestType)
	if err != nil {
		return nil, err
	}
	commitIDToModuleKey, err := slicesext.ToUniqueValuesMapError(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) (uuid.UUID, error) {
			return moduleKey.CommitID(), nil
		},
	)
	if err != nil {
		return nil, err
	}
	for _, protoCommit := range protoGraph.Commits {
		commitID, err := uuid.FromString(protoCommit.Id)
		if err != nil {
			return nil, err
		}
		moduleKey, ok := commitIDToModuleKey[commitID]
		if !ok {
			// This may be a transitive dependency that we don't have. In this case,
			// go out to the API and get the transitive dependency.
			moduleKey, err = getModuleKeyForProtoCommit(
				ctx,
				protoModuleProvider,
				protoOwnerProvider,
				registry,
				protoCommit,
			)
			if err != nil {
				return nil, err
			}
			commitIDToModuleKey[moduleKey.CommitID()] = moduleKey
		}
		graph.AddNode(moduleKey)
	}
	for _, protoEdge := range protoGraph.Edges {
		fromCommitID, err := uuid.FromString(protoEdge.FromCommitId)
		if err != nil {
			return nil, err
		}
		fromModuleKey, ok := commitIDToModuleKey[fromCommitID]
		if !ok {
			// We should always have this after our previous iteration.
			// This could be an API error, but regardless we consider it a system error here.
			return nil, syserror.Newf("did not have commit id %q in commitIDToModuleKey", fromCommitID)
		}
		toCommitID, err := uuid.FromString(protoEdge.ToCommitId)
		if err != nil {
			return nil, err
		}
		toModuleKey, ok := commitIDToModuleKey[toCommitID]
		if !ok {
			// We should always have this after our previous iteration.
			// This could be an API error, but regardless we consider it a system error here.
			return nil, syserror.Newf("did not have commit id %q in commitIDToModuleKey", toCommitID)
		}
		graph.AddEdge(fromModuleKey, toModuleKey)
	}
	return graph, nil
}

func (a *graphProvider) getProtoGraphForRegistryAndModuleKeys(
	ctx context.Context,
	registry string,
	moduleKeys []bufmodule.ModuleKey,
	digestType bufmodule.DigestType,
) (*modulev1beta1.Graph, error) {
	commitIDs := slicesext.Map(moduleKeys, bufmodule.ModuleKey.CommitID)
	protoDigestType, err := digestTypeToProto(digestType)
	if err != nil {
		return nil, err
	}
	response, err := a.clientProvider.GraphServiceClient(registry).GetGraph(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetGraphRequest{
				// TODO: chunking
				ResourceRefs: slicesext.Map(
					commitIDs,
					func(commitID uuid.UUID) *modulev1beta1.ResourceRef {
						return &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: commitID.String(),
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
			// Kind of an abuse of fs.PathError. Is there a way to get a specific ModuleKey out of this?
			return nil, &fs.PathError{Op: "read", Path: err.Error(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	return response.Msg.Graph, nil
}
