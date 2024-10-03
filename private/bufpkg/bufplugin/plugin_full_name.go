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

package bufplugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/netext"
)

// PluginFullName represents the full name of the Plugin, including its registry, owner, and name.
type PluginFullName interface {
	// String returns "registry/owner/name".
	fmt.Stringer

	// Registry returns the hostname of the BSR instance that this Plugin is contained within.
	Registry() string
	// Owner returns the name of the user or organization that owns this Plugin.
	Owner() string
	// Name returns the name of the Plugin.
	Name() string

	isPluginFullName()
}

// NewPluginFullName returns a new PluginFullName for the given components.
func NewPluginFullName(
	registry string,
	owner string,
	name string,
) (PluginFullName, error) {
	return newPluginFullName(
		registry,
		owner,
		name,
	)
}

// ParsePluginFullName parses a PluginFullName from a string in the form "registry/owner/name".
func ParsePluginFullName(pluginFullNameString string) (PluginFullName, error) {
	// parsePluiginFullNameComponents returns ParseErrors.
	registry, owner, name, err := parsePluginFullNameComponents(pluginFullNameString)
	if err != nil {
		return nil, err
	}

	// We need to validate the parameters of the PluginFullName.
	if err := validatePluginFullNameParameters(registry, owner, name); err != nil {
		return nil, &ParseError{
			typeString: "plugin name",
			input:      pluginFullNameString,
			err:        err,
		}
	}

	// We don't rely on constructors for ParseErrors.
	return NewPluginFullName(registry, owner, name)
}

// *** PRIVATE ***

type pluginFullName struct {
	registry string
	owner    string
	name     string
}

func newPluginFullName(
	registry string,
	owner string,
	name string,
) (*pluginFullName, error) {
	if err := validatePluginFullNameParameters(registry, owner, name); err != nil {
		return nil, err
	}
	return &pluginFullName{
		registry: registry,
		owner:    owner,
		name:     name,
	}, nil
}

func (m *pluginFullName) Registry() string {
	return m.registry
}

func (m *pluginFullName) Owner() string {
	return m.owner
}

func (m *pluginFullName) Name() string {
	return m.name
}

func (m *pluginFullName) String() string {
	return m.registry + "/" + m.owner + "/" + m.name
}

func (*pluginFullName) isPluginFullName() {}

// TODO(emcfarlane): move this to a shared location.
func validatePluginFullNameParameters(
	registry string,
	owner string,
	name string,
) error {
	if registry == "" {
		return errors.New("registry is empty")
	}
	if _, err := netext.ValidateHostname(registry); err != nil {
		return fmt.Errorf("registry %q is not a valid hostname: %w", registry, err)
	}
	if owner == "" {
		return errors.New("owner is empty")
	}
	if strings.Contains(owner, "/") {
		return fmt.Errorf("owner %q cannot contain slashes", owner)
	}
	if name == "" {
		return errors.New("name is empty")
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("name %q cannot contain slashes", name)
	}
	return nil
}
