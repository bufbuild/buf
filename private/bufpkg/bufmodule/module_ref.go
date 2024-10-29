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
)

// ModuleRef is an unresolved reference to a Module.
type ModuleRef interface {
	// String returns "registry/owner/name[:ref]".
	fmt.Stringer

	// ModuleFullName returns the full name of the Module.
	//
	// Always present.
	ModuleFullName() ModuleFullName
	// Ref returns the reference within the Module.
	//
	// May be a label or dashless commitID.
	//
	// May be empty, in which case this references the commit of the default label of the Module.
	Ref() string

	isModuleRef()
}

// NewModuleRef returns a new ModuleRef for the given compoonents.
func NewModuleRef(
	registry string,
	owner string,
	name string,
	ref string,
) (ModuleRef, error) {
	moduleFullName, err := NewModuleFullName(registry, owner, name)
	if err != nil {
		return nil, err
	}
	return newModuleRef(moduleFullName, ref)
}

// ParseModuleRef parses a ModuleRef from a string in the form "registry/owner/name[:ref]".
func ParseModuleRef(moduleRefString string) (ModuleRef, error) {
	// Returns ParseErrors.
	registry, owner, name, ref, err := parseModuleRefComponents(moduleRefString)
	if err != nil {
		return nil, err
	}
	// We don't rely on constructors for ParseErrors.
	return NewModuleRef(registry, owner, name, ref)
}

// *** PRIVATE ***

type moduleRef struct {
	moduleFullName ModuleFullName
	ref            string
}

func newModuleRef(
	moduleFullName ModuleFullName,
	ref string,
) (*moduleRef, error) {
	if moduleFullName == nil {
		return nil, errors.New("nil ModuleFullName when constructing ModuleRef")
	}
	return &moduleRef{
		moduleFullName: moduleFullName,
		ref:            ref,
	}, nil
}

func (m *moduleRef) ModuleFullName() ModuleFullName {
	return m.moduleFullName
}

func (m *moduleRef) Ref() string {
	return m.ref
}

func (m *moduleRef) String() string {
	if m.ref == "" {
		return m.moduleFullName.String()
	}
	return m.moduleFullName.String() + ":" + m.ref
}

func (*moduleRef) isModuleRef() {}
