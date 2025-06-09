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

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiowner"
	"github.com/bufbuild/buf/private/pkg/dag"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

// NewGraphProvider returns a new GraphProvider for the given API client.
func NewGraphProvider(
	logger *slog.Logger,
	moduleClientProvider interface {
		bufregistryapimodule.V1GraphServiceClientProvider
		bufregistryapimodule.V1ModuleServiceClientProvider
		bufregistryapimodule.V1Beta1GraphServiceClientProvider
	},
	ownerClientProvider bufregistryapiowner.V1OwnerServiceClientProvider,
	options ...GraphProviderOption,
) bufmodule.GraphProvider {
	return newGraphProvider(logger, moduleClientProvider, ownerClientProvider, options...)
}

// GraphProviderOption is an option for a new GraphProvider.
type GraphProviderOption func(*graphProvider)

// GraphProviderWithLegacyFederationRegistry returns a new GraphProviderOption that specifies
// the hostname of an additional registry that is allowed to use legacy federation. This should
// only be used in testing.
func GraphProviderWithLegacyFederationRegistry(legacyFederationRegistry string) GraphProviderOption {
	return func(graphProvider *graphProvider) {
		if legacyFederationRegistry != "" {
			graphProvider.legacyFederationRegistry = legacyFederationRegistry
		}
	}
}

// GraphProviderWithPublicRegistry returns a new GraphProviderOption that specifies
// the hostname of the public registry. By default this is "buf.build", however in testing,
// this may be something else. This is needed to discern which which registry to make calls
// against in the case where there is >1 registries represented in the ModuleKeys - we always
// want to call the non-public registry.
func GraphProviderWithPublicRegistry(publicRegistry string) GraphProviderOption {
	return func(graphProvider *graphProvider) {
		if publicRegistry != "" {
			graphProvider.publicRegistry = publicRegistry
		}
	}
}

// *** PRIVATE ***

type graphProvider struct {
	logger               *slog.Logger
	moduleClientProvider interface {
		bufregistryapimodule.V1GraphServiceClientProvider
		bufregistryapimodule.V1ModuleServiceClientProvider
		bufregistryapimodule.V1Beta1GraphServiceClientProvider
	}
	ownerClientProvider      bufregistryapiowner.V1OwnerServiceClientProvider
	legacyFederationRegistry string
	publicRegistry           string
}

func newGraphProvider(
	logger *slog.Logger,
	moduleClientProvider interface {
		bufregistryapimodule.V1GraphServiceClientProvider
		bufregistryapimodule.V1ModuleServiceClientProvider
		bufregistryapimodule.V1Beta1GraphServiceClientProvider
	},
	ownerClientProvider bufregistryapiowner.V1OwnerServiceClientProvider,
	options ...GraphProviderOption,
) *graphProvider {
	graphProvider := &graphProvider{
		logger:               logger,
		moduleClientProvider: moduleClientProvider,
		ownerClientProvider:  ownerClientProvider,
		publicRegistry:       defaultPublicRegistry,
	}
	for _, option := range options {
		option(graphProvider)
	}
	return graphProvider
}

