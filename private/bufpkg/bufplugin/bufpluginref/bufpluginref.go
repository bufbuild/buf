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

import (
	"fmt"
	"strings"
)

// PluginIdentity is a plugin identity.
//
// It just contains remote, owner, plugin.
type PluginIdentity interface {
	Remote() string
	Owner() string
	Plugin() string

	// IdentityString is the string remote/owner/plugin.
	IdentityString() string

	// Prevents this type from being implemented by
	// another package.
	isPluginIdentity()
}

// NewPluginIdentity returns a new PluginIdentity.
func NewPluginIdentity(
	remote string,
	owner string,
	plugin string,
) (PluginIdentity, error) {
	return newPluginIdentity(remote, owner, plugin)
}

// PluginIdentityForString returns a new PluginIdentity for the given string.
//
// This parses the path in the form remote/owner/plugin.
func PluginIdentityForString(path string) (PluginIdentity, error) {
	remote, owner, plugin, err := parsePluginIdentityComponents(path)
	if err != nil {
		return nil, err
	}
	return NewPluginIdentity(remote, owner, plugin)
}

// PluginReference is a plugin reference.
//
// It references a specific version, but it's modeled as a generic reference
// so that we can more easily support other reference types later.
type PluginReference interface {
	PluginIdentity

	// Prints remote/owner/plugin:<reference>
	fmt.Stringer

	// The plugin version (e.g. v1.0.0).
	Reference() string

	isPluginReference()
}

// NewPluginReference returns a new validated PluginReference.
func NewPluginReference(
	remote string,
	owner string,
	plugin string,
	reference string,
) (PluginReference, error) {
	return newPluginReference(remote, owner, plugin, reference)
}

// PluginReferenceForString returns a new PluginReference for the given string.
//
// This parses the path in the form remote/owner/plugin:reference
func PluginReferenceForString(path string) (PluginReference, error) {
	remote, owner, plugin, reference, err := parsePluginReferenceComponents(path)
	if err != nil {
		return nil, err
	}
	return NewPluginReference(remote, owner, plugin, reference)
}

// parsePluginReferenceComponents parses and returns the remote, owner, plugin,
// and reference from the given path.
func parsePluginReferenceComponents(path string) (remote string, owner string, plugin string, ref string, err error) {
	remote, owner, rest, err := parsePluginIdentityComponents(path)
	if err != nil {
		return "", "", "", "", newInvalidPluginReferenceStringError(path)
	}
	restSplit := strings.Split(rest, ":")
	plugin = strings.TrimSpace(restSplit[0])
	if len(restSplit) == 1 {
		return remote, owner, plugin, "", nil
	}
	if len(restSplit) == 2 {
		ref := strings.TrimSpace(restSplit[1])
		if ref == "" {
			return "", "", "", "", newInvalidPluginReferenceStringError(path)
		}
		return remote, owner, plugin, ref, nil
	}
	return "", "", "", "", newInvalidPluginReferenceStringError(path)
}

func parsePluginIdentityComponents(path string) (remote string, owner string, plugin string, err error) {
	slashSplit := strings.Split(path, "/")
	if len(slashSplit) != 3 {
		return "", "", "", newInvalidPluginIdentityStringError(path)
	}
	remote = strings.TrimSpace(slashSplit[0])
	if remote == "" {
		return "", "", "", newInvalidPluginIdentityStringError(path)
	}
	owner = strings.TrimSpace(slashSplit[1])
	if owner == "" {
		return "", "", "", newInvalidPluginIdentityStringError(path)
	}
	plugin = strings.TrimSpace(slashSplit[2])
	if plugin == "" {
		return "", "", "", newInvalidPluginIdentityStringError(path)
	}
	return remote, owner, plugin, nil
}

func newInvalidPluginIdentityStringError(s string) error {
	return fmt.Errorf("plugin identity %q is invalid: must be in the form remote/owner/plugin", s)
}

func newInvalidPluginReferenceStringError(s string) error {
	return fmt.Errorf("plugin reference %q is invalid: must be in the form remote/owner/plugin:reference", s)
}
