// Copyright 2020-2025 Buf Technologies, Inc.
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
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	connect "connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufprotopluginexec"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimageutil"
	"github.com/bufbuild/buf/private/bufpkg/bufprotoplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufprotoplugin/bufprotopluginos"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/thread"
	"google.golang.org/protobuf/types/pluginpb"
)

type generator struct {
	logger              *slog.Logger
	storageosProvider   storageos.Provider
	pluginexecGenerator bufprotopluginexec.Generator
	clientConfig        *connectclient.Config
}

func newGenerator(
	logger *slog.Logger,
	storageosProvider storageos.Provider,
	clientConfig *connectclient.Config,
) *generator {
	return &generator{
		logger:              logger,
		storageosProvider:   storageosProvider,
		pluginexecGenerator: bufprotopluginexec.NewGenerator(logger, storageosProvider),
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
// the bufprotopluginos.ResponseWriter. Once all of the CodeGeneratorResponses
// are written in-memory, we flush them to the OS filesystem by closing the
// bufprotopluginos.ResponseWriter.
//
// This behavior is equivalent to protoc, which only writes out the content
// for each of the plugins if all of the plugins are successful.
func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStdioContainer,
	config bufconfig.GenerateConfig,
	images []bufimage.Image,
	options ...GenerateOption,
) error {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	if !config.GenerateManagedConfig().Enabled() {
		if len(config.GenerateManagedConfig().Overrides()) != 0 || len(config.GenerateManagedConfig().Disables()) != 0 {
			g.logger.Warn("managed mode configs are set but are not enabled")
		}
	}
	for _, image := range images {
		if err := bufimagemodify.Modify(image, config.GenerateManagedConfig()); err != nil {
			return err
		}
	}
	shouldDeleteOuts := config.CleanPluginOuts()
	if generateOptions.deleteOuts != nil {
		shouldDeleteOuts = *generateOptions.deleteOuts
	}
	if shouldDeleteOuts {
		if err := g.deleteOuts(
			ctx,
			generateOptions.baseOutDirPath,
			config.GeneratePluginConfigs(),
		); err != nil {
			return err
		}
	}
	for _, image := range images {
		if err := g.generateCode(
			ctx,
			container,
			image,
			generateOptions.baseOutDirPath,
			config.GeneratePluginConfigs(),
			generateOptions.includeImportsOverride,
			generateOptions.includeWellKnownTypesOverride,
		); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) deleteOuts(
	ctx context.Context,
	baseOutDir string,
	pluginConfigs []bufconfig.GeneratePluginConfig,
) error {
	return bufprotopluginos.NewCleaner(g.storageosProvider).DeleteOuts(
		ctx,
		slicesext.Map(
			pluginConfigs,
			func(pluginConfig bufconfig.GeneratePluginConfig) string {
				out := pluginConfig.Out()
				if baseOutDir != "" && baseOutDir != "." {
					return filepath.Join(baseOutDir, out)
				}
				return out
			},
		),
	)
}

func (g *generator) generateCode(
	ctx context.Context,
	container app.EnvStdioContainer,
	inputImage bufimage.Image,
	baseOutDir string,
	pluginConfigs []bufconfig.GeneratePluginConfig,
	includeImportsOverride *bool,
	includeWellKnownTypesOverride *bool,
) error {
	responses, err := g.execPlugins(
		ctx,
		container,
		pluginConfigs,
		inputImage,
		includeImportsOverride,
		includeWellKnownTypesOverride,
	)
	if err != nil {
		return err
	}
	// Apply the CodeGeneratorResponses in the order they were specified.
	responseWriter := bufprotopluginos.NewResponseWriter(
		g.logger,
		g.storageosProvider,
		bufprotopluginos.ResponseWriterWithCreateOutDirIfNotExists(),
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

// hashPluginConfigForImage returns a hash of the plugin config for the image.
// This is used to batch plugins that have the same configuration.
// The hash is based on the following properties:
//   - Strategy
//   - ExcludeOptions
//   - RemoteHost
func hashPluginConfigForImage(pluginConfig bufconfig.GeneratePluginConfig) (string, error) {
	type imagePluginConfigKey struct {
		Strategy       Strategy
		ExcludeOptions []string
		RemoteHost     string
	}
	key := &imagePluginConfigKey{
		Strategy:       Strategy(pluginConfig.Strategy()),
		ExcludeOptions: pluginConfig.ExcludeOptions(),
		RemoteHost:     pluginConfig.RemoteHost(),
	}
	var str strings.Builder
	if err := gob.NewEncoder(&str).Encode(key); err != nil {
		return "", err
	}
	return str.String(), nil
}

func (g *generator) execPlugins(
	ctx context.Context,
	container app.EnvStdioContainer,
	pluginConfigs []bufconfig.GeneratePluginConfig,
	image bufimage.Image,
	includeImportsOverride *bool,
	includeWellKnownTypesOverride *bool,
) ([]*pluginpb.CodeGeneratorResponse, error) {
	// Collect all of the plugin jobs so that they can be executed in parallel.
	jobs := make([]func(context.Context) error, 0, len(pluginConfigs))
	responses := make([]*pluginpb.CodeGeneratorResponse, len(pluginConfigs))
	requiredFeatures := computeRequiredFeatures(image)

	pluginConfigsForImage, err := slicesext.ToIndexedValuesMapError(pluginConfigs, hashPluginConfigForImage)
	if err != nil {
		return nil, err
	}

	for _, indexedPluginConfigs := range pluginConfigsForImage {
		image := image
		currentPluginConfig := indexedPluginConfigs[0].Value

		// Apply per-config filters.
		if excludeOptions := currentPluginConfig.ExcludeOptions(); len(excludeOptions) > 0 {
			var err error
			image, err = bufimageutil.FilterImage(
				image,
				bufimageutil.WithExcludeOptions(excludeOptions...),
			)
			if err != nil {
				return nil, err
			}
		}

		// Batch for each remote.
		if remote := currentPluginConfig.RemoteHost(); remote != "" {
			jobs = append(jobs, func(ctx context.Context) error {
				results, err := g.execRemotePluginsV2(
					ctx,
					container,
					image,
					remote,
					indexedPluginConfigs,
					includeImportsOverride,
					includeWellKnownTypesOverride,
				)
				if err != nil {
					return err
				}
				for _, result := range results {
					responses[result.Index] = result.Value
				}
				return nil

			})
			continue
		}

		// Local plugins.
		var images []bufimage.Image
		switch Strategy(currentPluginConfig.Strategy()) {
		case StrategyAll:
			images = []bufimage.Image{image}
		case StrategyDirectory:
			var err error
			images, err = bufimage.ImageByDir(image)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown strategy: %v", currentPluginConfig.Strategy())
		}
		for _, indexedPluginConfig := range indexedPluginConfigs {
			jobs = append(jobs, func(ctx context.Context) error {
				includeImports := currentPluginConfig.IncludeImports()
				if includeImportsOverride != nil {
					includeImports = *includeImportsOverride
				}
				includeWellKnownTypes := currentPluginConfig.IncludeWKT()
				if includeWellKnownTypesOverride != nil {
					includeWellKnownTypes = *includeWellKnownTypesOverride
				}
				response, err := g.execLocalPlugin(
					ctx,
					container,
					images,
					indexedPluginConfig.Value,
					includeImports,
					includeWellKnownTypes,
				)
				if err != nil {
					return err
				}
				responses[indexedPluginConfig.Index] = response
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
	if err := thread.Parallelize(
		ctx,
		jobs,
		thread.ParallelizeWithCancelOnFailure(),
	); err != nil {
		return nil, err
	}
	if err := validateResponses(responses, pluginConfigs); err != nil {
		return nil, err
	}
	if err := checkRequiredFeatures(g.logger, requiredFeatures, responses, pluginConfigs); err != nil {
		return nil, err
	}
	return responses, nil
}

func (g *generator) execLocalPlugin(
	ctx context.Context,
	container app.EnvStdioContainer,
	pluginImages []bufimage.Image,
	pluginConfig bufconfig.GeneratePluginConfig,
	includeImports bool,
	includeWellKnownTypes bool,
) (*pluginpb.CodeGeneratorResponse, error) {
	requests, err := bufimage.ImagesToCodeGeneratorRequests(
		pluginImages,
		pluginConfig.Opt(),
		nil,
		includeImports,
		includeWellKnownTypes,
	)
	if err != nil {
		return nil, err
	}
	response, err := g.pluginexecGenerator.Generate(
		ctx,
		container,
		pluginConfig.Name(),
		requests,
		bufprotopluginexec.GenerateWithPluginPath(pluginConfig.Path()...),
		bufprotopluginexec.GenerateWithProtocPath(pluginConfig.ProtocPath()...),
	)
	if err != nil {
		return nil, fmt.Errorf("plugin %s: %v", pluginConfig.Name(), err)
	}
	return response, nil
}

func (g *generator) execRemotePluginsV2(
	ctx context.Context,
	container app.EnvStdioContainer,
	image bufimage.Image,
	remote string,
	indexedPluginConfigs []slicesext.Indexed[bufconfig.GeneratePluginConfig],
	includeImportsOverride *bool,
	includeWellKnownTypesOverride *bool,
) ([]slicesext.Indexed[*pluginpb.CodeGeneratorResponse], error) {
	requests := make([]*registryv1alpha1.PluginGenerationRequest, len(indexedPluginConfigs))
	for i, pluginConfig := range indexedPluginConfigs {
		includeImports := pluginConfig.Value.IncludeImports()
		if includeImportsOverride != nil {
			includeImports = *includeImportsOverride
		}
		includeWellKnownTypes := pluginConfig.Value.IncludeWKT()
		if includeWellKnownTypesOverride != nil {
			includeWellKnownTypes = *includeWellKnownTypesOverride
		}
		request, err := getPluginGenerationRequest(
			pluginConfig.Value,
			includeImports,
			includeWellKnownTypes,
		)
		if err != nil {
			return nil, err
		}
		requests[i] = request
	}
	codeGenerationService := connectclient.Make(g.clientConfig, remote, registryv1alpha1connect.NewCodeGenerationServiceClient)
	protoImage, err := bufimage.ImageToProtoImage(image)
	if err != nil {
		return nil, err
	}
	response, err := codeGenerationService.GenerateCode(
		ctx,
		connect.NewRequest(
			registryv1alpha1.GenerateCodeRequest_builder{
				Image:    protoImage,
				Requests: requests,
			}.Build(),
		),
	)
	if err != nil {
		return nil, err
	}
	responses := response.Msg.GetResponses()
	if len(responses) != len(requests) {
		return nil, fmt.Errorf("unexpected number of responses received, got %d, wanted %d", len(responses), len(requests))
	}
	result := make([]slicesext.Indexed[*pluginpb.CodeGeneratorResponse], 0, len(responses))
	for i := range requests {
		codeGeneratorResponse := responses[i].GetResponse()
		if codeGeneratorResponse == nil {
			return nil, errors.New("expected code generator response")
		}
		result = append(result, slicesext.Indexed[*pluginpb.CodeGeneratorResponse]{
			Value: codeGeneratorResponse,
			Index: indexedPluginConfigs[i].Index,
		})
	}
	return result, nil
}

func getPluginGenerationRequest(
	pluginConfig bufconfig.GeneratePluginConfig,
	includeImports bool,
	includeWellKnownTypes bool,
) (*registryv1alpha1.PluginGenerationRequest, error) {
	var curatedPluginReference *registryv1alpha1.CuratedPluginReference
	if reference, err := bufremotepluginref.PluginReferenceForString(pluginConfig.Name(), pluginConfig.Revision()); err == nil {
		curatedPluginReference = bufremoteplugin.PluginReferenceToProtoCuratedPluginReference(reference)
	} else {
		// Try parsing as a plugin identity (no version information)
		identity, err := bufremotepluginref.PluginIdentityForString(pluginConfig.Name())
		if err != nil {
			return nil, fmt.Errorf("invalid remote plugin %q", pluginConfig.Name())
		}
		curatedPluginReference = bufremoteplugin.PluginIdentityToProtoCuratedPluginReference(identity)
	}
	var options []string
	if len(pluginConfig.Opt()) > 0 {
		// Only include parameters if they're not empty.
		options = []string{pluginConfig.Opt()}
	}
	return registryv1alpha1.PluginGenerationRequest_builder{
		PluginReference:       curatedPluginReference,
		Options:               options,
		IncludeImports:        &includeImports,
		IncludeWellKnownTypes: &includeWellKnownTypes,
	}.Build(), nil
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
	pluginResponses := make([]*bufprotoplugin.PluginResponse, 0, len(responses))
	for i, response := range responses {
		pluginConfig := pluginConfigs[i]
		if response == nil {
			return fmt.Errorf("failed to create a response for %q", pluginConfig.Name())
		}
		pluginResponses = append(
			pluginResponses,
			bufprotoplugin.NewPluginResponse(
				response,
				pluginConfig.Name(),
				pluginConfig.Out(),
			),
		)
	}
	if err := bufprotoplugin.ValidatePluginResponses(pluginResponses); err != nil {
		return err
	}
	return nil
}

type generateOptions struct {
	baseOutDirPath                string
	deleteOuts                    *bool
	includeImportsOverride        *bool
	includeWellKnownTypesOverride *bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