func (a *graphProvider) GetGraphForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
) (*dag.Graph[bufmodule.RegistryCommitID, bufmodule.ModuleKey], error) {
	graph := dag.NewGraph(bufmodule.ModuleKeyToRegistryCommitID)
	if len(moduleKeys) == 0 {
		return graph, nil
	}
	digestType, err := bufmodule.UniqueDigestTypeForModuleKeys(moduleKeys)
	if err != nil {
		return nil, err
	}

	// We don't want to persist these across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	v1ProtoModuleProvider := newV1ProtoModuleProvider(a.logger, a.moduleClientProvider)
	v1ProtoOwnerProvider := newV1ProtoOwnerProvider(a.logger, a.ownerClientProvider)
	v1beta1ProtoGraph, err := a.getV1Beta1ProtoGraphForModuleKeys(ctx, moduleKeys, digestType)
	if err != nil {
		return nil, err
	}
	registryCommitIDToModuleKey, err := xslices.ToUniqueValuesMapError(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) (bufmodule.RegistryCommitID, error) {
			return bufmodule.ModuleKeyToRegistryCommitID(moduleKey), nil
		},
	)
	if err != nil {
		return nil, err
	}
	for _, v1beta1ProtoGraphCommit := range v1beta1ProtoGraph.Commits {
		v1beta1ProtoCommit := v1beta1ProtoGraphCommit.Commit
		registry := v1beta1ProtoGraphCommit.Registry
		commitID, err := uuidutil.FromDashless(v1beta1ProtoCommit.Id)
		if err != nil {
			return nil, err
		}
		registryCommitID := bufmodule.NewRegistryCommitID(registry, commitID)
		moduleKey, ok := registryCommitIDToModuleKey[registryCommitID]
		if !ok {
			universalProtoCommit, err := newUniversalProtoCommitForV1Beta1(v1beta1ProtoCommit)
			if err != nil {
				return nil, err
			}
			// This may be a transitive dependency that we don't have. In this case,
			// go out to the API and get the transitive dependency.
			moduleKey, err = getModuleKeyForUniversalProtoCommit(
				ctx,
				v1ProtoModuleProvider,
				v1ProtoOwnerProvider,
				registry,
				universalProtoCommit,
			)
			if err != nil {
				return nil, err
			}
			registryCommitIDToModuleKey[registryCommitID] = moduleKey
		}
		graph.AddNode(moduleKey)
	}
	for _, v1beta1ProtoEdge := range v1beta1ProtoGraph.Edges {
		fromRegistry := v1beta1ProtoEdge.FromNode.Registry
		fromCommitID, err := uuidutil.FromDashless(v1beta1ProtoEdge.FromNode.CommitId)
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
		toRegistry := v1beta1ProtoEdge.ToNode.Registry
		toCommitID, err := uuidutil.FromDashless(v1beta1ProtoEdge.ToNode.CommitId)
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

func (a *graphProvider) getV1Beta1ProtoGraphForModuleKeys(
	ctx context.Context,
	moduleKeys []bufmodule.ModuleKey,
	digestType bufmodule.DigestType,
) (*modulev1beta1.Graph, error) {
	primaryRegistry, legacyFederationAllowed, err := getPrimaryRegistryAndLegacyFederationAllowed(
		moduleKeys,
		a.publicRegistry,
		a.legacyFederationRegistry,
	)
	if err != nil {
		return nil, err
	}
	if !legacyFederationAllowed && digestType == bufmodule.DigestTypeB5 {
		// Legacy federation is not allowed, and we are using b5. Call the v1 API.
		graph, err := a.getV1ProtoGraphForRegistryAndModuleKeys(ctx, primaryRegistry, moduleKeys)
		if err != nil {
			return nil, err
		}
		return v1ProtoGraphToV1Beta1ProtoGraph(primaryRegistry, graph)
	}

	// Legacy federation is allowed, or we are using b4. We may have dependencies on modules from other registries, or we
	// are using a digest type not supported by the v1 API. Fall back to the v1beta1 API.

	v1beta1ProtoDigestType, err := digestTypeToV1Beta1Proto(digestType)
	if err != nil {
		return nil, err
	}
	response, err := a.moduleClientProvider.V1Beta1GraphServiceClient(primaryRegistry).GetGraph(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetGraphRequest{
				// TODO FUTURE: chunking
				ResourceRefs: moduleKeysToV1Beta1ProtoGetGraphRequestResourceRefs(moduleKeys),
				DigestType:   v1beta1ProtoDigestType,
			},
		),
	)
	if err != nil {
		return nil, maybeNewNotFoundError(err)
	}

	return response.Msg.Graph, nil
}

func (a *graphProvider) getV1ProtoGraphForRegistryAndModuleKeys(
	ctx context.Context,
	registry string,
	moduleKeys []bufmodule.ModuleKey,
) (*modulev1.Graph, error) {
	response, err := a.moduleClientProvider.V1GraphServiceClient(registry).GetGraph(
		ctx,
		connect.NewRequest(
			&modulev1.GetGraphRequest{
				// TODO FUTURE: chunking
				ResourceRefs: moduleKeysToV1ProtoResourceRefs(moduleKeys),
			},
		),
	)
	if err != nil {
		return nil, maybeNewNotFoundError(err)
	}
	return response.Msg.Graph, nil
}
