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

package bufgen

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	connect "connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/bufpkg/bufpluginexec"
	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/app/appproto/appprotoos"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/thread"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type generator struct {
	logger              *zap.Logger
	storageosProvider   storageos.Provider
	pluginexecGenerator bufpluginexec.Generator
	clientConfig        *connectclient.Config
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor bufwasm.PluginExecutor,
	clientConfig *connectclient.Config,
) *generator {
	return &generator{
		logger:              logger,
		storageosProvider:   storageosProvider,
		pluginexecGenerator: bufpluginexec.NewGenerator(logger, storageosProvider, runner, wasmPluginExecutor),
		clientConfig:        clientConfig,
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
		generateOptions.wasmEnabled,
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
	wasmEnabled bool,
) error {
	if err := modifyImage(ctx, g.logger, config, image); err != nil {
		return err
	}
	responses, err := g.execPlugins(
		ctx,
		container,
		config,
		image,
		includeImports,
		includeWellKnownTypes,
		wasmEnabled,
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
	config *Config,
	image bufimage.Image,
	includeImports bool,
	includeWellKnownTypes bool,
	wasmEnabled bool,
) ([]*pluginpb.CodeGeneratorResponse, error) {
	imageProvider := newImageProvider(image)
	// Collect all of the plugin jobs so that they can be executed in parallel.
	jobs := make([]func(context.Context) error, 0, len(config.PluginConfigs))
	responses := make([]*pluginpb.CodeGeneratorResponse, len(config.PluginConfigs))
	requiredFeatures := computeRequiredFeatures(image)
	remotePluginConfigTable := make(map[string][]*remotePluginExecArgs, len(config.PluginConfigs))
	for i, pluginConfig := range config.PluginConfigs {
		index := i
		currentPluginConfig := pluginConfig
		remote := currentPluginConfig.GetRemoteHostname()
		if remote != "" {
			remotePluginConfigTable[remote] = append(
				remotePluginConfigTable[remote],
				&remotePluginExecArgs{
					Index:        index,
					PluginConfig: currentPluginConfig,
				},
			)
		} else {
			jobs = append(jobs, func(ctx context.Context) error {
				response, err := g.execLocalPlugin(
					ctx,
					container,
					imageProvider,
					currentPluginConfig,
					includeImports,
					includeWellKnownTypes,
					wasmEnabled,
				)
				if err != nil {
					return err
				}
				responses[index] = response
				return nil
			})
		}
	}
	// Batch for each remote.
	for remote, indexedPluginConfigs := range remotePluginConfigTable {
		remote := remote
		indexedPluginConfigs := indexedPluginConfigs
		v2Args := make([]*remotePluginExecArgs, 0, len(indexedPluginConfigs))
		for _, param := range indexedPluginConfigs {
			if param.PluginConfig.Remote != "" {
				return nil, fmt.Errorf("invalid plugin reference: %s", param.PluginConfig.Remote)
			}
			v2Args = append(v2Args, param)
		}
		if len(v2Args) > 0 {
			jobs = append(jobs, func(ctx context.Context) error {
				results, err := g.execRemotePluginsV2(
					ctx,
					container,
					image,
					remote,
					v2Args,
					includeImports,
					includeWellKnownTypes,
				)
				if err != nil {
					return err
				}
				for _, result := range results {
					responses[result.Index] = result.CodeGeneratorResponse
				}
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
	//    - plugin: buf.build/org/insertion-point-receiver
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
	checkRequiredFeatures(container, requiredFeatures, responses, config.PluginConfigs)
	return responses, nil
}

func (g *generator) execLocalPlugin(
	ctx context.Context,
	container app.EnvStdioContainer,
	imageProvider *imageProvider,
	pluginConfig *PluginConfig,
	includeImports bool,
	includeWellKnownTypes bool,
	wasmEnabled bool,
) (*pluginpb.CodeGeneratorResponse, error) {
	pluginImages, err := imageProvider.GetImages(pluginConfig.Strategy)
	if err != nil {
		return nil, err
	}
	generateOptions := []bufpluginexec.GenerateOption{
		bufpluginexec.GenerateWithPluginPath(pluginConfig.Path...),
		bufpluginexec.GenerateWithProtocPath(pluginConfig.ProtocPath),
	}
	if wasmEnabled {
		generateOptions = append(
			generateOptions,
			bufpluginexec.GenerateWithWASMEnabled(),
		)
	}
	response, err := g.pluginexecGenerator.Generate(
		ctx,
		container,
		pluginConfig.PluginName(),
		bufimage.ImagesToCodeGeneratorRequests(
			pluginImages,
			pluginConfig.Opt,
			nil,
			includeImports,
			includeWellKnownTypes,
		),
		generateOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("plugin %s: %v", pluginConfig.PluginName(), err)
	}
	return response, nil
}

type remotePluginExecArgs struct {
	Index        int
	PluginConfig *PluginConfig
}

type remotePluginExecutionResult struct {
	CodeGeneratorResponse *pluginpb.CodeGeneratorResponse
	Index                 int
}

func (g *generator) execRemotePluginsV2(
	ctx context.Context,
	container app.EnvStdioContainer,
	image bufimage.Image,
	remote string,
	pluginConfigs []*remotePluginExecArgs,
	includeImports bool,
	includeWellKnownTypes bool,
) ([]*remotePluginExecutionResult, error) {
	requests := make([]*registryv1alpha1.PluginGenerationRequest, len(pluginConfigs))
	for i, pluginConfig := range pluginConfigs {
		request, err := getPluginGenerationRequest(pluginConfig.PluginConfig)
		if err != nil {
			return nil, err
		}
		requests[i] = request
	}
	codeGenerationService := connectclient.Make(g.clientConfig, remote, registryv1alpha1connect.NewCodeGenerationServiceClient)
	response, err := codeGenerationService.GenerateCode(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.GenerateCodeRequest{
				Image:                 bufimage.ImageToProtoImage(image),
				Requests:              requests,
				IncludeImports:        includeImports,
				IncludeWellKnownTypes: includeWellKnownTypes,
			},
		),
	)
	if err != nil {
		return nil, err
	}
	responses := response.Msg.Responses
	if len(responses) != len(requests) {
		return nil, fmt.Errorf("unexpected number of responses received, got %d, wanted %d", len(responses), len(requests))
	}
	result := make([]*remotePluginExecutionResult, 0, len(responses))
	for i := range requests {
		codeGeneratorResponse := responses[i].GetResponse()
		if codeGeneratorResponse == nil {
			return nil, errors.New("expected code generator response")
		}
		result = append(result, &remotePluginExecutionResult{
			CodeGeneratorResponse: codeGeneratorResponse,
			Index:                 pluginConfigs[i].Index,
		})
	}
	return result, nil
}

func getPluginGenerationRequest(
	pluginConfig *PluginConfig,
) (*registryv1alpha1.PluginGenerationRequest, error) {
	var curatedPluginReference *registryv1alpha1.CuratedPluginReference
	if reference, err := bufpluginref.PluginReferenceForString(pluginConfig.Plugin, pluginConfig.Revision); err == nil {
		curatedPluginReference = bufplugin.PluginReferenceToProtoCuratedPluginReference(reference)
	} else {
		// Try parsing as a plugin identity (no version information)
		identity, err := bufpluginref.PluginIdentityForString(pluginConfig.Plugin)
		if err != nil {
			return nil, fmt.Errorf("invalid remote plugin %q", pluginConfig.Plugin)
		}
		curatedPluginReference = bufplugin.PluginIdentityToProtoCuratedPluginReference(identity)
	}
	var options []string
	if len(pluginConfig.Opt) > 0 {
		// Only include parameters if they're not empty.
		options = []string{pluginConfig.Opt}
	}
	return &registryv1alpha1.PluginGenerationRequest{
		PluginReference: curatedPluginReference,
		Options:         options,
	}, nil
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
		bufimagemodify.JavaOuterClassname(
			logger,
			sweeper,
			managedConfig.Override[bufimagemodify.JavaOuterClassNameID],
			false, // preserveExistingValue
		),
		bufimagemodify.PhpNamespace(logger, sweeper, managedConfig.Override[bufimagemodify.PhpNamespaceID]),
		bufimagemodify.PhpMetadataNamespace(logger, sweeper, managedConfig.Override[bufimagemodify.PhpMetadataNamespaceID]),
	)
	javaPackagePrefix := &JavaPackagePrefixConfig{Default: bufimagemodify.DefaultJavaPackagePrefix}
	if managedConfig.JavaPackagePrefixConfig != nil {
		javaPackagePrefix = managedConfig.JavaPackagePrefixConfig
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
		false, // preserveExistingValue
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
	var (
		csharpNamespaceExcept   []bufmoduleref.ModuleIdentity
		csharpNamespaceOverride map[bufmoduleref.ModuleIdentity]string
	)
	if csharpNameSpaceConfig := managedConfig.CsharpNameSpaceConfig; csharpNameSpaceConfig != nil {
		csharpNamespaceExcept = csharpNameSpaceConfig.Except
		csharpNamespaceOverride = csharpNameSpaceConfig.Override
	}
	csharpNamespaceModifier := bufimagemodify.CsharpNamespace(
		logger,
		sweeper,
		csharpNamespaceExcept,
		csharpNamespaceOverride,
		managedConfig.Override[bufimagemodify.CsharpNamespaceID],
	)
	modifier = bufimagemodify.Merge(modifier, csharpNamespaceModifier)
	if managedConfig.OptimizeForConfig != nil {
		optimizeFor, err := bufimagemodify.OptimizeFor(
			logger,
			sweeper,
			managedConfig.OptimizeForConfig.Default,
			managedConfig.OptimizeForConfig.Except,
			managedConfig.OptimizeForConfig.Override,
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
	var (
		objcClassPrefixDefault  string
		objcClassPrefixExcept   []bufmoduleref.ModuleIdentity
		objcClassPrefixOverride map[bufmoduleref.ModuleIdentity]string
	)
	if objcClassPrefixConfig := managedConfig.ObjcClassPrefixConfig; objcClassPrefixConfig != nil {
		objcClassPrefixDefault = objcClassPrefixConfig.Default
		objcClassPrefixExcept = objcClassPrefixConfig.Except
		objcClassPrefixOverride = objcClassPrefixConfig.Override
	}
	objcClassPrefixModifier := bufimagemodify.ObjcClassPrefix(
		logger,
		sweeper,
		objcClassPrefixDefault,
		objcClassPrefixExcept,
		objcClassPrefixOverride,
		managedConfig.Override[bufimagemodify.ObjcClassPrefixID],
	)
	modifier = bufimagemodify.Merge(
		modifier,
		objcClassPrefixModifier,
	)
	var (
		rubyPackageExcept    []bufmoduleref.ModuleIdentity
		rubyPackageOverrides map[bufmoduleref.ModuleIdentity]string
	)
	if rubyPackageConfig := managedConfig.RubyPackageConfig; rubyPackageConfig != nil {
		rubyPackageExcept = rubyPackageConfig.Except
		rubyPackageOverrides = rubyPackageConfig.Override
	}
	rubyPackageModifier := bufimagemodify.RubyPackage(
		logger,
		sweeper,
		rubyPackageExcept,
		rubyPackageOverrides,
		managedConfig.Override[bufimagemodify.RubyPackageID],
	)
	modifier = bufimagemodify.Merge(
		modifier,
		rubyPackageModifier,
	)
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
	wasmEnabled           bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
