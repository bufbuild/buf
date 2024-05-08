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
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/netext"
)

// ModuleOwner represents a module owner, consisting of a registry and owner.
//
// This concept used to live in bufmodule but it doesn't really make sense there, as it isn't
// use anywhere else. We only use this in buf beta registry organization commands at the moment.
type ModuleOwner interface {
	// String returns "registry/owner".
	fmt.Stringer

	// Registry returns the hostname of the BSR instance that this owner is contained within.
	Registry() string
	// Owner returns the name of the user or organization.
	Owner() string

	isModuleOwner()
}

// NewModuleOwner returns a new ModuleOwner for the given components.
func NewModuleOwner(
	registry string,
	owner string,
) (ModuleOwner, error) {
	return newModuleOwner(
		registry,
		owner,
	)
}

// ParseModuleOwner parses a ModuleOwner from a string in the form "registry/owner".
func ParseModuleOwner(moduleOwnerString string) (ModuleOwner, error) {
	registry, owner, err := parseModuleOwnerComponents(moduleOwnerString)
	if err != nil {
		return nil, err
	}
	return NewModuleOwner(registry, owner)
}

// *** PRIVATE ***

type moduleOwner struct {
	registry string
	owner    string
}

func newModuleOwner(
	registry string,
	owner string,
) (*moduleOwner, error) {
	if registry == "" {
		return nil, errors.New("registry is empty")
	}
	if _, err := netext.ValidateHostname(registry); err != nil {
		return nil, fmt.Errorf("registry %q is not a valid hostname: %w", registry, err)
	}
	if owner == "" {
		return nil, errors.New("owner is empty")
	}
	if strings.Contains(owner, "/") {
		return nil, fmt.Errorf("owner %q cannot contain slashes", owner)
	}
	return &moduleOwner{
		registry: registry,
		owner:    owner,
	}, nil
}

func (m *moduleOwner) Registry() string {
	return m.registry
}

func (m *moduleOwner) Owner() string {
	return m.owner
}

func (m *moduleOwner) String() string {
	return m.registry + "/" + m.owner
}

func (*moduleOwner) isModuleOwner() {}

func parseModuleOwnerComponents(path string) (registry string, owner string, err error) {
	slashSplit := strings.Split(path, "/")
	if len(slashSplit) != 2 {
		return "", "", newInvalidModuleOwnerStringError(path)
	}
	registry = strings.TrimSpace(slashSplit[0])
	if registry == "" {
		return "", "", newInvalidModuleOwnerStringError(path)
	}
	owner = strings.TrimSpace(slashSplit[1])
	if owner == "" {
		return "", "", newInvalidModuleOwnerStringError(path)
	}
	return registry, owner, nil
}

func newInvalidModuleOwnerStringError(s string) error {
	return fmt.Errorf("invalid module owner %q: must be in the form registry/owner", s)
}
