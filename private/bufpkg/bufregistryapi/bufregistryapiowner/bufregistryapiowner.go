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

package bufregistryapiowner

import (
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/owner/v1/ownerv1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

var (
	// NopV1OrganizationServiceClientProvider is a V1OrganizationServiceClientProvider that provides unimplemented services for testing.
	NopV1OrganizationServiceClientProvider V1OrganizationServiceClientProvider = nopClientProvider{}
	// NopV1OwnerServiceClientProvider is a V1OwnerServiceClientProvider that provides unimplemented services for testing.
	NopV1OwnerServiceClientProvider V1OwnerServiceClientProvider = nopClientProvider{}
	// NopV1UserServiceClientProvider is a V1UserServiceClientProvider that provides unimplemented services for testing.
	NopV1UserServiceClientProvider V1UserServiceClientProvider = nopClientProvider{}
	// NopClientProvider is a ClientProvider that provides unimplemented services for testing.
	NopClientProvider ClientProvider = nopClientProvider{}
)

// V1OrganizationServiceClientProvider provides OrganizationServiceClients.
type V1OrganizationServiceClientProvider interface {
	V1OrganizationServiceClient(registry string) ownerv1connect.OrganizationServiceClient
}

// V1OwnerServiceClientProvider provides OwnerServiceClients.
type V1OwnerServiceClientProvider interface {
	V1OwnerServiceClient(registry string) ownerv1connect.OwnerServiceClient
}

// V1UserServiceClientProvider provides UserServiceClients.
type V1UserServiceClientProvider interface {
	V1UserServiceClient(registry string) ownerv1connect.UserServiceClient
}

// ClientProvider provides API clients for BSR services.
type ClientProvider interface {
	V1OrganizationServiceClientProvider
	V1OwnerServiceClientProvider
	V1UserServiceClientProvider
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

func (c *clientProvider) V1OrganizationServiceClient(registry string) ownerv1connect.OrganizationServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		ownerv1connect.NewOrganizationServiceClient,
	)
}

func (c *clientProvider) V1OwnerServiceClient(registry string) ownerv1connect.OwnerServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		ownerv1connect.NewOwnerServiceClient,
	)
}

func (c *clientProvider) V1UserServiceClient(registry string) ownerv1connect.UserServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		ownerv1connect.NewUserServiceClient,
	)
}

type nopClientProvider struct{}

func (nopClientProvider) V1OrganizationServiceClient(registry string) ownerv1connect.OrganizationServiceClient {
	return ownerv1connect.UnimplementedOrganizationServiceHandler{}
}

func (nopClientProvider) V1OwnerServiceClient(registry string) ownerv1connect.OwnerServiceClient {
	return ownerv1connect.UnimplementedOwnerServiceHandler{}
}

func (nopClientProvider) V1UserServiceClient(registry string) ownerv1connect.UserServiceClient {
	return ownerv1connect.UnimplementedUserServiceHandler{}
}
