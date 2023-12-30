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

var (
	// NopCommitServiceClientProvider is a CommitServiceClientProvider that provides unimplemented services for testing.
	NopCommitServiceClientProvider CommitServiceClientProvider = nopClientProvider{}
	// NopDownloadServiceClientProvider is a DownloadServiceClientProvider that provides unimplemented services for testing.
	NopDownloadServiceClientProvider DownloadServiceClientProvider = nopClientProvider{}
	// NopGraphServiceClientProvider is a GraphServiceClientProvider that provides unimplemented services for testing.
	NopGraphServiceClientProvider GraphServiceClientProvider = nopClientProvider{}
	// NopLabelServiceClientProvider is a LabelServiceClientProvider that provides unimplemented services for testing.
	NopLabelServiceClientProvider LabelServiceClientProvider = nopClientProvider{}
	// NopModuleServiceClientProvider is a ModuleServiceClientProvider that provides unimplemented services for testing.
	NopModuleServiceClientProvider ModuleServiceClientProvider = nopClientProvider{}
	// NopOrganizationServiceClientProvider is a OrganizationServiceClientProvider that provides unimplemented services for testing.
	NopOrganizationServiceClientProvider OrganizationServiceClientProvider = nopClientProvider{}
	// NopOwnerServiceClientProvider is a OwnerServiceClientProvider that provides unimplemented services for testing.
	NopOwnerServiceClientProvider OwnerServiceClientProvider = nopClientProvider{}
	// NopUploadServiceClientProvider is a UploadServiceClientProvider that provides unimplemented services for testing.
	NopUploadServiceClientProvider UploadServiceClientProvider = nopClientProvider{}
	// NopUserServiceClientProvider is a UserServiceClientProvider that provides unimplemented services for testing.
	NopUserServiceClientProvider UserServiceClientProvider = nopClientProvider{}
	// NopClientProvider is a ClientProvider that provides unimplemented services for testing.
	NopClientProvider ClientProvider = nopClientProvider{}
)

// CommitServiceClientProvider provides CommitServiceClients.
type CommitServiceClientProvider interface {
	CommitServiceClient(registry string) modulev1beta1connect.CommitServiceClient
}

// DownloadServiceClientProvider provides DownloadServiceClients.
type DownloadServiceClientProvider interface {
	DownloadServiceClient(registry string) modulev1beta1connect.DownloadServiceClient
}

// GraphServiceClientProvider provides GraphServiceClients.
type GraphServiceClientProvider interface {
	GraphServiceClient(registry string) modulev1beta1connect.GraphServiceClient
}

// LabelServiceClientProvider provides LabelServiceClients.
type LabelServiceClientProvider interface {
	LabelServiceClient(registry string) modulev1beta1connect.LabelServiceClient
}

// ModuleServiceClientProvider provides ModuleServiceClients.
type ModuleServiceClientProvider interface {
	ModuleServiceClient(registry string) modulev1beta1connect.ModuleServiceClient
}

// OrganizationServiceClientProvider provides OrganizationServiceClients.
type OrganizationServiceClientProvider interface {
	OrganizationServiceClient(registry string) ownerv1beta1connect.OrganizationServiceClient
}

// OwnerServiceClientProvider provides OwnerServiceClients.
type OwnerServiceClientProvider interface {
	OwnerServiceClient(registry string) ownerv1beta1connect.OwnerServiceClient
}

// UploadServiceClientProvider provides UploadServiceClients.
type UploadServiceClientProvider interface {
	UploadServiceClient(registry string) modulev1beta1connect.UploadServiceClient
}

// UserServiceClientProvider provides UserServiceClients.
type UserServiceClientProvider interface {
	UserServiceClient(registry string) ownerv1beta1connect.UserServiceClient
}

// ClientProvider provides API clients for BSR services.
type ClientProvider interface {
	CommitServiceClientProvider
	DownloadServiceClientProvider
	GraphServiceClientProvider
	LabelServiceClientProvider
	ModuleServiceClientProvider
	OrganizationServiceClientProvider
	OwnerServiceClientProvider
	UploadServiceClientProvider
	UserServiceClientProvider
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

func (c *clientProvider) CommitServiceClient(registry string) modulev1beta1connect.CommitServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewCommitServiceClient,
	)
}

func (c *clientProvider) DownloadServiceClient(registry string) modulev1beta1connect.DownloadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewDownloadServiceClient,
	)
}

func (c *clientProvider) GraphServiceClient(registry string) modulev1beta1connect.GraphServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewGraphServiceClient,
	)
}

func (c *clientProvider) LabelServiceClient(registry string) modulev1beta1connect.LabelServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewLabelServiceClient,
	)
}

func (c *clientProvider) ModuleServiceClient(registry string) modulev1beta1connect.ModuleServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewModuleServiceClient,
	)
}

func (c *clientProvider) OrganizationServiceClient(registry string) ownerv1beta1connect.OrganizationServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		ownerv1beta1connect.NewOrganizationServiceClient,
	)
}

func (c *clientProvider) OwnerServiceClient(registry string) ownerv1beta1connect.OwnerServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		ownerv1beta1connect.NewOwnerServiceClient,
	)
}

func (c *clientProvider) UploadServiceClient(registry string) modulev1beta1connect.UploadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewUploadServiceClient,
	)
}

func (c *clientProvider) UserServiceClient(registry string) ownerv1beta1connect.UserServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		ownerv1beta1connect.NewUserServiceClient,
	)
}

type nopClientProvider struct{}

func (nopClientProvider) CommitServiceClient(registry string) modulev1beta1connect.CommitServiceClient {
	return modulev1beta1connect.UnimplementedCommitServiceHandler{}
}

func (nopClientProvider) DownloadServiceClient(registry string) modulev1beta1connect.DownloadServiceClient {
	return modulev1beta1connect.UnimplementedDownloadServiceHandler{}
}

func (nopClientProvider) GraphServiceClient(registry string) modulev1beta1connect.GraphServiceClient {
	return modulev1beta1connect.UnimplementedGraphServiceHandler{}
}

func (nopClientProvider) LabelServiceClient(registry string) modulev1beta1connect.LabelServiceClient {
	return modulev1beta1connect.UnimplementedLabelServiceHandler{}
}

func (nopClientProvider) ModuleServiceClient(registry string) modulev1beta1connect.ModuleServiceClient {
	return modulev1beta1connect.UnimplementedModuleServiceHandler{}
}

func (nopClientProvider) OrganizationServiceClient(registry string) ownerv1beta1connect.OrganizationServiceClient {
	return ownerv1beta1connect.UnimplementedOrganizationServiceHandler{}
}

func (nopClientProvider) OwnerServiceClient(registry string) ownerv1beta1connect.OwnerServiceClient {
	return ownerv1beta1connect.UnimplementedOwnerServiceHandler{}
}

func (nopClientProvider) UploadServiceClient(registry string) modulev1beta1connect.UploadServiceClient {
	return modulev1beta1connect.UnimplementedUploadServiceHandler{}
}

func (nopClientProvider) UserServiceClient(registry string) ownerv1beta1connect.UserServiceClient {
	return ownerv1beta1connect.UnimplementedUserServiceHandler{}
}
