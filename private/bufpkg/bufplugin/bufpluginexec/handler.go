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

	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
	lintv1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/plugin/lint/v1beta1"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/slicesext"
)

type handler struct {
	runner     command.Runner
	pluginPath string
	pluginArgs []string
}

func newHandler(
	runner command.Runner,
	pluginPath string,
	pluginArgs []string,
) *handler {
	return &handler{
		runner:     runner,
		pluginPath: pluginPath,
		pluginArgs: pluginArgs,
	}
}

func (h *handler) Handle(
	ctx context.Context,
	env bufplugin.Env,
	responseWriter bufplugin.ResponseWriter,
	request bufplugin.Request,
) (retErr error) {
	protoRequestData, err := protoencoding.NewWireMarshaler().Marshal(request.ProtoRequest())
	if err != nil {
		return err
	}
	protoResponseBuffer := bytes.NewBuffer(nil)
	if err := h.runner.Run(
		ctx,
		h.pluginPath,
		command.RunWithStdin(bytes.NewReader(protoRequestData)),
		command.RunWithStdout(protoResponseBuffer),
		command.RunWithStderr(env.Stderr),
		command.RunWithArgs(h.pluginArgs...),
	); err != nil {
		return err
	}
	protoResponse := &lintv1beta1.Response{}
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(protoResponseBuffer.Bytes(), protoResponse); err != nil {
		return err
	}
	if errMessage := protoResponse.GetError(); errMessage != "" {
		return errors.New(errMessage)
	}
	responseWriter.AddAnnotations(slicesext.Map(protoResponse.GetAnnotations(), bufplugin.NewAnnotation)...)
	return nil
}
