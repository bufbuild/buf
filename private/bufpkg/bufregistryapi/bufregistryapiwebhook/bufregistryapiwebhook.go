// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufregistryapiwebhook

import (
	"buf.build/gen/go/bufbuild/registry/connectrpc/go/v2/buf/registry/webhook/v1/webhookv1connect"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

var (
	// NopV1WebhookServiceClientProvider is a V1WebhookServiceClientProvider that provides unimplemented services for testing.
	NopV1WebhookServiceClientProvider V1WebhookServiceClientProvider = nopClientProvider{}
	// NopClientProvider is a ClientProvider that provides unimplemented services for testing.
	NopClientProvider ClientProvider = nopClientProvider{}
)

// V1WebhookServiceClientProvider provides WebhookServiceClients.
type V1WebhookServiceClientProvider interface {
	V1WebhookServiceClient(registry string) webhookv1connect.WebhookServiceClient
}

// ClientProvider provides API clients for BSR services.
type ClientProvider interface {
	V1WebhookServiceClientProvider
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

func (c *clientProvider) V1WebhookServiceClient(registry string) webhookv1connect.WebhookServiceClient {
	return connectclient.Make(
		c.clientConfig,
		registry,
		webhookv1connect.NewWebhookServiceClient,
	)
}

type nopClientProvider struct{}

func (nopClientProvider) V1WebhookServiceClient(registry string) webhookv1connect.WebhookServiceClient {
	return webhookv1connect.UnimplementedWebhookServiceHandler{}
}
