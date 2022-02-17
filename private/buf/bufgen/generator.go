// Copyright 2020-2022 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/app/appproto/appprotoexec"
	"github.com/bufbuild/buf/private/pkg/app/appproto/appprotoos"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/thread"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type generator struct {
	logger                *zap.Logger
	storageosProvider     storageos.Provider
	appprotoexecGenerator appprotoexec.Generator
	registryProvider      registryv1alpha1apiclient.Provider
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	registryProvider registryv1alpha1apiclient.Provider,
) *generator {
	return &generator{
		logger:                logger,
		storageosProvider:     storageosProvider,
		appprotoexecGenerator: appprotoexec.NewGenerator(logger, storageosProvider, runner),
		registryProvider:      registryProvider,
	}
}

// Generate executes all of the plugins specified by the given Config, and
// consolidates the results in the same order that the plugins are listed.
// Order is particularly important for insertion points, which are used to
// modify the generated output from other plugins executed earlier in the chain.
//
// Note that insertion points will only have access to files that are written
// in the same protoc invocation; plugins will not be able to insert code into
// other files that already exist on disk (just like protoc).
//
// All of the plugins, both local and remote, are called concurrently. Each
// plugin returns a single CodeGeneratorResponse, which are cached in-memory in
// the appprotoos.ResponseWriter. Once all of the CodeGeneratorResponses
// are written in-memory, we flush them to the OS filesystem by closing the
// appprotoos.ResponseWriter.
//
// This behavior is equivalent to protoc, which only writes out the content
// for each of the plugins if all of the plugins are successful.
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
		generateOptions.includeWellKnownTypes,
	)
}

func (g *generator) generate(
	ctx context.Context,
	container app.EnvStdioContainer,
	config *Config,
	image bufimage.Image,
	baseOutDirPath string,
	includeImports bool,
	includeWellKnownTypes bool,
) error {
	if err := modifyImage(ctx, g.logger, config, image); err != nil {
		return err
	}
	responses, err := g.execPlugins(
		ctx,
		container,
		g.appprotoexecGenerator,
		config,
		image,
		includeImports,
		includeWellKnownTypes,
	)
	if err != nil {
		return err
	}
	// Apply the CodeGeneratorResponses in the order they were specified.
	responseWriter := appprotoos.NewResponseWriter(
		g.logger,
		g.storageosProvider,
		appprotoos.ResponseWriterWithCreateOutDirIfNotExists(),
	)
	for i, pluginConfig := range config.PluginConfigs {
		out := pluginConfig.Out
		if baseOutDirPath != "" && baseOutDirPath != "." {
			out = filepath.Join(baseOutDirPath, out)
		}
		response := responses[i]
		if response == nil {
			return fmt.Errorf("failed to get plugin response for %s", pluginConfig.PluginName())
		}
		if err := responseWriter.AddResponse(
			ctx,
			response,
			out,
		); err != nil {
			return fmt.Errorf("plugin %s: %v", pluginConfig.PluginName(), err)
		}
	}
	if err := responseWriter.Close(); err != nil {
		return err
	}
	return nil
}

func (g *generator) execPlugins(
	ctx context.Context,
	container app.EnvStdioContainer,
	appprotoexecGenerator appprotoexec.Generator,
	config *Config,
	image bufimage.Image,
	includeImports bool,
	includeWellKnownTypes bool,
) ([]*pluginpb.CodeGeneratorResponse, error) {
	imageProvider := newImageProvider(image)
	// Collect all of the plugin jobs so that they can be executed in parallel.
	jobs := make([]func(context.Context) error, 0, len(config.PluginConfigs))
	responses := make([]*pluginpb.CodeGeneratorResponse, len(config.PluginConfigs))
	for i, pluginConfig := range config.PluginConfigs {
		index := i
		currentPluginConfig := pluginConfig
		if pluginConfig.Remote != "" {
			jobs = append(jobs, func(ctx context.Context) error {
				response, err := g.execRemotePlugin(
					ctx,
					container,
					image,
					currentPluginConfig,
					includeImports,
					includeWellKnownTypes,
				)
				if err != nil {
					return err
				}
				responses[index] = response
				return nil
			})
		}
		if pluginConfig.Name != "" {
			jobs = append(jobs, func(ctx context.Context) error {
				response, err := g.execLocalPlugin(
					ctx,
					container,
					g.appprotoexecGenerator,
					imageProvider,
					currentPluginConfig,
					includeImports,
					includeWellKnownTypes,
				)
				if err != nil {
					return err
				}
				responses[index] = response
				return nil
			})
		}
	}
	// We execute all of the jobs in parallel, but apply them in order so that any
	// insertion points are handled correctly.
	//
	// For example,
	//
	//  # buf.gen.yaml
	//  version: v1
	//  plugins:
	//    - remote: buf.build/org/plugins/insertion-point-receiver
	//      out: gen/proto
	//    - name: insertion-point-writer
	//      out: gen/proto
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := thread.Parallelize(
		ctx,
		jobs,
		thread.ParallelizeWithCancel(cancel),
	); err != nil {
		if errs := multierr.Errors(err); len(errs) > 0 {
			return nil, errs[0]
		}
		return nil, err
	}
	if err := validateResponses(responses, config.PluginConfigs); err != nil {
		return nil, err
	}
	return responses, nil
}

