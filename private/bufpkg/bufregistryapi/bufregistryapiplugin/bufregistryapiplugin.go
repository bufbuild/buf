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

package bufregistryapiplugin

import (
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/plugin/v1beta1/pluginv1beta1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

var (
	// NopV1Beta1CollectionServiceClientProvider is a V1Beta1CollectionServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1CollectionServiceClientProvider V1Beta1CollectionServiceClientProvider = nopClientProvider{}
	// NopV1Beta1CommitServiceClientProvider is a V1Beta1CommitServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1CommitServiceClientProvider V1Beta1CommitServiceClientProvider = nopClientProvider{}
	// NopV1Beta1DownloadServiceClientProvider is a V1Beta1DownloadServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1DownloadServiceClientProvider V1Beta1DownloadServiceClientProvider = nopClientProvider{}
	// NopV1Beta1LabelServiceClientProvider is a V1Beta1LabelServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1LabelServiceClientProvider V1Beta1LabelServiceClientProvider = nopClientProvider{}
	// NopV1Beta1PluginServiceClientProvider is a V1Beta1PluginServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1PluginServiceClientProvider V1Beta1PluginServiceClientProvider = nopClientProvider{}
	// NopV1Beta1ResourceServiceClientProvider is a V1Beta1ResourceServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1ResourceServiceClientProvider V1Beta1ResourceServiceClientProvider = nopClientProvider{}
	// NopV1Beta1UploadServiceClientProvider is a V1Beta1UploadServiceClientProvider that provides unimplemented services for testing.
	NopV1Beta1UploadServiceClientProvider V1Beta1UploadServiceClientProvider = nopClientProvider{}
	// NopClientProvider is a ClientProvider that provides unimplemented services for testing.
	NopClientProvider ClientProvider = nopClientProvider{}
)

// V1Beta1CollectionServiceClientProvider provides CollectionServiceClients.
type V1Beta1CollectionServiceClientProvider interface {
	V1Beta1CollectionServiceClient(registry string) pluginv1beta1connect.CollectionServiceClient
}

// V1Beta1CommitServiceClientProvider provides CommitServiceClients.
type V1Beta1CommitServiceClientProvider interface {
	V1Beta1CommitServiceClient(registry string) pluginv1beta1connect.CommitServiceClient
}

// V1Beta1DownloadServiceClientProvider provides DownloadServiceClients.
type V1Beta1DownloadServiceClientProvider interface {
	V1Beta1DownloadServiceClient(registry string) pluginv1beta1connect.DownloadServiceClient
}

// V1Beta1LabelServiceClientProvider provides LabelServiceClients.
type V1Beta1LabelServiceClientProvider interface {
	V1Beta1LabelServiceClient(registry string) pluginv1beta1connect.LabelServiceClient
}

// V1Beta1ResourceServiceClientProvider provides ResourceServiceClients.
type V1Beta1ResourceServiceClientProvider interface {
	V1Beta1ResourceServiceClient(registry string) pluginv1beta1connect.ResourceServiceClient
}

// V1Beta1PluginServiceClientProvider provides PluginServiceClients.
type V1Beta1PluginServiceClientProvider interface {
	V1Beta1PluginServiceClient(registry string) pluginv1beta1connect.PluginServiceClient
}

// V1Beta1UploadServiceClientProvider provides UploadServiceClients.
type V1Beta1UploadServiceClientProvider interface {
	V1Beta1UploadServiceClient(registry string) pluginv1beta1connect.UploadServiceClient
}

// ClientProvider provides API clients for BSR services.
type ClientProvider interface {
	V1Beta1CollectionServiceClientProvider
	V1Beta1CommitServiceClientProvider
	V1Beta1DownloadServiceClientProvider
	V1Beta1LabelServiceClientProvider
	V1Beta1ResourceServiceClientProvider
	V1Beta1PluginServiceClientProvider
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

func (c *clientProvider) V1Beta1CollectionServiceClient(registry string) pluginv1beta1connect.CollectionServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		pluginv1beta1connect.NewCollectionServiceClient,
	)
}

func (c *clientProvider) V1Beta1CommitServiceClient(registry string) pluginv1beta1connect.CommitServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		pluginv1beta1connect.NewCommitServiceClient,
	)
}

func (c *clientProvider) V1Beta1DownloadServiceClient(registry string) pluginv1beta1connect.DownloadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		pluginv1beta1connect.NewDownloadServiceClient,
	)
}

func (c *clientProvider) V1Beta1LabelServiceClient(registry string) pluginv1beta1connect.LabelServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		pluginv1beta1connect.NewLabelServiceClient,
	)
}

func (c *clientProvider) V1Beta1ResourceServiceClient(registry string) pluginv1beta1connect.ResourceServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		pluginv1beta1connect.NewResourceServiceClient,
	)
}

func (c *clientProvider) V1Beta1PluginServiceClient(registry string) pluginv1beta1connect.PluginServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		pluginv1beta1connect.NewPluginServiceClient,
	)
}

func (c *clientProvider) V1Beta1UploadServiceClient(registry string) pluginv1beta1connect.UploadServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		pluginv1beta1connect.NewUploadServiceClient,
	)
}

type nopClientProvider struct{}

func (nopClientProvider) V1Beta1CollectionServiceClient(registry string) pluginv1beta1connect.CollectionServiceClient {
	return pluginv1beta1connect.UnimplementedCollectionServiceHandler{}
}

func (nopClientProvider) V1Beta1CommitServiceClient(registry string) pluginv1beta1connect.CommitServiceClient {
	return pluginv1beta1connect.UnimplementedCommitServiceHandler{}
}

func (nopClientProvider) V1Beta1DownloadServiceClient(registry string) pluginv1beta1connect.DownloadServiceClient {
	return pluginv1beta1connect.UnimplementedDownloadServiceHandler{}
}

func (nopClientProvider) V1Beta1LabelServiceClient(registry string) pluginv1beta1connect.LabelServiceClient {
	return pluginv1beta1connect.UnimplementedLabelServiceHandler{}
}

func (nopClientProvider) V1Beta1ResourceServiceClient(registry string) pluginv1beta1connect.ResourceServiceClient {
	return pluginv1beta1connect.UnimplementedResourceServiceHandler{}
}

func (nopClientProvider) V1Beta1PluginServiceClient(registry string) pluginv1beta1connect.PluginServiceClient {
	return pluginv1beta1connect.UnimplementedPluginServiceHandler{}
}

func (nopClientProvider) V1Beta1UploadServiceClient(registry string) pluginv1beta1connect.UploadServiceClient {
	return pluginv1beta1connect.UnimplementedUploadServiceHandler{}
}
