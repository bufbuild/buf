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

package bufapi

import (
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1beta1/modulev1beta1connect"
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/owner/v1beta1/ownerv1beta1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

// ClientProvider provides API clients for BSR services.
type ClientProvider interface {
	BranchServiceClient(registryHostname string) modulev1beta1connect.BranchServiceClient
	CommitServiceClient(registryHostname string) modulev1beta1connect.CommitServiceClient
	ModuleServiceClient(registryHostname string) modulev1beta1connect.ModuleServiceClient
	OrganizationServiceClient(registryHostname string) ownerv1beta1connect.OrganizationServiceClient
	OwnerServiceClient(registryHostname string) ownerv1beta1connect.OwnerServiceClient
	TagServiceClient(registryHostname string) modulev1beta1connect.TagServiceClient
	UserServiceClient(registryHostname string) ownerv1beta1connect.UserServiceClient
	VCSCommitServiceClient(registryHostname string) modulev1beta1connect.VCSCommitServiceClient
}

// NewClientProvider returns a new ClientProvider.
func NewClientProvider(clientConfig *connectclient.Config) ClientProvider {
	return newClientProvider(clientConfig)
}

// *** PRIVATE ***

type clientProvider struct {
	clientConfig *connectclient.Config
}

func newClientProvider(clientConfig *connectclient.Config) *clientProvider {
	return &clientProvider{
		clientConfig: clientConfig,
	}
}

func (c *clientProvider) BranchServiceClient(registryHostname string) modulev1beta1connect.BranchServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		modulev1beta1connect.NewBranchServiceClient,
	)
}

func (c *clientProvider) CommitServiceClient(registryHostname string) modulev1beta1connect.CommitServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		modulev1beta1connect.NewCommitServiceClient,
	)
}

func (c *clientProvider) ModuleServiceClient(registryHostname string) modulev1beta1connect.ModuleServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		modulev1beta1connect.NewModuleServiceClient,
	)
}

func (c *clientProvider) OrganizationServiceClient(registryHostname string) ownerv1beta1connect.OrganizationServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		ownerv1beta1connect.NewOrganizationServiceClient,
	)
}

func (c *clientProvider) OwnerServiceClient(registryHostname string) ownerv1beta1connect.OwnerServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		ownerv1beta1connect.NewOwnerServiceClient,
	)
}

func (c *clientProvider) TagServiceClient(registryHostname string) modulev1beta1connect.TagServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		modulev1beta1connect.NewTagServiceClient,
	)
}

func (c *clientProvider) UserServiceClient(registryHostname string) ownerv1beta1connect.UserServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		ownerv1beta1connect.NewUserServiceClient,
	)
}

func (c *clientProvider) VCSCommitServiceClient(registryHostname string) modulev1beta1connect.VCSCommitServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registryHostname,
		modulev1beta1connect.NewVCSCommitServiceClient,
	)
}
