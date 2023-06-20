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
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufgen/internal"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv1"
	"github.com/bufbuild/buf/private/buf/bufgen/internal/bufgenv2"
	"github.com/bufbuild/buf/private/buf/bufwire"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
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

type tmpGenerateOptions struct {
	configOverride        string
	typesIncludedOverride []string
	includeImports        bool
	includeWellKnownTypes bool
}

func newTmpGenerateOptions() *tmpGenerateOptions {
	return &tmpGenerateOptions{
		configOverride: internal.ExternalConfigFilePath,
	}
}

type TmpGenerateOption func(*tmpGenerateOptions)

func TmpGenerateWithConfigOverride(configOverride string) TmpGenerateOption {
	return func(options *tmpGenerateOptions) {
		if configOverride != "" {
			options.configOverride = configOverride
		}
	}
}

func TmpGenerateWithTypesIncludedOverride(typesIncludedOverride []string) TmpGenerateOption {
	return func(options *tmpGenerateOptions) {
		options.typesIncludedOverride = typesIncludedOverride
	}
}

func TmpGenerateWithIncludeImports() TmpGenerateOption {
	return func(options *tmpGenerateOptions) {
		options.includeImports = true
	}
}

func TmpGenerateWithIncludeWellKnownTypes() TmpGenerateOption {
	return func(options *tmpGenerateOptions) {
		options.includeWellKnownTypes = true
	}
}

func NewTmpGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	readWriteBucket storage.ReadWriteBucket,
	runner command.Runner,
	clientConfig *connectclient.Config,
	imageConfigReader bufwire.ImageConfigReader,
) *TmpGenerator {
	return &TmpGenerator{
		logger:            logger,
		storageosProvider: storageosProvider,
		readWriteBucket:   readWriteBucket,
		runner:            runner,
		clientConfig:      clientConfig,
		imageConfigReader: imageConfigReader,
	}
}

// TODO: unexport
type TmpGenerator struct {
	logger            *zap.Logger
	storageosProvider storageos.Provider
	readWriteBucket   storage.ReadWriteBucket
	runner            command.Runner
	clientConfig      *connectclient.Config
	imageConfigReader bufwire.ImageConfigReader
}

