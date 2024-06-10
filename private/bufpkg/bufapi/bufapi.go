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

package bufapi

import (
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1/modulev1connect"
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1beta1/modulev1beta1connect"
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/owner/v1/ownerv1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

var (
	// NopV1CommitServiceClientProvider is a V1CommitServiceClientProvider that provides unimplemented services for testing.
	NopV1CommitServiceClientProvider V1CommitServiceClientProvider = nopClientProvider{}
	// NopV1DownloadServiceClientProvider is a V1DownloadServiceClientProvider that provides unimplemented services for testing.
	NopV1DownloadServiceClientProvider V1DownloadServiceClientProvider = nopClientProvider{}
	// NopV1GraphServiceClientProvider is a V1GraphServiceClientProvider that provides unimplemented services for testing.
	NopV1GraphServiceClientProvider V1GraphServiceClientProvider = nopClientProvider{}
	// NopV1LabelServiceClientProvider is a V1LabelServiceClientProvider that provides unimplemented services for testing.
	NopV1LabelServiceClientProvider V1LabelServiceClientProvider = nopClientProvider{}
	// NopV1ModuleServiceClientProvider is a V1ModuleServiceClientProvider that provides unimplemented services for testing.
	NopV1ModuleServiceClientProvider V1ModuleServiceClientProvider = nopClientProvider{}
	// NopV1UploadServiceClientProvider is a V1UploadServiceClientProvider that provides unimplemented services for testing.
	NopV1UploadServiceClientProvider V1UploadServiceClientProvider = nopClientProvider{}
	// NopV1OrganizationServiceClientProvider is a V1OrganizationServiceClientProvider that provides unimplemented services for testing.
	NopV1OrganizationServiceClientProvider V1OrganizationServiceClientProvider = nopClientProvider{}
	// NopV1OwnerServiceClientProvider is a V1OwnerServiceClientProvider that provides unimplemented services for testing.
	NopV1OwnerServiceClientProvider V1OwnerServiceClientProvider = nopClientProvider{}
	// NopV1UserServiceClientProvider is a V1UserServiceClientProvider that provides unimplemented services for testing.
	NopV1UserServiceClientProvider V1UserServiceClientProvider = nopClientProvider{}
	// NopV1Beta1CommitServiceClientProvider is a V1Beta1CommitServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1CommitServiceClientProvider V1Beta1CommitServiceClientProvider = nopClientProvider{}
	// NopV1Beta1DownloadServiceClientProvider is a V1Beta1DownloadServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1DownloadServiceClientProvider V1Beta1DownloadServiceClientProvider = nopClientProvider{}
	// NopV1Beta1GraphServiceClientProvider is a V1Beta1GraphServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1GraphServiceClientProvider V1Beta1GraphServiceClientProvider = nopClientProvider{}
	// NopV1Beta1LabelServiceClientProvider is a V1Beta1LabelServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1LabelServiceClientProvider V1Beta1LabelServiceClientProvider = nopClientProvider{}
	// NopV1Beta1ModuleServiceClientProvider is a V1Beta1ModuleServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1ModuleServiceClientProvider V1Beta1ModuleServiceClientProvider = nopClientProvider{}
	// NopV1Beta1UploadServiceClientProvider is a V1Beta1UploadServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1UploadServiceClientProvider V1Beta1UploadServiceClientProvider = nopClientProvider{}
	// NopClientProvider is a ClientProvider that provides unimplemented services for testing.
	NopClientProvider ClientProvider = nopClientProvider{}
)

// V1CommitServiceClientProvider provides CommitServiceClients.
type V1CommitServiceClientProvider interface {
	V1CommitServiceClient(registry string) modulev1connect.CommitServiceClient
}

// V1DownloadServiceClientProvider provides DownloadServiceClients.
type V1DownloadServiceClientProvider interface {
	V1DownloadServiceClient(registry string) modulev1connect.DownloadServiceClient
}

// V1GraphServiceClientProvider provides GraphServiceClients.
type V1GraphServiceClientProvider interface {
	V1GraphServiceClient(registry string) modulev1connect.GraphServiceClient
}

// V1LabelServiceClientProvider provides LabelServiceClients.
type V1LabelServiceClientProvider interface {
	V1LabelServiceClient(registry string) modulev1connect.LabelServiceClient
}

// V1ModuleServiceClientProvider provides ModuleServiceClients.
type V1ModuleServiceClientProvider interface {
	V1ModuleServiceClient(registry string) modulev1connect.ModuleServiceClient
}

// V1ResourceServiceClientProvider provides ResourceServiceClients.
type V1ResourceServiceClientProvider interface {
	V1ResourceServiceClient(registry string) modulev1connect.ResourceServiceClient
}

// V1UploadServiceClientProvider provides UploadServiceClients.
type V1UploadServiceClientProvider interface {
	V1UploadServiceClient(registry string) modulev1connect.UploadServiceClient
}

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

// V1Beta1CommitServiceClientProvider provides CommitServiceClients.
type V1Beta1CommitServiceClientProvider interface {
	V1Beta1CommitServiceClient(registry string) modulev1beta1connect.CommitServiceClient
}

// V1Beta1DownloadServiceClientProvider provides DownloadServiceClients.
type V1Beta1DownloadServiceClientProvider interface {
	V1Beta1DownloadServiceClient(registry string) modulev1beta1connect.DownloadServiceClient
}

// V1Beta1GraphServiceClientProvider provides GraphServiceClients.
type V1Beta1GraphServiceClientProvider interface {
	V1Beta1GraphServiceClient(registry string) modulev1beta1connect.GraphServiceClient
}

// V1Beta1LabelServiceClientProvider provides LabelServiceClients.
type V1Beta1LabelServiceClientProvider interface {
	V1Beta1LabelServiceClient(registry string) modulev1beta1connect.LabelServiceClient
}

// V1Beta1ModuleServiceClientProvider provides ModuleServiceClients.
type V1Beta1ModuleServiceClientProvider interface {
	V1Beta1ModuleServiceClient(registry string) modulev1beta1connect.ModuleServiceClient
}

// V1Beta1UploadServiceClientProvider provides UploadServiceClients.
type V1Beta1UploadServiceClientProvider interface {
	V1Beta1UploadServiceClient(registry string) modulev1beta1connect.UploadServiceClient
}

// ClientProvider provides API clients for BSR services.
type ClientProvider interface {
	V1CommitServiceClientProvider
	V1DownloadServiceClientProvider
	V1GraphServiceClientProvider
	V1LabelServiceClientProvider
	V1ModuleServiceClientProvider
	V1ResourceServiceClientProvider
	V1UploadServiceClientProvider
	V1OrganizationServiceClientProvider
	V1OwnerServiceClientProvider
	V1UserServiceClientProvider
	V1Beta1CommitServiceClientProvider
	V1Beta1DownloadServiceClientProvider
	V1Beta1GraphServiceClientProvider
	V1Beta1LabelServiceClientProvider
	V1Beta1ModuleServiceClientProvider
	V1Beta1UploadServiceClientProvider
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

func (c *clientProvider) V1CommitServiceClient(registry string) modulev1connect.CommitServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1connect.NewCommitServiceClient,
	)
}

