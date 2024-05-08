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
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/pkg/app/appext"
)

// NewGraphProvider returns a new GraphProvider.
func NewGraphProvider(container appext.Container) (bufmodule.GraphProvider, error) {
	clientConfig, err := NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	return newGraphProvider(container, bufapi.NewClientProvider(clientConfig)), nil
}

func newGraphProvider(
	container appext.Container,
	clientProvider bufapi.ClientProvider,
) bufmodule.GraphProvider {
	return bufmoduleapi.NewGraphProvider(
		container.Logger(),
		clientProvider,
		// OK if empty
		bufmoduleapi.GraphProviderWithLegacyFederationRegistry(container.Env(legacyFederationRegistryEnvKey)),
		// OK if empty
		bufmoduleapi.GraphProviderWithPublicRegistry(container.Env(publicRegistryEnvKey)),
	)
}
