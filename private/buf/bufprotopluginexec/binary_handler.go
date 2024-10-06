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

package bufprotopluginexec

import (
	"bytes"
	"context"
	"io"
	"path/filepath"

	"log/slog"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/ioext"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/protoplugin"
	"google.golang.org/protobuf/types/pluginpb"
)

type binaryHandler struct {
	logger     *slog.Logger
	runner     command.Runner
	pluginPath string
	pluginArgs []string
}

func newBinaryHandler(
	logger *slog.Logger,
	runner command.Runner,
	pluginPath string,
	pluginArgs []string,
) *binaryHandler {
	return &binaryHandler{
		logger:     logger,
		runner:     runner,
		pluginPath: pluginPath,
		pluginArgs: pluginArgs,
	}
}

func (h *binaryHandler) Handle(
	ctx context.Context,
	pluginEnv protoplugin.PluginEnv,
	responseWriter protoplugin.ResponseWriter,
	request protoplugin.Request,
) (retErr error) {
	defer slogext.DebugProfile(h.logger, slog.String("plugin", filepath.Base(h.pluginPath)))()

	requestData, err := protoencoding.NewWireMarshaler().Marshal(request.CodeGeneratorRequest())
	if err != nil {
		return err
	}
	responseBuffer := bytes.NewBuffer(nil)
	stderrWriteCloser := newStderrWriteCloser(pluginEnv.Stderr, h.pluginPath)
	runOptions := []command.RunOption{
		command.RunWithEnviron(pluginEnv.Environ),
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
		return err
	}
	response := &pluginpb.CodeGeneratorResponse{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(responseBuffer.Bytes(), response); err != nil {
		return err
	}
	responseWriter.AddCodeGeneratorResponseFiles(response.GetFile()...)
	responseWriter.AddError(response.GetError())
	responseWriter.SetSupportedFeatures(response.GetSupportedFeatures())
	responseWriter.SetMinimumEdition(response.GetMinimumEdition())
	responseWriter.SetMaximumEdition(response.GetMaximumEdition())
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
		return ioext.NopWriteCloser(delegate)
	}
}
