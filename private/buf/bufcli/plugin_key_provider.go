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
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginapi"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapiplugin"
)

// NewPluginKeyProvider returns a new PluginKeyProvider.
func NewPluginKeyProvider(container appext.Container) (bufplugin.PluginKeyProvider, error) {
	clientConfig, err := NewConnectClientConfig(container)
	if err != nil {
		return nil, err
	}
	return bufpluginapi.NewPluginKeyProvider(
		container.Logger(),
		bufregistryapiplugin.NewClientProvider(
			clientConfig,
		),
	), nil
}
