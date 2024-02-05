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
	"io/fs"

	federationv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/legacy/federation/v1beta1"
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
		bufapi.LegacyFederationGraphServiceClientProvider
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
		bufapi.LegacyFederationGraphServiceClientProvider
		bufapi.ModuleServiceClientProvider
		bufapi.OwnerServiceClientProvider
	}
}

func newGraphProvider(
	logger *zap.Logger,
	clientProvider interface {
		bufapi.GraphServiceClientProvider
		bufapi.LegacyFederationGraphServiceClientProvider
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
) (*dag.Graph[bufmodule.RegistryCommitID, bufmodule.ModuleKey], error) {
	graph := dag.NewGraph[bufmodule.RegistryCommitID, bufmodule.ModuleKey](bufmodule.ModuleKeyToRegistryCommitID)
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
	protoLegacyFederationGraph, err := a.getProtoLegacyFederationGraphForModuleKeys(ctx, moduleKeys, digestType)
	if err != nil {
		return nil, err
	}
	registryCommitIDToModuleKey, err := slicesext.ToUniqueValuesMapError(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) (bufmodule.RegistryCommitID, error) {
			return bufmodule.ModuleKeyToRegistryCommitID(moduleKey), nil
		},
	)
	if err != nil {
		return nil, err
	}
	for _, protoLegacyFederationCommit := range protoLegacyFederationGraph.Commits {
		registry := protoLegacyFederationCommit.Registry
		commitID, err := uuid.FromString(protoLegacyFederationCommit.Commit.Id)
		if err != nil {
			return nil, err
		}
		registryCommitID := bufmodule.NewRegistryCommitID(registry, commitID)
		moduleKey, ok := registryCommitIDToModuleKey[registryCommitID]
		if !ok {
			// This may be a transitive dependency that we don't have. In this case,
			// go out to the API and get the transitive dependency.
			moduleKey, err = getModuleKeyForProtoCommit(
				ctx,
				protoModuleProvider,
				protoOwnerProvider,
				registry,
				protoLegacyFederationCommit.Commit,
			)
			if err != nil {
				return nil, err
			}
			registryCommitIDToModuleKey[registryCommitID] = moduleKey
		}
		graph.AddNode(moduleKey)
	}
	for _, protoLegacyFederationEdge := range protoLegacyFederationGraph.Edges {
		fromRegistry := protoLegacyFederationEdge.FromNode.Registry
		fromCommitID, err := uuid.FromString(protoLegacyFederationEdge.FromNode.CommitId)
		if err != nil {
			return nil, err
		}
		fromRegistryCommitID := bufmodule.NewRegistryCommitID(fromRegistry, fromCommitID)
		fromModuleKey, ok := registryCommitIDToModuleKey[fromRegistryCommitID]
		if !ok {
			// We should always have this after our previous iteration.
			// This could be an API error, but regardless we consider it a system error here.
			return nil, syserror.Newf("did not have RegistryCommitID %v in registryCommitIDToModuleKey", fromRegistryCommitID)
		}
		toRegistry := protoLegacyFederationEdge.ToNode.Registry
		toCommitID, err := uuid.FromString(protoLegacyFederationEdge.ToNode.CommitId)
		if err != nil {
			return nil, err
		}
		toRegistryCommitID := bufmodule.NewRegistryCommitID(toRegistry, toCommitID)
		toModuleKey, ok := registryCommitIDToModuleKey[toRegistryCommitID]
		if !ok {
			// We should always have this after our previous iteration.
			// This could be an API error, but regardless we consider it a system error here.
			return nil, syserror.Newf("did not have RegistryCommitID %v in registryCommitIDToModuleKey", toRegistryCommitID)
		}
		graph.AddEdge(fromModuleKey, toModuleKey)
	}
	return graph, nil
}

func (a *graphProvider) getProtoLegacyFederationGraphForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
	digestType bufmodule.DigestType,
) (*federationv1beta1.Graph, error) {
	primaryRegistry, secondaryRegistry, err := getPrimarySecondaryRegistry(moduleKeys)
	if err != nil {
		return nil, err
	}
	if secondaryRegistry == "" {
		// If we only have a single registry, invoke the new API endpoint that does not allow
		// for federation. Do this so that we can maintain federated API endpoint metrics.
		graph, err := a.getProtoGraphForRegistryAndModuleKeys(ctx, primaryRegistry, moduleKeys, digestType)
		if err != nil {
			return nil, err
		}
		return protoGraphToProtoLegacyFederationGraph(primaryRegistry, graph), nil
	}

	registryCommitIDs := slicesext.Map(moduleKeys, bufmodule.ModuleKeyToRegistryCommitID)
	protoDigestType, err := digestTypeToProto(digestType)
	if err != nil {
		return nil, err
	}
	response, err := a.clientProvider.LegacyFederationGraphServiceClient(primaryRegistry).GetGraph(
		ctx,
		connect.NewRequest(
			&federationv1beta1.GetGraphRequest{
				// TODO: chunking
				ResourceRefs: slicesext.Map(
					registryCommitIDs,
					func(registryCommitID bufmodule.RegistryCommitID) *federationv1beta1.ResourceRef {
						return &federationv1beta1.ResourceRef{
							Value: &federationv1beta1.ResourceRef_Id{
								Id: registryCommitID.CommitID.String(),
							},
							Registry: registryCommitID.Registry,
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

	for _, commit := range response.Msg.Graph.Commits {
		if err := validateRegistryIsPrimaryOrSecondary(commit.Registry, primaryRegistry, secondaryRegistry); err != nil {
			return nil, err
		}
	}
	for _, edge := range response.Msg.Graph.Edges {
		if err := validateRegistryIsPrimaryOrSecondary(edge.FromNode.Registry, primaryRegistry, secondaryRegistry); err != nil {
			return nil, err
		}
		if err := validateRegistryIsPrimaryOrSecondary(edge.ToNode.Registry, primaryRegistry, secondaryRegistry); err != nil {
			return nil, err
		}
	}

	return response.Msg.Graph, nil
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
