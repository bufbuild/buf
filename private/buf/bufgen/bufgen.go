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

// Package bufgen does configuration-based generation.
//
// It is used by the buf generate command.
package bufgen

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
)

const (
	// StrategyDirectory is the strategy that says to generate per directory.
	//
	// This is the default value.
	StrategyDirectory Strategy = 1
	// StrategyAll is the strategy that says to generate with all files at once.
	StrategyAll Strategy = 2
)

// Strategy is a generation stategy.
type Strategy int

// ParseStrategy parses the Strategy.
//
// If the empty string is provided, this is interpreted as StrategyDirectory.
func ParseStrategy(s string) (Strategy, error) {
	switch s {
	case "", "directory":
		return StrategyDirectory, nil
	case "all":
		return StrategyAll, nil
	default:
		return 0, fmt.Errorf("unknown strategy: %s", s)
	}
}

// String implements fmt.Stringer.
func (s Strategy) String() string {
	switch s {
	case StrategyDirectory:
		return "directory"
	case StrategyAll:
		return "all"
	default:
		return strconv.Itoa(int(s))
	}
}

// Generator generates Protobuf stubs based on configurations.
type Generator interface {
	// Generate calls the generation logic.
	//
	// The config is assumed to be valid. If created by ReadConfig, it will
	// always be valid.
	Generate(
		ctx context.Context,
		container app.EnvStdioContainer,
		config bufconfig.GenerateConfig,
		images []bufimage.Image,
		options ...GenerateOption,
	) error
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	tracer tracing.Tracer,
	storageosProvider storageos.Provider,
	runner command.Runner,
	// Pass a clientConfig instead of a CodeGenerationServiceClient because the
	// plugins' remotes/registries is not known at this time, and remotes/registries
	// may be different for different plugins.
	clientConfig *connectclient.Config,
) Generator {
	return newGenerator(
		logger,
		tracer,
		storageosProvider,
		runner,
		clientConfig,
	)
}

// GenerateOption is an option for Generate.
type GenerateOption func(*generateOptions)

// GenerateWithBaseOutDirPath returns a new GenerateOption that uses the given
// base directory as the output directory.
//
// The default is to use the current directory.
func GenerateWithBaseOutDirPath(baseOutDirPath string) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.baseOutDirPath = baseOutDirPath
	}
}

// GenerateWithIncludeImportsOverride is a strict override on whether imports are
// generated. This overrides IncludeImports from the GeneratePluginConfig.
//
// This option has presence, i.e. setting this option to false is not the same
// as not setting it, as the latter does not override the config.
//
// Note that this does NOT result in the Well-Known Types being generated when
// set to true, use GenerateWithIncludeWellKnownTypes to include the Well-Known Types.
func GenerateWithIncludeImportsOverride(includeImports bool) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.includeImportsOverride = &includeImports
	}
}

// GenerateWithIncludeWellKnownTypesOverride is a strict override on whether the
// well known types are generated. This overrides IncludeWKT from the GeneratePluginConfig.
//
// This option has presence, i.e. setting this option to false is not the same
// as not setting it, as the latter does not override the config.
//
// Setting this option to true has no effect if GenerateWithIncludeImports is not
// set to true.
func GenerateWithIncludeWellKnownTypesOverride(includeWellKnownTypes bool) GenerateOption {
	return func(generateOptions *generateOptions) {
		generateOptions.includeWellKnownTypesOverride = &includeWellKnownTypes
	}
}
