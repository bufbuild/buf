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

package appprotoexec

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	wasmtime "github.com/bytecodealliance/wasmtime-go"
	"github.com/wasmerio/wasmer-go/wasmer"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type wasmHandler struct {
	logger     *zap.Logger
	engine     *wasmtime.Engine
	pluginPath string
}

func newWASMHandler(
	logger *zap.Logger,
	pluginPath string,
) *wasmHandler {
	// TODO: This should be threaded in from the app package
	// (like command.Runner), but this is simpler for now.
	engine := wastime.NewEngine()
	return &wasmHandler{
		logger:     logger.Named("appprotoexec"),
		engine:     engine,
		pluginPath: pluginPath,
	}
}

func (h *wasmHandler) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) error {
	ctx, span := trace.StartSpan(ctx, "wasm_plugin")
	span.AddAttributes(trace.StringAttribute("plugin", filepath.Base(h.pluginPath)))
	defer span.End()
	wasmBytes, err := ioutil.ReadFile(h.pluginPath)
	if err != nil {
		return err
	}
	module, err := wasmtime.NewModule(h.store, wasmBytes)
	if err != nil {
		return err
	}
	instance, err := wasmer.NewInstance(module, wasmer.NewImportObject())
	if err != nil {
		return err
	}
	run, err := instance.Exports.GetFunction("_start")
	if err != nil {
		return err
	}
	requestData, err := protoencoding.NewWireMarshaler().Marshal(request)
	if err != nil {
		return err
	}
	result, err := run(requestData)
	if err != nil {
		return err
	}
	resultBytes, ok := result.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte result, but got %T", resultBytes)
	}
	response := &pluginpb.CodeGeneratorResponse{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(resultBytes, response); err != nil {
		return err
	}
	response, err = normalizeCodeGeneratorResponse(response)
	if err != nil {
		return err
	}
	if response.GetSupportedFeatures()&uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) != 0 {
		responseWriter.SetFeatureProto3Optional()
	}
	for _, file := range response.File {
		if err := responseWriter.AddFile(file); err != nil {
			return err
		}
	}
	// plugin.proto specifies that only non-empty errors are considered errors.
	// This is also consistent with protoc's behaviour.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/069f989b483e63005f87ab309de130677718bbec/src/google/protobuf/compiler/plugin.proto#L100-L108.
	if response.GetError() != "" {
		responseWriter.AddError(response.GetError())
	}
	return nil
}