func (c *clientProvider) V1DownloadServiceClient(registry string) modulev1connect.DownloadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1connect.NewDownloadServiceClient,
	)
}

func (c *clientProvider) V1GraphServiceClient(registry string) modulev1connect.GraphServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1connect.NewGraphServiceClient,
	)
}

func (c *clientProvider) V1LabelServiceClient(registry string) modulev1connect.LabelServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1connect.NewLabelServiceClient,
	)
}

func (c *clientProvider) V1ModuleServiceClient(registry string) modulev1connect.ModuleServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1connect.NewModuleServiceClient,
	)
}

func (c *clientProvider) V1ResourceServiceClient(registry string) modulev1connect.ResourceServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1connect.NewResourceServiceClient,
	)
}

func (c *clientProvider) V1UploadServiceClient(registry string) modulev1connect.UploadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1connect.NewUploadServiceClient,
	)
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

func (c *clientProvider) V1Beta1CommitServiceClient(registry string) modulev1beta1connect.CommitServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewCommitServiceClient,
	)
}

func (c *clientProvider) V1Beta1DownloadServiceClient(registry string) modulev1beta1connect.DownloadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewDownloadServiceClient,
	)
}

func (c *clientProvider) V1Beta1GraphServiceClient(registry string) modulev1beta1connect.GraphServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewGraphServiceClient,
	)
}

