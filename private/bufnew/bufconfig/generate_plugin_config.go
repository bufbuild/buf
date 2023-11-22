// Copyright 2020-2023 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
	"github.com/bufbuild/buf/private/pkg/encoding"
)

const remoteAlphaPluginDeprecationMessage = "the remote field no longer works as " +
	"the remote generation alpha has been deprecated, see the migration guide to " +
	"now-stable remote plugins: https://buf.build/docs/migration-guides/migrate-remote-generation-alpha/#migrate-to-remote-plugins"

// GenerateStrategy is the generation strategy for a protoc plugin.
//
// TODO: Should this type live in this package? Perhaps it should live in the package that handles generation?
// TODO: The same question can be asked for FieldOption and FileOption.
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

// PluginConfigType is a plugin configuration type.
type PluginConfigType int

const (
	// PluginConfigTypeRemote is the remote plugin config type.
	PluginConfigTypeRemote PluginConfigType = iota + 1
	// PluginConfigTypeBinary is the binary plugin config type.
	PluginConfigTypeBinary
	// PluginConfigTypeProtocBuiltin is the protoc built-in plugin config type.
	PluginConfigTypeProtocBuiltin
	// PluginConfigTypeLocal is the local plugin config type. This type indicates
	// it is to be determined whether the plugin is binary or protoc built-in.
	// We defer further classification to the plugin executor. In v2 the exact
	// plugin config type is always specified and it will never be just local.
	PluginConfigTypeLocal
)

// GeneratePluginConfig is a configuration for a plugin.
type GeneratePluginConfig interface {
	// Type returns the plugin type. This is never the zero value.
	Type() PluginConfigType
	// Name returns the plugin name. This is never empty.
	Name() string
	// Out returns the output directory for generation. This is never empty.
	Out() string
	// Opt returns the plugin options as a comma seperated string.
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
	// This is not empty only when the plugin is binary.
	Path() []string
	// ProtocPath returns a path to protoc.
	//
	// This is not empty only when the plugin is protoc-builtin
	ProtocPath() string
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

func parseStrategy(s string) (GenerateStrategy, error) {
	switch s {
	case "", "directory":
		return GenerateStrategyDirectory, nil
	case "all":
		return GenerateStrategyAll, nil
	default:
		return 0, fmt.Errorf("unknown strategy: %s", s)
	}
}

type pluginConfig struct {
	pluginConfigType PluginConfigType
	name             string
	out              string
	opt              string
	includeImports   bool
	includeWKT       bool
	strategy         GenerateStrategy
	path             []string
	protocPath       string
	remoteHost       string
	revision         int
}

func (p *pluginConfig) Type() PluginConfigType {
	return p.pluginConfigType
}

func (p *pluginConfig) Name() string {
	return p.name
}

func (p *pluginConfig) Out() string {
	return p.out
}

func (p *pluginConfig) Opt() string {
	return p.opt
}

func (p *pluginConfig) IncludeImports() bool {
	return p.includeImports
}

func (p *pluginConfig) IncludeWKT() bool {
	return p.includeWKT
}

func (p *pluginConfig) Strategy() GenerateStrategy {
	return p.strategy
}

func (p *pluginConfig) Path() []string {
	return p.path
}

func (p *pluginConfig) ProtocPath() string {
	return p.protocPath
}

func (p *pluginConfig) RemoteHost() string {
	return p.remoteHost
}

func (p *pluginConfig) Revision() int {
	return p.revision
}

func (p *pluginConfig) isGeneratePluginConfig() {}

// TODO: figure out where is the best place to do parameter validation, here or in new*plugin.
func newPluginConfigFromExternalV1(
	externalConfig externalGeneratePluginConfigV1,
) (GeneratePluginConfig, error) {
	if externalConfig.Remote != "" {
		return nil, errors.New(remoteAlphaPluginDeprecationMessage)
	}
	// In v1 config, only plugin and name are allowed, since remote alpha plugin
	// has been deprecated.
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
		if _, _, _, _, err := bufremoteplugin.ParsePluginVersionPath(pluginIdentifier); err == nil {
			// A remote alpha plugin name is not a valid remote plugin reference.
			return nil, fmt.Errorf("invalid remote plugin reference: %s", pluginIdentifier)
		}
	case externalConfig.Name != "":
		pluginIdentifier = externalConfig.Name
		if _, _, _, _, err := bufremoteplugin.ParsePluginVersionPath(pluginIdentifier); err == nil {
			return nil, fmt.Errorf("invalid plugin name %s, did you mean to use a remote plugin?", pluginIdentifier)
		}
		if bufpluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
			// A remote alpha plugin name is not a valid local plugin name.
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
	opt, err := encoding.InterfaceSliceOrStringToCommaSepString(externalConfig.Opt)
	if err != nil {
		return nil, err
	}
	path, err := encoding.InterfaceSliceOrStringToStringSlice(externalConfig.Path)
	if err != nil {
		return nil, err
	}
	if externalConfig.Plugin != "" && bufpluginref.IsPluginReferenceOrIdentity(pluginIdentifier) {
		// TODO: Is checkPathAndStrategyUnset the best way to validate this?
		if err := checkPathAndStrategyUnset(externalConfig, pluginIdentifier); err != nil {
			return nil, err
		}
		return newRemotePluginConfig(
			externalConfig.Plugin,
			externalConfig.Revision,
			externalConfig.Out,
			opt,
			false,
			false,
		)
	}
	// At this point the plugin must be local, regardless whehter it's specified
	// by key 'plugin' or 'name'.
	if len(path) > 0 {
		return newBinaryPluginConfig(
			pluginIdentifier,
			path,
			strategy,
			externalConfig.Out,
			opt,
			false,
			false,
		)
	}
	if externalConfig.ProtocPath != "" {
		return newProtocBuiltinPluginConfig(
			pluginIdentifier,
			externalConfig.ProtocPath,
			externalConfig.Out,
			opt,
			false,
			false,
			strategy,
		)
	}
	// It could be either binary or protoc built-in. We defer to the plugin executor
	// to decide whether the plugin is protoc-builtin or binary.
	return newLocalPluginConfig(
		pluginIdentifier,
		strategy,
		externalConfig.Out,
		opt,
		false,
		false,
	)
}

func newLocalPluginConfig(
	name string,
	strategy GenerateStrategy,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) (*pluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &pluginConfig{
		name:           name,
		strategy:       strategy,
		out:            out,
		opt:            opt,
		includeImports: includeImports,
		includeWKT:     includeWKT,
	}, nil
}

func newBinaryPluginConfig(
	name string,
	path []string,
	strategy GenerateStrategy,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) (*pluginConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("must specify a path to the plugin")
	}
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &pluginConfig{
		name:           name,
		path:           path,
		strategy:       strategy,
		out:            out,
		opt:            opt,
		includeImports: includeImports,
		includeWKT:     includeWKT,
	}, nil
}

