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

package bufpluginexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufwasm"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/types/pluginpb"
)

type wasmHandler struct {
	wasmPluginExecutor bufwasm.PluginExecutor
	pluginPath         string
	tracer             trace.Tracer
}

func newWasmHandler(
	wasmPluginExecutor bufwasm.PluginExecutor,
	pluginPath string,
) (*wasmHandler, error) {
	if pluginAbsPath, err := validateWASMFilePath(pluginPath); err != nil {
		return nil, err
	} else {
		return &wasmHandler{
			wasmPluginExecutor: wasmPluginExecutor,
			pluginPath:         pluginAbsPath,
			tracer:             otel.GetTracerProvider().Tracer("bufbuild/buf"),
		}, nil
	}
}

func (h *wasmHandler) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) (retErr error) {
	ctx, span := h.tracer.Start(ctx, "plugin_proxy", trace.WithAttributes(
		attribute.Key("plugin").String(filepath.Base(h.pluginPath)),
	))
	defer span.End()
	requestData, err := protoencoding.NewWireMarshaler().Marshal(request)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	pluginBytes, err := os.ReadFile(h.pluginPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	compiledPlugin, err := h.wasmPluginExecutor.CompilePlugin(ctx, pluginBytes)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, compiledPlugin.Close())
	}()

	responseBuffer := bytes.NewBuffer(nil)
	if err := h.wasmPluginExecutor.Run(
		ctx,
		compiledPlugin,
		// command.RunWithEnv(app.EnvironMap(container)), // TODO, not exposed right now
		bytes.NewReader(requestData),
		responseBuffer,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if pluginErr := new(bufwasm.PluginExecutionError); errors.As(err, &pluginErr) {
			_, _ = container.Stderr().Write([]byte(pluginErr.Stderr))
		}
		return err
	}
	response := &pluginpb.CodeGeneratorResponse{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(responseBuffer.Bytes(), response); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	response, err = normalizeCodeGeneratorResponse(response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if response.GetSupportedFeatures()&uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) != 0 {
		responseWriter.SetFeatureProto3Optional()
	}
	for _, file := range response.File {
		if err := responseWriter.AddFile(file); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	// plugin.proto specifies that only non-empty errors are considered errors.
	// This is also consistent with protoc's behavior.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/069f989b483e63005f87ab309de130677718bbec/src/google/protobuf/compiler/plugin.proto#L100-L108.
	if response.GetError() != "" {
		responseWriter.AddError(response.GetError())
	}
	return nil
}

func validateWASMFilePath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return path, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return path, err
	}
	if !info.Mode().IsRegular() || !strings.HasSuffix(path, ".wasm") {
		return path, fmt.Errorf("invalid WASM file: %s", path)
	}
	return path, nil
}
