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
	"bytes"
	"context"
	"io"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/ioextended"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/pluginpb"
)

type binaryHandler struct {
	runner     command.Runner
	pluginPath string
	tracer     trace.Tracer
	pluginArgs []string
}

func newBinaryHandler(
	runner command.Runner,
	pluginPath string,
	pluginArgs []string,
) *binaryHandler {
	return &binaryHandler{
		runner:     runner,
		pluginPath: pluginPath,
		tracer:     otel.GetTracerProvider().Tracer("bufbuild/buf"),
		pluginArgs: pluginArgs,
	}
}

func (h *binaryHandler) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) error {
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
	responseBuffer := bytes.NewBuffer(nil)
	stderrWriteCloser := newStderrWriteCloser(container.Stderr(), h.pluginPath)
	runOptions := []command.RunOption{
		command.RunWithEnv(app.EnvironMap(container)),
		command.RunWithStdin(bytes.NewReader(requestData)),
		command.RunWithStdout(responseBuffer),
		command.RunWithStderr(stderrWriteCloser),
	}
	if len(h.pluginArgs) > 0 {
		runOptions = append(runOptions, command.RunWithArgs(h.pluginArgs...))
	}
	if err := h.runner.Run(
		ctx,
		h.pluginPath,
		runOptions...,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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

func newStderrWriteCloser(delegate io.Writer, pluginPath string) io.WriteCloser {
	switch filepath.Base(pluginPath) {
	case "protoc-gen-swift":
		// https://github.com/bufbuild/buf/issues/1736
		// Swallowing specific stderr message for protoc-gen-swift as protoc-gen-swift, see issue.
		// This is all disgusting code but it's simple and it works.
		// We did not document if pluginPath is normalized or not, so
		return newProtocGenSwiftStderrWriteCloser(delegate)
	default:
		return ioextended.NopWriteCloser(delegate)
	}
}
