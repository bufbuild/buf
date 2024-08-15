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

package bufconfig

import (
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// GenerateStrategy is the generation strategy for a protoc plugin.
type GenerateStrategy int

const (
	// GenerateStrategyDirectory is the strategy to generate per directory.
	//
	// This is the default value for local plugins.
	GenerateStrategyDirectory GenerateStrategy = 1
	// GenerateStrategyAll is the strategy to generate with all files at once.
	//
	// This is the only strategy for remote plugins.
	GenerateStrategyAll GenerateStrategy = 2
)

// GeneratePluginConfigType is a generate plugin configuration type.
type GeneratePluginConfigType int

const (
	// GeneratePluginConfigTypeRemote is the remote plugin config type.
	GeneratePluginConfigTypeRemote GeneratePluginConfigType = iota + 1
	// GeneratePluginConfigTypeLocal is the local plugin config type.
	GeneratePluginConfigTypeLocal
	// GeneratePluginConfigTypeProtocBuiltin is the protoc built-in plugin config type.
	GeneratePluginConfigTypeProtocBuiltin
	// GeneratePluginConfigTypeLocalOrProtocBuiltin is a special plugin config type. This type indicates
	// it is to be determined whether the plugin is local or protoc built-in.
	// We defer further classification to the plugin executor. In v2 the exact
	// plugin config type is always specified and it will never be just local.
	GeneratePluginConfigTypeLocalOrProtocBuiltin
)

var (
	// ProtocProxyPluginNames are the names of the plugins that should be proxied through protoc
	// in the absence of a binary.
	ProtocProxyPluginNames = map[string]struct{}{
		"cpp":    {},
		"csharp": {},
		"java":   {},
		"js":     {},
		"objc":   {},
		"php":    {},
		"python": {},
		"pyi":    {},
		"ruby":   {},
		"kotlin": {},
		"rust":   {},
	}
)

// GeneratePluginConfig is a configuration for a plugin.
type GeneratePluginConfig interface {
	// Type returns the plugin type. This is never the zero value.
	Type() GeneratePluginConfigType
	// Name returns the plugin name. This is never empty.
	Name() string
	// Out returns the output directory for generation. This is never empty.
	Out() string
	// Opt returns the plugin options as a comma separated string.
	Opt() string
	// IncludeImports returns whether to generate code for imported files. This
	// is always false in v1.
	IncludeImports() bool
	// IncludeWKT returns whether to generate code for the well-known types.
	// This returns true only if IncludeImports returns true. This is always
	// false in v1.
	IncludeWKT() bool
	// Strategy returns the generation strategy.
	//
	// This is not empty only when the plugin is local, binary or protoc builtin.
	Strategy() GenerateStrategy
	// Path returns the path, including arguments, to invoke the binary plugin.
	//
	// This is not empty only when the plugin is local.
	Path() []string
	// ProtocPath returns a path to protoc, including any extra arguments.
	//
	// This is not empty only when the plugin is protoc-builtin.
	ProtocPath() []string
	// RemoteHost returns the remote host of the remote plugin.
	//
	// This is not empty only when the plugin is remote.
	RemoteHost() string
	// Revision returns the revision of the remote plugin.
	//
	// This is not empty only when the plugin is remote.
	Revision() int

	isGeneratePluginConfig()
}

// NewRemoteGeneratePluginConfig returns a new GeneratePluginConfig for a remote plugin.
func NewRemoteGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	revision int,
) (GeneratePluginConfig, error) {
	return newRemoteGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		revision,
	)
}

// NewLocalOrProtocBuiltinGeneratePluginConfig returns a new GeneratePluginConfig for a local or protoc builtin plugin.
func NewLocalOrProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	strategy *GenerateStrategy,
) (GeneratePluginConfig, error) {
	return newLocalOrProtocBuiltinGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		strategy,
	)
}

// NewLocalGeneratePluginConfig returns a new GeneratePluginConfig for a local plugin.
func NewLocalGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	strategy *GenerateStrategy,
	path []string,
) (GeneratePluginConfig, error) {
	return newLocalGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		strategy,
		path,
	)
}

// NewProtocBuiltinGeneratePluginConfig returns a new GeneratePluginConfig for a protoc
// builtin plugin.
func NewProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	strategy *GenerateStrategy,
	protocPath []string,
) (GeneratePluginConfig, error) {
	return newProtocBuiltinGeneratePluginConfig(
		name,
		out,
		opt,
		includeImports,
		includeWKT,
		strategy,
		protocPath,
	)
}

