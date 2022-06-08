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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	wasmtime "github.com/bytecodealliance/wasmtime-go"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
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
	engine := wasmtime.NewEngine()
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

	linker := wasmtime.NewLinker(h.engine)
	if err := linker.DefineWasi(); err != nil {
		return err
	}

	// TODO: Right now we're using the protojson format.
	// We'll need to update this to the binary form.
	stdinBlob, err := protojson.Marshal(request)
	if err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "out")
	if err != nil {
		return err
	}

	defer os.RemoveAll(dir)
	stdinPath := filepath.Join(dir, "stdin")
	stderrPath := filepath.Join(dir, "stderr")
	stdoutPath := filepath.Join(dir, "stdout")

	if err := os.WriteFile(stdinPath, stdinBlob, 0755); err != nil {
		return err
	}

	// Configure WASI imports to write stdout into a file.
	wasiConfig := wasmtime.NewWasiConfig()
	wasiConfig.SetStdinFile(stdinPath)
	wasiConfig.SetStdoutFile(stdoutPath)
	wasiConfig.SetStderrFile(stderrPath)

	store := wasmtime.NewStore(h.engine)
	store.SetWasi(wasiConfig)

	wasm, err := os.ReadFile(h.pluginPath)
	if err != nil {
		return err
	}

	module, err := wasmtime.NewModule(store.Engine, wasm)
	if err != nil {
		return err
	}

	instance, err := linker.Instantiate(store, module)
	if err != nil {
		return err
	}

	fn := instance.GetExport(store, "_start").Func()
	if _, err := fn.Call(store); err != nil {
		return err
	}

	stdoutBlob, err := os.ReadFile(stdoutPath)
	if err != nil {
		return err
	}

	response := &pluginpb.CodeGeneratorResponse{}
	if err := protoencoding.NewJSONUnmarshaler(nil).Unmarshal(stdoutBlob, response); err != nil {
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