func newProtocBuiltinPluginConfig(
	name string,
	protocPath string,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
	strategy GenerateStrategy,
) (*pluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	return &pluginConfig{
		name:           name,
		protocPath:     protocPath,
		out:            out,
		opt:            opt,
		strategy:       strategy,
		includeImports: includeImports,
		includeWKT:     includeWKT,
	}, nil
}

func newRemotePluginConfig(
	name string,
	revision int,
	out string,
	opt string,
	includeImports bool,
	includeWKT bool,
) (*pluginConfig, error) {
	if includeWKT && !includeImports {
		return nil, errors.New("cannot include well-known types without including imports")
	}
	remoteHost, err := parseRemoteHostName(name)
	if err != nil {
		return nil, err
	}
	return &pluginConfig{
		name:           name,
		remoteHost:     remoteHost,
		revision:       revision,
		out:            out,
		opt:            opt,
		includeImports: includeImports,
		includeWKT:     includeWKT,
	}, nil
}

func parseRemoteHostName(fullName string) (string, error) {
	if identity, err := bufpluginref.PluginIdentityForString(fullName); err == nil {
		return identity.Remote(), nil
	}
	reference, err := bufpluginref.PluginReferenceForString(fullName, 0)
	if err == nil {
		return reference.Remote(), nil
	}
	return "", err
}

func checkPathAndStrategyUnset(plugin externalGeneratePluginConfigV1, pluginIdentifier string) error {
	if plugin.Path != nil {
		return fmt.Errorf("remote plugin %s cannot specify a path", pluginIdentifier)
	}
	if plugin.Strategy != "" {
		return fmt.Errorf("remote plugin %s cannot specify a strategy", pluginIdentifier)
	}
	if plugin.ProtocPath != "" {
		return fmt.Errorf("remote plugin %s cannot specify a protoc path", pluginIdentifier)
	}
	return nil
}
