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

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufnew/bufapi"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"go.uber.org/zap"
)

// NewModuleKeyProvider returns a new ModuleKeyProvider for the given API clients.
func NewModuleKeyProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) bufmodule.ModuleKeyProvider {
	return newModuleKeyProvider(logger, clientProvider)
}

// *** PRIVATE ***

type moduleKeyProvider struct {
	logger         *zap.Logger
	clientProvider bufapi.ClientProvider
}

func newModuleKeyProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) *moduleKeyProvider {
	return &moduleKeyProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *moduleKeyProvider) GetModuleKeysForModuleRefs(ctx context.Context, moduleRefs ...bufmodule.ModuleRef) ([]bufmodule.ModuleKey, error) {
	// TODO: Do the work to coalesce ModuleRefs by registry hostname, make calls out to the CommitService
	// per registry, then get back the resulting data, and order it in the same order as the input ModuleRefs.
	// Make sure to respect 250 max.
	moduleKeys := make([]bufmodule.ModuleKey, len(moduleRefs))
	for i, moduleRef := range moduleRefs {
		moduleKey, err := a.getModuleKeyForModuleRef(ctx, moduleRef)
		if err != nil {
			return nil, err
		}
		moduleKeys[i] = moduleKey
	}
	return moduleKeys, nil
}

func (a *moduleKeyProvider) getModuleKeyForModuleRef(ctx context.Context, moduleRef bufmodule.ModuleRef) (bufmodule.ModuleKey, error) {
	protoCommit, err := a.getProtoCommitForModuleRef(ctx, moduleRef)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleKey(
		// Note we don't have to resolve owner_name and module_name since we already have them.
		moduleRef.ModuleFullName(),
		protoCommit.Id,
		func() (bufcas.Digest, error) {
			// Do not call getModuleKeyForProtoCommit, we already have the owner and module names.
			return bufcas.ProtoToDigest(protoCommit.Digest)
		},
	)
}

func (a *moduleKeyProvider) getProtoCommitForModuleRef(ctx context.Context, moduleRef bufmodule.ModuleRef) (*modulev1beta1.Commit, error) {
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
