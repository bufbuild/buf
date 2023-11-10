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
	ownerv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufnew/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

// ModuleKeyProvider provides ModuleKeys for ModuleRefs.
type ModuleKeyProvider interface {
	// GetModuleKeyForModuleRef gets the ModuleKey for the given ModuleRef.
	//
	// Resolution of the ModuleRef is done per the ModuleRef documentation.
	GetModuleKeyForModuleRef(context.Context, ModuleRef) (ModuleKey, error)
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

func (a *apiModuleKeyProvider) GetModuleKeyForModuleRef(ctx context.Context, moduleRef ModuleRef) (ModuleKey, error) {
	protoCommit, err := a.getProtoCommitForModuleRef(ctx, moduleRef)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return newModuleKeyForLazyDigest(
		moduleRef.ModuleFullName(),
		protoCommit.Id,
		func() (bufcas.Digest, error) {
			// Do not call getModuleKeyForProtoCommit, we already have the owner and module names.
			return bufcas.ProtoToDigest(protoCommit.Digest)
		},
	)
}

// All of this stuff we may want in some other common place such as bufapi, with interfaces.

// If you have the owner and module names, do not use this! This makes two extra calls to get the owner and module names.
func (a *apiModuleKeyProvider) getModuleKeyForProtoCommit(ctx context.Context, registryHostname string, protoCommit *modulev1beta1.Commit) (ModuleKey, error) {
	protoModule, err := a.getProtoModuleForModuleID(ctx, registryHostname, protoCommit.ModuleId)
	if err != nil {
		return nil, err
	}
	protoOwner, err := a.getProtoOwnerForOwnerID(ctx, registryHostname, protoCommit.OwnerId)
	if err != nil {
		return nil, err
	}
	var ownerName string
	switch {
	case protoOwner.GetUser() != nil:
		ownerName = protoOwner.GetUser().Name
	case protoOwner.GetOrganization() != nil:
		ownerName = protoOwner.GetOrganization().Name
	default:
		return nil, fmt.Errorf("proto Owner did not have a User or Organization: %v", protoOwner)
	}
	moduleFullName, err := newModuleFullName(
		registryHostname,
		ownerName,
		protoModule.Name,
	)
	if err != nil {
		return nil, err
	}
	return newModuleKeyForLazyDigest(
		moduleFullName,
		protoCommit.Id,
		func() (bufcas.Digest, error) {
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

func (a *apiModuleKeyProvider) getProtoModuleForModuleID(ctx context.Context, registryHostname string, moduleID string) (*modulev1beta1.Module, error) {
	response, err := a.clientProvider.ModuleServiceClient(registryHostname).GetModules(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetModulesRequest{
				ModuleRefs: []*modulev1beta1.ModuleRef{
					{
						Value: &modulev1beta1.ModuleRef_Id{
							Id: moduleID,
						},
					},
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Msg.Modules) != 1 {
		return nil, fmt.Errorf("expected 1 Module, got %d", len(response.Msg.Modules))
	}
	return response.Msg.Modules[0], nil
}

func (a *apiModuleKeyProvider) getProtoOwnerForOwnerID(ctx context.Context, registryHostname string, ownerID string) (*ownerv1beta1.Owner, error) {
	response, err := a.clientProvider.OwnerServiceClient(registryHostname).GetOwners(
		ctx,
		connect.NewRequest(
			&ownerv1beta1.GetOwnersRequest{
				OwnerRefs: []*ownerv1beta1.OwnerRef{
					{
						Value: &ownerv1beta1.OwnerRef_Id{
							Id: ownerID,
						},
					},
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Msg.Owners) != 1 {
		return nil, fmt.Errorf("expected 1 Owner, got %d", len(response.Msg.Owners))
	}
	return response.Msg.Owners[0], nil
}
