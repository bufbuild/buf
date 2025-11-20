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

package bufparse

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/netext"
)

// FullName represents the full name of the entity, including its registry, owner, and name.
type FullName interface {
	// String returns "registry/owner/name".
	fmt.Stringer

	// Registry returns the hostname of the BSR instance that this entity is contained within.
	Registry() string
	// Owner returns the name of the user or organization that owns this entity.
	Owner() string
	// Name returns the name of the entity.
	Name() string

	isFullName()
}

// NewFullName returns a new FullName for the given components.
func NewFullName(
	registry string,
	owner string,
	name string,
) (FullName, error) {
	return newFullName(
		registry,
		owner,
		name,
	)
}

// ParseFullName parses a FullName from a string in the form "registry/owner/name".
//
// Returns an error of type *ParseError if the string could not be parsed.
func ParseFullName(fullNameString string) (FullName, error) {
	// parseFullNameComponents returns ParseErrors.
	registry, owner, name, err := parseFullNameComponents(fullNameString)
	if err != nil {
		return nil, err
	}
	if err := validateFullNameParameters(registry, owner, name); err != nil {
		return nil, NewParseError(
			"full name",
			fullNameString,
			err,
		)
	}
	// We don't rely on constructors for ParseErrors.
	return NewFullName(registry, owner, name)
}

// FullNameEqual returns true if the FullNames are equal.
func FullNameEqual(one FullName, two FullName) bool {
	if (one == nil) != (two == nil) {
		return false
	}
	if one == nil {
		return true
	}
	return one.String() == two.String()
}

// HasFullName is any type that has a FullName() function.
type HasFullName interface {
	// FullName returns the ullName.
	//
	// May be empty.
	FullName() FullName
}

// FullNameStringToUniqueValue maps the values that implement HasFullName to a map
// from FullName string to the unique value that has this FullName.
//
// If any value has a nil FullName, this value is not added to the map. Therefore,
// for types that potentially have a nil FullName, you cannot reply on this function
// returning a map of the same length as the input values.
//
// Returns error if there are values with duplicate FullNames.
func FullNameStringToUniqueValue[T HasFullName, S ~[]T](values S) (map[string]T, error) {
	m := make(map[string]T, len(values))
	for _, value := range values {
		fullName := value.FullName()
		if fullName == nil {
			continue
		}
		existingValue, ok := m[fullName.String()]
		if ok {
			return nil, fmt.Errorf(
				"duplicate full names in input: %q, %q",
				existingValue.FullName().String(),
				fullName.String(),
			)
		}
		m[fullName.String()] = value
	}
	return m, nil
}

// *** PRIVATE ***

type fullName struct {
	registry string
	owner    string
	name     string
}

func newFullName(
	registry string,
	owner string,
	name string,
) (*fullName, error) {
	if err := validateFullNameParameters(registry, owner, name); err != nil {
		return nil, err
	}
	return &fullName{
		registry: registry,
		owner:    owner,
		name:     name,
	}, nil
}

func (m *fullName) Registry() string {
	return m.registry
}

func (m *fullName) Owner() string {
	return m.owner
}

func (m *fullName) Name() string {
	return m.name
}

func (m *fullName) String() string {
	return m.registry + "/" + m.owner + "/" + m.name
}

func (*fullName) isFullName() {}

func validateFullNameParameters(
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
