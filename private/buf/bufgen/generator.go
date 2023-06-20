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
	"fmt"

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

type generateOptions struct {
	genConfig             string
	moduleConfig          string
	input                 string
	baseOutDir            string
	typesIncluded         []string
	includeImports        bool
	includeWellKnownTypes bool
	pathsSpecified        []string
	pathsExcluded         []string
	errorFormat           string
	wasmEnbaled           bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}

type generator struct {
	logger             *zap.Logger
	storageosProvider  storageos.Provider
	readWriteBucket    storage.ReadWriteBucket
	runner             command.Runner
	clientConfig       *connectclient.Config
	imageConfigReader  bufwire.ImageConfigReader
	wasmPluginExecutor *bufwasm.WASMPluginExecutor
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	readWriteBucket storage.ReadWriteBucket,
	runner command.Runner,
	clientConfig *connectclient.Config,
	imageConfigReader bufwire.ImageConfigReader,
	wasmPluginExecutor *bufwasm.WASMPluginExecutor,
) *generator {
	return &generator{
		logger:             logger,
		storageosProvider:  storageosProvider,
		readWriteBucket:    readWriteBucket,
		runner:             runner,
		clientConfig:       clientConfig,
		imageConfigReader:  imageConfigReader,
		wasmPluginExecutor: wasmPluginExecutor,
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container appflag.Container,
	generateOptions ...GenerateOption,
) error {
	options := newGenerateOptions()
	for _, option := range generateOptions {
		option(options)
	}
	configVersion, err := internal.ReadConfigVersion(
		ctx,
		g.logger,
		g.readWriteBucket,
		internal.ReadConfigWithOverride(options.genConfig),
	)
	if err != nil {
		return err
	}
	var generatorForConfigVersion versionSpecificGenerator
	switch configVersion {
	case internal.V2Version:
		generatorForConfigVersion = bufgenv2.NewGenerator(
			g.logger,
			g.storageosProvider,
			g.runner,
			g.wasmPluginExecutor,
			g.clientConfig,
			g.imageConfigReader,
			g.readWriteBucket,
		)
	case internal.V1Version, internal.V1Beta1Version:
		generatorForConfigVersion = bufgenv1.NewGenerator(
			g.logger,
			g.storageosProvider,
			g.runner,
			g.wasmPluginExecutor,
			g.clientConfig,
			g.imageConfigReader,
			g.readWriteBucket,
		)
	default:
		return fmt.Errorf(`no version set. Please add "version: %s"`, internal.V2Version)
	}
	return generatorForConfigVersion.Generate(
		ctx,
		container,
		options.genConfig,
		options.moduleConfig,
		options.input,
		options.baseOutDir,
		options.typesIncluded,
		options.pathsSpecified,
		options.pathsExcluded,
		options.includeImports,
		options.includeWellKnownTypes,
		options.errorFormat,
		options.wasmEnbaled,
	)
}

type versionSpecificGenerator interface {
	Generate(
		ctx context.Context,
		container appflag.Container,
		genTemplatePath string,
		moduleConfigPathOverride string,
		inputSpecified string,
		baseOutDir string,
		typesIncludedOverride []string,
		pathsSpecifiedOverride []string,
		pathsExcludedOverride []string,
		includeImportsOverride bool,
		includeWellKnownTypesOverride bool,
		errorFormat string,
		wasmEnabled bool,
	) error
}
