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

package bufcheck

import (
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/bufplugin-go/check"
)

// checkClientSpec contains a check.Client and details on what to do about
// options it should pass when calling check.
//
// This allows us to take a bufconfig.PluginConfig and turn it into a client/options pair.
//
// options will be non-nil or useDefaultOptions will be true, but they will not be non-nil
// and true on the same checkClientSpec.
type checkClientSpec struct {
	client check.Client
	// options are plugin-specific Options to pass.
	options check.Options
	// useDefaultOptions says to use the DefaultOptions from a config instead of
	// the Options above.
	useDefaultOptions bool
}

func newDefaultCheckClientSpec(defaultClient check.Client) *checkClientSpec {
	return &checkClientSpec{
		client:            defaultClient,
		useDefaultOptions: true,
	}
}

func newPluginCheckClientSpec(pluginClient check.Client, options check.Options) *checkClientSpec {
	return &checkClientSpec{
		client:  pluginClient,
		options: options,
	}
}

func (c *checkClientSpec) Client() check.Client {
	return c.client
}

func (c *checkClientSpec) Options(config *config) (check.Options, error) {
	if c.options != nil {
		return c.options, nil
	}
	if c.useDefaultOptions {
		return config.DefaultOptions, nil
	}
	return nil, syserror.New("checkClientSpec did not have Options or UseDefaultOptions")
}
