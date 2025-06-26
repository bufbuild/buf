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

package bufregistryapipolicy

import (
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/policy/v1beta1/policyv1beta1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

var (
	// NopV1Beta1CommitServiceClientProvider is a V1Beta1CommitServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1CommitServiceClientProvider V1Beta1CommitServiceClientProvider = nopClientProvider{}
	// NopV1Beta1DownloadServiceClientProvider is a V1Beta1DownloadServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1DownloadServiceClientProvider V1Beta1DownloadServiceClientProvider = nopClientProvider{}
	// NopV1Beta1LabelServiceClientProvider is a V1Beta1LabelServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1LabelServiceClientProvider V1Beta1LabelServiceClientProvider = nopClientProvider{}
	// NopV1Beta1PolicyServiceClientProvider is a V1Beta1PolicyServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1PolicyServiceClientProvider V1Beta1PolicyServiceClientProvider = nopClientProvider{}
	// NopV1Beta1ResourceServiceClientProvider is a V1Beta1ResourceServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1ResourceServiceClientProvider V1Beta1ResourceServiceClientProvider = nopClientProvider{}
	// NopV1Beta1UploadServiceClientProvider is a V1Beta1UploadServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1UploadServiceClientProvider V1Beta1UploadServiceClientProvider = nopClientProvider{}
	// NopClientProvider is a ClientProvider that provides unimplemented services for testing.
	NopClientProvider ClientProvider = nopClientProvider{}
)

// V1Beta1CommitServiceClientProvider provides CommitServiceClients.
type V1Beta1CommitServiceClientProvider interface {
	V1Beta1CommitServiceClient(registry string) policyv1beta1connect.CommitServiceClient
}

// V1Beta1DownloadServiceClientProvider provides DownloadServiceClients.
type V1Beta1DownloadServiceClientProvider interface {
	V1Beta1DownloadServiceClient(registry string) policyv1beta1connect.DownloadServiceClient
}

// V1Beta1LabelServiceClientProvider provides LabelServiceClients.
type V1Beta1LabelServiceClientProvider interface {
	V1Beta1LabelServiceClient(registry string) policyv1beta1connect.LabelServiceClient
}

// V1Beta1ResourceServiceClientProvider provides ResourceServiceClients.
type V1Beta1ResourceServiceClientProvider interface {
	V1Beta1ResourceServiceClient(registry string) policyv1beta1connect.ResourceServiceClient
}

// V1Beta1PolicyServiceClientProvider provides PolicyServiceClients.
type V1Beta1PolicyServiceClientProvider interface {
	V1Beta1PolicyServiceClient(registry string) policyv1beta1connect.PolicyServiceClient
}

// V1Beta1UploadServiceClientProvider provides UploadServiceClients.
type V1Beta1UploadServiceClientProvider interface {
	V1Beta1UploadServiceClient(registry string) policyv1beta1connect.UploadServiceClient
}

// ClientProvider provides API clients for BSR services.
type ClientProvider interface {
	V1Beta1CommitServiceClientProvider
	V1Beta1DownloadServiceClientProvider
	V1Beta1LabelServiceClientProvider
	V1Beta1ResourceServiceClientProvider
	V1Beta1PolicyServiceClientProvider
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

func (c *clientProvider) V1Beta1CommitServiceClient(registry string) policyv1beta1connect.CommitServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		policyv1beta1connect.NewCommitServiceClient,
	)
}

func (c *clientProvider) V1Beta1DownloadServiceClient(registry string) policyv1beta1connect.DownloadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		policyv1beta1connect.NewDownloadServiceClient,
	)
}

func (c *clientProvider) V1Beta1LabelServiceClient(registry string) policyv1beta1connect.LabelServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		policyv1beta1connect.NewLabelServiceClient,
	)
}

func (c *clientProvider) V1Beta1ResourceServiceClient(registry string) policyv1beta1connect.ResourceServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		policyv1beta1connect.NewResourceServiceClient,
	)
}

func (c *clientProvider) V1Beta1PolicyServiceClient(registry string) policyv1beta1connect.PolicyServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		policyv1beta1connect.NewPolicyServiceClient,
	)
}

func (c *clientProvider) V1Beta1UploadServiceClient(registry string) policyv1beta1connect.UploadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		policyv1beta1connect.NewUploadServiceClient,
	)
}

type nopClientProvider struct{}

func (nopClientProvider) V1Beta1CommitServiceClient(registry string) policyv1beta1connect.CommitServiceClient {
	return policyv1beta1connect.UnimplementedCommitServiceHandler{}
}

func (nopClientProvider) V1Beta1DownloadServiceClient(registry string) policyv1beta1connect.DownloadServiceClient {
	return policyv1beta1connect.UnimplementedDownloadServiceHandler{}
}

func (nopClientProvider) V1Beta1LabelServiceClient(registry string) policyv1beta1connect.LabelServiceClient {
	return policyv1beta1connect.UnimplementedLabelServiceHandler{}
}

func (nopClientProvider) V1Beta1ResourceServiceClient(registry string) policyv1beta1connect.ResourceServiceClient {
	return policyv1beta1connect.UnimplementedResourceServiceHandler{}
}

func (nopClientProvider) V1Beta1PolicyServiceClient(registry string) policyv1beta1connect.PolicyServiceClient {
	return policyv1beta1connect.UnimplementedPolicyServiceHandler{}
}

func (nopClientProvider) V1Beta1UploadServiceClient(registry string) policyv1beta1connect.UploadServiceClient {
	return policyv1beta1connect.UnimplementedUploadServiceHandler{}
}
