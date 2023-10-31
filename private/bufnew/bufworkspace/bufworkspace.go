package bufworkspace

import "github.com/bufbuild/buf/private/bufnew/bufmodule"

type Workspace interface {
	ModuleSet() bufmodule.ModuleSet
	TargetPaths(moduleID string) ([]string, error)
	ModuleConfig(moduleID string) (ModuleConfig, error)
	DeclaredDeps() []bufmodule.ModuleReference

	isWorkspace()
}

type ModuleConfig interface {
	ExcludePaths() string
	Lint() LintConfig
	Breaking() BreakingConfig
}

type LintConfig interface{}

type BreakingConfig interface{}
