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
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/tracing"
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
	clientProvider := bufapi.NewClientProvider(clientConfig)
	moduleDataProvider, err := newModuleDataProvider(container, clientProvider)
	if err != nil {
		return nil, err
	}
	commitProvider, err := newCommitProvider(container, clientProvider)
	if err != nil {
		return nil, err
	}
	wktStore, err := newWKTStore(container)
	if err != nil {
		return nil, err
	}
	return bufctl.NewController(
		container.Logger(),
		tracing.NewTracer(container.Tracer()),
		container,
		newGraphProvider(container, clientProvider),
		bufmoduleapi.NewModuleKeyProvider(container.Logger(), clientProvider),
		moduleDataProvider,
		commitProvider,
		wktStore,
		// TODO FUTURE: Delete defaultHTTPClient and use the one from newConfig
		defaultHTTPClient,
		defaultHTTPAuthenticator,
		defaultGitClonerOptions,
		options...,
	)
}
