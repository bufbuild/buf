package bufworkspace

import (
	"context"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

type Workspace interface {
	ModuleSet() bufmodule.ModuleSet
	Config() WorkspaceConfig
	GetTargetPaths(moduleSetID string) ([]string, error)

	isWorkspace()
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

type workspaceOptions struct{}

type WorkspaceConfig interface {
	Version() ConfigVersion

	GetModuleConfig(moduleSetID string) (ModuleConfig, error)
	ModuleConfigs() []ModuleConfig
	//GenerateConfigs() []GenerateConfig
	DeclaredDeps() []bufmodule.ModuleRef

	isWorkspaceConfig()
}

type ModuleConfig interface {
	Version() ConfigVersion

	// Note: You could make the argument that you don't actually need this, however there
	// are situations where you just want to read a configuration on its own without
	// a corresponding Workspace.

	ModuleID() string
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
