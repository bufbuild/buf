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

	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto/appprotoos"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/zap"
)

const (
	// javaPackagePrefix is the default java_package prefix used in the JavaPackage modifier.
	javaPackagePrefix = "com."
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
	)
}

func (g *generator) generate(
	ctx context.Context,
	container app.EnvStdioContainer,
	config *Config,
	image bufimage.Image,
	baseOutDirPath string,
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
			bufimage.ImagesToCodeGeneratorRequests(pluginImages, pluginConfig.Opt),
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
	sweeper := bufimagemodify.NewFileOptionSweeper()
	modifier, err := modifierFromManagedConfig(config.ManagedConfig, sweeper)
	if err != nil {
		return err
	}
	if config.ManagedConfig != nil {
		managedModeModifier, err := managedModeModifier(config.PluginConfigs, sweeper)
		if err != nil {
			return err
		}
		modifier = bufimagemodify.Merge(modifier, managedModeModifier)
	}
	if modifier != nil {
		// Add the sweeper's modifier last so that all of its marks are swept up.
		modifier = bufimagemodify.Merge(modifier, bufimagemodify.ModifierFunc(sweeper.Sweep))
		if err := modifier.Modify(ctx, image); err != nil {
			return err
		}
	}
	return nil
}

// modifierFromManagedConfig returns a new Modifier for the given ManagedConfig.
func modifierFromManagedConfig(managedConfig *ManagedConfig, sweeper bufimagemodify.Sweeper) (bufimagemodify.Modifier, error) {
	if managedConfig == nil {
		return nil, nil
	}
	var modifier bufimagemodify.Modifier
	if managedConfig.CcEnableArenas != nil {
		modifier = bufimagemodify.Merge(
			modifier,
			bufimagemodify.CcEnableArenas(sweeper, *managedConfig.CcEnableArenas),
		)
	}
	if managedConfig.JavaMultipleFiles != nil {
		modifier = bufimagemodify.Merge(
			modifier,
			bufimagemodify.JavaMultipleFiles(sweeper, *managedConfig.JavaMultipleFiles),
		)
	}
	if managedConfig.OptimizeFor != nil {
		modifier = bufimagemodify.Merge(
			modifier,
			bufimagemodify.OptimizeFor(sweeper, *managedConfig.OptimizeFor),
		)
	}
	if managedConfig.GoPackagePrefixConfig != nil {
		goPackageModifier, err := bufimagemodify.GoPackage(
			sweeper,
			managedConfig.GoPackagePrefixConfig.Default,
			managedConfig.GoPackagePrefixConfig.Except,
			managedConfig.GoPackagePrefixConfig.Override,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to construct go_package modifier: %w", err)
		}
		modifier = bufimagemodify.Merge(
			modifier,
			goPackageModifier,
		)
	}
	// TODO: Add support for JavaStringCheckUTF8.
	return modifier, nil
}

// managedModeModifier returns the Managed Mode modifier.
func managedModeModifier(pluginConfigs []*PluginConfig, sweeper bufimagemodify.Sweeper) (bufimagemodify.Modifier, error) {
	// TODO: Implement the following modifiers and include them in
	// the NewMultiModifier below.
	//
	// bufimagemodify.SwiftPrefix(sweeper),
	// bufimagemodify.PhpClassPrefix(sweeper),
	// bufimagemodify.PhpNamespace(sweeper),
	// bufimagemodify.PhpMetadataNamespace(sweeper),
	// bufimagemodify.RubyPackage(sweeper),
	modifier := bufimagemodify.NewMultiModifier(
		bufimagemodify.JavaOuterClassname(sweeper),
		bufimagemodify.ObjcClassPrefix(sweeper),
		bufimagemodify.CsharpNamespace(sweeper),
	)
	javaPackageModifier, err := bufimagemodify.JavaPackage(sweeper, javaPackagePrefix)
	if err != nil {
		return nil, err
	}
	return bufimagemodify.Merge(modifier, javaPackageModifier), nil
}

type generateOptions struct {
	baseOutDirPath string
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
