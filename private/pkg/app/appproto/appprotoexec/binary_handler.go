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

package appprotoexec

import (
	"bytes"
	"context"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/pluginpb"
)

type binaryHandler struct {
	logger     *zap.Logger
	runner     command.Runner
	pluginPath string
}

func newBinaryHandler(
	logger *zap.Logger,
	runner command.Runner,
	pluginPath string,
) *binaryHandler {
	return &binaryHandler{
		logger:     logger.Named("appprotoexec"),
		runner:     runner,
		pluginPath: pluginPath,
	}
}

func (h *binaryHandler) Handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) error {
	ctx, span := trace.StartSpan(ctx, "plugin_proxy")
	span.AddAttributes(trace.StringAttribute("plugin", filepath.Base(h.pluginPath)))
	defer span.End()
	requestData, err := protoencoding.NewWireMarshaler().Marshal(request)
	if err != nil {
		return err
	}
	responseBuffer := bytes.NewBuffer(nil)
	// https://github.com/bufbuild/buf/issues/1736
	// Swallowing specific stderr message for protoc-gen-swift as protoc-gen-swift, see issue.
	// This is all disgusting code but its simple and it works.
	// We did not document if pluginPath is normalized or not, so
	isProtocGenSwift := filepath.Base(h.pluginPath) == "protoc-gen-swift"
	// protocGenSwiftStderrBuffer will be non-nil if isProtocGenSwift is true
	var protocGenSwiftStderrBuffer *bytes.Buffer
	// stderr is what we pass to Run regardless
	stderr := container.Stderr()
	if isProtocGenSwift {
		// If protoc-gen-swift, we want to capture all the stderr so we can process it.
		// Otherwise, we write stderr directly to the container.Stderr() as it is produced.
		protocGenSwiftStderrBuffer = bytes.NewBuffer(nil)
		stderr = protocGenSwiftStderrBuffer
	}
	if err := h.runner.Run(
		ctx,
		h.pluginPath,
		command.RunWithEnv(app.EnvironMap(container)),
		command.RunWithStdin(bytes.NewReader(requestData)),
		command.RunWithStdout(responseBuffer),
		command.RunWithStderr(stderr),
	); err != nil {
		// TODO: strip binary path as well?
		return handlePotentialTooManyFilesError(err)
	}
	if isProtocGenSwift {
		// If we had any stderr, then let's process it and print it.
		// protocGenSwiftStderrBuffer will always be non-nil if isProtocGenSwift is true
		if stderrData := protocGenSwiftStderrBuffer.Bytes(); len(stderrData) > 0 {
			// Just being extra careful to not initiate a Write call if we have len == 0, even though
			// in almost all io.Writer cases, this should have no side-effect (and this may even be
			// the documented behavior of io.Writer).
			if newStderrData := bytes.ReplaceAll(
				stderrData,
				// If swift-protobuf changes their error message, this may not longer filter properly
				// but this is OK - this filtering should be treated as non-critical.
				// https://github.com/apple/swift-protobuf/blob/c3d060478fcf1f564be0a3876bde8c04247793ae/Sources/protoc-gen-swift/main.swift#L244
				//
				// Note that our heuristic as to whether this is protoc-gen-swift or not for isProtocGenSwift
				// is that the binary is named protoc-gen-swift, and buf/protoc will print the binary name
				// before any message to stderr, so given our protoc-gen-swift heuristic, this is the
				// error message that will be printed.
				//
				// Tested manually on Mac.
				// TODO: Test manually on Windows.
				[]byte("protoc-gen-swift: WARNING: unknown version of protoc, use 3.2.x or later to ensure JSON support is correct.\n"),
				nil,
			); len(newStderrData) > 0 {
				if _, err := container.Stderr().Write(newStderrData); err != nil {
					return err
				}
			}
		}
	}
	response := &pluginpb.CodeGeneratorResponse{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(responseBuffer.Bytes(), response); err != nil {
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
	// This is also consistent with protoc's behavior.
	// Ref: https://github.com/protocolbuffers/protobuf/blob/069f989b483e63005f87ab309de130677718bbec/src/google/protobuf/compiler/plugin.proto#L100-L108.
	if response.GetError() != "" {
		responseWriter.AddError(response.GetError())
	}
	return nil
}
