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
	"errors"
	"fmt"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	storagev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/storage/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufnew/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/storage"
)

var (
	// NopModuleDataProvider is a no-op ModuleDataProvider.
	NopModuleDataProvider = nopModuleDataProvider{}
)

// ModuleDataProvider provides ModulesDatas.
type ModuleDataProvider interface {
	// GetModuleDataForModuleKey gets the ModuleData for the ModuleKey.
	//
	// If there is no error, the length of the ModuleDatas returned will match the length of the ModuleKeys.
	// If there is an error, no ModuleDatas will be returned.
	GetModuleDatasForModuleKeys(ctx context.Context, moduleKeys ...ModuleKey) ([]ModuleData, error)
}

// NewAPIModuleDataProvider returns a new ModuleDataProvider for the given API client.
func NewAPIModuleDataProvider(clientProvider bufapi.ClientProvider) ModuleDataProvider {
	return newAPIModuleDataProvider(clientProvider)
}

// *** PRIVATE ***

// apiModuleDataProvider

type apiModuleDataProvider struct {
	clientProvider bufapi.ClientProvider
}

func newAPIModuleDataProvider(clientProvider bufapi.ClientProvider) *apiModuleDataProvider {
	return &apiModuleDataProvider{
		clientProvider: clientProvider,
	}
}

func (a *apiModuleDataProvider) GetModuleDataForModuleKey(
	ctx context.Context,
	moduleKey ModuleKey,
) (ModuleData, error) {
	registryHostname := moduleKey.ModuleFullName().Registry()
	// Note that we could actually just use the Digest. However, we want to force the caller
	// to provide a CommitID, so that we can document that all Modules returned from a
	// ModuleDataProvider will have a CommitID. We also want to prevent callers from having
	// to invoke moduleKey.Digest() unnecessarily, as this could cause unnecessary lazy loading.
	// If we were to instead have GetModuleDataForDigest(context.Context, ModuleFullName, bufcas.Digest),
	// we would never have the CommitID, even in cases where we have it via the ModuleKey.
	// If we were to provide both GetModuleDataForModuleKey and GetModuleForDigest, then why would anyone
	// ever call GetModuleDataForModuleKey? This forces a single call pattern for now.
	response, err := a.clientProvider.CommitServiceClient(registryHostname).GetCommitNodes(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitNodesRequest{
				Values: []*modulev1beta1.GetCommitNodesRequest_Value{
					{
						ResourceRef: &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: moduleKey.CommitID(),
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
	if len(response.Msg.CommitNodes) != 1 {
		return nil, fmt.Errorf("expected 1 CommitNode, got %d", len(response.Msg.CommitNodes))
	}
	protoCommitNode := response.Msg.CommitNodes[0]
	digest, err := bufcas.ProtoToDigest(protoCommitNode.Commit.Digest)
	if err != nil {
		return nil, err
	}
	return NewModuleData(
		moduleKey,
		func() (storage.ReadBucket, error) {
			return a.getBucketForProtoFileNodes(ctx, registryHostname, protoCommitNode.FileNodes)
		},
		func() ([]ModuleKey, error) {
			return a.getModuleKeysForProtoCommits(ctx, registryHostname, protoCommitNode.Deps)
		},
		// TODO: Is this enough for tamper-proofing? With this, we are just calculating the
		// digest that we got back from the API, as opposed to re-calculating the digest based
		// on the data. This is saying we trust the API to produce the correct digest for the
		// data it is returning. An argument could be made we should not, but that argument is shaky.
		//
		// We could go a step further and calculate based on the actual data, but doing this lazily
		// is additional work (but very possible).
		ModuleDataWithActualDigest(digest),
	)
}

func (a *apiModuleDataProvider) getBucketForProtoFileNodes(
	ctx context.Context,
	registryHostname string,
	protoFileNodes []*storagev1beta1.FileNode,
) (storage.ReadBucket, error) {
	return nil, errors.New("TODO")
}

func (a *apiModuleDataProvider) getModuleKeysForProtoCommits(
	ctx context.Context,
	registryHostname string,
	protoCommits []*modulev1beta1.Commit,
) ([]ModuleKey, error) {
	return nil, errors.New("TODO")
}

// nopModuleDataProvider

type nopModuleDataProvider struct{}

func (nopModuleDataProvider) GetModuleDataForModuleKey(context.Context, ModuleKey) (ModuleData, error) {
	return nil, errors.New("nopModuleDataProvider")
}
