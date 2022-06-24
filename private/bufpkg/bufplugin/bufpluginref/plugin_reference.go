// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufpluginref

import "errors"

type pluginReference struct {
	remote    string
	owner     string
	plugin    string
	reference string
}

func newPluginReference(
	remote string,
	owner string,
	plugin string,
	reference string,
) (*pluginReference, error) {
	pluginReference := &pluginReference{
		remote:    remote,
		owner:     owner,
		plugin:    plugin,
		reference: reference,
	}
	if err := validatePluginReference(pluginReference); err != nil {
		return nil, err
	}
	return pluginReference, nil
}

func (m *pluginReference) Remote() string {
	return m.remote
}

func (m *pluginReference) Owner() string {
	return m.owner
}

func (m *pluginReference) Plugin() string {
	return m.plugin
}

func (m *pluginReference) Reference() string {
	return m.reference
}

func (m *pluginReference) IdentityString() string {
	return m.remote + "/" + m.owner + "/" + m.plugin
}

func (m *pluginReference) String() string {
	return m.remote + "/" + m.owner + "/" + m.plugin + ":" + m.reference
}

func (*pluginReference) isPluginIdentity()  {}
func (*pluginReference) isPluginReference() {}

func validatePluginReference(pluginReference PluginReference) error {
	if pluginReference == nil {
		return errors.New("plugin reference is required")
	}
	if err := validateRemote(pluginReference.Remote()); err != nil {
		return err
	}
	if pluginReference.Owner() == "" {
		return errors.New("owner name is required")
	}
	if pluginReference.Plugin() == "" {
		return errors.New("plugin name is required")
	}
	if pluginReference.Reference() == "" {
		return errors.New("reference is required")
	}
	return nil
}
