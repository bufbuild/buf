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

package bufmodule

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/netext"
)

// ModuleFullName represents the full name of the Module, including its registry, owner, and name.
type ModuleFullName interface {
	// String returns "registry/owner/name".
	fmt.Stringer

	// Registry returns the hostname of the BSR instance that this Module is contained within.
	Registry() string
	// Owner returns the name of the user or organization that owns this Module.
	Owner() string
	// Name returns the name of the Module.
	Name() string

	isModuleFullName()
}

// NewModuleFullName returns a new ModuleFullName for the given components.
func NewModuleFullName(
	registry string,
	owner string,
	name string,
) (ModuleFullName, error) {
	return newModuleFullName(
		registry,
		owner,
		name,
	)
}

// ParseModuleFullName parses a ModuleFullName from a string in the form "registry/owner/name".
func ParseModuleFullName(moduleFullNameString string) (ModuleFullName, error) {
	// parseModuleFullNameComponents returns ParseErrors.
	registry, owner, name, err := parseModuleFullNameComponents(moduleFullNameString)
	if err != nil {
		return nil, err
	}
	if err := validateModuleFullNameParameters(registry, owner, name); err != nil {
		return nil, &ParseError{
			typeString: "module name",
			input:      moduleFullNameString,
			err:        err,
		}
	}
	// We don't rely on constructors for ParseErrors.
	return NewModuleFullName(registry, owner, name)
}

// ModuleFullNameEqual returns true if the ModuleFullNames are equal.
func ModuleFullNameEqual(one ModuleFullName, two ModuleFullName) bool {
	if (one == nil) != (two == nil) {
		return false
	}
	if one == nil {
		return true
	}
	return one.String() == two.String()
}

// HasModuleFullName is any type that has a ModuleFullName() function.
type HasModuleFullName interface {
	// ModuleFullName returns the ModuleullName.
	//
	// May be empty.
	ModuleFullName() ModuleFullName
}

// ModuleFullNameStringToValue maps the values that implement HasModuleFullName to a map
// from ModuleFullName string to the unique value that has this ModuleFullName.
//
// If any value has a nil ModuleFullName, this value is not added to the map. Therefore,
// for types that potentially have a nil ModuleFullName, you cannot reply on this function
// returning a map of the same length as the input values.
//
// Returns error if there are values with duplicate ModuleFullNames.
func ModuleFullNameStringToUniqueValue[T HasModuleFullName, S ~[]T](values S) (map[string]T, error) {
	m := make(map[string]T, len(values))
	for _, value := range values {
		moduleFullName := value.ModuleFullName()
		if moduleFullName == nil {
			continue
		}
		existingValue, ok := m[moduleFullName.String()]
		if ok {
			return nil, fmt.Errorf(
				"duplicate module names in input: %q, %q",
				existingValue.ModuleFullName().String(),
				moduleFullName.String(),
			)
		}
		m[moduleFullName.String()] = value
	}
	return m, nil
}

// *** PRIVATE ***

type moduleFullName struct {
	registry string
	owner    string
	name     string
}

func newModuleFullName(
	registry string,
	owner string,
	name string,
) (*moduleFullName, error) {
	if err := validateModuleFullNameParameters(registry, owner, name); err != nil {
		return nil, err
	}
	return &moduleFullName{
		registry: registry,
		owner:    owner,
		name:     name,
	}, nil
}

func (m *moduleFullName) Registry() string {
	return m.registry
}

func (m *moduleFullName) Owner() string {
	return m.owner
}

func (m *moduleFullName) Name() string {
	return m.name
}

func (m *moduleFullName) String() string {
	return m.registry + "/" + m.owner + "/" + m.name
}

func (*moduleFullName) isModuleFullName() {}

func validateModuleFullNameParameters(
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
