package bufmodule

import "context"

// ModuleSet is a set of Modules.
//
// Within the CLI, this is the set of Modules that comprises a workspace.
//
// Modules within a ModuleSet have a ModuleSetID, which can be used only in the context of
// Modules in a common ModuleSet.
type ModuleSet interface {
	// Modules returns the Modules within this ModuleSet.
	Modules() []Module

	isModuleSet()
}

// ModuleSetExternalDependencyModulePins returns the combined list of external
// dependencies for all Modules in a ModuleSet.
//
// Since ExternalDependencyModulePins is defined to have the same commit for a given dependency,
// this is just the union of ModulePins from the Modules.
func ModuleSetExternalDependencyModulePins(ctx context.Context, moduleSet ModuleSet) ([]ModulePin, error) {
	var combinedModulePins []ModulePin
	for _, module := range moduleSet.Modules() {
		modulePins, err := module.ExternalDependencyModulePins(ctx)
		if err != nil {
			return nil, err
		}
		combinedModulePins = append(combinedModulePins, modulePins...)
	}
	return uniqueSortedModulePins(combinedModulePins), nil
}

func GetFileInfo(
	ctx context.Context,
	moduleSet ModuleSet,
	path string,
) (FileInfo, error) {
	return nil, nil
}

func ModuleSetToExportedProtoFileBucket(
	ctx context.Context,
	//moduleReader ModuleReader,
	moduleSet ModuleSet,
) {
}
