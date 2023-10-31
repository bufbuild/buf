package bufworkspace

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

const (
	ConfigVersionV1Beta1 ConfigVersion = iota + 1
	ConfigVersionV1
)

var (
	configVersionToString = map[ConfigVersion]string{
		ConfigVersionV1Beta1: "v1beta1",
		ConfigVersionV1:      "v1",
	}
	stringToConfigVersion = map[string]ConfigVersion{
		"v1beta1": ConfigVersionV1Beta1,
		"v1":      ConfigVersionV1,
	}
)

type ConfigVersion int

func (c ConfigVersion) String() string {
	s, ok := configVersionToString[c]
	if !ok {
		return strconv.Itoa(int(c))
	}
	return s
}

func ParseConfigVersion(s string) (ConfigVersion, error) {
	c, ok := stringToConfigVersion[s]
	if !ok {
		return 0, fmt.Errorf("unknown ConfigVersion: %q", s)
	}
	return c, nil
}

type Workspace interface {
	Version() ConfigVersion

	ModuleSet() bufmodule.ModuleSet
	GetTargetPaths(moduleID string) ([]string, error)
	DeclaredDeps() []bufmodule.ModuleReference
	Config() WorkspaceConfig

	isWorkspace()
}

// Can read a single buf.yaml v 1
// Can read a buf.work.yaml
// Can read a buf.yaml v2
func NewWorkspaceForBucket(ctx context.Context, bucket storage.ReadBucket) (Workspace, error) {
	return nil, nil
}

type WorkspaceConfig interface {
	Version() ConfigVersion

	GetModuleConfig(moduleID string) (ModuleConfig, error)
	ModuleConfigs() []ModuleConfig
	//GenerateConfigs() []GenerateConfig

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
