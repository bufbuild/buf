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
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleapi"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginapi"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapimodule"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
	"github.com/bufbuild/buf/private/pkg/app/appext"
)

// NewModuleUploader returns a new Uploader for ModuleSets.
func NewModuleUploader(container appext.Container) (bufmodule.Uploader, error) {
	clientConfig, err := NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	return newModuleUploader(container, bufregistryapimodule.NewClientProvider(clientConfig)), nil
}

// NewPluginUploader returns a new Uploader for Plugins.
func NewPluginUploader(container appext.Container) (bufplugin.Uploader, error) {
	clientConfig, err := NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	return newPluginUploader(container, bufregistryapiplugin.NewClientProvider(clientConfig)), nil
}

func newModuleUploader(
	container appext.Container,
	clientProvider bufregistryapimodule.ClientProvider,
) bufmodule.Uploader {
	return bufmoduleapi.NewUploader(
		container.Logger(),
		clientProvider,
		// OK if empty
		bufmoduleapi.UploaderWithPublicRegistry(container.Env(publicRegistryEnvKey)),
	)
}

func newPluginUploader(
	container appext.Container,
	clientProvider bufregistryapiplugin.ClientProvider,
) bufplugin.Uploader {
	return bufpluginapi.NewUploader(
		container.Logger(),
		clientProvider,
	)
}
