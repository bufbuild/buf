package bufmodule

import "context"

// ModulePinProvider provides ModulePins for ModuleRefs.
type ModulePinProvider interface {
	GetModulePinForModuleRef(context.Context, ModuleRef) (ModulePin, error)
}
