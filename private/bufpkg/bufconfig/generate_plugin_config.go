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

	isPluginConfig()
}