// NewGeneratePluginConfigWithIncludeImportsAndWKT returns a GeneratePluginConfig the
// same as the input, with include imports and include wkt overridden.
func NewGeneratePluginConfigWithIncludeImportsAndWKT(
	config GeneratePluginConfig,
	includeImports bool,
	includeWKT bool,
) (GeneratePluginConfig, error) {
	originalConfig, ok := config.(*generatePluginConfig)
	if !ok {
		return nil, syserror.Newf("unknown implementation of GeneratePluginConfig: %T", config)
	}
	generatePluginConfig := *originalConfig
	if includeImports {
		generatePluginConfig.includeImports = true
	}
	if includeWKT {
		generatePluginConfig.includeWKT = true
	}
	return &generatePluginConfig, nil
}

// *** PRIVATE ***

type generatePluginConfig struct {
	generatePluginConfigType GeneratePluginConfigType
	name                     string
	out                      string
	opts                     []string
	includeImports           bool
	includeWKT               bool
	strategy                 *GenerateStrategy
	path                     []string
	protocPath               []string
	remoteHost               string
	revision                 int
}

func newGeneratePluginConfigFromExternalV1Beta1(
	externalConfig externalGeneratePluginConfigV1Beta1,
) (GeneratePluginConfig, error) {
	if externalConfig.Name == "" {
		return nil, errors.New("plugin name is required")
	}
	if externalConfig.Out == "" {
		return nil, fmt.Errorf("out is required for plugin %s", externalConfig.Name)
	}
	strategy, err := parseStrategy(externalConfig.Strategy)
	if err != nil {
		return nil, err
	}
	opt, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	if externalConfig.Path != "" {
		return newLocalGeneratePluginConfig(
			externalConfig.Name,
			externalConfig.Out,
			opt,
			false,
			false,
			strategy,
			[]string{externalConfig.Path},
		)
	}
	return newLocalOrProtocBuiltinGeneratePluginConfig(
		externalConfig.Name,
		externalConfig.Out,
		opt,
		false,
		false,
		strategy,
	)
}

func newGeneratePluginConfigFromExternalV1(
	externalConfig externalGeneratePluginConfigV1,
) (GeneratePluginConfig, error) {
	if externalConfig.Plugin == "" && externalConfig.Name == "" {
		return nil, fmt.Errorf("one of plugin or name is required")
	}
	if externalConfig.Plugin != "" && externalConfig.Name != "" {
		return nil, fmt.Errorf("only one of plugin or name can be set")
	}
	var pluginIdentifier string
	switch {
	case externalConfig.Plugin != "":
		pluginIdentifier = externalConfig.Plugin
	case externalConfig.Name != "":
		pluginIdentifier = externalConfig.Name
		if bufremotepluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
			// A plugin reference is not a valid local plugin name.
			return nil, fmt.Errorf("invalid local plugin name: %s", pluginIdentifier)
		}
	}
	if externalConfig.Out == "" {
		return nil, fmt.Errorf("out is required for plugin %s", pluginIdentifier)
	}
	strategy, err := parseStrategy(externalConfig.Strategy)
	if err != nil {
		return nil, err
	}
	opt, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Path)
	if err != nil {
		return nil, err
	}
	protocPath, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.ProtocPath)
	if err != nil {
		return nil, err
	}
	if externalConfig.Plugin != "" && bufremotepluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
		if externalConfig.Path != nil {
			return nil, fmt.Errorf("cannot specify path for remote plugin %s", externalConfig.Plugin)
		}
		if externalConfig.Strategy != "" {
			return nil, fmt.Errorf("cannot specify strategy for remote plugin %s", externalConfig.Plugin)
		}
		if externalConfig.ProtocPath != nil {
			return nil, fmt.Errorf("cannot specify protoc_path for remote plugin %s", externalConfig.Plugin)
		}
		return newRemoteGeneratePluginConfig(
			externalConfig.Plugin,
			externalConfig.Out,
			opt,
			false,
			false,
			externalConfig.Revision,
		)
	}
	// At this point the plugin must be local, regardless whehter it's specified
	// by key 'plugin' or 'name'.
	if len(path) > 0 {
		return newLocalGeneratePluginConfig(
			pluginIdentifier,
			externalConfig.Out,
			opt,
			false,
			false,
			strategy,
			path,
		)
	}
	if externalConfig.ProtocPath != nil {
		return newProtocBuiltinGeneratePluginConfig(
			pluginIdentifier,
			externalConfig.Out,
			opt,
			false,
			false,
			strategy,
			protocPath,
		)
	}
	// It could be either local or protoc built-in. We defer to the plugin executor
	// to decide whether the plugin is protoc-builtin or local.
	return newLocalOrProtocBuiltinGeneratePluginConfig(
		pluginIdentifier,
		externalConfig.Out,
		opt,
		false,
		false,
		strategy,
	)
}

