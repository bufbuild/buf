// Copyright 2020-2023 Buf Technologies, Inc.
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
	if registry == "" {
		return nil, errors.New("new ModuleFullName: registry is empty")
	}
	if owner == "" {
		return nil, errors.New("new ModuleFullName: owner is empty")
	}
	if name == "" {
		return nil, errors.New("new ModuleFullName: name is empty")
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