func (c *clientProvider) V1Beta1LabelServiceClient(registry string) modulev1beta1connect.LabelServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewLabelServiceClient,
	)
}

func (c *clientProvider) V1Beta1ModuleServiceClient(registry string) modulev1beta1connect.ModuleServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewModuleServiceClient,
	)
}

func (c *clientProvider) V1Beta1UploadServiceClient(registry string) modulev1beta1connect.UploadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		modulev1beta1connect.NewUploadServiceClient,
	)
}

type nopClientProvider struct{}

func (nopClientProvider) V1CommitServiceClient(registry string) modulev1connect.CommitServiceClient {
	return modulev1connect.UnimplementedCommitServiceHandler{}
}

func (nopClientProvider) V1DownloadServiceClient(registry string) modulev1connect.DownloadServiceClient {
	return modulev1connect.UnimplementedDownloadServiceHandler{}
}

func (nopClientProvider) V1GraphServiceClient(registry string) modulev1connect.GraphServiceClient {
	return modulev1connect.UnimplementedGraphServiceHandler{}
}

func (nopClientProvider) V1LabelServiceClient(registry string) modulev1connect.LabelServiceClient {
	return modulev1connect.UnimplementedLabelServiceHandler{}
}

func (nopClientProvider) V1ModuleServiceClient(registry string) modulev1connect.ModuleServiceClient {
	return modulev1connect.UnimplementedModuleServiceHandler{}
}

func (nopClientProvider) V1ResourceServiceClient(registry string) modulev1connect.ResourceServiceClient {
	return modulev1connect.UnimplementedResourceServiceHandler{}
}

func (nopClientProvider) V1UploadServiceClient(registry string) modulev1connect.UploadServiceClient {
	return modulev1connect.UnimplementedUploadServiceHandler{}
}

func (nopClientProvider) V1OrganizationServiceClient(registry string) ownerv1connect.OrganizationServiceClient {
	return ownerv1connect.UnimplementedOrganizationServiceHandler{}
}

func (nopClientProvider) V1OwnerServiceClient(registry string) ownerv1connect.OwnerServiceClient {
	return ownerv1connect.UnimplementedOwnerServiceHandler{}
}

func (nopClientProvider) V1UserServiceClient(registry string) ownerv1connect.UserServiceClient {
	return ownerv1connect.UnimplementedUserServiceHandler{}
}

func (nopClientProvider) V1Beta1CommitServiceClient(registry string) modulev1beta1connect.CommitServiceClient {
	return modulev1beta1connect.UnimplementedCommitServiceHandler{}
}

func (nopClientProvider) V1Beta1DownloadServiceClient(registry string) modulev1beta1connect.DownloadServiceClient {
	return modulev1beta1connect.UnimplementedDownloadServiceHandler{}
}

func (nopClientProvider) V1Beta1GraphServiceClient(registry string) modulev1beta1connect.GraphServiceClient {
	return modulev1beta1connect.UnimplementedGraphServiceHandler{}
}

func (nopClientProvider) V1Beta1LabelServiceClient(registry string) modulev1beta1connect.LabelServiceClient {
	return modulev1beta1connect.UnimplementedLabelServiceHandler{}
}

func (nopClientProvider) V1Beta1ModuleServiceClient(registry string) modulev1beta1connect.ModuleServiceClient {
	return modulev1beta1connect.UnimplementedModuleServiceHandler{}
}

func (nopClientProvider) V1Beta1UploadServiceClient(registry string) modulev1beta1connect.UploadServiceClient {
	return modulev1beta1connect.UnimplementedUploadServiceHandler{}
}
