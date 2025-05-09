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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiowner"
)

// NewGraphProvider returns a new GraphProvider.
func NewGraphProvider(container appext.Container) (bufmodule.GraphProvider, error) {
	clientConfig, err := NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	return newGraphProvider(
		container,
		bufregistryapimodule.NewClientProvider(clientConfig),
		bufregistryapiowner.NewClientProvider(clientConfig),
	), nil
}

func newGraphProvider(
	container appext.Container,
	moduleClientProvider bufregistryapimodule.ClientProvider,
	ownerClientProvider bufregistryapiowner.ClientProvider,
) bufmodule.GraphProvider {
	return bufmoduleapi.NewGraphProvider(
		container.Logger(),
		moduleClientProvider,
		ownerClientProvider,
		// OK if empty
		bufmoduleapi.GraphProviderWithLegacyFederationRegistry(container.Env(legacyFederationRegistryEnvKey)),
		// OK if empty
		bufmoduleapi.GraphProviderWithPublicRegistry(container.Env(publicRegistryEnvKey)),
	)
}