func (g *generator) execLocalPlugin(
	ctx context.Context,
	container app.EnvStdioContainer,
	appprotoexecGenerator appprotoexec.Generator,
	imageProvider *imageProvider,
	pluginConfig *PluginConfig,
	includeImports bool,
	includeWellKnownTypes bool,
) (*pluginpb.CodeGeneratorResponse, error) {
	pluginImages, err := imageProvider.GetImages(pluginConfig.Strategy)
	if err != nil {
		return nil, err
	}
	response, err := appprotoexecGenerator.Generate(
		ctx,
		container,
		pluginConfig.Name,
		bufimage.ImagesToCodeGeneratorRequests(
			pluginImages,
			pluginConfig.Opt,
			nil,
			includeImports,
			includeWellKnownTypes,
		),
		appprotoexec.GenerateWithPluginPath(pluginConfig.Path),
	)
	if err != nil {
		return nil, fmt.Errorf("plugin %s: %v", pluginConfig.PluginName(), err)
	}
	return response, nil
}

func (g *generator) execRemotePlugin(
	ctx context.Context,
	container app.EnvStdioContainer,
	image bufimage.Image,
	pluginConfig *PluginConfig,
	includeImports bool,
	includeWellKnownTypes bool,
) (*pluginpb.CodeGeneratorResponse, error) {
	remote, owner, name, version, err := bufplugin.ParsePluginVersionPath(pluginConfig.Remote)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin path: %w", err)
	}
	generateService, err := g.registryProvider.NewGenerateService(ctx, remote)
	if err != nil {
		return nil, fmt.Errorf("failed to create generate service for remote %q: %w", remote, err)
	}
	var parameters []string
	if len(pluginConfig.Opt) > 0 {
		// Only include parameters if they're not empty.
		parameters = []string{pluginConfig.Opt}
	}
	responses, _, err := generateService.GeneratePlugins(
		ctx,
		bufimage.ImageToProtoImage(image),
		[]*registryv1alpha1.PluginReference{
			{
				Owner:      owner,
				Name:       name,
				Version:    version,
				Parameters: parameters,
			},
		},
		includeImports,
		includeWellKnownTypes,
	)
	if err != nil {
		return nil, err
	}
	if len(responses) != 1 {
		return nil, fmt.Errorf("unexpected number of responses received, got %d, wanted %d", len(responses), 1)
	}
	pluginService, err := g.registryProvider.NewPluginService(ctx, remote)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin service for remote %q: %w", remote, err)
	}
	plugin, err := pluginService.GetPlugin(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	if plugin.Deprecated {
		warnMsg := fmt.Sprintf(`Plugin "%s/%s/%s" is deprecated`, remote, owner, name)
		if plugin.DeprecationMessage != "" {
			warnMsg = fmt.Sprintf("%s: %s", warnMsg, plugin.DeprecationMessage)
		}
		g.logger.Sugar().Warn(warnMsg)
	}
	return responses[0], nil
}

// modifyImage modifies the image according to the given configuration (i.e. managed mode).
func modifyImage(
	ctx context.Context,
	logger *zap.Logger,
	config *Config,
	image bufimage.Image,
) error {
	if config.ManagedConfig == nil {
		// If the config is nil, it implies that the
		// user has not enabled managed mode.
		return nil
	}
	sweeper := bufimagemodify.NewFileOptionSweeper()
	modifier, err := newModifier(logger, config.ManagedConfig, sweeper)
	if err != nil {
		return err
	}
	modifier = bufimagemodify.Merge(modifier, bufimagemodify.ModifierFunc(sweeper.Sweep))
	return modifier.Modify(ctx, image)
}

