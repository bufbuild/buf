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
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appproto/appprotoos"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/osextended"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/zap"
	"golang.org/x/mod/modfile"
)

const (
	// goModuleFile is the name of the Go module file.
	goModuleFile = "go.mod"
	// goPluginName is the name used to configure the protoc-gen-go plugin.
	goPluginName = "go"
	// goGrpcPluginName is the name used to configure the protoc-gen-go-grpc plugin.
	goGrpcPluginName = "go-grpc"
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
	if err := g.modifyImage(ctx, config, image); err != nil {
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

type generateOptions struct {
	baseOutDirPath string
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}

// modifyImage modifies the image according to the given configuration (i.e. Managed Mode).
func (g *generator) modifyImage(
	ctx context.Context,
	config *Config,
	image bufimage.Image,
) error {
	sweeper := bufimagemodify.NewFileOptionSweeper()
	modifier := modifierFromOptions(config.Options, sweeper)
	if config.Managed {
		managedModeModifier, err := g.managedModeModifier(config.PluginConfigs, sweeper)
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

// managedModeModifier returns the Managed Mode modifier.
func (g *generator) managedModeModifier(pluginConfigs []*PluginConfig, sweeper bufimagemodify.Sweeper) (bufimagemodify.Modifier, error) {
	modifier := bufimagemodify.NewMultiModifier(
		// TODO: Implement the following modifiers.
		//
		// bufimagemodify.CsharpNamespace(sweeper),
		bufimagemodify.JavaOuterClassname(sweeper),
	// bufimagemodify.ObjcClassPrefix(sweeper),
	// bufimagemodify.SwiftPrefix(sweeper),
	// bufimagemodify.PhpClassPrefix(sweeper),
	// bufimagemodify.PhpNamespace(sweeper),
	// bufimagemodify.PhpMetadataNamespace(sweeper),
	// bufimagemodify.RubyPackage(sweeper),
	)
	javaPackageModifier, err := bufimagemodify.JavaPackage(sweeper, javaPackagePrefix)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodify.Merge(modifier, javaPackageModifier)

	goPackageModifier, err := g.goPackageModifierFromPluginConfigs(pluginConfigs, sweeper)
	if err != nil {
		return nil, err
	}
	return bufimagemodify.Merge(modifier, goPackageModifier), nil
}

// goPackageModifierFromPluginConfigs returns a new Modifier that sets
// the go_package file option based on the configured output directory
// for the protoc-gen-go[-grpc] plugin. If the protoc-gen-go[-grpc] plugin
// is not configured, a 'nil' Modifier is returned. Otherwise, we attempt to
// resolve the user's Go module name and error if it cannot be resolved.
//
// Note that we can resolve the relative output directory from either the
// protoc-gen-go or protoc-gen-go-grpc plugins because they MUST be placed
// in the same directory to compile.
func (g *generator) goPackageModifierFromPluginConfigs(
	pluginConfigs []*PluginConfig,
	sweeper bufimagemodify.Sweeper,
) (bufimagemodify.Modifier, error) {
	var goPluginOut string
	for _, pluginConfig := range pluginConfigs {
		if pluginConfig.Name == goPluginName || pluginConfig.Name == goGrpcPluginName {
			goPluginOut = pluginConfig.Out
			break
		}
	}
	if goPluginOut == "" {
		// The protoc-gen-go[-grpc] plugin was not configured,
		// so there's nothing to do here.
		return nil, nil
	}
	goModulePath, relativePath, err := resolveGoModulePath()
	if err != nil {
		// The user specified the protoc-gen-go[-grpc] plugin,
		// but a go.mod file could not be resolved. We don't
		// want to fail entirely, but we should at least warn.
		g.logger.Sugar().Warnf("Managed Mode skipping go_package option: %v", err)
		return nil, nil
	}
	return bufimagemodify.GoPackage(
		sweeper,
		normalpath.Join(goModulePath, relativePath, goPluginOut),
	)
}

// modifierFromOptions returns a new Modifier for the given options.
func modifierFromOptions(options *Options, sweeper bufimagemodify.Sweeper) bufimagemodify.Modifier {
	if options == nil {
		return nil
	}
	var modifier bufimagemodify.Modifier
	if options.CcEnableArenas != nil {
		modifier = bufimagemodify.Merge(
			modifier,
			bufimagemodify.CcEnableArenas(sweeper, *options.CcEnableArenas),
		)
	}
	if options.JavaMultipleFiles != nil {
		modifier = bufimagemodify.Merge(
			modifier,
			bufimagemodify.JavaMultipleFiles(sweeper, *options.JavaMultipleFiles),
		)
	}
	if options.OptimizeFor != nil {
		modifier = bufimagemodify.Merge(
			modifier,
			bufimagemodify.OptimizeFor(sweeper, *options.OptimizeFor),
		)
	}
	// TODO: Add support for JavaStringCheckUTF8.
	return modifier
}

// resolveGoModulePath returns the Go module path specified in the
// user's go.mod file, if one exists. The relative path between
// the current working directory and the module root path is also
// returned, if a go.mod file exists.
func resolveGoModulePath() (string, string, error) {
	wd, err := osextended.Getwd()
	if err != nil {
		return "", "", err
	}
	goModuleRoot, err := findGoModuleFromRoot(wd)
	if err != nil {
		return "", "", err
	}
	relativePath, err := normalpath.Rel(goModuleRoot, wd)
	if err != nil {
		return "", "", err
	}
	goModuleFilePath := normalpath.Join(goModuleRoot, goModuleFile)
	goModuleFileData, err := os.ReadFile(goModuleFilePath)
	if err != nil {
		return "", "", err
	}
	modulePath := modfile.ModulePath(goModuleFileData)
	if modulePath == "" {
		return "", "", fmt.Errorf("%s does not define a module path", goModuleFilePath)
	}
	return modulePath, relativePath, nil
}

// findGoModuleRoot returns the relative path to the go.mod
// based on the current directory. This is referenced from
// the internal implementation found at:
// https://github.com/golang/go/blob/4520da486b6d236090b1d98ce4707c5bcd19cb70/src/cmd/go/internal/modload/init.go#L794
func findGoModuleFromRoot(root string) (string, error) {
	dir := normalpath.Normalize(root)
	for {
		if fileInfo, err := os.Stat(normalpath.Join(dir, goModuleFile)); err == nil && !fileInfo.IsDir() {
			return dir, nil
		}
		parent := normalpath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("failed to find %s within %s", goModuleFile, root)
}
