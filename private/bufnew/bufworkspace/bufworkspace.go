package bufworkspace

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type Workspace interface {
	Modules() []WorkspaceModule
	DeclaredDeps() []bufmodule.ModuleRef
	//GenerateConfigs() []GenerateConfig

	isWorkspace()
}

type WorkspaceModule interface {
	bufmodule.Module

	ModuleConfig() ModuleConfig
	TargetPaths() []string
}

// Can read a single buf.yaml v 1
// Can read a buf.work.yaml
// Can read a buf.yaml v2
func NewWorkspaceForBucket(ctx context.Context, bucket storage.ReadBucket, options ...WorkspaceOption) (Workspace, error) {
	return nil, nil
}

type WorkspaceOption func(*workspaceOptions)

func WorkspaceWithDirPaths(dirPaths []string) WorkspaceOption {
	return nil
}

func WorkspaceWithProtoFilterPaths(paths []string, excludePaths []string) WorkspaceOption {
	return nil
}

// ModuleSetExternalDependencyModulePins returns the combined list of external
// dependencies for all Modules in a ModuleSet.

// Since ExternalDependencyModulePins is defined to have the same commit for a given dependency,
// this is just the union of ModulePins from the Modules.
func WorkspaceNonColocatedModuleDeps(ctx context.Context, workspace Workspace) ([]bufmodule.ModuleDep, error) {
	//var combinedModulePins []ModulePin
	//for _, module := range moduleSet.Modules() {
	//modulePins, err := module.ExternalDependencyModulePins(ctx)
	//if err != nil {
	//return nil, err
	//}
	//combinedModulePins = append(combinedModulePins, modulePins...)
	//}
	//return uniqueSortedModulePins(combinedModulePins), nil
	return nil, errors.New("TODO")
}

func GetWorkspaceFileInfo(
	ctx context.Context,
	workspace Workspace,
	path string,
) (bufmodule.FileInfo, error) {
	return nil, errors.New("TODO")
}

func WorkspaceToModuleReadBucketWithOnlyProtoFiles(
	ctx context.Context,
	workspace Workspace,
) (bufmodule.ModuleReadBucket, error) {
	return nil, errors.New("TODO")
}

type workspaceOptions struct{}

type ModuleConfig interface {
	Version() ConfigVersion

	// Note: You could make the argument that you don't actually need this, however there
	// are situations where you just want to read a configuration on its own without
	// a corresponding Workspace.
	ModuleFullName() bufmodule.ModuleFullName

	RootToExcludes() map[string][]string
	LintConfig() LintConfig
	BreakingConfig() BreakingConfig

	isModuleConfig()
}

type LintConfig interface {
	Version() ConfigVersion

	UseIDs() []string
	ExceptIDs() string
	IgnoreRootPaths() []string
	IgnoreIDToRootPaths() map[string][]string
	EnumZeroValueSuffix() string
	RPCAllowSameRequestResponse() bool
	RPCAllowGoogleProtobufEmptyRequests() bool
	RPCAllowGoogleProtobufEmptyResponses() bool
	ServiceSuffix() string
	AllowCommentIgnores() bool

	isLintConfig()
}

type BreakingConfig interface {
	Version() ConfigVersion

	UseIDs() []string
	ExceptIDs() string
	IgnoreRootPaths() []string
	IgnoreIDToRootPaths() map[string][]string
	IgnoreUnstablePackages() bool

	isBreakingConfig()
}

//type GenerateConfig interface{}