func newModifier(
	logger *zap.Logger,
	managedConfig *ManagedConfig,
	sweeper bufimagemodify.Sweeper,
) (bufimagemodify.Modifier, error) {
	modifier := bufimagemodify.NewMultiModifier(
		bufimagemodify.JavaOuterClassname(logger, sweeper, managedConfig.Override[bufimagemodify.JavaOuterClassNameID]),
		bufimagemodify.ObjcClassPrefix(logger, sweeper, managedConfig.Override[bufimagemodify.ObjcClassPrefixID]),
		bufimagemodify.CsharpNamespace(logger, sweeper, managedConfig.Override[bufimagemodify.CsharpNamespaceID]),
		bufimagemodify.PhpNamespace(logger, sweeper, managedConfig.Override[bufimagemodify.PhpNamespaceID]),
		bufimagemodify.PhpMetadataNamespace(logger, sweeper, managedConfig.Override[bufimagemodify.PhpMetadataNamespaceID]),
		bufimagemodify.RubyPackage(logger, sweeper, managedConfig.Override[bufimagemodify.RubyPackageID]),
	)
	javaPackagePrefix := &JavaPackagePrefixConfig{Default: bufimagemodify.DefaultJavaPackagePrefix}
	if managedConfig.JavaPackagePrefix != nil {
		javaPackagePrefix = managedConfig.JavaPackagePrefix
	}
	javaPackageModifier, err := bufimagemodify.JavaPackage(
		logger,
		sweeper,
		javaPackagePrefix.Default,
		javaPackagePrefix.Except,
		javaPackagePrefix.Override,
		managedConfig.Override[bufimagemodify.JavaPackageID],
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct java_package modifier: %w", err)
	}
	modifier = bufimagemodify.Merge(
		modifier,
		javaPackageModifier,
	)
	javaMultipleFilesValue := bufimagemodify.DefaultJavaMultipleFilesValue
	if managedConfig.JavaMultipleFiles != nil {
		javaMultipleFilesValue = *managedConfig.JavaMultipleFiles
	}
	javaMultipleFilesModifier, err := bufimagemodify.JavaMultipleFiles(
		logger,
		sweeper,
		javaMultipleFilesValue,
		managedConfig.Override[bufimagemodify.JavaMultipleFilesID],
	)
	if err != nil {
		return nil, err
	}
	modifier = bufimagemodify.Merge(modifier, javaMultipleFilesModifier)
	if managedConfig.CcEnableArenas != nil {
		ccEnableArenasModifier, err := bufimagemodify.CcEnableArenas(
			logger,
			sweeper,
			*managedConfig.CcEnableArenas,
			managedConfig.Override[bufimagemodify.CcEnableArenasID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(modifier, ccEnableArenasModifier)
	}
	if managedConfig.JavaStringCheckUtf8 != nil {
		javaStringCheckUtf8, err := bufimagemodify.JavaStringCheckUtf8(
			logger,
			sweeper,
			*managedConfig.JavaStringCheckUtf8,
			managedConfig.Override[bufimagemodify.JavaStringCheckUtf8ID],
		)
		if err != nil {
			return nil, err
		}
		modifier = bufimagemodify.Merge(modifier, javaStringCheckUtf8)
	}
	if managedConfig.OptimizeFor != nil {
		optimizeFor, err := bufimagemodify.OptimizeFor(
			logger,
			sweeper,
			*managedConfig.OptimizeFor,
			managedConfig.Override[bufimagemodify.OptimizeForID],
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
			logger,
			sweeper,
			managedConfig.GoPackagePrefixConfig.Default,
			managedConfig.GoPackagePrefixConfig.Except,
			managedConfig.GoPackagePrefixConfig.Override,
			managedConfig.Override[bufimagemodify.GoPackageID],
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

// validateResponses verifies that a response is set for each of the
// pluginConfigs, and that each generated file is generated by a single
// plugin.
func validateResponses(
	responses []*pluginpb.CodeGeneratorResponse,
	pluginConfigs []*PluginConfig,
) error {
	if len(responses) != len(pluginConfigs) {
		return fmt.Errorf("unexpected number of responses: expected %d but got %d", len(pluginConfigs), len(responses))
	}
	pluginResponses := make([]*appproto.PluginResponse, 0, len(responses))
	for i, response := range responses {
		pluginConfig := pluginConfigs[i]
		if response == nil {
			return fmt.Errorf("failed to create a response for %q", pluginConfig.PluginName())
		}
		pluginResponses = append(
			pluginResponses,
			appproto.NewPluginResponse(
				response,
				pluginConfig.PluginName(),
				pluginConfig.Out,
			),
		)
	}
	if err := appproto.ValidatePluginResponses(pluginResponses); err != nil {
		return err
	}
	return nil
}

type generateOptions struct {
	baseOutDirPath        string
	includeImports        bool
	includeWellKnownTypes bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
