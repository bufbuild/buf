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
