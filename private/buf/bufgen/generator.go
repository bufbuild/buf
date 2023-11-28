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
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	connect "connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
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

const defaultInput = "."

type generator struct {
	logger              *zap.Logger
	controller          bufctl.Controller
	storageosProvider   storageos.Provider
	pluginexecGenerator bufpluginexec.Generator
	clientConfig        *connectclient.Config
}

func newGenerator(
	logger *zap.Logger,
	controller bufctl.Controller,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor bufwasm.PluginExecutor,
	clientConfig *connectclient.Config,
) *generator {
	return &generator{
		logger:              logger,
		controller:          controller,
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
	config bufconfig.GenerateConfig,
	options ...GenerateOption,
) error {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	inputImages, err := getInputImages(
		ctx,
		g.logger,
		g.controller,
		generateOptions.input,
		config,
		generateOptions.moduleConfigPath,
		generateOptions.includePaths,
		generateOptions.excludePaths,
		generateOptions.includeTypes,
	)
	if err != nil {
		return err
	}
	for _, inputImage := range inputImages {
		if err := bufimagemodify.Modify(ctx, inputImage, config.GenerateManagedConfig()); err != nil {
			return err
		}
		if err := g.generateCode(
			ctx,
			container,
			inputImage,
			generateOptions.baseOutDirPath,
			config.GeneratePluginConfigs(),
			generateOptions.includeImports,
			generateOptions.includeWellKnownTypes,
		); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) generateCode(
	ctx context.Context,
	container app.EnvStdioContainer,
	inputImage bufimage.Image,
	baseOutDir string,
	pluginConfigs []bufconfig.GeneratePluginConfig,
	alwaysIncludeImports bool,
	alwaysIncludeWKT bool,
) error {
	responses, err := g.execPlugins(
		ctx,
		container,
		pluginConfigs,
		inputImage,
		alwaysIncludeImports,
		alwaysIncludeWKT,
		false, // wasm enabled is false
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
	for i, pluginConfig := range pluginConfigs {
		out := pluginConfig.Out()
		if baseOutDir != "" && baseOutDir != "." {
			out = filepath.Join(baseOutDir, out)
		}
		response := responses[i]
		if response == nil {
			return fmt.Errorf("failed to get plugin response for %s", pluginConfig.Name())
		}
		if err := responseWriter.AddResponse(
			ctx,
			response,
			out,
		); err != nil {
			return fmt.Errorf("plugin %s: %v", pluginConfig.Name(), err)
		}
	}
	if err := responseWriter.Close(); err != nil {
		return err
	}
	return nil
}

// TODO: this is a very temporary solution, although it would be nice if buffetch exposes function that parses ref from a map
func refStringForInputConfig(
	ctx context.Context,
	logger *zap.Logger,
	inputConfig bufconfig.GenerateInputConfig,
) string {
	var refString = inputConfig.Location()
	refOptionKeyToValue := map[string]string{}
	if inputConfig.Compression() != "" {
		refOptionKeyToValue["compression"] = inputConfig.Compression()
	}
	if inputConfig.StripComponents() != nil {
		refOptionKeyToValue["strip_components"] = strconv.FormatUint(uint64(*inputConfig.StripComponents()), 10)
	}
	if inputConfig.Subdir() != "" {
		refOptionKeyToValue["subdir"] = inputConfig.Subdir()
	}
	if inputConfig.Branch() != "" {
		refOptionKeyToValue["branch"] = inputConfig.Branch()
	}
	if inputConfig.Tag() != "" {
		refOptionKeyToValue["tag"] = inputConfig.Tag()
	}
	if inputConfig.Ref() != "" {
		refOptionKeyToValue["ref"] = inputConfig.Ref()
	}
	// TODO: != 0
	if inputConfig.Depth() != nil {
		refOptionKeyToValue["depth"] = strconv.FormatUint(uint64(*inputConfig.Depth()), 10)
	}
	if inputConfig.RecurseSubmodules() {
		refOptionKeyToValue["recurse_submodules"] = "true"
	}
	if inputConfig.IncludePackageFiles() {
		refOptionKeyToValue["include_package_files"] = "true"
	}
	if len(refOptionKeyToValue) == 0 {
		return refString
	}
	refString += "#"
	for key, value := range refOptionKeyToValue {
		refString += key + "=" + value
	}
	return refString
}

func getInputImages(
	ctx context.Context,
	logger *zap.Logger,
	controller bufctl.Controller,
	inputSpecified string,
	config bufconfig.GenerateConfig,
	moduleConfigOverride string,
	includePathsOverride []string,
	excludePathsOverride []string,
	includeTypesOverride []string,
) ([]bufimage.Image, error) {
	var inputImages []bufimage.Image
	// If input is specified on the command line, we use that. If input is not
	// specified on the command line, but the config has no inputs, use the default input.
	if inputSpecified != "" || len(config.GenerateInputConfigs()) == 0 {
		input := defaultInput
		if inputSpecified != "" {
			input = inputSpecified
		}
		var includeTypes []string
		if typesConfig := config.GenerateTypeConfig(); typesConfig != nil {
			includeTypes = typesConfig.IncludeTypes()
		}
		if len(includeTypesOverride) > 0 {
			includeTypes = includeTypesOverride
		}
		inputImage, err := controller.GetImage(
			ctx,
			input,
			bufctl.WithConfigOverride(moduleConfigOverride),
			bufctl.WithTargetPaths(includePathsOverride, excludePathsOverride),
			bufctl.WithImageTypes(includeTypes),
		)
		if err != nil {
			return nil, err
		}
		inputImages = []bufimage.Image{inputImage}
	} else {
		for _, inputConfig := range config.GenerateInputConfigs() {
			includePaths := inputConfig.IncludePaths()
			if len(includePathsOverride) > 0 {
				includePaths = includePathsOverride
			}
			excludePaths := inputConfig.ExcludePaths()
			if len(excludePathsOverride) > 0 {
				excludePaths = excludePathsOverride
			}
			// In V2 we do not need to look at inputConfig.GenerateTypeConfig().IncludeTypes()
			// because inputConfig.GenerateTypeConfig() is always nil.
			// TODO: document the above in godoc
			includeTypes := inputConfig.IncludeTypes()
			if len(includeTypesOverride) > 0 {
				includeTypes = includeTypesOverride
			}
			input := refStringForInputConfig(ctx, logger, inputConfig)
			inputImage, err := controller.GetImage(
				ctx,
				input,
				bufctl.WithConfigOverride(moduleConfigOverride),
				bufctl.WithTargetPaths(includePaths, excludePaths),
				bufctl.WithImageTypes(includeTypes),
			)
			if err != nil {
				return nil, err
			}
			inputImages = append(inputImages, inputImage)
		}
	}
	return inputImages, nil
}

func (g *generator) execPlugins(
	ctx context.Context,
	container app.EnvStdioContainer,
	pluginConfigs []bufconfig.GeneratePluginConfig,
	image bufimage.Image,
	alwaysIncludeImports bool,
	alwaysIncludeWellKnownTypes bool,
	wasmEnabled bool,
) ([]*pluginpb.CodeGeneratorResponse, error) {
	imageProvider := newImageProvider(image)
	// Collect all of the plugin jobs so that they can be executed in parallel.
	jobs := make([]func(context.Context) error, 0, len(pluginConfigs))
	responses := make([]*pluginpb.CodeGeneratorResponse, len(pluginConfigs))
	requiredFeatures := computeRequiredFeatures(image)
	remotePluginConfigTable := make(map[string][]*remotePluginExecArgs, len(pluginConfigs))
	for i, pluginConfig := range pluginConfigs {
		index := i
		currentPluginConfig := pluginConfig
		remote := currentPluginConfig.RemoteHost()
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
					// TODO: can the user override this to false on the command line? i.e. is `buf generate --include-imports=false` possible?
					alwaysIncludeImports || currentPluginConfig.IncludeImports(),
					alwaysIncludeWellKnownTypes || currentPluginConfig.IncludeWKT(),
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
		if len(indexedPluginConfigs) > 0 {
			jobs = append(jobs, func(ctx context.Context) error {
				results, err := g.execRemotePluginsV2(
					ctx,
					container,
					image,
					remote,
					indexedPluginConfigs,
					alwaysIncludeImports,
					alwaysIncludeWellKnownTypes,
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
	if err := validateResponses(responses, pluginConfigs); err != nil {
		return nil, err
	}
	checkRequiredFeatures(container, requiredFeatures, responses, pluginConfigs)
	return responses, nil
}

func (g *generator) execLocalPlugin(
	ctx context.Context,
	container app.EnvStdioContainer,
	imageProvider *imageProvider,
	pluginConfig bufconfig.GeneratePluginConfig,
	includeImports bool,
	includeWellKnownTypes bool,
	wasmEnabled bool,
) (*pluginpb.CodeGeneratorResponse, error) {
	pluginImages, err := imageProvider.GetImages(Strategy(pluginConfig.Strategy()))
	if err != nil {
		return nil, err
	}
	generateOptions := []bufpluginexec.GenerateOption{
		bufpluginexec.GenerateWithPluginPath(pluginConfig.Path()...),
		bufpluginexec.GenerateWithProtocPath(pluginConfig.ProtocPath()),
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
		pluginConfig.Name(),
		bufimage.ImagesToCodeGeneratorRequests(
			pluginImages,
			pluginConfig.Opt(),
			nil,
			includeImports,
			includeWellKnownTypes,
		),
		generateOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("plugin %s: %v", pluginConfig.Name(), err)
	}
	return response, nil
}

type remotePluginExecArgs struct {
	Index        int
	PluginConfig bufconfig.GeneratePluginConfig
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
	alwaysIncludeImports bool,
	alwaysIncludeWellKnownTypes bool,
) ([]*remotePluginExecutionResult, error) {
	requests := make([]*registryv1alpha1.PluginGenerationRequest, len(pluginConfigs))
	for i, pluginConfig := range pluginConfigs {
		request, err := getPluginGenerationRequest(
			pluginConfig.PluginConfig,
			alwaysIncludeImports || pluginConfig.PluginConfig.IncludeImports(),
			alwaysIncludeWellKnownTypes || pluginConfig.PluginConfig.IncludeWKT(),
		)
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
				IncludeImports:        alwaysIncludeImports,
				IncludeWellKnownTypes: alwaysIncludeWellKnownTypes,
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
	pluginConfig bufconfig.GeneratePluginConfig,
	includeImports bool,
	includeWKT bool,
) (*registryv1alpha1.PluginGenerationRequest, error) {
	var curatedPluginReference *registryv1alpha1.CuratedPluginReference
	if reference, err := bufpluginref.PluginReferenceForString(pluginConfig.Name(), pluginConfig.Revision()); err == nil {
		curatedPluginReference = bufplugin.PluginReferenceToProtoCuratedPluginReference(reference)
	} else {
		// Try parsing as a plugin identity (no version information)
		identity, err := bufpluginref.PluginIdentityForString(pluginConfig.Name())
		if err != nil {
			return nil, fmt.Errorf("invalid remote plugin %q", pluginConfig.Name())
		}
		curatedPluginReference = bufplugin.PluginIdentityToProtoCuratedPluginReference(identity)
	}
	var options []string
	if len(pluginConfig.Opt()) > 0 {
		// Only include parameters if they're not empty.
		options = []string{pluginConfig.Opt()}
	}
	return &registryv1alpha1.PluginGenerationRequest{
		PluginReference:       curatedPluginReference,
		Options:               options,
		IncludeImports:        &includeImports,
		IncludeWellKnownTypes: &includeWKT,
	}, nil
}

// validateResponses verifies that a response is set for each of the
// pluginConfigs, and that each generated file is generated by a single
// plugin.
func validateResponses(
	responses []*pluginpb.CodeGeneratorResponse,
	pluginConfigs []bufconfig.GeneratePluginConfig,
) error {
	if len(responses) != len(pluginConfigs) {
		return fmt.Errorf("unexpected number of responses: expected %d but got %d", len(pluginConfigs), len(responses))
	}
	pluginResponses := make([]*appproto.PluginResponse, 0, len(responses))
	for i, response := range responses {
		pluginConfig := pluginConfigs[i]
		if response == nil {
			return fmt.Errorf("failed to create a response for %q", pluginConfig.Name())
		}
		pluginResponses = append(
			pluginResponses,
			appproto.NewPluginResponse(
				response,
				pluginConfig.Name(),
				pluginConfig.Out(),
			),
		)
	}
	if err := appproto.ValidatePluginResponses(pluginResponses); err != nil {
		return err
	}
	return nil
}

type generateOptions struct {
	// plugin specific options:
	baseOutDirPath        string
	includeImports        bool
	includeWellKnownTypes bool
	wasmEnabled           bool
	// image/input specific options:
	input            string
	moduleConfigPath string
	// TODO: unify naming: includePaths / pathsIncluded / pathSpecified
	includePaths []string
	excludePaths []string
	includeTypes []string
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
