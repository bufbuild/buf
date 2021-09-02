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

package bufgen

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto/appprotoexec"
	"github.com/bufbuild/buf/private/pkg/app/appproto/appprotoos"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

type generator struct {
	logger              *zap.Logger
	appprotoosGenerator appprotoos.Generator
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
) *generator {
	return &generator{
		logger:              logger,
		appprotoosGenerator: appprotoos.NewGenerator(logger, storageosProvider),
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStdioContainer,
	config *Config,
	image bufimage.Image,
	options ...GenerateOption,
) error {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	return g.generate(
		ctx,
		container,
		config,
		image,
		generateOptions.baseOutDirPath,
		generateOptions.includeImports,
	)
}

func (g *generator) generate(
	ctx context.Context,
	container app.EnvStdioContainer,
	config *Config,
	image bufimage.Image,
	baseOutDirPath string,
	includeImports bool,
) error {
	if err := modifyImage(ctx, config, image); err != nil {
		return err
	}
	// We keep this as a variable so we can cache it if we hit StrategyDirectory.
	var imagesByDir []bufimage.Image
	var err error
	for _, pluginConfig := range config.PluginConfigs {
		out := pluginConfig.Out
		if baseOutDirPath != "" && baseOutDirPath != "." {
			out = filepath.Join(baseOutDirPath, out)
		}
		var pluginImages []bufimage.Image
		switch pluginConfig.Strategy {
		case StrategyAll:
			pluginImages = []bufimage.Image{image}
		case StrategyDirectory:
			// If we have not already called this, call it.
			if imagesByDir == nil {
				imagesByDir, err = bufimage.ImageByDir(image)
				if err != nil {
					return err
				}
			}
			pluginImages = imagesByDir
		default:
			return fmt.Errorf("unknown strategy: %v", pluginConfig.Strategy)
		}
		if err := g.appprotoosGenerator.Generate(
			ctx,
			container,
			pluginConfig.Name,
			out,
			bufimage.ImagesToCodeGeneratorRequests(
				pluginImages,
				pluginConfig.Opt,
				appprotoexec.DefaultVersion,
				includeImports,
			),
			appprotoos.GenerateWithPluginPath(pluginConfig.Path),
			appprotoos.GenerateWithCreateOutDirIfNotExists(),
		); err != nil {
			return fmt.Errorf("plugin %s: %v", pluginConfig.Name, err)
		}
	}
	return nil
}

// modifyImage modifies the image according to the given configuration (i.e. Managed Mode).
func modifyImage(
	ctx context.Context,
	config *Config,
	image bufimage.Image,
) error {
	if config.ManagedConfig == nil {
		// If the config is nil, it implies that the
		// user has not enabled managed mode.
		return nil
	}
	sweeper := bufimagemodify.NewFileOptionSweeper()
	modifier, err := newModifier(config.ManagedConfig, sweeper)
	if err != nil {
		return err
	}
	modifier = bufimagemodify.Merge(modifier, bufimagemodify.ModifierFunc(sweeper.Sweep))
	return modifier.Modify(ctx, image)
}

func newModifier(managedConfig *ManagedConfig, sweeper bufimagemodify.Sweeper) (bufimagemodify.Modifier, error) {
	modifier := bufimagemodify.NewMultiModifier(
		bufimagemodify.JavaOuterClassname(sweeper, managedConfig.Overrides[bufimagemodify.JavaOuterClassNameID]),
		bufimagemodify.ObjcClassPrefix(sweeper, managedConfig.Overrides[bufimagemodify.ObjcClassPrefixID]),
		bufimagemodify.CsharpNamespace(sweeper, managedConfig.Overrides[bufimagemodify.CsharpNamespaceID]),
		bufimagemodify.PhpNamespace(sweeper, managedConfig.Overrides[bufimagemodify.PhpNamespaceID]),
		bufimagemodify.PhpMetadataNamespace(sweeper, managedConfig.Overrides[bufimagemodify.PhpMetadataNamespaceID]),
		bufimagemodify.RubyPackage(sweeper, managedConfig.Overrides[bufimagemodify.RubyPackageID]),
	)
	javaPackagePrefix := bufimagemodify.DefaultJavaPackagePrefix
	if managedConfig.JavaPackagePrefix != "" {
		javaPackagePrefix = managedConfig.JavaPackagePrefix
	}
	javaPackageModifier, err := bufimagemodify.JavaPackage(
		sweeper,
		javaPackagePrefix,
		managedConfig.Overrides[bufimagemodify.JavaPackageID],
	)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodify.Merge(modifier, javaPackageModifier)
	javaMultipleFilesValue := bufimagemodify.DefaultJavaMultipleFilesValue
	if managedConfig.JavaMultipleFiles != nil {
		javaMultipleFilesValue = *managedConfig.JavaMultipleFiles
	}
	javaMultipleFilesModifier, err := bufimagemodify.JavaMultipleFiles(
		sweeper,
		javaMultipleFilesValue,
		managedConfig.Overrides[bufimagemodify.JavaMultipleFilesID],
	)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodify.Merge(modifier, javaMultipleFilesModifier)
	if managedConfig.CcEnableArenas != nil {
		ccEnableArenasModifier, err := bufimagemodify.CcEnableArenas(
			sweeper,
			*managedConfig.CcEnableArenas,
			managedConfig.Overrides[bufimagemodify.CcEnableArenasID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(modifier, ccEnableArenasModifier)
	}
	if managedConfig.JavaStringCheckUtf8 != nil {
		javaStringCheckUtf8, err := bufimagemodify.JavaStringCheckUtf8(
			sweeper,
			*managedConfig.JavaStringCheckUtf8,
			managedConfig.Overrides[bufimagemodify.JavaStringCheckUtf8ID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(modifier, javaStringCheckUtf8)
	}
	if managedConfig.OptimizeFor != nil {
		optimizeFor, err := bufimagemodify.OptimizeFor(
			sweeper,
			*managedConfig.OptimizeFor,
			managedConfig.Overrides[bufimagemodify.OptimizeForID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(
			modifier,
			optimizeFor,
		)
	}
	if managedConfig.GoPackagePrefixConfig != nil {
		goPackageModifier, err := bufimagemodify.GoPackage(
			sweeper,
			managedConfig.GoPackagePrefixConfig.Default,
			managedConfig.GoPackagePrefixConfig.Except,
			managedConfig.GoPackagePrefixConfig.Override,
			managedConfig.Overrides[bufimagemodify.GoPackageID],
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct go_package modifier: %w", err)
		}
		modifier = bufimagemodify.Merge(
			modifier,
			goPackageModifier,
		)
	}
	return modifier, nil
}

type generateOptions struct {
	baseOutDirPath string
	includeImports bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
