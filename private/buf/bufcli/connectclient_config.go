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

package bufcli

import (
	"connectrpc.com/connect"
	otelconnect "connectrpc.com/otelconnect"
	"github.com/bufbuild/buf/private/buf/bufapp"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/bufpkg/buftransport"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/netrc"
	"github.com/bufbuild/buf/private/pkg/transport/http/httpclient"
)

// NewConnectClientConfig creates a new connect.ClientConfig which uses a token reader to look
// up the token in the container or in netrc based on the address of each individual client.
// It is then set in the header of all outgoing requests from clients created using this config.
func NewConnectClientConfig(container appext.Container) (*connectclient.Config, error) {
	envTokenProvider, err := bufconnect.NewTokenProviderFromContainer(container)
	if err != nil {
		return nil, err
	}
	netrcTokenProvider := bufconnect.NewNetrcTokenProvider(container, netrc.GetMachineForName)
	return newConnectClientConfigWithOptions(
		container,
		connectclient.WithAuthInterceptorProvider(
			bufconnect.NewAuthorizationInterceptorProvider(envTokenProvider, netrcTokenProvider),
		),
	)
}

// NewConnectClientConfigWithToken creates a new connect.ClientConfig with a given token. The provided token is
// set in the header of all outgoing requests from this provider
func NewConnectClientConfigWithToken(container appext.Container, token string) (*connectclient.Config, error) {
	tokenProvider, err := bufconnect.NewTokenProviderFromString(token)
	if err != nil {
		return nil, err
	}
	return newConnectClientConfigWithOptions(
		container,
		connectclient.WithAuthInterceptorProvider(
			bufconnect.NewAuthorizationInterceptorProvider(tokenProvider),
		),
	)
}

// Returns a registry provider with the given options applied in addition to default ones for all providers
func newConnectClientConfigWithOptions(container appext.Container, opts ...connectclient.ConfigOption) (*connectclient.Config, error) {
	config, err := newConfig(container)
	if err != nil {
		return nil, err
	}
	otelconnectInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return nil, err
	}
	client := httpclient.NewClient(config.TLS)
	options := []connectclient.ConfigOption{
		connectclient.WithAddressMapper(func(address string) string {
			if config.TLS == nil {
				return buftransport.PrependHTTP(address)
			}
			return buftransport.PrependHTTPS(address)
		}),
		connectclient.WithInterceptors(
			[]connect.Interceptor{
				bufconnect.NewAugmentedConnectErrorInterceptor(),
				bufconnect.NewSetCLIVersionInterceptor(Version),
				bufconnect.NewCLIWarningInterceptor(container),
				otelconnectInterceptor,
			},
		),
	}
	return connectclient.NewConfig(client, append(options, opts...)...), nil
}

// newConfig creates a new Config.
func newConfig(container appext.Container) (*bufapp.Config, error) {
	externalConfig := bufapp.ExternalConfig{}
	if err := appext.ReadConfig(container, &externalConfig); err != nil {
		return nil, err
	}
	return bufapp.NewConfig(container, externalConfig)
}
