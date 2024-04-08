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

package bufremotepluginref

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/netext"
)

type pluginIdentity struct {
	remote string
	owner  string
	plugin string
}

func newPluginIdentity(
	remote string,
	owner string,
	plugin string,
) (*pluginIdentity, error) {
	pluginIdentity := &pluginIdentity{
		remote: remote,
		owner:  owner,
		plugin: plugin,
	}
	if err := ValidatePluginIdentity(pluginIdentity); err != nil {
		return nil, err
	}
	return pluginIdentity, nil
}

func (m *pluginIdentity) Remote() string {
	return m.remote
}

func (m *pluginIdentity) Owner() string {
	return m.owner
}

func (m *pluginIdentity) Plugin() string {
	return m.plugin
}

func (m *pluginIdentity) IdentityString() string {
	return m.remote + "/" + m.owner + "/" + m.plugin
}

func (*pluginIdentity) isPluginIdentity() {}

func ValidatePluginIdentity(pluginIdentity PluginIdentity) error {
	if pluginIdentity == nil {
		return errors.New("plugin identity is required")
	}
	if err := ValidateRemote(pluginIdentity.Remote()); err != nil {
		return err
	}
	if pluginIdentity.Owner() == "" {
		return errors.New("owner name is required")
	}
	if pluginIdentity.Plugin() == "" {
		return errors.New("plugin name is required")
	}
	return nil
}

func ValidateRemote(remote string) error {
	if remote == "" {
		return errors.New("remote name is required")
	}
	if _, err := netext.ValidateHostname(remote); err != nil {
		return fmt.Errorf("invalid remote %q: %w", remote, err)
	}
	return nil
}
