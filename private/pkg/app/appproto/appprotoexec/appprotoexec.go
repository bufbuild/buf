// Copyright 2020-2021 Buf Technologies, Inc.
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

// Package appprotoexec provides protoc plugin handling and execution.
//
// Note this is currently implicitly tested through buf's protoc command.
// If this were split out into a separate package, testing would need to be moved to this package.
package appprotoexec

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	// DefaultMajorVersion is the default major version.
	defaultMajorVersion = 3
	// DefaultMinorVersion is the default minor version.
	defaultMinorVersion = 18
	// DefaultPatchVersion is the default patch version.
	defaultPatchVersion = 0
	// DefaultSuffixVersion is the default suffix version.
	defaultSuffixVersion = ""
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
		"ruby":   {},
		"kotlin": {},
	}

	// DefaultVersion represents the default version to use as compiler version for codegen requests.
	DefaultVersion = newVersion(
		defaultMajorVersion,
		defaultMinorVersion,
		defaultPatchVersion,
		defaultSuffixVersion,
	)
)

// Generator is used to generate code with plugins found on the local filesystem.
type Generator interface {
	// Generate generates a CodeGeneratorResponse for the given pluginName. The
	// pluginName must be available on the system's PATH or one of the plugins
	// built-in to protoc. The plugin path can be overridden via the
	// GenerateWithPluginPath option.
	Generate(
		ctx context.Context,
		container app.EnvStderrContainer,
		pluginName string,
		requests []*pluginpb.CodeGeneratorRequest,
		options ...GenerateOption,
	) (*pluginpb.CodeGeneratorResponse, error)
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) Generator {
	return newGenerator(logger, storageosProvider)
}

// GenerateOption is an option for Generate.
type GenerateOption func(*generateOptions)

// GenerateWithPluginPath returns a new GenerateOption that uses the given
// path to the plugin.
func GenerateWithPluginPath(pluginPath string) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.pluginPath = pluginPath
	}
}
