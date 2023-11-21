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

// TODO: should this type live here? Perhaps it should live in the package that handles generation?
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

// GeneratePluginConfig is a configuration for a plugin.
type GeneratePluginConfig interface {
	Name() string
	Out() string
	Opt() string
	IncludeImports() bool
	IncludeWKT() bool

	isPluginConfig()
}

// LocalPluginConfig is a plugin configuration for a local plugin. It maybe a
// binary or protoc builtin plugin, but its type is undetermined. We defer to
// the plugin executor to decide which type it is.
type LocalPluginConfig interface {
	GeneratePluginConfig
	Strategy() GenerateStrategy

	isLocalPluginConfig()
}

// BinaryPluginConfig is a binary plugin configuration.
type BinaryPluginConfig interface {
	LocalPluginConfig
	Path() []string

	isBinaryPluginConfig()
}

// ProtocBuiltinPluginConfig is a protoc builtin plugin configuration.
type ProtocBuiltinPluginConfig interface {
	LocalPluginConfig
	ProtocPath() string

	isProtocBuiltinPluginConfig()
}

// RemotePluginConfig is a remote plugin configuration.
type RemotePluginConfig interface {
	GeneratePluginConfig
	RemoteHost() string
	Revision() int

	isRemotePluginConfig()
}
