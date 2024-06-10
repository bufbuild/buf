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

// PluginReference uniquely references a plugin (including version and revision information).
//
// It can be used to identify dependencies on other plugins.
type PluginReference interface {
	PluginIdentity

	// ReferenceString is the string representation of identity:version:revision.
	ReferenceString() string

	// Version is the plugin's semantic version.
	Version() string

	// Revision is the plugin's revision number.
	//
	// The accepted range for this value is 0 - math.MaxInt32.
	Revision() int

	// Prevents this type from being implemented by
	// another package.
	isPluginReference()
}

// NewPluginReference returns a new PluginReference.
func NewPluginReference(
	identity PluginIdentity,
	version string,
	revision int,
) (PluginReference, error) {
	return newPluginReference(identity, version, revision)
}

// PluginReferenceForString returns a new PluginReference for the given string.
//
// This parses the path in the form remote/owner/plugin:version.
func PluginReferenceForString(reference string, revision int) (PluginReference, error) {
	return parsePluginReference(reference, revision)
}

// ParsePluginIdentityOptionalVersion returns the PluginIdentity and version for the given string.
// If the string does not contain a version, the version is assumed to be an empty string.
// This parses the path in the form remote/owner/plugin:version.
func ParsePluginIdentityOptionalVersion(rawReference string) (PluginIdentity, string, error) {
	if reference, err := PluginReferenceForString(rawReference, 0); err == nil {
		return reference, reference.Version(), nil
	}
	// Try parsing as a plugin identity (no version information)
	identity, err := PluginIdentityForString(rawReference)
	if err != nil {
		return nil, "", fmt.Errorf("invalid remote plugin %s", rawReference)
	}
	return identity, "", nil
}

// IsPluginReferenceOrIdentity returns true if the argument matches a plugin
// reference (with version) or a plugin identity (without version).
func IsPluginReferenceOrIdentity(plugin string) bool {
	if _, err := PluginReferenceForString(plugin, 0); err == nil {
		return true
	}
	if _, err := PluginIdentityForString(plugin); err == nil {
		return true
	}
	return false
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
	if plugin == "" || strings.ContainsRune(plugin, ':') {
		return "", "", "", newInvalidPluginIdentityStringError(path)
	}
	return remote, owner, plugin, nil
}

func newInvalidPluginIdentityStringError(s string) error {
	return fmt.Errorf("plugin identity %q is invalid: must be in the form remote/owner/plugin", s)
}

func parsePluginReference(reference string, revision int) (PluginReference, error) {
	name, version, ok := strings.Cut(reference, ":")
	if !ok {
		return nil, fmt.Errorf("plugin references must be specified as \"<name>:<version>\" strings")
	}
	identity, err := PluginIdentityForString(name)
	if err != nil {
		return nil, err
	}
	return NewPluginReference(identity, version, revision)
}
