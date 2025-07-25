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

package bufcli

import (
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginapi"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy/bufpolicyapi"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiowner"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapipolicy"
)

// NewController returns a new Controller.
func NewController(
	container appext.Container,
	options ...bufctl.ControllerOption,
) (bufctl.Controller, error) {
	if container.Env(copyToInMemoryEnvKey) != "" {
		options = append(
			options,
			bufctl.WithCopyToInMemory(),
		)
	}
	clientConfig, err := NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	moduleClientProvider := bufregistryapimodule.NewClientProvider(clientConfig)
	ownerClientProvider := bufregistryapiowner.NewClientProvider(clientConfig)
	pluginClientProvider := bufregistryapiplugin.NewClientProvider(clientConfig)
	policyClientProvider := bufregistryapipolicy.NewClientProvider(clientConfig)
	moduleDataProvider, err := newModuleDataProvider(container, moduleClientProvider, ownerClientProvider)
	if err != nil {
		return nil, err
	}
	commitProvider, err := newCommitProvider(container, moduleClientProvider, ownerClientProvider)
	if err != nil {
		return nil, err
	}
	pluginDataProvider, err := newPluginDataProvider(container, pluginClientProvider)
	if err != nil {
		return nil, err
	}
	policyDataProvider, err := newPolicyDataProvider(container, policyClientProvider)
	if err != nil {
		return nil, err
	}
	wktStore, err := NewWKTStore(container)
	if err != nil {
		return nil, err
	}
	return bufctl.NewController(
		container.Logger(),
		container,
		newGraphProvider(container, moduleClientProvider, ownerClientProvider),
		bufmoduleapi.NewModuleKeyProvider(container.Logger(), moduleClientProvider),
		moduleDataProvider,
		commitProvider,
		bufpluginapi.NewPluginKeyProvider(container.Logger(), pluginClientProvider),
		pluginDataProvider,
		bufpolicyapi.NewPolicyKeyProvider(container.Logger(), policyClientProvider),
		policyDataProvider,
		wktStore,
		// TODO FUTURE: Delete defaultHTTPClient and use the one from newConfig
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		defaultGitClonerOptions,
		options...,
	)
}
