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

package bufgen

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv1"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv2"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

type ExternalConfigVersion = internal.ExternalConfigVersion
type ExternalConfigV2 = bufgenv2.ExternalConfigV2
type ExternalPluginConfigV2 = bufgenv2.ExternalPluginConfigV2
type ExternalConfigV1 = bufgenv1.ExternalConfigV1
type ExternalPluginConfigV1 = bufgenv1.ExternalPluginConfigV1
type ExternalManagedConfigV1 = bufgenv1.ExternalManagedConfigV1
type ExternalOptimizeForConfigV1 = bufgenv1.ExternalOptimizeForConfigV1
type ExternalConfigV1Beta1 = bufgenv1.ExternalConfigV1Beta1

const (
	// ExternalConfigFilePath is the default external configuration file path.
	ExternalConfigFilePath = internal.ExternalConfigFilePath
	// V1Version is the string used to identify the v1 version of the generate template.
	V1Version = internal.V1Version
	// V1Beta1Version is the string used to identify the v1beta1 version of the generate template.
	V1Beta1Version = internal.V1Beta1Version
	// V2Version is the string used to identify the v2 version of the generate template.
	V2Version = internal.V2Version
)

// Generators generates.
type Generator interface {
	// Generate reads inputs into images, modifies them and generates code.
	Generate(
		ctx context.Context,
		container appflag.Container,
		generateOptions ...GenerateOption,
	) error
}

// NewGenerator returns a new Generator.
func NewGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	readWriteBucket storage.ReadWriteBucket,
	runner command.Runner,
	clientConfig *connectclient.Config,
	imageConfigReader bufwire.ImageConfigReader,
	wasmPluginExecutor *bufwasm.WASMPluginExecutor,
) Generator {
	return newGenerator(
		logger,
		storageosProvider,
		readWriteBucket,
		runner,
		clientConfig,
		imageConfigReader,
		wasmPluginExecutor,
	)
}

// GenerateOption is an option for Generate.
type GenerateOption func(*generateOptions)

// GenerateWithGenConfig sets generation configuration, which can be
// a path to a local file or config data in json.
func GenerateWithGenConfig(genConfig string) GenerateOption {
	return func(options *generateOptions) {
		options.genConfig = genConfig
	}
}

// GenerateWithModuleConfig sets module configuration, which can be
// a path to a local file or config data in json.
func GenerateWithModuleConfig(moduleConfig string) GenerateOption {
	return func(options *generateOptions) {
		options.moduleConfig = moduleConfig
	}
}

// GenerateWithInputSpecified sets the input to generate code for.
func GenerateWithInputSpecified(input string) GenerateOption {
	return func(options *generateOptions) {
		options.input = input
	}
}

// GenerateWithBaseOutDir sets the base output directory.
func GenerateWithBaseOutDir(baseoutDir string) GenerateOption {
	return func(options *generateOptions) {
		options.baseOutDir = baseoutDir
	}
}

// GenerateWithTypesIncluded sets types to generate code for.
func GenerateWithTypesIncluded(typesIncluded []string) GenerateOption {
	return func(options *generateOptions) {
		options.typesIncluded = typesIncluded
	}
}

// GenerateWithIncludeImports includes inputs' imports.
func GenerateWithIncludeImports() GenerateOption {
	return func(options *generateOptions) {
		options.includeImports = true
	}
}

// GenerateWithIncludeWellKnownTypes includes Well-Known Types.
func GenerateWithIncludeWellKnownTypes() GenerateOption {
	return func(options *generateOptions) {
		options.includeWellKnownTypes = true
	}
}

// GenerateWithPathsSpecified sets the paths in inputs to generate code for.
func GenerateWithPathsSpecified(pathsSpecified []string) GenerateOption {
	return func(options *generateOptions) {
		options.pathsSpecified = pathsSpecified
	}
}

// GenerateWithPathsExcluded sets the paths to exclude from code generation.
func GenerateWithPathsExcluded(pathsExcluded []string) GenerateOption {
	return func(options *generateOptions) {
		options.pathsExcluded = pathsExcluded
	}
}

// GenerateWithErrorFormat sets the error format.
func GenerateWithErrorFormat(errorFormat string) GenerateOption {
	return func(options *generateOptions) {
		options.errorFormat = errorFormat
	}
}

// GenerateWithFileAnnotationErr sets the error to return when file annotations
// are printed.
func GenerateWithFileAnnotationErr(annotationErr error) GenerateOption {
	return func(options *generateOptions) {
		options.fileAnnotationErr = annotationErr
	}
}

// GenerateWithWasmEnabled enables Wasm plugins.
func GenerateWithWasmEnabled() GenerateOption {
	return func(options *generateOptions) {
		options.wasmEnbaled = true
	}
}
