package bufmodule

import "fmt"

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
