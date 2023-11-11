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

package bufmodule

import (
	"context"
	"fmt"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufnew/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// ModuleKeyProvider provides ModuleKeys for ModuleRefs.
type ModuleKeyProvider interface {
	// GetModuleKeysForModuleRefs gets the ModuleKeys for the given ModuleRefs.
	//
	// Resolution of the ModuleRefs is done per the ModuleRef documentation.
	//
	// If there is no error, the length of the ModuleKeys returned will match the length of the ModuleRefs.
	// If there is an error, no ModuleKeys will be returned.
	GetModuleKeysForModuleRefs(context.Context, ...ModuleRef) ([]ModuleKey, error)
}

// NewAPIModuleKeyProvider returns a new ModuleKeyProvider for the given API clients.
func NewAPIModuleKeyProvider(clientProvider bufapi.ClientProvider) ModuleKeyProvider {
	return newAPIModuleKeyProvider(clientProvider)
}

type apiModuleKeyProvider struct {
	clientProvider bufapi.ClientProvider
}

func newAPIModuleKeyProvider(clientProvider bufapi.ClientProvider) *apiModuleKeyProvider {
	return &apiModuleKeyProvider{
		clientProvider: clientProvider,
	}
}

func (a *apiModuleKeyProvider) GetModuleKeysForModuleRefs(ctx context.Context, moduleRefs ...ModuleRef) ([]ModuleKey, error) {
	// TODO: Do the work to coalesce ModuleRefs by registry hostname, make calls out to the CommitService
	// per registry, then get back the resulting data, and order it in the same order as the input ModuleRefs.
	// Make sure to respect 250 max.
	moduleKeys := make([]ModuleKey, len(moduleRefs))
	for i, moduleRef := range moduleRefs {
		moduleKey, err := a.getModuleKeyForModuleRef(ctx, moduleRef)
		if err != nil {
			return nil, err
		}
		moduleKeys[i] = moduleKey
	}
	return moduleKeys, nil
}

func (a *apiModuleKeyProvider) getModuleKeyForModuleRef(ctx context.Context, moduleRef ModuleRef) (ModuleKey, error) {
	protoCommit, err := a.getProtoCommitForModuleRef(ctx, moduleRef)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return newModuleKeyForLazyDigest(
		// Note we don't have to resolve owner_name and module_name since we already have them.
		moduleRef.ModuleFullName(),
		protoCommit.Id,
		func() (bufcas.Digest, error) {
			// Do not call getModuleKeyForProtoCommit, we already have the owner and module names.
			return bufcas.ProtoToDigest(protoCommit.Digest)
		},
	)
}

func (a *apiModuleKeyProvider) getProtoCommitForModuleRef(ctx context.Context, moduleRef ModuleRef) (*modulev1beta1.Commit, error) {
	response, err := a.clientProvider.CommitServiceClient(moduleRef.ModuleFullName().Registry()).ResolveCommits(
		ctx,
		connect.NewRequest(
			&modulev1beta1.ResolveCommitsRequest{
				ResourceRefs: []*modulev1beta1.ResourceRef{
					{
						Value: &modulev1beta1.ResourceRef_Name_{
							Name: &modulev1beta1.ResourceRef_Name{
								Owner:  moduleRef.ModuleFullName().Owner(),
								Module: moduleRef.ModuleFullName().Name(),
								Child: &modulev1beta1.ResourceRef_Name_Ref{
									Ref: moduleRef.Ref(),
								},
							},
						},
					},
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Msg.Commits) != 1 {
		return nil, fmt.Errorf("expected 1 Commit, got %d", len(response.Msg.Commits))
	}
	return response.Msg.Commits[0], nil
}