func newGeneratePluginConfigFromExternalV2(
	externalConfig externalGeneratePluginConfigV2,
) (GeneratePluginConfig, error) {
	var pluginTypeCount int
	if externalConfig.Remote != nil {
		pluginTypeCount++
	}
	if externalConfig.Local != nil {
		pluginTypeCount++
	}
	if externalConfig.ProtocBuiltin != nil {
		pluginTypeCount++
	}
	if pluginTypeCount == 0 {
		return nil, errors.New("must specify one of remote, local or protoc_builtin")
	}
	if pluginTypeCount > 1 {
		return nil, errors.New("only one of remote, local or protoc_builtin")
	}
	if externalConfig.Out == "" {
		return nil, errors.New("must specify out")
	}
	var strategy string
	if externalConfig.Strategy != nil {
		strategy = *externalConfig.Strategy
	}
	parsedStrategy, err := parseStrategy(strategy)
	if err != nil {
		return nil, err
	}
	opt, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	switch {
	case externalConfig.Remote != nil:
		var revision int
		if externalConfig.Revision != nil {
			revision = *externalConfig.Revision
		}
		if externalConfig.Strategy != nil {
			return nil, fmt.Errorf("cannot specify strategy for remote plugin %s", *externalConfig.Remote)
		}
		if externalConfig.ProtocPath != nil {
			return nil, fmt.Errorf("cannot specify protoc_path for remote plugin %s", *externalConfig.Remote)
		}
		return newRemoteGeneratePluginConfig(
			*externalConfig.Remote,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			revision,
		)
	case externalConfig.Local != nil:
		path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Local)
		if err != nil {
			return nil, err
		}
		localPluginName := strings.Join(path, " ")
		if externalConfig.Revision != nil {
			return nil, fmt.Errorf("cannot specify revision for local plugin %s", localPluginName)
		}
		if externalConfig.ProtocPath != nil {
			return nil, fmt.Errorf("cannot specify protoc_path for local plugin %s", localPluginName)
		}
		return newLocalGeneratePluginConfig(
			strings.Join(path, " "),
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			parsedStrategy,
			path,
		)
	case externalConfig.ProtocBuiltin != nil:
		protocPath, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.ProtocPath)
		if err != nil {
			return nil, err
		}
		if externalConfig.Revision != nil {
			return nil, fmt.Errorf("cannot specify revision for protoc built-in plugin %s", *externalConfig.ProtocBuiltin)
		}
		return newProtocBuiltinGeneratePluginConfig(
			*externalConfig.ProtocBuiltin,
			externalConfig.Out,
			opt,
			externalConfig.IncludeImports,
			externalConfig.IncludeWKT,
			parsedStrategy,
			protocPath,
		)
	default:
		return nil, syserror.Newf("must specify one of remote, binary and protoc_builtin")
	}
}

func newRemoteGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	revision int,
) (*generatePluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	remoteHost, err := parseRemoteHostName(name)
	if err != nil {
		return nil, err
	}
	if revision < 0 || revision > math.MaxInt32 {
		return nil, fmt.Errorf("revision %d is out of accepted range %d-%d", revision, 0, math.MaxInt32)
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeRemote,
		name:                     name,
		remoteHost:               remoteHost,
		revision:                 revision,
		out:                      out,
		opts:                     opt,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
	}, nil
}

func newLocalOrProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	strategy *GenerateStrategy,
) (*generatePluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeLocalOrProtocBuiltin,
		name:                     name,
		strategy:                 strategy,
		out:                      out,
		opts:                     opt,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
	}, nil
}

func newLocalGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	strategy *GenerateStrategy,
	path []string,
) (*generatePluginConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeLocal,
		name:                     name,
		path:                     path,
		strategy:                 strategy,
		out:                      out,
		opts:                     opt,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
	}, nil
}

func newProtocBuiltinGeneratePluginConfig(
	name string,
	out string,
	opt []string,
	includeImports bool,
	includeWKT bool,
	strategy *GenerateStrategy,
	protocPath []string,
) (*generatePluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &generatePluginConfig{
		generatePluginConfigType: GeneratePluginConfigTypeProtocBuiltin,
		name:                     name,
		protocPath:               protocPath,
		out:                      out,
		opts:                     opt,
		strategy:                 strategy,
		includeImports:           includeImports,
		includeWKT:               includeWKT,
	}, nil
}

func (p *generatePluginConfig) Type() GeneratePluginConfigType {
	return p.generatePluginConfigType
}

func (p *generatePluginConfig) Name() string {
	return p.name
}

func (p *generatePluginConfig) Out() string {
	return p.out
}

