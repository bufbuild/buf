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

package bufpluginexec

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type generator struct {
	logger             *zap.Logger
	storageosProvider  storageos.Provider
	runner             command.Runner
	wasmPluginExecutor bufwasm.PluginExecutor
}

func newGenerator(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	runner command.Runner,
	wasmPluginExecutor bufwasm.PluginExecutor,
) *generator {
	return &generator{
		logger:             logger,
		storageosProvider:  storageosProvider,
		runner:             runner,
		wasmPluginExecutor: wasmPluginExecutor,
	}
}

func (g *generator) Generate(
	ctx context.Context,
	container app.EnvStderrContainer,
	pluginName string,
	requests []*pluginpb.CodeGeneratorRequest,
	options ...GenerateOption,
) (*pluginpb.CodeGeneratorResponse, error) {
	generateOptions := newGenerateOptions()
	for _, option := range options {
		option(generateOptions)
	}
	handlerOptions := []HandlerOption{
		HandlerWithPluginPath(generateOptions.pluginPath...),
		HandlerWithProtocPath(generateOptions.protocPath),
	}
	if generateOptions.wasmEnabled {
		handlerOptions = append(
			handlerOptions,
			HandlerWithWASMEnabled(),
		)
	}
	handler, err := NewHandler(
		g.storageosProvider,
		g.runner,
		g.wasmPluginExecutor,
		pluginName,
		handlerOptions...,
	)
	if err != nil {
		return nil, err
	}
	return appproto.NewGenerator(
		g.logger,
		handler,
	).Generate(
		ctx,
		container,
		requests,
	)
}

func (g *generator) GenerateWithBinary(
	ctx context.Context,
	container app.EnvStderrContainer,
	pluginPath string,
	pluginArgs []string,
	requests []*pluginpb.CodeGeneratorRequest,
) (*pluginpb.CodeGeneratorResponse, error) {
	handler, err := NewBinaryHandler(
		g.runner,
		pluginPath,
		pluginArgs,
	)
	if err != nil {
		return nil, err
	}
	return appproto.NewGenerator(
		g.logger,
		handler,
	).Generate(
		ctx,
		container,
		requests,
	)
}

func (g *generator) GenerateWithProtocBuiltin(
	ctx context.Context,
	container app.EnvStderrContainer,
	pluginName string,
	requests []*pluginpb.CodeGeneratorRequest,
	options ...GenerateWithProtocBuiltinOption,
) (*pluginpb.CodeGeneratorResponse, error) {
	generateWithProtocBuiltinOptions := newGenerateWithProtocBuiltinOptions()
	for _, option := range options {
		option(generateWithProtocBuiltinOptions)
	}
	handler, err := NewProtocProxyHandler(
		g.storageosProvider,
		g.runner,
		generateWithProtocBuiltinOptions.protocPath,
		pluginName,
	)
	if err != nil {
		return nil, err
	}
	return appproto.NewGenerator(
		g.logger,
		handler,
	).Generate(
		ctx,
		container,
		requests,
	)
}

type generateOptions struct {
	pluginPath  []string
	protocPath  string
	wasmEnabled bool
}

func newGenerateOptions() *generateOptions {
	return &generateOptions{}
}

type generateWithProtocBuiltinOptions struct {
	protocPath string
}

func newGenerateWithProtocBuiltinOptions() *generateWithProtocBuiltinOptions {
	return &generateWithProtocBuiltinOptions{}
}
