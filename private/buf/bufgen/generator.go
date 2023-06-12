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

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin/bufpluginref"
	"github.com/bufbuild/buf/private/bufpkg/bufpluginexec"
	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin"
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
	connect "github.com/bufbuild/connect-go"
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
	pluginConfigs []PluginConfig,
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
		pluginConfigs,
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
	pluginConfigs []PluginConfig,
	image bufimage.Image,
	baseOutDirPath string,
	includeImports bool,
	includeWellKnownTypes bool,
	wasmEnabled bool,
) error {
	// TODO: move this somewhere
	// if err := modifyImage(ctx, g.logger, config, image); err != nil {
	// 	return err
	// }
	responses, err := g.execPlugins(
		ctx,
		container,
		pluginConfigs,
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
	for i, pluginConfig := range pluginConfigs {
		out := pluginConfig.Out()
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
	pluginConfigs []PluginConfig,
	image bufimage.Image,
	includeImports bool,
	includeWellKnownTypes bool,
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
		switch t := currentPluginConfig.(type) {
		case RemotePluginConfig:
			remotePluginConfigTable[t.Remote()] = append(
				remotePluginConfigTable[t.Remote()],
				&remotePluginExecArgs{
					Index:        index,
					PluginConfig: t,
				},
			)
		case LocalPluginConfig:
			jobs = append(jobs, func(ctx context.Context) error {
				response, err := g.execLocalPlugin(
					ctx,
					container,
					imageProvider,
					t,
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
		v1Args := make([]*v1RemotePluginExecArgs, 0, len(indexedPluginConfigs))
		v2Args := make([]*curatedPluginExecArgs, 0, len(indexedPluginConfigs))
		for _, param := range indexedPluginConfigs {
			switch t := param.PluginConfig.(type) {
			case LegacyRemotePluginConfig:
				v1Args = append(
					v1Args,
					&v1RemotePluginExecArgs{
						Index:        param.Index,
						PluginConfig: t,
					},
				)
			case CuratedPluginConfig:
				v2Args = append(
					v2Args,
					&curatedPluginExecArgs{
						Index:        param.Index,
						PluginConfig: t,
					},
				)
			}
		}
		if len(v1Args) > 0 {
			jobs = append(jobs, func(ctx context.Context) error {
				results, err := g.executeRemotePlugins(
					ctx,
					container,
					image,
					remote,
					v1Args,
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
	pluginConfig LocalPluginConfig,
	includeImports bool,
	includeWellKnownTypes bool,
	wasmEnabled bool,
) (*pluginpb.CodeGeneratorResponse, error) {
	pluginImages, err := imageProvider.GetImages(pluginConfig.Strategy())
	if err != nil {
		return nil, err
	}
	requests := bufimage.ImagesToCodeGeneratorRequests(
		pluginImages,
		pluginConfig.Opt(),
		nil,
		includeImports,
		includeWellKnownTypes,
	)
	var response *pluginpb.CodeGeneratorResponse
	switch t := pluginConfig.(type) {
	case BinaryPluginConfig:
		response, err = g.pluginexecGenerator.GenerateWithBinary(
			ctx,
			container,
			t.Path()[0],
			t.Path()[1:],
			requests,
		)
	case ProtocBuiltinPluginConfig:
		var options []bufpluginexec.GenerateWithProtocBuiltinOption
		if t.ProtocPath() != "" {
			options = append(
				options,
				bufpluginexec.GenerateWithProtocBuiltinWithPath(t.ProtocPath()),
			)
		}
		response, err = g.pluginexecGenerator.GenerateWithProtocBuiltin(
			ctx,
			container,
			pluginConfig.PluginName(),
			requests,
			options...,
		)
	default:
		response, err = g.pluginexecGenerator.Generate(
			ctx,
			container,
			t.PluginName(),
			requests,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("plugin %s: %v", pluginConfig.PluginName(), err)
	}
	return response, nil
}

type remotePluginExecArgs struct {
	Index        int
	PluginConfig RemotePluginConfig
}

type v1RemotePluginExecArgs struct {
	Index        int
	PluginConfig LegacyRemotePluginConfig
}

type curatedPluginExecArgs struct {
	Index        int
	PluginConfig CuratedPluginConfig
}

type remotePluginExecutionResult struct {
	CodeGeneratorResponse *pluginpb.CodeGeneratorResponse
	Index                 int
}

func (g *generator) executeRemotePlugins(
	ctx context.Context,
	container app.EnvStdioContainer,
	image bufimage.Image,
	remote string,
	pluginConfigs []*v1RemotePluginExecArgs,
	includeImports bool,
	includeWellKnownTypes bool,
) ([]*remotePluginExecutionResult, error) {
	pluginReferences := make([]*registryv1alpha1.PluginReference, len(pluginConfigs))
	for i, pluginConfig := range pluginConfigs {
		pluginReference, err := getPluginReference(pluginConfig.PluginConfig)
		if err != nil {
			return nil, err
		}
		pluginReferences[i] = pluginReference
	}
	generateService := connectclient.Make(g.clientConfig, remote, registryv1alpha1connect.NewGenerateServiceClient)
	response, err := generateService.GeneratePlugins(
		ctx,
		connect.NewRequest(
			&registryv1alpha1.GeneratePluginsRequest{
				Image:                 bufimage.ImageToProtoImage(image),
				Plugins:               pluginReferences,
				IncludeImports:        includeImports,
				IncludeWellKnownTypes: includeWellKnownTypes,
			},
		),
	)
	if err != nil {
		return nil, err
	}
	responses := response.Msg.Responses
	if len(responses) != len(pluginReferences) {
		return nil, fmt.Errorf("unexpected number of responses, got %d, wanted: %d", len(responses), len(pluginReferences))
	}
	pluginService := connectclient.Make(g.clientConfig, remote, registryv1alpha1connect.NewPluginServiceClient)
	result := make([]*remotePluginExecutionResult, 0, len(pluginReferences))
	for i, pluginReference := range pluginReferences {
		result = append(result, &remotePluginExecutionResult{
			Index:                 pluginConfigs[i].Index,
			CodeGeneratorResponse: responses[i],
		})
		resp, err := pluginService.GetPlugin(
			ctx,
			connect.NewRequest(
				&registryv1alpha1.GetPluginRequest{
					Owner: pluginReference.Owner,
					Name:  pluginReference.Name,
				},
			),
		)
		if err != nil {
			return nil, err
		}
		plugin := resp.Msg.Plugin
		if plugin.Deprecated {
			warnMsg := fmt.Sprintf(`Plugin "%s/%s/%s" is deprecated`, remote, pluginReference.Owner, pluginReference.Name)
			if plugin.DeprecationMessage != "" {
				warnMsg = fmt.Sprintf("%s: %s", warnMsg, plugin.DeprecationMessage)
			}
			g.logger.Sugar().Warn(warnMsg)
		}
	}
	return result, nil
}

func (g *generator) execRemotePluginsV2(
	ctx context.Context,
	container app.EnvStdioContainer,
	image bufimage.Image,
	remote string,
	pluginConfigs []*curatedPluginExecArgs,
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
	pluginConfig CuratedPluginConfig,
) (*registryv1alpha1.PluginGenerationRequest, error) {
	var curatedPluginReference *registryv1alpha1.CuratedPluginReference
	if reference, err := bufpluginref.PluginReferenceForString(pluginConfig.Remote(), pluginConfig.Revision()); err == nil {
		curatedPluginReference = bufplugin.PluginReferenceToProtoCuratedPluginReference(reference)
	} else {
		// Try parsing as a plugin identity (no version information)
		identity, err := bufpluginref.PluginIdentityForString(pluginConfig.Remote())
		if err != nil {
			return nil, fmt.Errorf("invalid remote plugin %q", pluginConfig.Remote())
		}
		curatedPluginReference = bufplugin.PluginIdentityToProtoCuratedPluginReference(identity)
	}
	var options []string
	if len(pluginConfig.Opt()) > 0 {
		// Only include parameters if they're not empty.
		options = []string{pluginConfig.Opt()}
	}
	return &registryv1alpha1.PluginGenerationRequest{
		PluginReference: curatedPluginReference,
		Options:         options,
	}, nil
}

// getPluginReference returns the plugin reference and remote for the given plugin configuration.
func getPluginReference(pluginConfig LegacyRemotePluginConfig) (*registryv1alpha1.PluginReference, error) {
	_, owner, name, version, err := bufremoteplugin.ParsePluginVersionPath(pluginConfig.Remote())
	if err != nil {
		return nil, fmt.Errorf("invalid plugin path: %w", err)
	}
	var parameters []string
	if len(pluginConfig.Opt()) > 0 {
		// Only include parameters if they're not empty.
		parameters = []string{pluginConfig.Opt()}
	}
	return &registryv1alpha1.PluginReference{
		Owner:      owner,
		Name:       name,
		Version:    version,
		Parameters: parameters,
	}, nil
}

// validateResponses verifies that a response is set for each of the
// pluginConfigs, and that each generated file is generated by a single
// plugin.
func validateResponses(
	responses []*pluginpb.CodeGeneratorResponse,
	pluginConfigs []PluginConfig,
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
	baseOutDirPath        string
	includeImports        bool
	includeWellKnownTypes bool
	wasmEnabled           bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}