func (g *TmpGenerator) Generate(
	ctx context.Context,
	container appflag.Container,
	baseOutDir string,
	tmpGenerateOptions ...TmpGenerateOption,
) error {
	options := newTmpGenerateOptions()
	for _, option := range tmpGenerateOptions {
		option(options)
	}
	configVersion, err := internal.ReadConfigVersion(
		ctx,
		g.logger,
		g.readWriteBucket,
		internal.ReadConfigWithOverride(options.configOverride),
	)
	if err != nil {
		return err
	}
	switch configVersion {
	case internal.V2Version:
	case internal.V1Beta1Version, internal.V1Version:
	}
	// typesIncludedOverride := options.typesIncludedOverride
	var (
		inputImages   []bufimage.Image
		imageModifier bufimagemodify.Modifier
		plugins       []internal.PluginConfig
	)
	switch configVersion {
	case internal.V2Version:
		// genConfigV2, err := bufgenv2.ReadConfigV2(
		// 	ctx,
		// 	logger,
		// 	readWriteBucket,
		// 	bufgen.ReadConfigWithOverride(flags.Template),
		// )
		// if err != nil {
		// 	return err
		// }
		// // TODO: implement managed mode
		// imageModifier = nopModifier{}
		// plugins = genConfigV2.Plugins
		// if bufcli.IsInputSpecified(container, flags.InputHashtag) {
		// 	inputRef, err := getInputRefFromCLI(
		// 		ctx,
		// 		container,
		// 		flags.InputHashtag,
		// 	)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	inputImage, err := getInputImage(
		// 		ctx,
		// 		container,
		// 		inputRef,
		// 		imageConfigReader,
		// 		flags.Config,
		// 		flags.Paths,
		// 		flags.ExcludePaths,
		// 		flags.ErrorFormat,
		// 		includedTypesFromCLI,
		// 	)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	inputImages = append(inputImages, inputImage)
		// 	break
		// }
		// for _, inputConfig := range genConfigV2.Inputs {
		// 	includePaths := inputConfig.IncludePaths
		// 	if len(flags.Paths) > 0 {
		// 		includePaths = flags.Paths
		// 	}
		// 	excludePaths := inputConfig.ExcludePaths
		// 	if len(flags.ExcludePaths) > 0 {
		// 		excludePaths = flags.ExcludePaths
		// 	}
		// 	includedTypes := inputConfig.Types
		// 	if len(includedTypesFromCLI) > 0 {
		// 		includedTypes = includedTypesFromCLI
		// 	}
		// 	inputImage, err := getInputImage(
		// 		ctx,
		// 		container,
		// 		inputConfig.InputRef,
		// 		imageConfigReader,
		// 		flags.Config,
		// 		includePaths,
		// 		excludePaths,
		// 		flags.ErrorFormat,
		// 		includedTypes,
		// 	)
		// 	if err != nil {
		// 		return err
		// 	}
		// 	inputImages = append(inputImages, inputImage)
		// }
	case internal.V1Version, internal.V1Beta1Version:
		// genConfigV1, err := bufgenv1.ReadConfigV1(
		// 	ctx,
		// 	logger,
		// 	readWriteBucket,
		// 	bufgen.ReadConfigWithOverride(flags.Template),
		// )
		// if err != nil {
		// 	return err
		// }
		// if imageModifier, err = bufgenv1.NewModifier(
		// 	logger,
		// 	genConfigV1,
		// ); err != nil {
		// 	return err
		// }
		// plugins = genConfigV1.PluginConfigs
		// inputRef, err := getInputRefFromCLI(
		// 	ctx,
		// 	container,
		// 	flags.InputHashtag,
		// )
		// if err != nil {
		// 	return err
		// }
		// var includedTypes []string
		// if typesConfig := genConfigV1.TypesConfig; typesConfig != nil {
		// 	includedTypes = typesConfig.Include
		// }
		// if len(includedTypesFromCLI) > 0 {
		// 	includedTypes = includedTypesFromCLI
		// }
		// inputImage, err := getInputImage(
		// 	ctx,
		// 	container,
		// 	inputRef,
		// 	imageConfigReader,
		// 	flags.Config,
		// 	flags.Paths,
		// 	flags.ExcludePaths,
		// 	flags.ErrorFormat,
		// 	includedTypes,
		// )
		// if err != nil {
		// 	return err
		// }
		// inputImages = append(inputImages, inputImage)
	default:
		return fmt.Errorf(`no version set. Please add "version: %s"`, internal.V2Version)
	}
	generateOptions := []internal.GenerateOption{
		internal.GenerateWithBaseOutDirPath(baseOutDir),
	}
	if options.includeImports {
		generateOptions = append(
			generateOptions,
			internal.GenerateWithIncludeImports(),
		)
	}
	if options.includeWellKnownTypes {
		generateOptions = append(
			generateOptions,
			internal.GenerateWithIncludeWellKnownTypes(),
		)
	}
	wasmEnabled, err := bufcli.IsAlphaWASMEnabled(container)
	if err != nil {
		return err
	}
	if wasmEnabled {
		generateOptions = append(
			generateOptions,
			internal.GenerateWithWASMEnabled(),
		)
	}
	wasmPluginExecutor, err := bufwasm.NewPluginExecutor(
		filepath.Join(
			container.CacheDirPath(),
			bufcli.WASMCompilationCacheDir,
		),
	)
	if err != nil {
		return err
	}
	generator := internal.NewGenerator(
		g.logger,
		g.storageosProvider,
		g.runner,
		wasmPluginExecutor,
		g.clientConfig,
	)
	for _, image := range inputImages {
		if err := imageModifier.Modify(
			ctx,
			image,
		); err != nil {
			return err
		}
		if err := generator.Generate(
			ctx,
			container,
			plugins,
			image,
			generateOptions...,
		); err != nil {
			return err
		}
	}
	return nil
}