func (p *generatePluginConfig) Opt() string {
	return strings.Join(p.opts, ",")
}

func (p *generatePluginConfig) IncludeImports() bool {
	return p.includeImports
}

func (p *generatePluginConfig) IncludeWKT() bool {
	return p.includeWKT
}

func (p *generatePluginConfig) Strategy() GenerateStrategy {
	if p.strategy == nil {
		return GenerateStrategyDirectory
	}
	return *p.strategy
}

func (p *generatePluginConfig) Path() []string {
	return p.path
}

func (p *generatePluginConfig) ProtocPath() []string {
	return p.protocPath
}

func (p *generatePluginConfig) RemoteHost() string {
	return p.remoteHost
}

func (p *generatePluginConfig) Revision() int {
	return p.revision
}

func (p *generatePluginConfig) isGeneratePluginConfig() {}

func newExternalGeneratePluginConfigV2FromPluginConfig(
	pluginConfig GeneratePluginConfig,
) (externalGeneratePluginConfigV2, error) {
	generatePluginConfig, ok := pluginConfig.(*generatePluginConfig)
	if !ok {
		return externalGeneratePluginConfigV2{}, syserror.Newf("unknown implementation of GeneratePluginConfig: %T", generatePluginConfig)
	}
	externalPluginConfigV2 := externalGeneratePluginConfigV2{
		Out:            generatePluginConfig.Out(),
		IncludeImports: generatePluginConfig.IncludeImports(),
		IncludeWKT:     generatePluginConfig.IncludeWKT(),
	}
	opts := generatePluginConfig.opts
	switch {
	case len(opts) == 1:
		externalPluginConfigV2.Opt = opts[0]
	case len(opts) > 1:
		externalPluginConfigV2.Opt = opts
	}
	strategy := generatePluginConfig.strategy
	switch {
	case strategy != nil && *strategy == GenerateStrategyDirectory:
		externalPluginConfigV2.Strategy = toPointer("directory")
	case strategy != nil && *strategy == GenerateStrategyAll:
		externalPluginConfigV2.Strategy = toPointer("all")
	}
	switch generatePluginConfig.Type() {
	case GeneratePluginConfigTypeRemote:
		externalPluginConfigV2.Remote = toPointer(generatePluginConfig.Name())
		if revision := generatePluginConfig.Revision(); revision != 0 {
			externalPluginConfigV2.Revision = &revision
		}
	case GeneratePluginConfigTypeLocal:
		path := generatePluginConfig.Path()
		switch {
		case len(path) == 1:
			externalPluginConfigV2.Local = path[0]
		case len(path) > 1:
			externalPluginConfigV2.Local = path
		}
	case GeneratePluginConfigTypeProtocBuiltin:
		externalPluginConfigV2.ProtocBuiltin = toPointer(generatePluginConfig.Name())
		if protocPath := generatePluginConfig.ProtocPath(); len(protocPath) > 0 {
			if len(protocPath) == 1 {
				externalPluginConfigV2.ProtocPath = protocPath[0]
			} else {
				externalPluginConfigV2.ProtocPath = protocPath
			}
		}
	case GeneratePluginConfigTypeLocalOrProtocBuiltin:
		binaryName := "protoc-gen-" + generatePluginConfig.Name()
		// First, check if this is a binary.
		_, err := exec.LookPath(binaryName)
		if err == nil || errors.Is(err, exec.ErrDot) {
			externalPluginConfigV2.Local = binaryName
			break
		}
		// If not, check if it is a protoc plugin.
		if _, isProtocBuiltin := ProtocProxyPluginNames[generatePluginConfig.Name()]; isProtocBuiltin {
			externalPluginConfigV2.ProtocBuiltin = toPointer(generatePluginConfig.Name())
			break
		}
		// Otherwise, assume this is a binary.
		externalPluginConfigV2.Local = binaryName
	}
	return externalPluginConfigV2, nil
}

func parseStrategy(s string) (*GenerateStrategy, error) {
	var strategy GenerateStrategy
	switch s {
	case "":
		return nil, nil
	case "directory":
		strategy = GenerateStrategyDirectory
	case "all":
		strategy = GenerateStrategyAll
	default:
		return nil, fmt.Errorf("unknown strategy: %s", s)
	}
	return &strategy, nil
}

func parseRemoteHostName(fullName string) (string, error) {
	if identity, err := bufremotepluginref.PluginIdentityForString(fullName); err == nil {
		return identity.Remote(), nil
	}
	reference, err := bufremotepluginref.PluginReferenceForString(fullName, 0)
	if err == nil {
		return reference.Remote(), nil
	}
	return "", err
}

func toPointer[T any](value T) *T {
	return &value
}
