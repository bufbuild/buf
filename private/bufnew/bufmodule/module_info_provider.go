package bufmodule

import "context"

// ModuleInfoProvider provides ModuleInfos for ModuleRefs.
type ModuleInfoProvider interface {
	GetModuleInfoForModuleRef(context.Context, ModuleRef) (ModuleInfo, error)
}
