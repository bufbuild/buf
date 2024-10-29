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
)

// PluginRef is an unresolved reference to a Plugin.
type PluginRef interface {
	// String returns "registry/owner/name[:ref]".
	fmt.Stringer

	// PluginFullName returns the full name of the Plugin.
	//
	// Always present.
	PluginFullName() PluginFullName
	// Ref returns the reference within the Plugin.
	//
	// May be a label or dashless commitID.
	//
	// May be empty, in which case this references the commit of the default label of the Plugin.
	Ref() string

	isPluginRef()
}

// NewPluginRef returns a new PluginRef for the given compoonents.
func NewPluginRef(
	registry string,
	owner string,
	name string,
	ref string,
) (PluginRef, error) {
	pluginFullName, err := NewPluginFullName(registry, owner, name)
	if err != nil {
		return nil, err
	}
	return newPluginRef(pluginFullName, ref)
}

// ParsePluginRef parses a PluginRef from a string in the form "registry/owner/name[:ref]".
func ParsePluginRef(pluginRefString string) (PluginRef, error) {
	// Returns ParseErrors.
	registry, owner, name, ref, err := parsePluginRefComponents(pluginRefString)
	if err != nil {
		return nil, err
	}
	// We don't rely on constructors for ParseErrors.
	return NewPluginRef(registry, owner, name, ref)
}

// *** PRIVATE ***

type pluginRef struct {
	pluginFullName PluginFullName
	ref            string
}

func newPluginRef(
	pluginFullName PluginFullName,
	ref string,
) (*pluginRef, error) {
	if pluginFullName == nil {
		return nil, errors.New("nil PluginFullName when constructing PluginRef")
	}
	return &pluginRef{
		pluginFullName: pluginFullName,
		ref:            ref,
	}, nil
}

func (m *pluginRef) PluginFullName() PluginFullName {
	return m.pluginFullName
}

func (m *pluginRef) Ref() string {
	return m.ref
}

func (m *pluginRef) String() string {
	if m.ref == "" {
		return m.pluginFullName.String()
	}
	return m.pluginFullName.String() + ":" + m.ref
}

func (*pluginRef) isPluginRef() {}
