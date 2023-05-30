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

package generate

import (
	"context"
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
)

func runV1(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) (retErr error) {
	logger := container.Logger()
	if flags.IncludeWKT && !flags.IncludeImports {
		// You need to set --include-imports if you set --include-wkt, which isn’t great. The alternative is to have
		// --include-wkt implicitly set --include-imports, but this could be surprising. Or we could rename
		// --include-wkt to --include-imports-and/with-wkt. But the summary is that the flag only makes sense
		// in the context of including imports.
		return appcmd.NewInvalidArgumentErrorf("Cannot set --%s without --%s", includeWKTFlagName, includeImportsFlagName)
	}
	if err := bufcli.ValidateErrorFormatFlag(flags.ErrorFormat, errorFormatFlagName); err != nil {
		return err
	}
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, ".")
	if err != nil {
		return err
	}
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, input)
	if err != nil {
		return err
	}
	storageosProvider := bufcli.NewStorageosProvider(flags.DisableSymlinks)
	runner := command.NewRunner()
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	genConfig, err := bufgen.ReadConfig(
		ctx,
		logger,
		bufgen.NewProvider(logger),
		readWriteBucket,
		bufgen.ReadConfigWithOverride(flags.Template),
	)
	if err != nil {
		return err
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	imageConfigReader, err := bufcli.NewWireImageConfigReader(
		container,
		storageosProvider,
		runner,
		clientConfig,
	)
	if err != nil {
		return err
	}
	imageConfigs, fileAnnotations, err := imageConfigReader.GetImageConfigs(
		ctx,
		container,
		ref,
		flags.Config,
		flags.Paths,        // we filter on files
		flags.ExcludePaths, // we exclude these paths
		false,              // input files must exist
		false,              // we must include source info for generation
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(container.Stderr(), fileAnnotations, flags.ErrorFormat); err != nil {
			return err
		}
		return bufcli.ErrFileAnnotation
	}
	images := make([]bufimage.Image, 0, len(imageConfigs))
	for _, imageConfig := range imageConfigs {
		images = append(images, imageConfig.Image())
	}
	image, err := bufimage.MergeImages(images...)
	if err != nil {
		return err
	}
	generateOptions := []bufgen.GenerateOption{
		bufgen.GenerateWithBaseOutDirPath(flags.BaseOutDirPath),
	}
	if flags.IncludeImports {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithIncludeImports(),
		)
	}
	if flags.IncludeWKT {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithIncludeWellKnownTypes(),
		)
	}
	wasmEnabled, err := bufcli.IsAlphaWASMEnabled(container)
	if err != nil {
		return err
	}
	if wasmEnabled {
		generateOptions = append(
			generateOptions,
			bufgen.GenerateWithWASMEnabled(),
		)
	}
	var includedTypes []string
	if len(flags.Types) > 0 || len(flags.TypesDeprecated) > 0 {
		// command-line flags take precedence
		includedTypes = append(flags.Types, flags.TypesDeprecated...)
	} else if genConfig.TypesConfig != nil {
		includedTypes = genConfig.TypesConfig.Include
	}
	if len(includedTypes) > 0 {
		image, err = bufimageutil.ImageFilteredByTypes(image, includedTypes...)
		if err != nil {
			return err
		}
	}
	wasmPluginExecutor, err := bufwasm.NewPluginExecutor(
		filepath.Join(container.CacheDirPath(), bufcli.WASMCompilationCacheDir))
	if err != nil {
		return err
	}
	return bufgen.NewGenerator(
		logger,
		storageosProvider,
		runner,
		wasmPluginExecutor,
		clientConfig,
	).Generate(
		ctx,
		container,
		genConfig,
		image,
		generateOptions...,
	)
}
